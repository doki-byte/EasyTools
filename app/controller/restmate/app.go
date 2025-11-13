package restmate

import (
	"EasyTools/app/controller/proxy"
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/goccy/go-json"
	cookiejar "github.com/juju/persistent-cookiejar"
	"github.com/matoous/go-nanoid/v2"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type RestMate struct {
	ctx        context.Context
	db         string
	env        string
	settings   string
	jarFile    string
	requestCtx context.CancelFunc
}

func NewRestMate() *RestMate {
	return &RestMate{}
}

func (a *RestMate) InitRestMateDbFile() {
	a.InitFile("restmate_db.json", "db")
	a.InitFile("restmate_env.json", "env")
	a.InitFile("restmate_jar.json", "jar")
}

func (a *RestMate) Startup(ctx context.Context) {
	a.ctx = ctx
	a.InitRestMateDbFile()
	runtime.EventsOn(ctx, "cancelRequest", func(_ ...interface{}) {
		if a.requestCtx != nil {
			a.requestCtx()
			a.requestCtx = nil
		}
	})
}

func (a *RestMate) InvokeRequest(r Request) (resp JSResp) {
	if a.requestCtx != nil {
		a.requestCtx()
		a.requestCtx = nil
	}
	if r.Url == "" {
		resp.Msg = "Error! URL cannot be empty"
		return
	}
	ctx, cancel := context.WithTimeout(a.ctx, 300*time.Second)
	a.requestCtx = cancel
	defer func() {
		cancel()
		a.requestCtx = nil
	}()

	vars := make(map[string]string)
	f, err := os.ReadFile(a.env)
	if err != nil {
		resp.Msg = "Error! Failed to read env file"
		return
	}
	var e []Env
	err = json.Unmarshal(f, &e)
	if err != nil {
		resp.Msg = "Error! Failed to unmarshal Env file"
		return
	}
	for i := range e {
		if e[i].Selected == true {
			vars = e[i].Variable
			break
		}
	}
	interpolateVars := func(v string) string {
		if len(vars) > 0 {
			re := regexp.MustCompile(`\{\{([\w.-]+)\}\}`)
			o := re.ReplaceAllStringFunc(v, func(m string) string {
				key := re.FindStringSubmatch(m)[1]
				if val, ok := vars[key]; ok {
					return val
				}
				return m
			})
			return o

		} else {
			return v
		}
	}
	var rgxURL = interpolateVars(r.Url)
	if !strings.HasPrefix(rgxURL, "http://") && !strings.HasPrefix(rgxURL, "https://") {
		rgxURL = "https://" + rgxURL
	}
	u, err := url.Parse(rgxURL)
	if err != nil {
		resp.Msg = "Error! cannot parse URL"
		return
	}
	var body io.Reader
	autoHeaders := make(http.Header)
	p := u.Query()
	for i := range r.Params {
		if strings.TrimSpace(r.Params[i].Key) != "" {
			var val = interpolateVars(r.Params[i].Value)
			p.Add(r.Params[i].Key, val)
		}
	}
	u.RawQuery = p.Encode()
	method := parseMethod(r.Method)
	if method != "GET" && method != "DELETE" {
		if r.Body.BodyType == "formdata" {
			var b bytes.Buffer
			writer := multipart.NewWriter(&b)
			for i := range r.Body.FormData {
				fd := r.Body.FormData[i]
				if fd.Type != "file" && fd.Key != "" {
					var val = interpolateVars(fd.Value)
					err := writer.WriteField(fd.Key, val)
					if err != nil {
						resp.Msg = "Error! Failed to write field formdata"
						return
					}
				}
				if fd.Type == "file" && fd.Key != "" {
					for f := range fd.Files {
						file, err := os.Open(fd.Files[f])
						if err != nil {
							resp.Msg = "Error! Cannot open file"
							return
						}
						defer file.Close()
						part, err := writer.CreateFormFile(fd.Key, filepath.Base(fd.Files[f]))
						if err != nil {
							resp.Msg = "Error! Failed to write image in formdata"
							return
						}
						if _, err := io.Copy(part, file); err != nil {
							resp.Msg = "Error! Failed to copy file content"
							return
						}
					}
				}
			}
			writer.Close()
			body = &b
			autoHeaders.Set("Content-Type", writer.FormDataContentType())
		} else if r.Body.BodyType == "json" {
			body = strings.NewReader(interpolateVars(r.Body.BodyRaw))
			autoHeaders.Set("Content-Type", "application/json")
		} else {
			body = nil
		}
	} else {
		body = nil
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		resp.Msg = "Error! Request initiation failed"
		return
	}
	req.Header.Set("User-Agent", "EasyTools/1.8.7")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Connection", "close")
	for k, values := range autoHeaders {
		for _, v := range values {
			req.Header.Add(k, v)
		}
	}
	for i := range r.Headers {
		if strings.Contains(r.Headers[i].Key, " ") {
			continue
		}
		if strings.TrimSpace(r.Headers[i].Key) != "" {
			var val = interpolateVars(r.Headers[i].Value)
			req.Header.Set(r.Headers[i].Key, val)
		}
	}

	jar, err := cookiejar.New(&cookiejar.Options{Filename: a.jarFile})
	if err != nil {
		resp.Msg = "Error! Failed to read cookies"
		return
	}
	var result Result
	client := proxy.GlobalProxyManager.GetHTTPClient()
	cli := &http.Client{
		Jar:           jar,                  // 使用自定义 Cookie Jar
		Transport:     client.Transport,     // 使用代理客户端的 Transport
		Timeout:       client.Timeout,       // 使用代理客户端的超时设置
		CheckRedirect: client.CheckRedirect, // 继承重定向策略
	}
	startTime := time.Now()
	response, err := cli.Do(req)
	if err != nil {
		result.ErrorContent = err.Error()
		resp.Success = true
		resp.Msg = "Error! Request action failed"
		resp.Data = result
		return
	}
	d := time.Since(startTime).Milliseconds()
	duration := fmt.Sprintf("%d", d)
	jar.SetCookies(u, response.Cookies())
	err = jar.Save()
	if err != nil {
		resp.Msg = "Error! Failed to read cookies"
		return
	}
	defer response.Body.Close()
	result.StatusCode = response.StatusCode
	result.HttpStatus = response.Status
	result.Headers = parseResponseHeaders(&response.Header)
	result.Duration = duration

	buf, err := io.ReadAll(response.Body)
	if err != nil {
		resp.Msg = "Error! Failed to read response body"
		return
	}
	ct := strings.ToLower(strings.Split(response.Header.Get("Content-Type"), ";")[0])
	switch {
	case strings.Contains(ct, "application/json"):
		result.BodyContent = string(buf)
		result.ContentType = "JSON"
	case strings.Contains(ct, "text/html"):
		result.BodyContent = string(buf)
		result.ContentType = "HTML"
	case strings.Contains(ct, "text/plain"):
		result.BodyContent = string(buf)
		result.ContentType = "TEXT"
	case strings.HasPrefix(ct, "image/"):
		result.BodyContent = "data:" + ct + ";base64," + base64.StdEncoding.EncodeToString(buf)
		result.ContentType = "IMAGE"
	case strings.Contains(ct, "application/pdf"):
		result.BodyContent = "data:" + ct + ";base64," + base64.StdEncoding.EncodeToString(buf)
		result.ContentType = "PDF"
	default:
		result.BodyContent = string(buf)
		result.ContentType = "TEXT"
	}
	resp.Success = true
	resp.Msg = "Request was successful"
	resp.Data = result
	return
}

func (a *RestMate) ChoseFile() (resp JSResp) {
	s, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Choose files",
	})
	if err != nil || len(s) <= 0 {
		resp.Msg = "Error! Cannot Chose file"
		return
	}
	resp.Success = true
	resp.Msg = "Files selected successfully!"
	resp.Data = s
	return
}

