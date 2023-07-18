package opensearch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/opensearch-project/opensearch-go"
	"github.com/opensearch-project/opensearch-go/opensearchapi"
	"github.com/tidwall/gjson"
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
		log.Println("failed to create index ", errCreateIndex)
		return errCreateIndex
	}

	defer createIndexResponse.Body.Close()

	log.Println(createIndexResponse)
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
		log.Println("failed to insert document: ", err)
		return err
	}
	defer insertResponse.Body.Close()

	log.Println(insertResponse)
	log.Println("add success")
	return nil
}

//單一刪除
func SingleDeleteIndex(client *opensearch.Client, index []string) error {
	deleteIndex := opensearchapi.IndicesDeleteRequest{
		Index: index,
	}

	deleteIndexResponse, errDeleteIndex := deleteIndex.Do(context.Background(), client.Transport)
	if errDeleteIndex != nil {
		log.Println("failed to delete index ", errDeleteIndex)
		return errDeleteIndex
	}
	defer deleteIndexResponse.Body.Close()

	log.Println(deleteIndexResponse)

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

	// log.Println(res.StatusCode)
	if res.StatusCode > 299 {
		var errorResult SearchErrorResponse
		json.NewDecoder(res.Body).Decode(&errorResult)
		return result, errors.New(errorResult.Error.Reason)
	}

	// log.Printf("search response: [%+v]", res)

	json.NewDecoder(res.Body).Decode(&result)

	return result, nil
}

func BulkPrevious(client *opensearch.Client, mode string, data BulkPreviousUse) (result *opensearchapi.Response, err error) {

	switch mode {
	case "create":
		if data.Create.Data == nil || data.Create.Index == "" {
			errCreateData := errors.New("create issue is illegal,please check it ")
			return nil, errCreateData
		}
		createData, errCreate := BulkCreate(data.Create.Index, data.Create.Data)
		if errCreate != nil {
			return nil, errCreate
		}

		result, errExecute := BulkExecute(client, createData)
		if errExecute != nil {
			return nil, errExecute
		}

		return result, nil

	case "update":
		if data.Update.Id == "" || data.Update.Index == "" || len(data.Update.Data.Data) == 0 {
			errUpdateData := errors.New("update issue is illegal,please check it ")
			return nil, errUpdateData
		}
		updateData, errUpdate := BulkUpdate(data.Update.Index, data.Update.Id, data.Update.Data)
		if errUpdate != nil {
			return nil, errUpdate
		}

		result, errExecute := BulkExecute(client, updateData)
		if errExecute != nil {
			return nil, errExecute
		}

		return result, nil

	case "delete":
		if len(data.Delete) == 0 {
			errDeleteData := errors.New("delete issue is illegal,please check it")
			return nil, errDeleteData
		}
		deleteData, errDelete := BulkDelete(data.Delete)
		if errDelete != nil {
			return nil, errDelete
		}

		result, errExecute := BulkExecute(client, deleteData)
		if errExecute != nil {
			return nil, errExecute
		}

		return result, nil

	default:
		return nil, errors.New("invalid mode")
	}

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

func removeMapKey(c InsertData) (r string) {
	dataBytes, _ := json.Marshal(c)

	var data map[string]interface{}

	if err := json.Unmarshal(dataBytes, &data); err != nil {
		log.Println("JSON 解析錯誤：", err)
		return
	}

	dataValue, ok := data["data"]
	if !ok {
		log.Println("未找到 'data' key")
		return
	}

	r1, err := json.Marshal(dataValue)
	if err != nil {
		log.Println("JSON 編碼錯誤：", err)
		return
	}

	var data2 map[string]interface{}

	if err := json.Unmarshal(r1, &data2); err != nil {
		log.Println("JSON 解析錯誤：", err)
		return
	}

	data2["@timestamp"] = c.Timestamp

	result, err := json.Marshal(data2)
	if err != nil {
		log.Println("JSON 編碼錯誤：", err)
		return
	}

	return string(result)

}

func removeMapKeyRemoteWrite(c InsertData) (r string, err error) {

	defer func() {
		if r := recover(); r != nil {

			log.Println("panic_InsertData: ", string(c.Data), c.Timestamp)

			fmt.Println("Recovered from panic in processInsert, ", r)
		}
	}()

	dataBytes, _ := json.Marshal(c)

	var data map[string]interface{}

	if errUnmarshal := json.Unmarshal(dataBytes, &data); errUnmarshal != nil {
		log.Println("removeMapKeyRemoteWrite Unmarshal error: ", errUnmarshal)

		return "", errUnmarshal
	}

	dataValue, ok := data["data"]
	if !ok {
		return "", errors.New("未找到 'data' key")
	}

	r1, errMarshal := json.Marshal(dataValue)
	if errMarshal != nil {
		log.Println("removeMapKeyRemoteWrite Marshal error: ", errMarshal)

		return "", errMarshal
	}

	var data2 map[string]interface{}

	if errUnmarshal2 := json.Unmarshal(r1, &data2); errUnmarshal2 != nil {
		log.Println("removeMapKeyRemoteWrite Unmarshal2 error: ", errUnmarshal2)
		return "", errUnmarshal2
	}

	location, errLocation := time.LoadLocation("UTC")
	if errLocation != nil {
		log.Println("removeMapKeyRemoteWrite errLocation error: ", errLocation)
		return "", errLocation
	}

	existTimeStamp := gjson.Get(string(dataBytes), "data.samples.0.timestamp")
	if existTimeStamp.Exists() {
		timestampMillis := existTimeStamp.Int() // 獲取時間戳（毫秒）
		timestampSeconds := timestampMillis / 1000
		timestamp := time.Unix(timestampSeconds, 0).In(location)
		formattedTimestamp := timestamp.Format("2006-01-02T15:04:05.000Z")
		data2["@timestamp"] = formattedTimestamp
	} else {
		data2["@timestamp"] = c.Timestamp
	}

	result, errMarshal2 := json.Marshal(data2)
	if errMarshal2 != nil {
		log.Println("removeMapKeyRemoteWrite Marshal2 error: ", errMarshal2)

		return "", errMarshal2
	}

	// log.Println("data result: ", string(result))

	return string(result), nil

}

func BulkCreate(index string, data map[string]interface{}) (result string, err error) {

	r := []interface{}{}

	Action := actionCreate(index)

	ContentDetail1 := contentDetailCreate(data)

	c := removeMapKey(ContentDetail1)
	// log.Println("c: ", c)

	r = dataMix(r, Action, c)
	// log.Println("r: ", r)

	buf := &bytes.Buffer{}

	for _, v := range r {
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			log.Println("JSON 編碼錯誤：", err)
			return "", err
		}
		// 移除反斜線
		jsonString := strings.Replace(string(jsonBytes), "\\", "", -1)
		// 移除前後的雙引號
		jsonString = strings.TrimPrefix(jsonString, `"`)
		jsonString = strings.TrimSuffix(jsonString, `"`)

		buf.WriteString(jsonString)
		buf.WriteByte('\n')
	}

	// log.Println("buf.String(): ", buf.String())

	return buf.String(), nil

}

