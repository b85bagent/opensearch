package opensearch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/opensearch-project/opensearch-go"
	"github.com/opensearch-project/opensearch-go/opensearchapi"
	"github.com/tidwall/gjson"
)

var (
	dataInsertMutex sync.Mutex
)

func removeMapKey(c InsertData) (r string) {
	c.Data["@timestamp"] = c.Timestamp

	result, err := json.Marshal(c.Data)
	if err != nil {
		log.Println("JSON 編碼錯誤：", err)
		return
	}

	return string(result)

}

func flattenData(data map[string]interface{}) map[string]interface{} {
	if innerData, ok := data["data"].(map[string]interface{}); ok {
		for k, v := range innerData {
			data[k] = v
		}
		delete(data, "data")
	}
	return data
}

func removeMapKeyRemoteWrite(data map[string]interface{}) (r string, err error) {
	data2 := make(map[string]interface{})

	dataValue, ok := data["data"]
	if !ok {
		return "", errors.New("未找到 'data' key")
	}

	r1, errMarshal := json.Marshal(dataValue)
	if errMarshal != nil {
		log.Println("removeMapKeyRemoteWrite Marshal error: ", errMarshal)
		return "", errMarshal
	}

	if errUnmarshal2 := json.Unmarshal(r1, &data2); errUnmarshal2 != nil {
		log.Println("removeMapKeyRemoteWrite Unmarshal2 error: ", errUnmarshal2)
		return "", errUnmarshal2
	}

	location, errLocation := time.LoadLocation("UTC")
	if errLocation != nil {
		log.Println("removeMapKeyRemoteWrite errLocation error: ", errLocation)
		return "", errLocation
	}

	existTimeStamp := gjson.Get(string(r1), "samples.0.timestamp")
	if existTimeStamp.Exists() {
		timestampMillis := existTimeStamp.Int()
		timestampSeconds := timestampMillis / 1000
		timestamp := time.Unix(timestampSeconds, 0).In(location)
		formattedTimestamp := timestamp.Format("2006-01-02T15:04:05.000Z")
		data2["@timestamp"] = formattedTimestamp
	}

	result, errMarshal2 := json.Marshal(data2)
	if errMarshal2 != nil {
		log.Println("removeMapKeyRemoteWrite Marshal2 error: ", errMarshal2)
		return "", errMarshal2
	}

	defer func() {
		if r := recover(); r != nil {
			log.Println("panic_InsertData2: ", data2)
			log.Println("panic_InsertDataValue: ", dataValue)
			fmt.Println("Recovered from panic in processInsert, ", r)
		}
	}()

	return string(result), nil
}

func BulkCreate(index string, data map[string]interface{}) (result string, err error) {

	preprocessData(data)

	data = flattenData(data) // 调用flattenData函数来修改数据结构

	r := []interface{}{}

	Action := actionCreate(index)
	r = append(r, Action)

	ContentDetail1 := contentDetailCreate(data)
	// ContentDetail1 已經是一個準備好的 map[string]interface{}，所以不再需要 removeMapKey
	r = append(r, ContentDetail1)

	buf := &bytes.Buffer{}
	for _, v := range r {
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			log.Println("JSON 編碼錯誤：", err)
			return "", err
		}
		buf.Write(jsonBytes)
		buf.WriteByte('\n')
	}

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

// Bulk Execute
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

func BulkInsert(data string, client *opensearch.Client) error {
	dataInsertMutex.Lock()
	defer dataInsertMutex.Unlock()

	result, err := BulkExecute(client, data)
	if err != nil {
		log.Println("BulkExecute error: ", err)
		return err
	}

	defer result.Body.Close()

	body, err := io.ReadAll(result.Body)
	if err != nil {
		log.Println("result.IsError ReadAll error: ", err)

		return err
	}

	var (
		e                        error
		errType, reason, Creason string
	)

	gjson.Get(string(body), "items.#.create.error.reason").ForEach(func(_, value gjson.Result) bool {
		if value.String() != "" {

			errType = gjson.Get(string(body), "items.#.create.error.type").String()
			reason = gjson.Get(string(body), "items.#.create.error.reason").String()
			Creason = gjson.Get(string(body), "items.#.create.error.caused_by.reason").String()
			e = errors.New("Insert error: " + errType + "_" + reason + ": " + Creason)
		}
		fmt.Println("error result: ", string(body))
		return true // keep iterating
	})

	if e != nil {
		return e
	}

	return nil
}
