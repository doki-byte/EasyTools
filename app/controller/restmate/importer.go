package restmate

import (
	"EasyTools/app/controller/proxy"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/goccy/go-json"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func (a *RestMate) ImportCollection() (resp JSResp) {
	s, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "选择文件",
	})
	if err != nil || s == "" {
		resp.Msg = "错误！无法选择文件"
		return
	}
	j := strings.HasSuffix(strings.ToLower(s), ".json")
	if !j {
		resp.Msg = "错误！所选文件不是json"
		return
	}
	f, err := os.ReadFile(s)
	if err != nil {
		resp.Msg = "错误！读取文件失败"
		return
	}
	type checkinfo struct {
		Info struct {
			Schema string `json:"schema"`
			Name   string `json:"name"`
		} `json:"info"`
	}
	var checkin checkinfo
	err = json.Unmarshal(f, &checkin)
	if err != nil {
		// 可能是 Swagger/OpenAPI 文档，尝试解析
		return a.importSwagger(f, s)
	}
	var pmSchema = "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
	var rmSchema = "https://schema.restmate.com/json/collection/collection.json"

	restFile, err := os.ReadFile(a.db)
	if err != nil {
		resp.Msg = "错误！读取文件失败"
		return
	}
	var c []Collection
	newcols := false
	err = json.Unmarshal(restFile, &c)
	if err != nil {
		resp.Msg = "错误！读取文件失败"
		return
	}
	if checkin.Info.Schema == pmSchema {
		var pm PMCollection
		err = json.Unmarshal(f, &pm)
		if err != nil {
			resp.Msg = "错误！读取文件失败"
			return
		}
		if pm.Info.Name == "" {
			resp.Msg = "错误！集合文件无效"
			return
		}
		var col Collection
		col.Name = pm.Info.Name
		col_id, err := gonanoid.New()
		if err != nil {
			resp.Msg = "错误！读取文件失败"
			return
		}
		col.ID = col_id
		PMRecursion(&pm.Item, &col.Requests, col_id)
		c = append(c, col)
		newcols = true
	} else if checkin.Info.Schema == rmSchema {
		var expCol ExportCollection
		err = json.Unmarshal(f, &expCol)
		if err != nil {
			resp.Msg = "错误！读取文件失败"
			return
		}
		if expCol.Collection.ID == "" || expCol.Collection.Name == "" {
			resp.Msg = "错误！读取文件失败"
			return
		}
		col_id, err := gonanoid.New()
		if err != nil {
			resp.Msg = "错误！读取文件失败"
			return
		}
		expCol.Collection.ID = col_id
		//new IDs for requests
		for i := range expCol.Collection.Requests {
			if expCol.Collection.Requests[i].ID == "" || expCol.Collection.Requests[i].Name == "" {
				continue
			}
			req_id, err := gonanoid.New()
			if err != nil {
				continue
			}
			expCol.Collection.Requests[i].ID = req_id
			expCol.Collection.Requests[i].Method = validateMethod(expCol.Collection.Requests[i].Method)
			expCol.Collection.Requests[i].CollId = col_id
		}
		c = append(c, expCol.Collection)
		newcols = true
	} else {
		// 尝试作为 Swagger/OpenAPI 文档解析
		return a.importSwagger(f, s)
	}
	if newcols {
		b, err := json.Marshal(c)
		if err != nil {
			resp.Msg = "错误！无法导入集合"
			return
		}
		err = os.WriteFile(a.db, b, 0644)
		if err != nil {
			resp.Msg = "错误！无法导入集合"
			return
		}
		collRspSlice := makeCollRsp(&c)
		resp.Success = true
		resp.Msg = "集合导入成功"
		resp.Data = collRspSlice
		return
	}
	resp.Msg = "错误！无法导入集合"
	return
}

func (a *RestMate) importSwagger(f []byte, filePath string) (resp JSResp) {
	return a.importSwaggerWithName(f, filePath, "")
}