func (a *RestMate) ExportCollection(id string) (resp JSResp) {
	if id == "" {
		resp.Msg = "错误！无法导出集合"
		return
	}
	m, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "保存集合",
		DefaultFilename: "export_collection.json",
	})
	if err != nil {
		resp.Msg = "错误！无法导出集合"
		return
	}
	f, err := os.ReadFile(a.db)
	if err != nil {
		resp.Msg = "错误！无法导出集合"
		return
	}
	var c []Collection
	err = json.Unmarshal(f, &c)
	if err != nil {
		resp.Msg = "错误！无法导出集合"
		return
	}
	for i := range c {
		if c[i].ID == id {
			var exp ExportCollection
			exp.Info.Schema = "https://schema.restmate.com/json/collection/collection.json"
			exp.Info.Name = "Restmate"
			exp.Collection = c[i]
			b, err := json.Marshal(exp)
			if err != nil {
				resp.Msg = "错误！无法导出集合"
				return
			}
			err = os.WriteFile(m, b, 0644)
			if err != nil {
				resp.Msg = "错误！无法导出集合"
				return
			}
			resp.Success = true
			resp.Msg = "集合导入成功！"
			return
		}
	}
	resp.Msg = "错误！无法导出集合"
	return
}

func (a *RestMate) MoveRequest(id, new_id, coll_id, new_coll_id string) (resp JSResp) {
	if id == "" || coll_id == "" || new_coll_id == "" {
		resp.Msg = "错误！无法移动请求"
		return
	}
	f, err := os.ReadFile(a.db)
	if err != nil {
		resp.Msg = "错误！无法移动请求"
		return
	}
	var c []Collection
	err = json.Unmarshal(f, &c)
	if err != nil {
		resp.Msg = "错误！无法移动请求"
		return
	}
	var mReq Request
	for i := range c {
		if c[i].ID == coll_id {
			for j := range c[i].Requests {
				if c[i].Requests[j].ID == id {
					mReq = c[i].Requests[j]
					c[i].Requests = append(c[i].Requests[:j], c[i].Requests[j+1:]...)
					break
				}
			}
		}
	}
	if mReq.ID == "" {
		resp.Msg = "错误！未找到请求"
		return
	}
	opr := false
	for i := range c {
		if c[i].ID == new_coll_id {
			mReq.ID = new_id
			mReq.CollId = new_coll_id
			c[i].Requests = append(c[i].Requests, mReq)
			opr = true
			break
		}
	}
	if !opr {
		resp.Msg = "错误！未找到目标集合"
		return
	}
	b, err := json.Marshal(c)
	if err != nil {
		resp.Msg = "错误！无法移动请求"
		return
	}
	err = os.WriteFile(a.db, b, 0644)
	if err != nil {
		resp.Msg = "错误！无法移动请求"
		return
	}
	collRspSlice := makeCollRsp(&c)
	resp.Success = true
	resp.Msg = "请求已成功移动"
	resp.Data = collRspSlice
	return
}

