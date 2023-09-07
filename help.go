package opensearch

import (
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func dataMix(data []interface{}, Action any, ContentDetail any) []interface{} {

	data = append(data, Action)
	data = append(data, ContentDetail)

	return data
}

func actionCreate(index string) ActionCreate {
	return ActionCreate{Create: IndexDetail{Index: index}}
}

func contentDetailCreate(data map[string]interface{}) InsertData {
	// 嘗試從data中提取time_received_ns
	if source, ok := data["_source"].(map[string]interface{}); ok {
		if timeReceived, ok := source["time_received_ns"].(int64); ok {
			// 轉換time_received_ns（納秒）為RFC3339格式並返回
			timestamp := time.Unix(0, timeReceived).Format("2006-01-02T15:04:05.000Z")
			dataBytes, _ := json.Marshal(data)
			return InsertData{Data: json.RawMessage(dataBytes), Timestamp: timestamp}
		}
	}

	// 如果time_received_ns不存在，則使用當前時間作為timestamp
	t := time.Now().In(time.FixedZone("UTC+0", 0))
	timestamp := t.Format("2006-01-02T15:04:05.000Z")
	dataBytes, _ := json.Marshal(data)
	return InsertData{Data: json.RawMessage(dataBytes), Timestamp: timestamp}
}

func actionDelete(index, id string) ActionDelete {
	return ActionDelete{Delete: IndexAndIDDetail{Index: index, Id: id}}
}

func actionUpdate(index, id string) ActionUpdate {
	return ActionUpdate{Update: IndexAndIDDetail{Index: index, Id: id}}
}

func contentDetailUpdate(data InsertData) UpdateData {
	return UpdateData{Doc: data}
}

// 把data 轉成 字串
func DataCompressionRemoteWrite(data map[string]interface{}, index string) string {

	if len(data) == 0 {
		return ""
	}

	index = indexRegexp(index)

	result, err := BulkCreateRemoteWrite(index, data)
	if err != nil {
		log.Println("Bulk Create error: ", err)
		return ""
	}

	return result
}

func DataCompression(data map[string]interface{}, index string) string {

	if len(data) == 0 {
		return ""
	}

	index = indexRegexp(index)

	result, err := BulkCreate(index, data)
	if err != nil {
		log.Println("Bulk Create error: ", err)
		return ""
	}

	return result
}

func indexRegexp(index string) string {
	pattern := `%\{YYYY([-.\/]M{1,2}([-.\/]D{1,2})?)?\}`

	// 編籍正則表達式
	re, _ := regexp.Compile(pattern)

	// 檢查這個參數是否符合正則表達式的模式
	matches := re.FindStringSubmatch(index)

	if len(matches) > 0 {
		// 基於匹配的字符串決定需要的日期格式
		var separator string
		hasSeparator := false

		// 獲取日期部分和分隔符
		datePart := strings.Trim(matches[0], "%{}")
		if strings.Contains(datePart, "-") {
			separator = "-"
			hasSeparator = true
		} else if strings.Contains(datePart, ".") {
			separator = "."
			hasSeparator = true
		} else if strings.Contains(datePart, "/") {
			separator = "-"
			hasSeparator = true
		}

		// 獲取當前的日期
		currentDate := time.Now()

		// 獲取日期格式
		var year, month, day string
		if hasSeparator {
			year = strconv.Itoa(currentDate.Year())

			if strings.Count(datePart, "M") == 1 {
				month = strconv.Itoa(int(currentDate.Month()))
			} else {
				month = fmt.Sprintf("%02d", currentDate.Month())
			}

			if strings.Count(datePart, "D") == 1 {
				day = strconv.Itoa(currentDate.Day())
			} else {
				day = fmt.Sprintf("%02d", currentDate.Day())
			}
		} else {
			year = strconv.Itoa(currentDate.Year())
		}

		// 根據日期部分的格式，組合出最終的日期
		if strings.Contains(datePart, "D") {
			index = re.ReplaceAllString(index, year+separator+month+separator+day)
		} else if strings.Contains(datePart, "M") {
			index = re.ReplaceAllString(index, year+separator+month)
		} else {
			index = re.ReplaceAllString(index, year)
		}
	}

	return index
}