func (a *RestMate) importSwaggerWithName(f []byte, filePath string, collectionName string) (resp JSResp) {
	restFile, err := os.ReadFile(a.db)
	if err != nil {
		resp.Msg = "错误！读取文件失败"
		return
	}
	var c []Collection
	err = json.Unmarshal(restFile, &c)
	if err != nil {
		resp.Msg = "错误！读取文件失败"
		return
	}

	// 尝试解析 OpenAPI 3.0
	var openAPI3 struct {
		OpenAPI string `json:"openapi"`
		Info    struct {
			Title   string `json:"title"`
			Version string `json:"version"`
		} `json:"info"`
		Servers []struct {
			URL string `json:"url"`
		} `json:"servers"`
		Paths map[string]map[string]interface{} `json:"paths"`
	}

	err = json.Unmarshal(f, &openAPI3)
	if err == nil && openAPI3.OpenAPI != "" {
		col_id, err := gonanoid.New()
		if err != nil {
			resp.Msg = "错误！生成ID失败"
			return
		}
		colName := openAPI3.Info.Title + " (OpenAPI " + openAPI3.OpenAPI + ")"
		if collectionName != "" {
			colName = collectionName
		}
		col := Collection{
			ID:       col_id,
			Name:     colName,
			Requests: []Request{},
		}

		for path, methods := range openAPI3.Paths {
			for method, op := range methods {
				method = strings.ToLower(method)
				if method != "get" && method != "post" && method != "put" && method != "delete" {
					continue
				}
				req_id, err := gonanoid.New()
				if err != nil {
					continue
				}
				opMap, ok := op.(map[string]interface{})
				if !ok {
					continue
				}
				summary, _ := opMap["summary"].(string)
				operationId, _ := opMap["operationId"].(string)
				name := summary
				if name == "" {
					name = operationId
				}
				if name == "" {
					name = strings.ToUpper(method) + " " + path
				}

				// 构建URL，使用{{baseurl}}占位符
				urlPath := strings.TrimPrefix(path, "/")
				finalURL := "{{baseurl}}/" + urlPath

				req := Request{
					ID:      req_id,
					Name:    name,
					Method:  validateMethod(method),
					Url:     finalURL,
					CollId:  col_id,
					Headers: []KeyValue{},
					Params:  []KeyValue{},
					Body: Body{
						BodyType: "json",
						BodyRaw:  "{}",
						FormData: []FormData{},
					},
				}

				// 解析参数
				if parameters, ok := opMap["parameters"].([]interface{}); ok {
					for _, param := range parameters {
						paramMap, ok := param.(map[string]interface{})
						if !ok {
							continue
						}
						paramName, _ := paramMap["name"].(string)
						paramIn, _ := paramMap["in"].(string)
						paramRequired, _ := paramMap["required"].(bool)
						if paramName == "" {
							continue
						}

						paramID, err := gonanoid.New()
						if err != nil {
							continue
						}

						// 提取参数值：优先使用example，然后是default，最后是schema生成的示例值
						paramValue := ""
						if example, ok := paramMap["example"]; ok {
							paramValue = fmt.Sprintf("%v", example)
						} else if defaultValue, ok := paramMap["default"]; ok {
							paramValue = fmt.Sprintf("%v", defaultValue)
						} else if schema, ok := paramMap["schema"].(map[string]interface{}); ok {
							paramValue = generateExampleValueFromSchema(schema)
						}

						kv := KeyValue{
							ID:     paramID,
							Key:    paramName,
							Value:  paramValue,
							Active: paramRequired,
						}

						if paramIn == "query" {
							req.Params = append(req.Params, kv)
						} else if paramIn == "header" {
							req.Headers = append(req.Headers, kv)
						} else if paramIn == "path" {
							// Path参数需要替换到URL中，这里先添加到Params中，后续可以处理
							req.Params = append(req.Params, kv)
						}
					}
				}

				// 解析请求体
				if requestBody, ok := opMap["requestBody"].(map[string]interface{}); ok {
					if content, ok := requestBody["content"].(map[string]interface{}); ok {
						if jsonContent, ok := content["application/json"].(map[string]interface{}); ok {
							if schema, ok := jsonContent["schema"].(map[string]interface{}); ok {
								exampleBody := generateExampleFromSchema(schema)
								req.Body.BodyRaw = exampleBody
							}
						}
					}
				}

				col.Requests = append(col.Requests, req)
			}
		}

		c = append(c, col)
		b, err := json.Marshal(c)
		if err != nil {
			resp.Msg = "错误！无法导入集合"
			return
		}
		err = os.WriteFile(a.db, b, 0644)
		if err != nil {
			resp.Msg = "错误！无法导入集合"
			return
		}
		collRspSlice := makeCollRsp(&c)
		resp.Success = true
		resp.Msg = "Swagger/OpenAPI 文档导入成功"
		resp.Data = collRspSlice
		return
	}

	// 尝试解析 Swagger 2.0
	var swagger2 struct {
		Swagger string `json:"swagger"`
		Info    struct {
			Title   string `json:"title"`
			Version string `json:"version"`
		} `json:"info"`
		Host     string                            `json:"host"`
		BasePath string                            `json:"basePath"`
		Schemes  []string                          `json:"schemes"`
		Paths    map[string]map[string]interface{} `json:"paths"`
	}

	err = json.Unmarshal(f, &swagger2)
	if err == nil && swagger2.Swagger != "" {
		col_id, err := gonanoid.New()
		if err != nil {
			resp.Msg = "错误！生成ID失败"
			return
		}
		colName := swagger2.Info.Title + " (Swagger " + swagger2.Swagger + ")"
		if collectionName != "" {
			colName = collectionName
		}
		col := Collection{
			ID:       col_id,
			Name:     colName,
			Requests: []Request{},
		}

		for path, methods := range swagger2.Paths {
			for method, op := range methods {
				method = strings.ToLower(method)
				if method != "get" && method != "post" && method != "put" && method != "delete" {
					continue
				}
				req_id, err := gonanoid.New()
				if err != nil {
					continue
				}
				opMap, ok := op.(map[string]interface{})
				if !ok {
					continue
				}
				summary, _ := opMap["summary"].(string)
				operationId, _ := opMap["operationId"].(string)
				name := summary
				if name == "" {
					name = operationId
				}
				if name == "" {
					name = strings.ToUpper(method) + " " + path
				}

				// 构建URL，使用{{baseurl}}占位符
				urlPath := strings.TrimPrefix(path, "/")
				finalURL := "{{baseurl}}/" + urlPath

				req := Request{
					ID:      req_id,
					Name:    name,
					Method:  validateMethod(method),
					Url:     finalURL,
					CollId:  col_id,
					Headers: []KeyValue{},
					Params:  []KeyValue{},
					Body: Body{
						BodyType: "json",
						BodyRaw:  "{}",
						FormData: []FormData{},
					},
				}

				// 解析参数
				if parameters, ok := opMap["parameters"].([]interface{}); ok {
					for _, param := range parameters {
						paramMap, ok := param.(map[string]interface{})
						if !ok {
							continue
						}
						paramName, _ := paramMap["name"].(string)
						paramIn, _ := paramMap["in"].(string)
						paramRequired, _ := paramMap["required"].(bool)
						if paramName == "" {
							continue
						}

						paramID, err := gonanoid.New()
						if err != nil {
							continue
						}

						// 提取参数值：优先使用example，然后是default，最后是schema生成的示例值
						paramValue := ""
						if example, ok := paramMap["example"]; ok {
							paramValue = fmt.Sprintf("%v", example)
						} else if defaultValue, ok := paramMap["default"]; ok {
							paramValue = fmt.Sprintf("%v", defaultValue)
						} else if schema, ok := paramMap["schema"].(map[string]interface{}); ok {
							paramValue = generateExampleValueFromSchema(schema)
						}

						kv := KeyValue{
							ID:     paramID,
							Key:    paramName,
							Value:  paramValue,
							Active: paramRequired,
						}

						if paramIn == "query" {
							req.Params = append(req.Params, kv)
						} else if paramIn == "header" {
							req.Headers = append(req.Headers, kv)
						} else if paramIn == "path" {
							// Path参数需要替换到URL中，这里先添加到Params中，后续可以处理
							req.Params = append(req.Params, kv)
						}
					}
				}

				col.Requests = append(col.Requests, req)
			}
		}

		c = append(c, col)
		b, err := json.Marshal(c)
		if err != nil {
			resp.Msg = "错误！无法导入集合"
			return
		}
		err = os.WriteFile(a.db, b, 0644)
		if err != nil {
			resp.Msg = "错误！无法导入集合"
			return
		}
		collRspSlice := makeCollRsp(&c)
		resp.Success = true
		resp.Msg = "Swagger/OpenAPI 文档导入成功"
		resp.Data = collRspSlice
		return
	}

	resp.Msg = "错误！无法识别文件格式，请确保是 Postman 集合、Restmate 集合或 Swagger/OpenAPI 文档"
	return
}

