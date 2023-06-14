# Opensearch Api Usage

## Using

```bash
go get -u github.com/b85bagent/opensearch@版本號
go get -u github.com/b85bagent/opensearch@c478316
```

## Create

### Request

#### Params

| Name  | Type  | Necessary |
| :------------: |:---------------:|:-----:|
| data.Create.Data  | map[string]interface{} | O |
| data.Create.Index | String                 | O |

#### code

```go
var data model.BulkPrevious
 data.Create.Data = map[string]interface{}{
  "host": "10.42.11.258",
  "http": map[string]interface{}{
   "method":  "GET",
   "request": 3369,
   "version": "HTTP/1.1",
  },
  "url": map[string]interface{}{
   "domain": "10.42.11.255",
   "path":   "/",
   "port":   9090,
  },
  "timestamp": time.Now(),
 }
 data.Create.Index = osIndex
 if err := opensearch.BulkPrevious("create", data); err != nil {
  log.Println(err)
 }
```

### Response

- 正確響應

```bash
[200 OK] {"took":37,"errors":false,"items":[{"create":{"_index":"lex-test66","_id":"EJazs4gBM-XHgcOmDej3","_version":1,"result":"created","_shards":{"total":2,"successful":2,"failed":0},"_seq_no":2,"_primary_term":1,"status":201}}]}
```

- 錯誤響應
  - 沒有帶參數

```bash
create issue is illegal,please check it
```

## Update

### Request

#### Params

| Name  | Type  | Necessary |
| :------------: |:---------------:|:-----:|
| data.Update.Data  | model.InsertData | O |
| data.Update.Id    | String           | O |
| data.Update.Index | String           | O |

#### code 

```go
var data model.BulkPrevious
 data.Update.Data = model.InsertData{
  Data: map[string]interface{}{"host": "10.40.192.277"},
 }
 data.Update.Index = osIndex
 data.Update.Id = "DZags4gBM-XHgcOmtuhm"

 if err := opensearch.BulkPrevious("update", data); err != nil {
  log.Println(err)
 }
```

### Response

- 正確響應

```bash
[200 OK] {"took":21,"errors":false,"items":[{"update":{"_index":"lex-test66","_id":"DZags4gBM-XHgcOmtuhm","_version":2,"result":"updated","_shards":{"total":2,"successful":2,"failed":0},"_seq_no":1,"_primary_term":1,"status":200}}]}
```

- 錯誤響應
  - 沒有帶參數

```bash
update issue is illegal,please check it
```

## Delete

### Request

#### Params

| Name  | Type  | Necessary |
| :------------: |:---------------:|:-----:|
| data.Delete  | map[string]string | O |

> map內 key 為 **index** / value 為 **id**

#### code

```go
var data model.BulkPrevious
 data.Delete = map[string]string{
  "lex-test66": "EJazs4gBM-XHgcOmDej3",
 }

 if err := opensearch.BulkPrevious("delete", data); err != nil {
  log.Println(err)
 }
```

### Response

- 正確響應

  - 參數正確，且找的到相對應的index && id

    ```bash
    [200 OK] {"took":28,"errors":false,"items":[{"delete":{"_index":"lex-test66","_id":"EJazs4gBM-XHgcOmDej3",      "_version":2,"result":"deleted","_shards":{"total":2,"successful":2,"failed":0},"_seq_no":4,"_primary_term":1,      "status":200}}]}
    ```

    > result 為 "deleted"

  - 參數正確，但index && id 找不到(兩者要完全匹配才能正確刪除)

    ```bash
    [200 OK] {"took":5,"errors":false,"items":[{"delete":{"_index":"lex-test66","_id":"EJazs4gBM-XHgcOmDej3",       "_version":1,"result":"not_found","_shards":{"total":2,"successful":2,"failed":0},"_seq_no":5,"_primary_term":1,    "status":404}}]}
    ```

    > result 為 "not_found"

- 錯誤響應
  - 沒有帶參數

```bash
delete issue is illegal,please check it
```

## 用錯mode

- example

### Code

```go

var data model.BulkPrevious
 data.Delete = map[string]string{
  "lex-test66": "EJazs4gBM-XHgcOmDej3",
 }
// 要刪除，但是用到create 模式
 if err := opensearch.BulkPrevious("create", data); err != nil {
  log.Println(err)
 }
```

### Response

```bash
 create issue is illegal,please check it 
```

## Search

### Request

- func Search(client *opensearch.Client, Index string, key string, value string)

| Name  | Type  | Necessary |
| :------------: |:---------------:|:-----:|
| client  | *opensearch.Client | O |
| Index  | string | O |
| key  | string | X |
| value  | string | X |

> key && value 非必帶，不帶就會回 搜尋的index 的全部資料 (不帶key&value 請用空字串)
> key 可以帶 \* 會把所有key都帶出，或是 data.url.\* 等方式
> **key 有帶 value就一定要帶**

#### code

```go
os, ok := server.GetServerInstance().GetOpensearch()["One"]
 if !ok {
  fmt.Println("No OK")
  return
 }
 //Search key
 result, err := opensearch.Search(os, "lex-test66")
 if err != nil {
  fmt.Println("No OK")
  return
 }

 log.Println(result.Hits.Hits[0].Source)
```

### Response

- 正確響應

```bash
[[200 OK] {"took":1,"timed_out":false,"_shards":{"total":1,"successful":1,"skipped":0,"failed":0},"hits":{"total":{"value":1,"relation":"eq"},"max_score":1.0,"hits":[{"_index":"lex-test","_id":"75b7sogBM-XHgcOm7ufE","_score":1.0,"_source":{"data":{"host":"10.40.192.213","http":{"method":"POST","request":1669,"version":"HTTP/1.1"},"timestamp":"2023-06-13T12:20:38.5735803+08:00","url":{"domain":"10.11.233.11","path":"/","port":8080}}}}]}}]
```


