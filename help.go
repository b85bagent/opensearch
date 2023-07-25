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

//把data 轉成 字串
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

	pattern := `%\{YYYY([-.\/]MM([-.\/]DD)?)?\}`

	// 編譯正則表達式
	re, _ := regexp.Compile(pattern)

	// 檢查這個參數是否符合正則表達式的模式
	matches := re.FindStringSubmatch(index)

	if len(matches) > 0 {
		// 基於匹配的字符串決定需要的日期格式
		var layout string
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

		// 獲取日期格式
		if hasSeparator {
			switch strings.Count(datePart, separator) {
			case 1:
				layout = "2006" + separator + "01"
			case 2:
				layout = "2006" + separator + "01" + separator + "02"
			}
		} else {
			layout = "2006"
		}

		// 獲取當前的日期
		currentDate := time.Now().Format(layout)

		// 將匹配的部分替換為當天的日期
		index = re.ReplaceAllString(index, currentDate)
	}

	return index
}