func (a *RestMate) ImportSwaggerFromUrl(url string) (resp JSResp) {
	if url == "" {
		resp.Msg = "错误！URL不能为空"
		return
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		resp.Msg = "错误！URL必须以http://或https://开头"
		return
	}

	// 创建HTTP客户端，使用全局代理
	client := proxy.GlobalProxyManager.GetHTTPClient()

	// 发送请求获取Swagger文档
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		resp.Msg = "错误！创建请求失败: " + err.Error()
		return
	}
	req.Header.Set("User-Agent", "EasyTools/1.8.7")
	req.Header.Set("Accept", "application/json, application/yaml, */*")

	response, err := client.Do(req)
	if err != nil {
		resp.Msg = "错误！请求失败: " + err.Error()
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		resp.Msg = "错误！服务器返回状态码: " + fmt.Sprintf("%d", response.StatusCode)
		return
	}

	// 读取响应内容
	f, err := io.ReadAll(response.Body)
	if err != nil {
		resp.Msg = "错误！读取响应失败: " + err.Error()
		return
	}

	// 调用importSwagger方法解析
	return a.importSwaggerWithName(f, url, "")
}

func (a *RestMate) ImportSwaggerFromUrlWithName(url string, collectionName string) (resp JSResp) {
	if url == "" {
		resp.Msg = "错误！URL不能为空"
		return
	}
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		resp.Msg = "错误！URL必须以http://或https://开头"
		return
	}

	// 创建HTTP客户端，使用全局代理
	client := proxy.GlobalProxyManager.GetHTTPClient()

	// 发送请求获取Swagger文档
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		resp.Msg = "错误！创建请求失败: " + err.Error()
		return
	}
	req.Header.Set("User-Agent", "EasyTools/1.8.7")
	req.Header.Set("Accept", "application/json, application/yaml, */*")

	response, err := client.Do(req)
	if err != nil {
		resp.Msg = "错误！请求失败: " + err.Error()
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		resp.Msg = "错误！服务器返回状态码: " + fmt.Sprintf("%d", response.StatusCode)
		return
	}

	// 读取响应内容
	f, err := io.ReadAll(response.Body)
	if err != nil {
		resp.Msg = "错误！读取响应失败: " + err.Error()
		return
	}

	// 调用importSwaggerWithName方法解析
	return a.importSwaggerWithName(f, url, collectionName)
}

