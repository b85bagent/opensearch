package opensearch

import (
	"encoding/json"
	"log"
	"regexp"
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
	var timestamp string
	if ts, ok := data["time"].(string); ok { // check for LokiReceiver
		timestamp = ts
	} else if ts, ok := data["ts"].(string); ok { // check for LokiReceiver
		timestamp = ts
	} else if ts, ok := data["timestamp"].(string); ok { // check for LokiReceiver
		timestamp = ts
	} else if timeReceived, ok := data["time_received_ns"].(int64); ok {
		// If timestamp does not exist, but time_received_ns does, convert it to RFC3339 format
		timestamp = time.Unix(0, timeReceived).Format("2006-01-02T15:04:05.000Z")
	}

	// If neither timestamp nor time_received_ns exists, use the current time as the timestamp
	if timestamp == "" {
		t := time.Now().In(time.FixedZone("UTC+0", 0))
		timestamp = t.Format("2006-01-02T15:04:05.000Z")
	}

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

	index = IndexRegexp(index)

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

	index = IndexRegexp(index)

	result, err := BulkCreate(index, data)
	if err != nil {
		log.Println("Bulk Create error: ", err)
		return ""
	}

	return result
}

func IndexRegexp(index string) string {

	pattern := `YYYY(([-./]))?(MM)?(M)?(([-./]))?(DD)?(D)?`

	// 編譯正則表達式
	re, _ := regexp.Compile(pattern)

	// 檢查這個參數是否符合正則表達式的模式
	matches := re.FindStringSubmatch(index)

	if len(matches) > 0 {
		// 根據匹配的字符串決定需要的日期格式
		var layout strings.Builder

		// 用於年份的格式
		layout.WriteString("2006")

		// 如果有匹配到月份
		if matches[3] != "" || matches[4] != "" {
			separator := matches[2] //獲取分隔符
			//如果分隔符是 / 把他替換掉
			if separator == "/" {
				separator = "-"
			}

			if matches[3] != "" {
				layout.WriteString(separator + "01")
			} else {
				layout.WriteString(separator + "1")
			}
		}

		// 如果有匹配到日期
		if matches[7] != "" || matches[8] != "" {
			separator := matches[6] //獲取分隔符
			//如果分隔符是 / 把他替換掉
			if separator == "/" {
				separator = "-"
			}

			if matches[7] != "" {
				layout.WriteString(separator + "02")
			} else {
				layout.WriteString(separator + "2")
			}
		}

		currentDate := time.Now().Format(layout.String())

		// 將匹配的部分替換為當前的日期
		index = re.ReplaceAllString(index, currentDate)
	}

	// 輸出結果

	return index
}