func (a *RestMate) GetRequest(id, coll_id string) (resp JSResp) {
	if id == "" || coll_id == "" {
		resp.Msg = "错误！无法获取请求"
		return
	}
	f, err := os.ReadFile(a.db)
	if err != nil {
		resp.Msg = "错误！无法获取请求"
		return
	}
	var c []Collection
	err = json.Unmarshal(f, &c)
	if err != nil {
		resp.Msg = "错误！无法获取请求"
		return
	}
	for i := range c {
		if c[i].ID == coll_id {
			for j := range c[i].Requests {
				if c[i].Requests[j].ID == id {
					resp.Success = true
					resp.Msg = "成功！已找到请求"
					resp.Data = c[i].Requests[j]
					return
				}
			}
		}
	}
	resp.Msg = "错误！无法获取请求"
	return
}

func (a *RestMate) UpsertRequest(r Request) (resp JSResp) {
	opr := false
	if r.CollId == "" || r.ID == "" {
		resp.Msg = "错误！无法保存请求"
		return
	}
	f, err := os.ReadFile(a.db)
	if err != nil {
		resp.Msg = "错误！无法保存请求"
		return
	}
	var c []Collection
	err = json.Unmarshal(f, &c)
	if err != nil {
		resp.Msg = "错误！无法保存请求"
		return
	}
	for i := range c {
		if c[i].ID == r.CollId {
			found := false
			for j := range c[i].Requests {
				if c[i].Requests[j].ID == r.ID {
					c[i].Requests[j] = r
					found = true
					opr = true
					break
				}
			}
			if !found {
				c[i].Requests = append(c[i].Requests, r)
				opr = true
			}
			break
		}
	}
	if !opr {
		resp.Msg = "错误！无法保存请求"
		return
	}
	b, err := json.Marshal(c)
	if err != nil {
		resp.Msg = "错误！无法保存请求"
		return
	}
	err = os.WriteFile(a.db, b, 0644)
	if err != nil {
		resp.Msg = "错误！无法保存请求"
		return
	}
	collRspSlice := makeCollRsp(&c)
	resp.Success = true
	resp.Msg = "请求已成功保存"
	resp.Data = collRspSlice
	return
}

func (a *RestMate) GetCollections() (resp JSResp) {
	f, err := os.ReadFile(a.db)
	if err != nil {
		resp.Msg = "错误！无法添加集合"
		return
	}
	var c []Collection
	err = json.Unmarshal(f, &c)
	if err != nil {
		resp.Msg = "错误！无法添加集合"
		return
	}
	collRspSlice := makeCollRsp(&c)
	resp.Success = true
	resp.Msg = "已成功提取集合内容"
	resp.Data = collRspSlice
	return
}