func generateExampleValueFromSchema(schema map[string]interface{}) string {
	// 提取单个值的示例
	if example, ok := schema["example"]; ok {
		return fmt.Sprintf("%v", example)
	}
	if defaultValue, ok := schema["default"]; ok {
		return fmt.Sprintf("%v", defaultValue)
	}
	typ, _ := schema["type"].(string)
	switch typ {
	case "string":
		return "string"
	case "integer", "number":
		return "0"
	case "boolean":
		return "true"
	case "array":
		return "[]"
	case "object":
		return "{}"
	default:
		return ""
	}
}

func generateExampleFromSchema(schema map[string]interface{}) string {
	typ, _ := schema["type"].(string)
	if typ == "object" {
		result := make(map[string]interface{})
		if properties, ok := schema["properties"].(map[string]interface{}); ok {
			for key, prop := range properties {
				propMap, ok := prop.(map[string]interface{})
				if !ok {
					continue
				}
				propType, _ := propMap["type"].(string)
				switch propType {
				case "string":
					result[key] = ""
				case "number", "integer":
					result[key] = 0
				case "boolean":
					result[key] = false
				case "array":
					result[key] = []interface{}{}
				case "object":
					result[key] = map[string]interface{}{}
				default:
					result[key] = nil
				}
			}
		}
		jsonBytes, _ := json.MarshalIndent(result, "", "  ")
		return string(jsonBytes)
	}
	return "{}"
}
func PMRecursion(items *[]Item, reqs *[]Request, coll_id string) {
	for i := range *items {
		itm := (*items)[i]
		if len(itm.Item) > 0 {
			PMRecursion(&itm.Item, reqs, coll_id)
			continue
		}
		if itm.Request.Method == "" {
			continue
		}
		var rq Request
		req_id, err := gonanoid.New()
		if err != nil {
			continue
		}
		rq.ID = req_id
		rq.Name = itm.Name
		rq.Method = validateMethod(itm.Request.Method)
		rq.Url = itm.Request.URL.Raw
		rq.CollId = coll_id
		for h := range itm.Request.Header {
			hl := itm.Request.Header[h]
			if hl.Key == "" {
				continue
			}
			hid, err := gonanoid.New()
			if err != nil {
				continue
			}
			var myhead KeyValue
			myhead.ID = hid
			myhead.Active = true
			myhead.Key = hl.Key
			myhead.Value = hl.Value
			rq.Headers = append(rq.Headers, myhead)
		}
		if itm.Request.Body.Mode == "formdata" {
			rq.Body.BodyType = "formdata"
			for f := range itm.Request.Body.FormData {
				fdl := itm.Request.Body.FormData[f]
				if fdl.Type != "text" {
					continue
				}
				var formDataSet FormData
				nid, err := gonanoid.New()
				if err != nil {
					continue
				}
				formDataSet.ID = nid
				formDataSet.Key = fdl.Key
				formDataSet.Type = "text"
				formDataSet.Value = fdl.Value
				formDataSet.Active = true
				rq.Body.FormData = append(rq.Body.FormData, formDataSet)
			}
		} else if itm.Request.Body.Mode == "raw" {
			rq.Body.BodyType = "json"
		} else {
			rq.Body.BodyType = "none"
		}
		rq.Body.BodyRaw = itm.Request.Body.Raw
		*reqs = append(*reqs, rq)
	}
}
func validateMethod(str string) string {
	str = strings.ToLower(str)
	switch str {
	case "get", "post", "put", "delete", "patch", "head", "options":
		return str
	default:
		return "get"
	}
}
