package opensearch

import (
	"encoding/json"
	"time"
)

type SearchResponse struct {
	Took     int  `json:"took"`
	TimedOut bool `json:"timed_out"`
	Shards   struct {
		Total      int `json:"total"`
		Successful int `json:"successful"`
		Skipped    int `json:"skipped"`
		Failed     int `json:"failed"`
	} `json:"_shards"`
	Hits struct {
		Total struct {
			Value    int    `json:"value"`
			Relation string `json:"relation"`
		} `json:"total"`
		MaxScore float64 `json:"max_score"`
		Hits     []struct {
			Index  string  `json:"_index"`
			ID     string  `json:"_id"`
			Score  float64 `json:"_score"`
			Source struct {
				Data ContentDetail
			} `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

type SearchErrorResponse struct {
	Error struct {
		RootCause []struct {
			Type         string `json:"type"`
			Reason       string `json:"reason"`
			Index        string `json:"index"`
			ResourceID   string `json:"resource.id"`
			ResourceType string `json:"resource.type"`
			IndexUUID    string `json:"index_uuid"`
		} `json:"root_cause"`
		Type         string `json:"type"`
		Reason       string `json:"reason"`
		Index        string `json:"index"`
		ResourceID   string `json:"resource.id"`
		ResourceType string `json:"resource.type"`
		IndexUUID    string `json:"index_uuid"`
	} `json:"error"`
	Status int `json:"status"`
}

type BulkError struct {
	Error struct {
		RootCause []struct {
			Type   string `json:"type"`
			Reason string `json:"reason"`
		} `json:"root_cause"`
		Type   string `json:"type"`
		Reason string `json:"reason"`
	} `json:"error"`
	Status int `json:"status"`
}

type BulkCreateResponse struct {
	Took   int  `json:"took"`
	Errors bool `json:"errors"`
	Items  []struct {
		Create struct {
			Index   string `json:"_index"`
			Id      string `json:"_id"`
			Version int    `json:"_version"`
			Result  string `json:"result"`
			Shards  struct {
				Total      int `json:"total"`
				Successful int `json:"successful"`
				Failed     int `json:"failed"`
			} `json:"_shards"`
			SeqNo       int `json:"_seq_no"`
			PrimaryTerm int `json:"_primary_term"`
			Status      int `json:"status"`
		} `json:"create,omitempty"`
	} `json:"items"`
}

// ---bulk Use ---
type ActionCreate struct {
	Create IndexDetail `json:"create,omitempty"`
}

type ActionDelete struct {
	Delete IndexAndIDDetail `json:"delete,omitempty"`
}

type ActionUpdate struct {
	Update IndexAndIDDetail `json:"update,omitempty"`
}

type IndexAndIDDetail struct {
	Index string `json:"_index"`
	Id    string `json:"_id"`
}

type IndexDetail struct {
	Index string `json:"_index"`
}

//若有其他content需求 改這邊
type ContentDetail struct {
	Host string `json:"host"`
	HTTP struct {
		Method  string `json:"method"`
		Request int    `json:"request"`
		Version string `json:"version"`
	} `json:"http"`
	URL struct {
		Domain string `json:"domain"`
		Path   string `json:"path"`
		Port   int    `json:"port"`
	} `json:"url"`
	Timestamp time.Time `json:"timestamp"`
}

type InsertData struct {
	Timestamp string          `json:"@timestamp"`
	Data      json.RawMessage `json:"data"`
}

type UpdateData struct {
	Doc InsertData `json:"doc"`
}

type BulkPreviousUse struct {
	Delete map[string]string
	Create struct {
		Index string
		Data  map[string]interface{}
	}
	Update struct {
		Index, Id string
		Data      InsertData
	}
}