func (a *RestMate) AddCollection(id, name string) (resp JSResp) {
	if id == "" || name == "" {
		resp.Msg = "错误！无法添加集合"
		return
	}
	f, err := os.ReadFile(a.db)
	if err != nil {
		resp.Msg = "错误！无法添加集合"
		return
	}
	var c []Collection
	err = json.Unmarshal(f, &c)
	if err != nil {
		resp.Msg = "错误！无法添加集合"
		return
	}
	newCol := Collection{
		ID:       id,
		Name:     name,
		Requests: []Request{},
	}
	c = append(c, newCol)
	b, err := json.Marshal(c)
	if err != nil {
		resp.Msg = "错误！无法添加集合"
		return
	}
	err = os.WriteFile(a.db, b, 0644)
	if err != nil {
		resp.Msg = "错误！无法添加集合"
		return
	}
	collRspSlice := makeCollRsp(&c)
	resp.Success = true
	resp.Msg = "集合保存成功"
	resp.Data = collRspSlice
	return
}

func (a *RestMate) RenameCollection(id, name string) (resp JSResp) {
	if id == "" || name == "" {
		resp.Msg = "错误！无法重命名集合"
		return
	}
	f, err := os.ReadFile(a.db)
	if err != nil {
		resp.Msg = "错误！无法重命名集合"
		return
	}
	var c []Collection
	err = json.Unmarshal(f, &c)
	if err != nil {
		resp.Msg = "错误！无法重命名集合"
		return
	}
	for i := range c {
		if c[i].ID == id {
			c[i].Name = name
			break
		}
	}
	b, err := json.Marshal(c)
	if err != nil {
		resp.Msg = "错误！无法重命名集合"
		return
	}
	err = os.WriteFile(a.db, b, 0644)
	if err != nil {
		resp.Msg = "错误！无法重命名集合"
		return
	}
	collRspSlice := makeCollRsp(&c)
	resp.Success = true
	resp.Msg = "集合重命名成功"
	resp.Data = collRspSlice
	return
}

func (a *RestMate) DuplicateRequest(coll_id, req_id string) (resp JSResp) {
	if coll_id == "" || req_id == "" {
		resp.Msg = "错误！无法复制请求"
		return
	}
	f, err := os.ReadFile(a.db)
	if err != nil {
		resp.Msg = "错误！无法复制请求"
		return
	}
	var c []Collection
	err = json.Unmarshal(f, &c)
	if err != nil {
		resp.Msg = "错误！无法复制请求"
		return
	}
	for i := range c {
		if c[i].ID == coll_id {
			var newReq Request
			for j := range c[i].Requests {
				if c[i].Requests[j].ID == req_id {
					newReq = c[i].Requests[j]
					break
				}
			}
			if newReq.ID != "" {
				nid, err := gonanoid.New()
				if err != nil {
					resp.Msg = "错误！无法复制请求"
					return
				}
				newReq.ID = nid
				newReq.Name += " Copy"
				c[i].Requests = append(c[i].Requests, newReq)
			}
			break
		}
	}
	b, err := json.Marshal(c)
	if err != nil {
		resp.Msg = "错误！无法复制请求"
		return
	}
	err = os.WriteFile(a.db, b, 0644)
	if err != nil {
		resp.Msg = "错误！无法复制请求"
		return
	}
	collRspSlice := makeCollRsp(&c)
	resp.Success = true
	resp.Msg = "请求复制成功"
	resp.Data = collRspSlice
	return
}

func (a *RestMate) RenameRequest(coll_id, req_id, req_name string) (resp JSResp) {
	if coll_id == "" || req_id == "" || req_name == "" {
		resp.Msg = "错误！无法重命名请求"
		return
	}
	f, err := os.ReadFile(a.db)
	if err != nil {
		resp.Msg = "错误！无法重命名请求"
		return
	}
	var c []Collection
	err = json.Unmarshal(f, &c)
	if err != nil {
		resp.Msg = "错误！无法重命名请求"
		return
	}
	for i := range c {
		if c[i].ID == coll_id {
			for j := range c[i].Requests {
				if c[i].Requests[j].ID == req_id {
					c[i].Requests[j].Name = req_name
					break
				}
			}
			break
		}
	}
	b, err := json.Marshal(c)
	if err != nil {
		resp.Msg = "错误！无法重命名请求"
		return
	}
	err = os.WriteFile(a.db, b, 0644)
	if err != nil {
		resp.Msg = "错误！无法重命名请求"
		return
	}
	collRspSlice := makeCollRsp(&c)
	resp.Success = true
	resp.Msg = "请求重命名成功"
	resp.Data = collRspSlice
	return
}