func BulkCreateRemoteWrite(index string, data map[string]interface{}) (result string, err error) {

	if len(data) == 0 {
		log.Println("BulkCreateRemoteWrite data 0")

		return "", errors.New("missing data format")
	}

	r := []interface{}{}

	Action := actionCreate(index)

	ContentDetail1 := contentDetailCreate(data)

	c, err := removeMapKeyRemoteWrite(ContentDetail1)
	if err != nil {
		return "", err
	}
	// log.Println("c: ", c)

	r = dataMix(r, Action, c)
	// log.Println("r: ", r)

	buf := &bytes.Buffer{}

	for _, v := range r {
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			log.Println("JSON 編碼錯誤：", err)
			return "", err
		}
		// 移除反斜線
		jsonString := strings.Replace(string(jsonBytes), "\\", "", -1)
		// 移除前後的雙引號
		jsonString = strings.TrimPrefix(jsonString, `"`)
		jsonString = strings.TrimSuffix(jsonString, `"`)

		buf.WriteString(jsonString)
		buf.WriteByte('\n')
	}

	// log.Println("buf.String(): ", buf.String())

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

	return buf.String(), nil

}

//Bulk Execute
func BulkExecute(client *opensearch.Client, documents string) (result *opensearchapi.Response, err error) {

	// log.Println("documents: ", documents)
	blk, errBulk := client.Bulk(strings.NewReader(documents))
	if errBulk != nil {
		log.Println("failed to perform bulk operations", errBulk)
		return nil, errBulk
	}

	if blk.IsError() {
		var errBulk BulkError

		json.NewDecoder(blk.Body).Decode(&errBulk)

		errBody := errors.New(errBulk.Error.Reason)
		return nil, errBody
	}

	return blk, nil
}
