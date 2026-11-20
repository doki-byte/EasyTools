package restmate

import (
	"EasyTools/app/controller/proxy"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/andybalholm/brotli"
	"github.com/goccy/go-json"
	"github.com/gogs/chardet"
	cookiejar "github.com/juju/persistent-cookiejar"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

type RestMate struct {
	ctx           context.Context
	db            string
	env           string
	jarFile       string
	requestCtx    context.CancelFunc
	batchTestCtx  context.CancelFunc
	bypassTestCtx context.CancelFunc
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
	runtime.EventsOn(ctx, "cancelBatchTest", func(_ ...interface{}) {
		if a.batchTestCtx != nil {
			a.batchTestCtx()
			a.batchTestCtx = nil
		}
	})
	runtime.EventsOn(ctx, "cancelBypassTest", func(_ ...interface{}) {
		if a.bypassTestCtx != nil {
			a.bypassTestCtx()
			a.bypassTestCtx = nil
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
	var bodyBytes []byte // 用于构建请求包
	autoHeaders := make(http.Header)
	method := parseMethod(r.Method)
	var userCT string
	for i := range r.Headers {
		if strings.ToLower(r.Headers[i].Key) == "content-type" {
			userCT = strings.ToLower(strings.TrimSpace(r.Headers[i].Value))
			break
		}
	}

	// 查询参数（Params）始终放在 URL 查询字符串中，不管什么请求方法
	p := u.Query()
	for i := range r.Params {
		if strings.TrimSpace(r.Params[i].Key) != "" {
			var val = interpolateVars(r.Params[i].Value)
			p.Add(r.Params[i].Key, val)
		}
	}
	u.RawQuery = p.Encode()

	// 请求体（Body）根据方法和 bodyType 进行处理
	if method == "GET" || method == "DELETE" {
		// GET/DELETE 请求：没有请求体
		body = nil
	} else {
		// POST/PUT/PATCH 请求：根据 bodyType 处理请求体
		if r.Body.BodyType == "formdata" {
			// 如果用户已经提供 content-type 且有原始 body，则直接使用原始 body，避免破坏 boundary
			if strings.Contains(userCT, "multipart/form-data") && strings.TrimSpace(r.Body.BodyRaw) != "" {
				bodyStr := interpolateVars(r.Body.BodyRaw)
				bodyBytes = []byte(bodyStr)
				body = strings.NewReader(bodyStr)
				// 不自动写 Content-Type，沿用用户头
			} else {
				// formdata 模式：FormData 放在 body 中（multipart/form-data）
				var b bytes.Buffer
				writer := multipart.NewWriter(&b)

				// 添加 FormData 中的字段
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
							filePath := fd.Files[f]
							file, err := os.Open(filePath)
							if err != nil {
								resp.Msg = "Error! Cannot open file"
								return
							}
							defer file.Close()

							filename := filepath.Base(filePath)
							ext := strings.ToLower(filepath.Ext(filename))
							contentType := mime.TypeByExtension(ext)
							if contentType == "" {
								contentType = "application/octet-stream"
							}

							header := make(textproto.MIMEHeader)
							header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fd.Key, filename))
							header.Set("Content-Type", contentType)

							part, err := writer.CreatePart(header)
							if err != nil {
								resp.Msg = "Error! Failed to write file in formdata"
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
				bodyBytes = b.Bytes()
				body = bytes.NewReader(bodyBytes)
				autoHeaders.Set("Content-Type", writer.FormDataContentType())
			}
		} else if r.Body.BodyType == "urlencoded" {
			// x-www-form-urlencoded 模式：FormData 放在 body 中（application/x-www-form-urlencoded）
			formValues := url.Values{}

			// 添加 FormData 中的字段
			for i := range r.Body.FormData {
				fd := r.Body.FormData[i]
				if fd.Type != "file" && fd.Key != "" {
					var val = interpolateVars(fd.Value)
					formValues.Add(fd.Key, val)
				}
			}

			bodyStr := formValues.Encode()
			bodyBytes = []byte(bodyStr)
			body = strings.NewReader(bodyStr)
			autoHeaders.Set("Content-Type", "application/x-www-form-urlencoded")
		} else if r.Body.BodyType == "json" {
			// JSON 模式：BodyRaw 放在 body 中（application/json）
			bodyStr := interpolateVars(r.Body.BodyRaw)
			bodyBytes = []byte(bodyStr)
			body = strings.NewReader(bodyStr)
			autoHeaders.Set("Content-Type", "application/json")
		} else if r.Body.BodyType == "raw" {
			bodyStr := interpolateVars(r.Body.BodyRaw)
			// 若用户在 multipart/raw 场景下换行被压成 LF，自动补回 CRLF，避免 boundary 长度不匹配
			if strings.Contains(userCT, "multipart/form-data") && !strings.Contains(bodyStr, "\r\n") && strings.Contains(bodyStr, "\n") {
				bodyStr = strings.ReplaceAll(bodyStr, "\n", "\r\n")
			}
			bodyBytes = []byte(bodyStr)
			body = strings.NewReader(bodyStr)
			// Content-Type 走用户传入，如未传可由服务器兜底
		} else {
			// 其他情况（none 等）：没有请求体
			body = nil
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		resp.Msg = "Error! 请求失败，请查看站点是否可以访问，或者代理是否可用"
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

	// 构建请求包字符串
	var reqBuilder strings.Builder
	reqBuilder.WriteString(fmt.Sprintf("%s %s %s\r\n", req.Method, req.URL.RequestURI(), req.Proto))
	reqBuilder.WriteString(fmt.Sprintf("Host: %s\r\n", req.Host))
	// 按字母顺序排序请求头，保证顺序一致
	headerKeys := make([]string, 0, len(req.Header))
	for k := range req.Header {
		headerKeys = append(headerKeys, k)
	}
	sort.Strings(headerKeys)
	for _, k := range headerKeys {
		v := req.Header[k]
		for _, val := range v {
			reqBuilder.WriteString(fmt.Sprintf("%s: %s\r\n", k, val))
		}
	}
	reqBuilder.WriteString("\r\n")
	// 添加请求体（如果存在）
	if len(bodyBytes) > 0 {
		reqBuilder.Write(bodyBytes)
	}
	requestPacket := reqBuilder.String()

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

	// 读取原始响应体（用于构建响应包，保持原始格式）
	rawBuf, err := io.ReadAll(response.Body)
	if err != nil {
		resp.Msg = "Error! Failed to read response body"
		return
	}

	// 处理响应体解压（用于前端显示，支持 gzip/deflate/br）
	bodyReader := bytes.NewReader(rawBuf)
	contentEncoding := strings.ToLower(strings.TrimSpace(response.Header.Get("Content-Encoding")))
	var decodedBody io.ReadCloser
	switch contentEncoding {
	case "gzip":
		if gr, err := gzip.NewReader(bodyReader); err == nil {
			decodedBody = gr
		}
	case "deflate":
		if zr, err := zlib.NewReader(bodyReader); err == nil {
			decodedBody = zr
		}
	case "br":
		decodedBody = io.NopCloser(brotli.NewReader(bodyReader))
	}

	var buf []byte
	if decodedBody != nil {
		defer decodedBody.Close()
		buf, err = io.ReadAll(decodedBody)
		if err != nil {
			buf = rawBuf // 如果解压失败，使用原始数据
		}
	} else {
		buf = rawBuf
	}
	if err != nil {
		resp.Msg = "Error! Failed to read response body"
		return
	}

	contentTypeHeader := response.Header.Get("Content-Type")
	ct := strings.ToLower(strings.Split(contentTypeHeader, ";")[0])
	declaredCharset := parseCharsetFromContentType(contentTypeHeader)

	// 判断是否为二进制类型（基于 Content-Type）
	isBinaryByType := strings.HasPrefix(ct, "image/") ||
		strings.Contains(ct, "application/pdf") ||
		strings.Contains(ct, "application/octet-stream") ||
		strings.Contains(ct, "application/zip") ||
		strings.Contains(ct, "application/x-") ||
		strings.Contains(ct, "video/") ||
		strings.Contains(ct, "audio/")

	// 检查实际内容是否为二进制
	isBinaryByContent := !isTextContent(buf)

	// 如果 Content-Type 或实际内容显示为二进制，则按二进制处理
	isBinary := isBinaryByType || isBinaryByContent

	if isBinary {
		// 二进制数据：直接使用原始字节，不做任何转换
		result.BodyContent = string(buf) // 直接使用解压后的字节
		result.IsBinary = true
		result.BinaryData = base64.StdEncoding.EncodeToString(rawBuf) // 保存原始字节的 Base64 编码，用于保存文件

		// 根据 Content-Type 推断文件扩展名
		switch {
		case strings.HasPrefix(ct, "image/png"):
			result.ContentType = "IMAGE"
			result.SuggestedExt = "png"
		case strings.HasPrefix(ct, "image/jpeg") || strings.HasPrefix(ct, "image/jpg"):
			result.ContentType = "IMAGE"
			result.SuggestedExt = "jpg"
		case strings.HasPrefix(ct, "image/gif"):
			result.ContentType = "IMAGE"
			result.SuggestedExt = "gif"
		case strings.HasPrefix(ct, "image/"):
			result.ContentType = "IMAGE"
			result.SuggestedExt = "bin"
		case strings.Contains(ct, "application/pdf"):
			result.ContentType = "PDF"
			result.SuggestedExt = "pdf"
		case strings.Contains(ct, "application/zip"):
			result.ContentType = "BINARY"
			result.SuggestedExt = "zip"
		default:
			result.ContentType = "BINARY"
			result.SuggestedExt = "bin"
		}
	} else {
		// 文本类型，进行编码转换
		bodyContent, converted := convertToUTF8(buf, declaredCharset)
		// 如果转换失败或结果看起来不像有效文本，直接使用原始字节
		if !converted || !isValidUTF8Text(bodyContent) {
			result.BodyContent = string(buf)
			result.ContentType = "BINARY"
			result.IsBinary = true
			result.BinaryData = base64.StdEncoding.EncodeToString(rawBuf) // 保存原始字节的 Base64 编码
			result.SuggestedExt = "bin"
		} else {
			result.BodyContent = bodyContent
			switch {
			case strings.Contains(ct, "application/json"):
				result.ContentType = "JSON"
			case strings.Contains(ct, "text/html"):
				result.ContentType = "HTML"
			case strings.Contains(ct, "text/plain"):
				result.ContentType = "TEXT"
			case strings.Contains(ct, "application/xml") || strings.Contains(ct, "text/xml"):
				result.ContentType = "XML"
			default:
				result.ContentType = "TEXT"
			}
		}
	}

	// 构建响应包字符串
	var respBuilder strings.Builder
	respBuilder.WriteString(fmt.Sprintf("%s %d %s\r\n", response.Proto, response.StatusCode, response.Status))
	// 按字母顺序排序响应头，保证顺序一致
	respHeaderKeys := make([]string, 0, len(response.Header))
	for k := range response.Header {
		respHeaderKeys = append(respHeaderKeys, k)
	}
	sort.Strings(respHeaderKeys)
	for _, k := range respHeaderKeys {
		v := response.Header[k]
		for _, val := range v {
			respBuilder.WriteString(fmt.Sprintf("%s: %s\r\n", k, val))
		}
	}
	respBuilder.WriteString("\r\n")
	// 响应包使用解压后的字节（buf），这样显示的是实际内容而不是压缩数据
	// 如果响应是压缩的，buf 已经是解压后的数据；如果不是压缩的，buf 等于 rawBuf
	respBuilder.Write(buf)
	result.ResponsePacket = respBuilder.String()
	result.RequestPacket = requestPacket

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
		// 如果文件不存在，返回空数组而不是错误
		if os.IsNotExist(err) {
			resp.Success = true
			resp.Msg = "集合列表为空"
			resp.Data = []interface{}{}
			return
		}
		resp.Msg = "错误！无法读取集合文件: " + err.Error()
		return
	}
	var c []Collection
	err = json.Unmarshal(f, &c)
	if err != nil {
		// JSON解析失败，尝试返回空数组
		resp.Success = true
		resp.Msg = "集合文件格式错误，已重置为空"
		resp.Data = []interface{}{}
		// 尝试修复：写入空数组
		emptyData, _ := json.Marshal([]Collection{})
		_ = os.WriteFile(a.db, emptyData, 0644)
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
	m = strings.ToLower(m)
	switch m {
	case "get":
		return "GET"
	case "post":
		return "POST"
	case "put":
		return "PUT"
	case "delete":
		return "DELETE"
	case "patch":
		return "PATCH"
	case "head":
		return "HEAD"
	case "options":
		return "OPTIONS"
	default:
		return "GET"
	}
}

// 从 Content-Type 头中解析 charset 参数
func parseCharsetFromContentType(contentType string) string {
	contentType = strings.ToLower(contentType)
	// 查找 charset= 参数
	if idx := strings.Index(contentType, "charset="); idx != -1 {
		charset := strings.TrimSpace(contentType[idx+8:])
		// 移除可能的分号和空格
		if semicolonIdx := strings.Index(charset, ";"); semicolonIdx != -1 {
			charset = charset[:semicolonIdx]
		}
		charset = strings.TrimSpace(charset)
		// 移除引号
		charset = strings.Trim(charset, `"'`)
		return strings.ToLower(charset)
	}
	return ""
}

// 检测内容是否为文本
func isTextContent(data []byte) bool {
	if len(data) == 0 {
		return true
	}

	// 检查前1KB的内容是否主要是可打印字符
	checkSize := len(data)
	if checkSize > 1024 {
		checkSize = 1024
	}

	printableCount := 0
	controlCount := 0

	for i := 0; i < checkSize; i++ {
		b := data[i]
		// 可打印字符：ASCII 32-126，以及制表符(9)、换行(10)、回车(13)
		if b >= 32 && b <= 126 || b == 9 || b == 10 || b == 13 {
			printableCount++
		} else if b < 32 || b == 127 {
			controlCount++
		}
	}

	// 如果控制字符太多，可能不是文本
	if controlCount > checkSize/10 {
		return false
	}

	// 如果可打印字符占比超过80%，认为是文本
	return float32(printableCount)/float32(checkSize) > 0.8
}

// 检测UTF-8字符串是否为有效的文本内容
func isValidUTF8Text(s string) bool {
	if len(s) == 0 {
		return true
	}

	// 检查字符串中是否有过多无法显示的字符（乱码通常包含大量无法显示的字符）
	unprintableCount := 0
	totalRunes := 0

	for _, r := range s {
		totalRunes++
		// 检查是否为不可打印的控制字符（除了常见的空白字符）
		if (r < 32 && r != 9 && r != 10 && r != 13) || r == 127 {
			unprintableCount++
		}
		// 检查是否为替换字符（UTF-8解码失败时会使用）
		if r == '\uFFFD' {
			unprintableCount++
		}
	}

	// 如果不可打印字符占比超过10%，可能不是有效文本
	if totalRunes > 0 {
		return float32(unprintableCount)/float32(totalRunes) < 0.1
	}

	return true
}

// 构建十六进制预览（最多预览前 4096 字节）
func makeHexPreview(data []byte) string {
	if len(data) == 0 {
		return "[空响应]"
	}
	max := len(data)
	if max > 4096 {
		max = 4096
	}
	preview := hex.Dump(data[:max])
	if len(data) > max {
		preview += fmt.Sprintf("... (剩余 %d 字节未显示)\n", len(data)-max)
	}
	return preview
}

// 检测并转换编码到 UTF-8
func convertToUTF8(data []byte, declaredCharset string) (string, bool) {
	// 如果已经是有效的 UTF-8，直接返回
	if utf8.Valid(data) {
		return string(data), true
	}

	// 如果有声明的 charset，优先使用
	if declaredCharset != "" {
		result, ok := convertByCharset(data, declaredCharset)
		if ok {
			return result, true
		}
	}

	// 尝试自动检测编码
	detector := chardet.NewTextDetector()
	result, err := detector.DetectBest(data)
	if err == nil && result != nil && result.Charset != "" {
		if converted, ok := convertByCharset(data, result.Charset); ok {
			return converted, true
		}
	}

	// 如果都失败了，尝试常见的编码
	commonCharsets := []string{"gbk", "gb2312", "gb18030", "big5", "windows-1252", "iso-8859-1"}
	for _, charset := range commonCharsets {
		if converted, ok := convertByCharset(data, charset); ok && utf8.ValidString(converted) {
			return converted, true
		}
	}

	// 最后，使用 ToValidUTF8 进行安全转换
	return strings.ToValidUTF8(string(data), ""), false
}

// 根据指定的 charset 转换编码
func convertByCharset(data []byte, charset string) (string, bool) {
	charset = strings.ToLower(charset)

	var t transform.Transformer
	switch charset {
	case "utf-8", "utf8":
		return string(data), true
	case "gbk", "gb2312", "gb18030":
		t = simplifiedchinese.GBK.NewDecoder()
	case "big5":
		// Big5 编码处理（如果需要可以添加）
		return "", false
	case "utf-16le", "utf16-le":
		t = unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder()
	case "utf-16be", "utf16-be":
		t = unicode.UTF16(unicode.BigEndian, unicode.UseBOM).NewDecoder()
	default:
		return "", false
	}

	if t != nil {
		reader := transform.NewReader(bytes.NewReader(data), t)
		decoded, err := io.ReadAll(reader)
		if err == nil && utf8.Valid(decoded) {
			return string(decoded), true
		}
	}
	return "", false
}

func parseResponseHeaders(data *http.Header) []KV {
	var result []KV
	// 按字母顺序排序响应头，保证顺序一致
	headerKeys := make([]string, 0, len(*data))
	for k := range *data {
		headerKeys = append(headerKeys, k)
	}
	sort.Strings(headerKeys)
	for _, k := range headerKeys {
		v := (*data)[k]
		h := KV{}
		h.Key = k
		h.Value = strings.Join(v, "")
		result = append(result, h)
	}
	return result
}

type BatchRequestResult struct {
	RequestID      string  `json:"requestId"`
	RequestName    string  `json:"requestName"`
	Success        bool    `json:"success"`
	Result         *Result `json:"result,omitempty"`
	Error          string  `json:"error,omitempty"`
	RequestPacket  string  `json:"requestPacket,omitempty"`
	ResponsePacket string  `json:"responsePacket,omitempty"`
}

type BatchInvokeResponse struct {
	Total   int                  `json:"total"`
	Success int                  `json:"success"`
	Failed  int                  `json:"failed"`
	Results []BatchRequestResult `json:"results"`
}

// isSensitiveRequest 检查请求是否为敏感接口
// 过滤规则：
// 1. DELETE 协议
// 2. URL 中包含敏感关键词：delete, insert, update, remove, drop, truncate, clear, destroy
func (a *RestMate) isSensitiveRequest(req Request) bool {
	// 检查是否为DELETE协议
	method := strings.ToLower(req.Method)
	if method == "delete" {
		return true
	}

	// 检查URL中是否包含敏感关键词
	url := strings.ToLower(req.Url)
	sensitiveKeywords := []string{"delete", "insert", "update", "remove", "drop", "truncate", "clear", "destroy"}

	for _, keyword := range sensitiveKeywords {
		if strings.Contains(url, keyword) {
			return true
		}
	}

	return false
}

func (a *RestMate) BatchInvokeRequests(coll_id string, concurrency int, retryCount int, filterSensitive bool) (resp JSResp) {
	if coll_id == "" {
		resp.Msg = "错误！集合ID不能为空"
		return
	}

	// 创建批量测试的 context，用于取消
	if a.batchTestCtx != nil {
		a.batchTestCtx()
	}
	batchCtx, cancel := context.WithCancel(a.ctx)
	a.batchTestCtx = cancel
	defer func() {
		cancel()
		a.batchTestCtx = nil
	}()

	f, err := os.ReadFile(a.db)
	if err != nil {
		resp.Msg = "错误！读取文件失败"
		return
	}
	var c []Collection
	err = json.Unmarshal(f, &c)
	if err != nil {
		resp.Msg = "错误！读取文件失败"
		return
	}

	var targetCollection *Collection
	for i := range c {
		if c[i].ID == coll_id {
			targetCollection = &c[i]
			break
		}
	}

	if targetCollection == nil {
		resp.Msg = "错误！未找到集合"
		return
	}

	if len(targetCollection.Requests) == 0 {
		resp.Msg = "错误！集合中没有请求"
		return
	}

	// 过滤敏感接口
	requestsToTest := targetCollection.Requests
	if filterSensitive {
		requestsToTest = make([]Request, 0)
		for _, req := range targetCollection.Requests {
			if !a.isSensitiveRequest(req) {
				requestsToTest = append(requestsToTest, req)
			}
		}
		if len(requestsToTest) == 0 {
			resp.Msg = "错误！过滤后没有可测试的请求"
			return
		}
	}

	// 使用传入的配置参数
	config := BatchConfig{
		Concurrency: concurrency,
		RetryCount:  retryCount,
	}
	// 验证配置值
	if config.Concurrency <= 0 {
		config.Concurrency = 10
	}
	if config.Concurrency > 1000 {
		config.Concurrency = 1000
	}
	if config.RetryCount < 0 {
		config.RetryCount = 0
	}
	if config.RetryCount > 10 {
		config.RetryCount = 10
	}

	batchResp := BatchInvokeResponse{
		Total:   len(requestsToTest),
		Success: 0,
		Failed:  0,
		Results: make([]BatchRequestResult, 0, len(requestsToTest)),
	}

	// 使用并发控制发送请求
	sem := make(chan struct{}, config.Concurrency) // 并发控制信号量
	var wg sync.WaitGroup
	var mu sync.Mutex // 保护共享数据
	results := make([]BatchRequestResult, len(requestsToTest))

	// 速率限制：计算每个请求之间的间隔时间（毫秒）
	rateLimitInterval := time.Duration(1000/config.Concurrency) * time.Millisecond
	if rateLimitInterval < 1 {
		rateLimitInterval = 1 * time.Millisecond
	}

	for i := range requestsToTest {
		// 检查是否已取消
		select {
		case <-batchCtx.Done():
			resp.Msg = "批量测试已取消"
			resp.Success = false
			runtime.EventsEmit(a.ctx, "batchTestCancelled", map[string]interface{}{
				"total":   batchResp.Total,
				"success": batchResp.Success,
				"failed":  batchResp.Failed,
			})
			return
		default:
		}

		// 速率限制：等待适当的间隔
		if i > 0 {
			time.Sleep(rateLimitInterval)
		}

		wg.Add(1)
		sem <- struct{}{} // 获取信号量

		go func(idx int, req Request) {
			defer wg.Done()
			defer func() { <-sem }() // 释放信号量

			// 检查是否已取消
			select {
			case <-batchCtx.Done():
				return
			default:
			}

			result := BatchRequestResult{
				RequestID:   req.ID,
				RequestName: req.Name,
			}

			// 发送请求开始事件
			runtime.EventsEmit(a.ctx, "batchRequestResult", map[string]interface{}{
				"index":   idx,
				"result":  nil,
				"total":   batchResp.Total,
				"success": batchResp.Success,
				"failed":  batchResp.Failed,
			})

			// 执行请求（带重试）
			var invokeResp JSResp
			var lastErr error
			for retry := 0; retry <= config.RetryCount; retry++ {
				// 检查是否已取消
				select {
				case <-batchCtx.Done():
					return
				default:
				}

				invokeResp = a.InvokeRequest(req)
				if invokeResp.Success && invokeResp.Data != nil {
					// 检查是否有错误
					if resMap, ok := invokeResp.Data.(map[string]interface{}); ok {
						if errorContent, ok := resMap["errorContent"].(string); ok && errorContent != "" {
							lastErr = fmt.Errorf(errorContent)
							if retry < config.RetryCount {
								// 继续重试
								continue
							}
						}
					} else if res, ok := invokeResp.Data.(Result); ok {
						if res.ErrorContent != "" {
							lastErr = fmt.Errorf(res.ErrorContent)
							if retry < config.RetryCount {
								// 继续重试
								continue
							}
						}
					}
					// 成功，跳出重试循环
					break
				} else {
					lastErr = fmt.Errorf(invokeResp.Msg)
					if retry < config.RetryCount {
						// 继续重试
						continue
					}
				}
			}

			// 处理响应结果
			if invokeResp.Success && invokeResp.Data != nil {
				// 使用类型断言处理 Result
				resMap, ok := invokeResp.Data.(map[string]interface{})
				if ok {
					// 从 map 转换为 Result 结构
					var res Result
					if statusCode, ok := resMap["statusCode"].(float64); ok {
						res.StatusCode = int(statusCode)
					}
					if httpStatus, ok := resMap["httpStatus"].(string); ok {
						res.HttpStatus = httpStatus
					}
					if bodyContent, ok := resMap["bodyContent"].(string); ok {
						res.BodyContent = bodyContent
					}
					if errorContent, ok := resMap["errorContent"].(string); ok {
						res.ErrorContent = errorContent
					}
					if contentType, ok := resMap["contentType"].(string); ok {
						res.ContentType = contentType
					}
					if duration, ok := resMap["duration"].(string); ok {
						res.Duration = duration
					}
					if headers, ok := resMap["headers"].([]interface{}); ok {
						for _, h := range headers {
							if hMap, ok := h.(map[string]interface{}); ok {
								kv := KV{}
								if key, ok := hMap["key"].(string); ok {
									kv.Key = key
								}
								if value, ok := hMap["value"].(string); ok {
									kv.Value = value
								}
								res.Headers = append(res.Headers, kv)
							}
						}
					}
					if requestPacket, ok := resMap["requestPacket"].(string); ok {
						res.RequestPacket = requestPacket
					}
					if responsePacket, ok := resMap["responsePacket"].(string); ok {
						res.ResponsePacket = responsePacket
					}
					if res.ErrorContent == "" {
						result.Success = true
						result.Result = &res
						result.RequestPacket = res.RequestPacket
						result.ResponsePacket = res.ResponsePacket
					} else {
						result.Success = false
						result.Error = res.ErrorContent
						result.RequestPacket = res.RequestPacket
						result.ResponsePacket = res.ResponsePacket
						if lastErr != nil {
							result.Error = lastErr.Error()
						}
					}
				} else {
					// 尝试直接类型断言
					if res, ok := invokeResp.Data.(Result); ok {
						if res.ErrorContent == "" {
							result.Success = true
							result.Result = &res
							result.RequestPacket = res.RequestPacket
							result.ResponsePacket = res.ResponsePacket
						} else {
							result.Success = false
							result.Error = res.ErrorContent
							result.RequestPacket = res.RequestPacket
							result.ResponsePacket = res.ResponsePacket
						}
					} else {
						result.Success = false
						result.Error = "响应数据格式错误"
						if lastErr != nil {
							result.Error = lastErr.Error()
						}
					}
				}
			} else {
				result.Success = false
				if invokeResp.Data != nil {
					if resMap, ok := invokeResp.Data.(map[string]interface{}); ok {
						if errorContent, ok := resMap["errorContent"].(string); ok {
							result.Error = errorContent
						} else {
							result.Error = invokeResp.Msg
						}
					} else if res, ok := invokeResp.Data.(Result); ok && res.ErrorContent != "" {
						result.Error = res.ErrorContent
					} else {
						result.Error = invokeResp.Msg
					}
				} else {
					result.Error = invokeResp.Msg
				}
				if lastErr != nil {
					result.Error = lastErr.Error()
				}
			}

			// 更新结果到数组并更新统计
			mu.Lock()
			results[idx] = result
			if result.Success {
				batchResp.Success++
			} else {
				batchResp.Failed++
			}
			currentSuccess := batchResp.Success
			currentFailed := batchResp.Failed
			mu.Unlock()

			// 实时推送单个请求结果
			runtime.EventsEmit(a.ctx, "batchRequestResult", map[string]interface{}{
				"index":   idx,
				"result":  result,
				"total":   batchResp.Total,
				"success": currentSuccess,
				"failed":  currentFailed,
			})
		}(i, requestsToTest[i])
	}

	// 等待所有goroutine完成
	wg.Wait()

	// 组装最终结果
	batchResp.Results = results

	// 推送完成事件
	runtime.EventsEmit(a.ctx, "batchTestComplete", batchResp)

	resp.Success = true
	resp.Msg = fmt.Sprintf("批量测试完成：成功 %d，失败 %d", batchResp.Success, batchResp.Failed)
	resp.Data = batchResp
	return
}
