package opensearch

import "time"

type opensearchM struct {
	timestamp time.Time `json:"@timestamp"`
	source    any       `json:"_source"`
}

func dataMix(data []interface{}, Action any, ContentDetail any) []interface{} {
	data = append(data, Action)

	r := opensearchM{}
	r.source = ContentDetail
	r.timestamp = time.Now()

	data = append(data, r)

	return data
}

func actionCreate(index string) ActionCreate {
	return ActionCreate{Create: IndexDetail{Index: index}}
}

func contentDetailCreate(data map[string]interface{}) InsertData {
	return InsertData{Data: data}
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
