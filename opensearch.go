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

	data2 := make(map[string]interface{})

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

			return
		}
	}()

	dataValue = nil
	data2 = nil
	data = nil
	r1 = nil

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
		e               error
		errType, reason string
	)

	gjson.Get(string(body), "items.#.create.error.reason").ForEach(func(_, value gjson.Result) bool {
		if value.String() != "" {
			errType = gjson.Get(string(body), "items.#.create.error.type").String()
			reason = gjson.Get(string(body), "items.#.create.error.reason").String()
			e = errors.New("Insert error: " + errType + ": " + reason)
		}
		return true // keep iterating
	})

	if e != nil {
		return e
	}

	return nil
}
