package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/opensearch-project/opensearch-go"
	"github.com/opensearch-project/opensearch-go/opensearchapi"
)

//建立Index
func CreateIndex(client *opensearch.Client, IndexName string) error {
	//設定Index
	settings := strings.NewReader(`{
		"settings": {
	    	"index": {
	        	"number_of_shards": 1,
	        	"number_of_replicas": 2
	        }
	    },
		"mappings": {
			"properties": {
				"timestamp": {
					"type": "date"
				}
			}
		}
	}`)

	res := opensearchapi.IndicesCreateRequest{
		Index: IndexName,
		Body:  settings,
	}

	createIndexResponse, errCreateIndex := res.Do(context.Background(), client)
	if errCreateIndex != nil {
		fmt.Println("failed to create index ", errCreateIndex)
		return errCreateIndex
	}

	defer createIndexResponse.Body.Close()

	fmt.Println(createIndexResponse)
	return nil
}

//單一插入
func SingleInsert(client *opensearch.Client, index, document string) error {
	req := opensearchapi.IndexRequest{
		Index: index,
		Body:  strings.NewReader(document),
	}
	insertResponse, err := req.Do(context.Background(), client.Transport)
	if err != nil {
		fmt.Println("failed to insert document: ", err)
		return err
	}
	defer insertResponse.Body.Close()

	fmt.Println(insertResponse)
	fmt.Println("add success")
	return nil
}

//單一刪除
func SingleDeleteIndex(client *opensearch.Client, index []string) error {
	deleteIndex := opensearchapi.IndicesDeleteRequest{
		Index: index,
	}

	deleteIndexResponse, errDeleteIndex := deleteIndex.Do(context.Background(), client.Transport)
	if errDeleteIndex != nil {
		fmt.Println("failed to delete index ", errDeleteIndex)
		return errDeleteIndex
	}
	defer deleteIndexResponse.Body.Close()

	fmt.Println(deleteIndexResponse)

	return nil

}

//Search something
func Search(client *opensearch.Client, Index, key, value string) (result SearchResponse, err error) {

	var res *opensearchapi.Response

	if key != "" && value != "" {
		queryString := fmt.Sprintf(`%s: "%s"`, "*", value)

		res, err = client.Search(
			client.Search.WithIndex(Index),

			client.Search.WithQuery(queryString),
		)
		if err != nil {
			log.Printf("error Search: [%s]", err.Error())
		}
	} else {
		res, err = client.Search(
			client.Search.WithIndex(Index),
			client.Search.WithSize(5),
		)
		if err != nil {
			log.Printf("error Search: [%s]", err.Error())
		}
	}

	// fmt.Println(res.StatusCode)
	if res.StatusCode > 299 {
		var errorResult SearchErrorResponse
		json.NewDecoder(res.Body).Decode(&errorResult)
		return result, errors.New(errorResult.Error.Reason)
	}

	log.Printf("response: [%+v]", res)

	json.NewDecoder(res.Body).Decode(&result)

	return result, nil
}

func BulkPrevious(client *opensearch.Client, mode string, data BulkPreviousUse) error {

	switch mode {
	case "create":
		if data.Create.Data == nil || data.Create.Index == "" {
			errCreateData := errors.New("create issue is illegal,please check it ")
			return errCreateData
		}
		result, errCreate := BulkCreate(data.Create.Index, data.Create.Data)
		if errCreate != nil {
			return errCreate
		}

		if errExecute := BulkExecute(client, result); errExecute != nil {
			return errExecute
		}

	case "update":
		if data.Update.Id == "" || data.Update.Index == "" || len(data.Update.Data.Data) == 0 {
			errCreateData := errors.New("update issue is illegal,please check it ")
			return errCreateData
		}
		result, errCreate := BulkUpdate(data.Update.Index, data.Update.Id, data.Update.Data)
		if errCreate != nil {
			return errCreate
		}

		if errExecute := BulkExecute(client, result); errExecute != nil {
			return errExecute
		}
	case "delete":
		if len(data.Delete) == 0 {
			errCreateData := errors.New("delete issue is illegal,please check it")
			return errCreateData
		}
		result, errCreate := BulkDelete(data.Delete)
		if errCreate != nil {
			return errCreate
		}

		if errExecute := BulkExecute(client, result); errExecute != nil {
			return errExecute
		}
	default:
		return errors.New("invalid mode")
	}

	fmt.Println("BulkPrevious End")

	return nil

}

func BulkDelete(Delete map[string]string) (result string, err error) {

	r := []interface{}{}

	for key, value := range Delete {
		deleteIndex := actionDelete(key, value)
		r = append(r, deleteIndex)
	}

	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)

	for _, v := range r {
		if errEncode := enc.Encode(v); errEncode != nil {
			log.Fatal(err)
			return "", errEncode
		}
	}

	return buf.String(), nil

}

func BulkCreate(index string, data map[string]interface{}) (result string, err error) {

	r := []interface{}{}

	Action := actionCreate(index)

	ContentDetail1 := contentDetailCreate(data)

	r = dataMix(r, Action, ContentDetail1)

	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)

	for _, v := range r {
		if errEncode := enc.Encode(v); errEncode != nil {
			log.Fatal(err)
			return "", errEncode
		}
	}

	fmt.Println(buf.String())

	return buf.String(), nil

}

func BulkUpdate(index, id string, data InsertData) (result string, err error) {

	r := []interface{}{}

	Action := actionUpdate(index, id)

	ContentDetail1 := contentDetailUpdate(data)

	r = dataMix(r, Action, ContentDetail1)

	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)

	for _, v := range r {
		if errEncode := enc.Encode(v); errEncode != nil {
			log.Fatal(err)
			return "", errEncode
		}
	}

	fmt.Println(buf.String())

	return buf.String(), nil

}

//Bulk Execute
func BulkExecute(client *opensearch.Client, documents string) error {

	// fmt.Println("documents: ", documents)

	blk, errBulk := client.Bulk(strings.NewReader(documents))
	defer blk.Body.Close()

	if errBulk != nil {
		fmt.Println("failed to perform bulk operations", errBulk)
		return errBulk
	}
	defer blk.Body.Close()

	fmt.Println("Performing bulk operations")
	fmt.Println(blk)

	if blk.IsError() {
		var errBulk BulkError

		json.NewDecoder(blk.Body).Decode(&errBulk)

		errBody := errors.New(errBulk.Error.Reason)
		return errBody
	}

	body, errReadAll := io.ReadAll(blk.Body)
	if errReadAll != nil {
		log.Printf("error occurred: [%s]", errReadAll.Error())
		return errReadAll
	}

	var response BulkCreateResponse
	if errUnmarshal := json.Unmarshal(body, &response); errUnmarshal != nil {
		log.Printf("error Unmarshal blkResponse: [%s]", errUnmarshal.Error())
		return errUnmarshal
	}

	for _, item := range response.Items {
		if item.Create.Status > 299 {
			log.Printf("error occurred: [%s]", item.Create.Result)
		} else {
			log.Printf("success: [%s]", item.Create.Index)
		}
	}

	return nil
}
