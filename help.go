package opensearch

import (
	"encoding/json"
	"time"
)

type opensearchM struct {
	timestamp time.Time `json:"@timestamp"`
	source    any       `json:"_source"`
}

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