func (a *RestMate) DeleteRequest(coll_id, req_id string) (resp JSResp) {
	if coll_id == "" || req_id == "" {
		resp.Msg = "错误！无法删除请求"
		return
	}
	str, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:          runtime.QuestionDialog,
		Title:         "删除请求",
		Message:       "您确定要删除该请求吗？该请求将被永久删除。",
		Buttons:       []string{"Yes", "No"},
		DefaultButton: "No",
	})
	if err != nil {
		resp.Msg = "错误！无法删除请求"
		return
	}
	if str != "Yes" && str != "Ok" {
		resp.Msg = "错误！无法删除请求"
		return
	}

	f, err := os.ReadFile(a.db)
	if err != nil {
		resp.Msg = "错误！无法删除请求"
		return
	}
	var c []Collection
	err = json.Unmarshal(f, &c)
	if err != nil {
		resp.Msg = "错误！无法删除请求"
		return
	}
	for i := range c {
		if c[i].ID == coll_id {
			for j := range c[i].Requests {
				if c[i].Requests[j].ID == req_id {
					c[i].Requests = append(c[i].Requests[:j], c[i].Requests[j+1:]...)
					break
				}
			}
			break
		}
	}
	b, err := json.Marshal(c)
	if err != nil {
		resp.Msg = "错误！无法删除请求"
		return
	}
	err = os.WriteFile(a.db, b, 0644)
	if err != nil {
		resp.Msg = "错误！无法删除请求"
		return
	}
	collRspSlice := makeCollRsp(&c)
	resp.Success = true
	resp.Msg = "请求删除成功"
	resp.Data = collRspSlice
	return
}

func (a *RestMate) DeleteCollection(id string) (resp JSResp) {
	if id == "" {
		resp.Msg = "错误！无法删除请求"
		return
	}
	str, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:          runtime.QuestionDialog,
		Title:         "删除集合",
		Message:       "您确定要删除该集合吗？此集合中的所有请求都将被永久删除。",
		Buttons:       []string{"Yes", "No"},
		DefaultButton: "No",
	})
	if err != nil {
		resp.Msg = "错误！无法删除请求"
		return
	}
	if str != "Yes" && str != "Ok" {
		resp.Msg = "错误！无法删除请求"
		return
	}
	f, err := os.ReadFile(a.db)
	if err != nil {
		resp.Msg = "错误！无法删除集合"
		return
	}
	var c []Collection
	err = json.Unmarshal(f, &c)
	if err != nil {
		resp.Msg = "错误！无法删除集合"
		return
	}
	for i := range c {
		if c[i].ID == id {
			c = append(c[:i], c[i+1:]...)
			break
		}
	}
	b, err := json.Marshal(c)
	if err != nil {
		resp.Msg = "错误！无法删除集合"
		return
	}
	err = os.WriteFile(a.db, b, 0644)
	if err != nil {
		resp.Msg = "错误！无法删除集合"
		return
	}
	collRspSlice := makeCollRsp(&c)
	resp.Success = true
	resp.Msg = "集合删除成功"
	resp.Data = collRspSlice
	return
}

func makeCollRsp(c *[]Collection) []CollRsp {
	collRspSlice := make([]CollRsp, 0, len(*c))
	for i := range *c {
		reqRspSlice := make([]ReqRsp, 0, len((*c)[i].Requests))
		for j := range (*c)[i].Requests {
			r := (*c)[i].Requests[j]
			reqRspSlice = append(reqRspSlice, ReqRsp{
				ID:     r.ID,
				Name:   r.Name,
				Url:    r.Url,
				Method: r.Method,
				CollId: r.CollId,
			})
		}

		collRspSlice = append(collRspSlice, CollRsp{
			ID:       (*c)[i].ID,
			Name:     (*c)[i].Name,
			Requests: reqRspSlice,
		})
	}
	return collRspSlice
}

func parseMethod(m string) string {
	if m == "get" {
		return "GET"
	} else if m == "post" {
		return "POST"
	} else if m == "put" {
		return "PUT"
	} else if m == "delete" {
		return "DELETE"
	}
	return "GET"
}

func parseResponseHeaders(data *http.Header) []KV {
	var result []KV
	for k, v := range *data {
		h := KV{}
		h.Key = k
		h.Value = strings.Join(v, "")
		result = append(result, h)
	}
	return result
}
