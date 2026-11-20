package cloudSecurity

import (
	"EasyTools/app/controller/proxy"
	"EasyTools/app/controller/system"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/antchfx/xmlquery"
	"github.com/tealeg/xlsx"
)

// 风险等级类型
type RiskLevel string

const (
	CRITICAL1 RiskLevel = "CRITICAL1"
	CRITICAL2 RiskLevel = "CRITICAL2"
	CRITICAL3 RiskLevel = "CRITICAL3"
	CRITICAL4 RiskLevel = "CRITICAL4"
	HIGH1     RiskLevel = "HIGH1"
	HIGH2     RiskLevel = "HIGH2"
	MEDIUM1   RiskLevel = "MEDIUM1"
	MEDIUM2   RiskLevel = "MEDIUM2"
	MEDIUM3   RiskLevel = "MEDIUM3"
	LOW1      RiskLevel = "LOW1"
	LOW2      RiskLevel = "LOW2"
	LOW3      RiskLevel = "LOW3"
	ERROR     RiskLevel = "ERROR"
)

// 漏洞扫描结果
type VulnScanResult struct {
	Risk RiskLevel `json:"risk"`
	Msg  string    `json:"msg"`
	URL  string    `json:"url"`
}

// 扫描配置
type VulnScanConfig struct {
	ScanOptions []string `json:"scanOptions"`
	Threads     int      `json:"threads"`
	Timeout     int      `json:"timeout"`
}

// 云服务商检测器
type CloudDetector struct{}

// 检测云服务商类型
func (d *CloudDetector) DetectCloudProvider(urlStr string) string {
	urlStr = strings.ToLower(urlStr)

	// 阿里云
	if strings.Contains(urlStr, ".aliyuncs.com") || strings.Contains(urlStr, ".oss-") {
		return "aliyun"
	}
	// 腾讯云
	if strings.Contains(urlStr, ".myqcloud.com") || strings.Contains(urlStr, ".cos.") {
		return "tencent"
	}
	// 华为云
	if strings.Contains(urlStr, ".myhuaweicloud.com") || strings.Contains(urlStr, ".obs-") {
		return "huawei"
	}
	// AWS
	if strings.Contains(urlStr, ".amazonaws.com") || strings.Contains(urlStr, ".s3-") {
		return "aws"
	}
	// Azure
	if strings.Contains(urlStr, ".blob.core.windows.net") {
		return "azure"
	}
	// GCP
	if strings.Contains(urlStr, ".storage.googleapis.com") {
		return "gcp"
	}

	return "unknown"
}

// 从URL中提取Bucket和Region信息
func (d *CloudDetector) ExtractBucketInfo(urlStr, cloudProvider string) (bucket, region string) {
	switch cloudProvider {
	case "aliyun":
		// 格式: https://bucket.oss-region.aliyuncs.com
		re := regexp.MustCompile(`https?://([^.]+)\.oss-([^.]+)\.aliyuncs\.com`)
		matches := re.FindStringSubmatch(urlStr)
		if len(matches) == 3 {
			return matches[1], matches[2]
		}
	case "tencent":
		// 格式: https://bucket.cos.region.myqcloud.com
		re := regexp.MustCompile(`https?://([^.]+)\.cos\.([^.]+)\.myqcloud\.com`)
		matches := re.FindStringSubmatch(urlStr)
		if len(matches) == 3 {
			return matches[1], matches[2]
		}
	case "huawei":
		// 格式: https://bucket.obs-region.myhuaweicloud.com
		re := regexp.MustCompile(`https?://([^.]+)\.obs-([^.]+)\.myhuaweicloud\.com`)
		matches := re.FindStringSubmatch(urlStr)
		if len(matches) == 3 {
			return matches[1], matches[2]
		}
	case "aws":
		// 格式: https://bucket.s3.region.amazonaws.com
		re := regexp.MustCompile(`https?://([^.]+)\.s3\.([^.]+)\.amazonaws\.com`)
		matches := re.FindStringSubmatch(urlStr)
		if len(matches) == 3 {
			return matches[1], matches[2]
		}
	}
	return "", ""
}

// 漏洞扫描器
type VulnScanner struct {
	baseURL       string
	cloudProvider string
	bucket        string
	region        string
	config        *VulnScanConfig
	client        *http.Client
	results       []VulnScanResult
	mu            sync.Mutex
	// mutationObjectKey 单次扫描内 PUT/POST/DELETE 探测共用的临时对象名，避免多处创建不同测试文件
	mutationObjectKey string
}

func NewVulnScanner(urlStr string, config *VulnScanConfig) *VulnScanner {
	detector := &CloudDetector{}
	cloudProvider := detector.DetectCloudProvider(urlStr)
	bucket, region := detector.ExtractBucketInfo(urlStr, cloudProvider)

	return &VulnScanner{
		baseURL:       urlStr,
		cloudProvider: cloudProvider,
		bucket:        bucket,
		region:        region,
		config:        config,
		client:        proxy.GlobalProxyManager.GetHTTPClient(),
		results:       make([]VulnScanResult, 0),
	}
}

// HTTP请求工具
func (v *VulnScanner) httpRequest(method, urlStr string, headers map[string]string, body string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(context.Background(), method, urlStr, strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	// 设置headers
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// 设置默认User-Agent
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0 Safari/537.36")
	}

	return v.client.Do(req)
}

// 添加扫描结果
func (v *VulnScanner) addResult(risk RiskLevel, msg, url string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.results = append(v.results, VulnScanResult{
		Risk: risk,
		Msg:  msg,
		URL:  url,
	})
}

// 检测Bucket是否存在
func (v *VulnScanner) checkBucketExist() bool {
	resp, err := v.httpRequest("HEAD", v.baseURL, nil, "")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode != 404
}

// 检测列表权限
func (v *VulnScanner) checkListPermission() {
	resp, err := v.httpRequest("HEAD", v.baseURL+"/", nil, "")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		contentType := resp.Header.Get("Content-Type")
		if strings.Contains(contentType, "xml") {
			v.addResult(HIGH1, "Bucket公开可列目录", v.baseURL+"/")
		}
	}
}

// mutationTestObjectKey 返回本轮扫描共用的临时对象键名（仅用于匿名读写探测，不影响业务自选路径）
func (v *VulnScanner) mutationTestObjectKey() string {
	if v.mutationObjectKey == "" {
		v.mutationObjectKey = fmt.Sprintf("easytools_oss_scan_%d.txt", time.Now().UnixNano())
	}
	return v.mutationObjectKey
}

func (v *VulnScanner) closeRespBody(resp *http.Response) {
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
}

// runMutationScans 将 PUT/POST/DELETE 探测统一到同一临时对象：先按需上传，再测删除，最后尽力 DELETE 清理，避免遗留测试文件
func (v *VulnScanner) runMutationScans(scanOptions map[string]bool) {
	wantPut := scanOptions["scan_put_upload"]
	wantPost := scanOptions["scan_post_upload"]
	wantDel := scanOptions["scan_delete_perm"]
	if !wantPut && !wantPost && !wantDel {
		return
	}

	key := v.mutationTestObjectKey()
	fileURL := v.baseURL + "/" + key
	probeBody := "easytools_oss_scan_probe"

	objectLikelyExists := false

	if wantPut {
		resp, err := v.httpRequest("PUT", fileURL, nil, probeBody)
		if err == nil {
			code := resp.StatusCode
			v.closeRespBody(resp)
			if code == 200 || code == 201 {
				objectLikelyExists = true
				v.addResult(CRITICAL3, "Bucket允许匿名PUT上传", fileURL)
			}
		}
	}

	if wantPost {
		body := fmt.Sprintf("------WebKitFormBoundary7MA4YWxkTrZu0gW\r\nContent-Disposition: form-data; name=\"file\"; filename=\"%s\"\r\nContent-Type: text/plain\r\n\r\n%s\r\n------WebKitFormBoundary7MA4YWxkTrZu0gW--", key, probeBody)
		headers := map[string]string{
			"Content-Type": "multipart/form-data; boundary=----WebKitFormBoundary7MA4YWxkTrZu0gW",
		}
		resp, err := v.httpRequest("POST", v.baseURL+"/", headers, body)
		if err == nil {
			code := resp.StatusCode
			v.closeRespBody(resp)
			if code == 200 || code == 201 {
				objectLikelyExists = true
				v.addResult(CRITICAL3, "Bucket允许匿名POST上传", v.baseURL+"/")
			}
		}
	}

	if wantDel {
		if !objectLikelyExists {
			uploadResp, err := v.httpRequest("PUT", fileURL, nil, probeBody)
			if err == nil {
				code := uploadResp.StatusCode
				v.closeRespBody(uploadResp)
				objectLikelyExists = code == 200 || code == 201
			}
		}
		if objectLikelyExists {
			deleteResp, err := v.httpRequest("DELETE", fileURL, nil, "")
			if err == nil {
				code := deleteResp.StatusCode
				v.closeRespBody(deleteResp)
				if code == 200 || code == 204 {
					v.addResult(CRITICAL4, "Bucket允许匿名DELETE", fileURL)
				}
			}
		}
	}

	// 最终尽力删除临时对象，避免残留（失败则忽略，不改变扫描结论）
	cleanResp, _ := v.httpRequest("DELETE", fileURL, nil, "")
	v.closeRespBody(cleanResp)
}

// 检测CORS配置
func (v *VulnScanner) checkCORS() {
	headers := map[string]string{
		"Origin": "https://malicious.com",
	}

	resp, err := v.httpRequest("OPTIONS", v.baseURL, headers, "")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	allowMethods := resp.Header.Get("Access-Control-Allow-Methods")

	if allowOrigin == "*" && (strings.Contains(allowMethods, "PUT") || strings.Contains(allowMethods, "POST")) {
		v.addResult(MEDIUM2, "Bucket CORS过度宽松（允许所有域名+上传方法）", v.baseURL)
	}
}

// 检测访问日志泄露
func (v *VulnScanner) checkLogLeak() {
	logPaths := []string{
		"/logs/", "/accesslog/", "/oss-logs/",
		"/cos-logs/", "/tencent-logs/", "/s3-logs/", "/aws-logs/",
	}

	for _, path := range logPaths {
		logURL := v.baseURL + path
		resp, err := v.httpRequest("HEAD", logURL, nil, "")
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			v.addResult(HIGH2, "访问日志泄露", logURL)
			return
		}
	}
}

// 检测目录遍历漏洞
func (v *VulnScanner) checkDirectoryTraversal() {
	testPaths := []string{
		"../../etc/passwd",
		"../../../etc/passwd",
		"../windows/system32/drivers/etc/hosts",
		"../secret.txt",
	}

	for _, path := range testPaths {
		traversalURL := v.baseURL + "/" + path
		resp, err := v.httpRequest("GET", traversalURL, nil, "")
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			// 读取内容检查
			body, _ := io.ReadAll(resp.Body)
			if strings.Contains(string(body), "root:") {
				v.addResult(MEDIUM3, "目录遍历漏洞（可访问系统文件）", traversalURL)
				return
			}
		}
	}
}

// 检测敏感HTTP头泄露
func (v *VulnScanner) checkSensitiveHeaders() {
	resp, err := v.httpRequest("HEAD", v.baseURL, nil, "")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	sensitivePrefixes := []string{
		"X-OSS-Meta-", "X-Amz-Meta-", "X-Cos-Meta-", "X-Obs-Meta-",
		"X-OSS-Storage-Class", "X-Amz-Storage-Class",
	}

	for header := range resp.Header {
		for _, prefix := range sensitivePrefixes {
			if strings.HasPrefix(header, prefix) {
				v.addResult(LOW3, fmt.Sprintf("敏感HTTP头泄露（%s）", header), v.baseURL)
				return
			}
		}
	}
}

// 检测Bucket策略漏洞
func (v *VulnScanner) checkBucketPolicy() {
	policyURL := v.baseURL + "/?policy"
	resp, err := v.httpRequest("GET", policyURL, nil, "")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		body, _ := io.ReadAll(resp.Body)
		content := string(body)

		if strings.Contains(content, `"Effect": "Allow"`) && strings.Contains(content, `"Principal": "*"`) {
			if strings.Contains(content, `"oss:GetBucketPolicy"`) {
				v.addResult(HIGH2, "Bucket允许匿名获取Policy配置", policyURL)
			} else if strings.Contains(content, `"oss:PutObject"`) || strings.Contains(content, `"oss:DeleteObject"`) {
				v.addResult(CRITICAL2, "Bucket存在高危宽松策略（允许匿名写入/删除）", policyURL)
			} else {
				v.addResult(MEDIUM2, "Bucket存在宽松策略（允许匿名读取）", policyURL)
			}
		}
	}
}

// 检测加密配置漏洞
func (v *VulnScanner) checkEncryptionConfig() {
	encryptionURL := v.baseURL + "/?encryption"
	resp, err := v.httpRequest("GET", encryptionURL, nil, "")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		body, _ := io.ReadAll(resp.Body)
		content := string(body)

		if !strings.Contains(content, "ServerSideEncryptionConfiguration") {
			v.addResult(HIGH2, "Bucket未启用服务端加密", encryptionURL)
		} else if strings.Contains(content, "KMS") && strings.Contains(content, "KMSMasterKeyID") && !strings.Contains(strings.ToLower(content), "byok") {
			v.addResult(MEDIUM3, "Bucket使用KMS加密但未使用BYOK（自定义密钥）", encryptionURL)
		}
	}
}

// 运行漏洞扫描
func (v *VulnScanner) RunScan() []VulnScanResult {
	// 检查Bucket是否存在
	if !v.checkBucketExist() {
		v.addResult(LOW1, "Bucket不存在", v.baseURL)
		return v.results
	}

	// 构建扫描选项映射
	scanOptions := make(map[string]bool)
	for _, opt := range v.config.ScanOptions {
		scanOptions[opt] = true
	}

	// 执行基础检测
	v.checkListPermission()

	// PUT/POST/DELETE 使用同一临时对象键，按序探测并统一清理
	v.runMutationScans(scanOptions)
	if scanOptions["scan_cors"] {
		v.checkCORS()
	}
	if scanOptions["scan_logs"] {
		v.checkLogLeak()
	}
	if scanOptions["scan_directory_traversal"] {
		v.checkDirectoryTraversal()
	}
	if scanOptions["scan_sensitive_headers"] {
		v.checkSensitiveHeaders()
	}
	if scanOptions["scan_bucket_policy"] {
		v.checkBucketPolicy()
	}
	if scanOptions["scan_kms_encryption"] {
		v.checkEncryptionConfig()
	}

	return v.results
}

// 原有的Scanner结构体（文件遍历功能）
type Scanner struct {
	totalKeys map[string]bool
	keysMutex sync.Mutex
	baseURL   string
	system.Base
}

func NewScanner(url string) *Scanner {
	base := url
	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	return &Scanner{
		totalKeys: make(map[string]bool),
		baseURL:   base,
	}
}

// 原有的文件遍历处理方法
func (s *Scanner) Process() (string, error) {
	fmt.Printf("[*] 开始解析 URL：%s\n", s.baseURL)

	maxKeys, _, childTags, filename, err := s.getBucketInfo()
	if err != nil {
		return "", fmt.Errorf("[-] 获取信息失败: %v", err)
	}

	if len(childTags) == 0 {
		return "", fmt.Errorf("[-] 该存储桶不支持遍历, 或检查 URL 是否有误")
	}

	if err := s.fetchData(maxKeys, filename, childTags, "", 0); err != nil {
		return "", fmt.Errorf("[-] 获取数据失败: %v", err)
	}
	fmt.Println(filename)
	return filename, nil
}

// 新增漏洞扫描方法
func (s *Scanner) VulnScan(configJSON string) ([]VulnScanResult, error) {
	var config VulnScanConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, fmt.Errorf("配置解析失败: %v", err)
	}

	// 设置默认值
	if config.Threads == 0 {
		config.Threads = 5
	}
	if config.Timeout == 0 {
		config.Timeout = 10
	}

	scanner := NewVulnScanner(s.baseURL, &config)
	results := scanner.RunScan()

	return results, nil
}

// 原有的文件遍历方法保持不变
func (s *Scanner) getBucketInfo() (int, string, []string, string, error) {
	client := s.createClient()
	resp, err := client.Get(s.baseURL)
	if err != nil {
		return 0, "", nil, "", err
	}
	defer resp.Body.Close()

	xmlData, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", nil, "", err
	}

	doc, err := xmlquery.Parse(strings.NewReader(string(xmlData)))
	if err != nil {
		return 0, "", nil, "", err
	}

	maxKeys := 1000
	if node := xmlquery.FindOne(doc, "//MaxKeys"); node != nil {
		if v, err := strconv.Atoi(node.InnerText()); err == nil {
			maxKeys = v
		}
	}

	nextMarker := ""
	if node := xmlquery.FindOne(doc, "//NextMarker"); node != nil {
		nextMarker = node.InnerText()
	}

	var tags []string
	if contents := xmlquery.Find(doc, "//Contents"); len(contents) > 0 {
		for n := contents[0].FirstChild; n != nil; n = n.NextSibling {
			if n.Type == xmlquery.ElementNode {
				tagName := n.Data
				if strings.Contains(tagName, "}") {
					parts := strings.SplitN(tagName, "}", 2)
					tagName = parts[1]
				}
				tags = append(tags, tagName)
			}
		}
	}

	filename, err := s.createCSV(tags)
	if err != nil {
		return 0, "", nil, "", err
	}

	return maxKeys, nextMarker, tags, filename, nil
}

func (s *Scanner) createCSV(headers []string) (string, error) {
	baseDir := s.GetAppPath()
	fileDir := filepath.Join(baseDir, "file")
	cleanURL := regexp.MustCompile(`[:/?]`).ReplaceAllString(strings.TrimPrefix(s.baseURL, "http://"), "_")
	filename := filepath.Join(fileDir, fmt.Sprintf("%s.csv", cleanURL))

	file, err := os.Create(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 定义新的字段顺序：key、LastModified、filetype、url、Size、ETag、StorageClass、Owner、Type
	orderedHeaders := s.reorderHeaders(headers)
	return filename, writer.Write(orderedHeaders)
}

// reorderHeaders 重新排序CSV表头
// 目标顺序：Key、LastModified、filetype、url、Size、ETag、StorageClass、Owner、Type
func (s *Scanner) reorderHeaders(headers []string) []string {
	// 定义目标顺序（注意：Key在XML中可能是大写K）
	targetOrder := []string{"Key", "LastModified", "filetype", "url", "Size", "ETag", "StorageClass", "Owner", "Type"}

	// 创建一个map来存储原始headers中存在的字段（不区分大小写匹配）
	headerMap := make(map[string]string)
	headerLowerMap := make(map[string]string) // 用于不区分大小写查找
	for _, h := range headers {
		headerMap[h] = h
		headerLowerMap[strings.ToLower(h)] = h
	}

	// 按照目标顺序构建新的headers
	orderedHeaders := make([]string, 0, len(targetOrder)+len(headers))

	for _, target := range targetOrder {
		// filetype和url是固定添加的，不需要从原始headers中查找
		if target == "filetype" || target == "url" {
			orderedHeaders = append(orderedHeaders, target)
		} else {
			// 尝试精确匹配
			if original, exists := headerMap[target]; exists {
				orderedHeaders = append(orderedHeaders, original)
			} else if original, exists := headerLowerMap[strings.ToLower(target)]; exists {
				// 不区分大小写匹配
				orderedHeaders = append(orderedHeaders, original)
			}
		}
	}

	// 添加原始headers中不在目标顺序中的其他字段（如果有）
	for _, h := range headers {
		found := false
		for _, target := range targetOrder {
			if strings.EqualFold(h, target) {
				found = true
				break
			}
		}
		if !found {
			orderedHeaders = append(orderedHeaders, h)
		}
	}

	return orderedHeaders
}

func (s *Scanner) fetchData(maxKeys int, filename string, headers []string, marker string, page int) error {
	client := s.createClient()
	encodedMarker := url.QueryEscape(marker)
	url := fmt.Sprintf("%s?max-keys=%d&marker=%s", s.baseURL, maxKeys, encodedMarker)

	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	xmlData, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	doc, err := xmlquery.Parse(strings.NewReader(string(xmlData)))
	if err != nil {
		return err
	}

	contents := xmlquery.Find(doc, "//Contents")
	nextMarker := ""
	if node := xmlquery.FindOne(doc, "//NextMarker"); node != nil {
		nextMarker = node.InnerText()
	}

	_, repeat, _, err := s.writeCSV(filename, headers, contents)
	if err != nil {
		return err
	}

	page++

	if nextMarker == "" || repeat {
		s.createExcel(filename)
		return nil
	}

	return s.fetchData(maxKeys, filename, headers, nextMarker, page)
}

func (s *Scanner) writeCSV(filename string, headers []string, nodes []*xmlquery.Node) (int, bool, int, error) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return 0, false, 0, err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// 获取重新排序后的headers（与createCSV保持一致）
	orderedHeaders := s.reorderHeaders(headers)

	var newCount, repeatCount int
	for _, node := range nodes {
		keyNode := xmlquery.FindOne(node, ".//Key")
		if keyNode == nil {
			continue
		}
		key := keyNode.InnerText()

		s.keysMutex.Lock()
		if s.totalKeys[key] {
			repeatCount++
			s.keysMutex.Unlock()
			continue
		}
		s.totalKeys[key] = true
		s.keysMutex.Unlock()

		newCount++

		// 创建按照新顺序排列的数据行
		row := make([]string, len(orderedHeaders))

		// 填充原始字段的值（从XML节点中提取）
		for _, h := range headers {
			// 查找该字段在新顺序中的位置
			var targetIdx int = -1
			for idx, orderedH := range orderedHeaders {
				if strings.EqualFold(orderedH, h) {
					targetIdx = idx
					break
				}
			}

			if targetIdx >= 0 {
				if n := xmlquery.FindOne(node, fmt.Sprintf(".//%s", h)); n != nil {
					row[targetIdx] = cleanExcelValue(n.InnerText())
				}
			}
		}

		// 计算filetype
		fileType := "unknown"
		if parts := strings.Split(key, "."); len(parts) > 1 {
			if ext := parts[len(parts)-1]; regexp.MustCompile(`^[a-zA-Z0-9]+$`).MatchString(ext) {
				fileType = ext
			}
		}

		// 填充filetype和url（按照新顺序）
		for idx, orderedH := range orderedHeaders {
			if strings.EqualFold(orderedH, "filetype") {
				row[idx] = cleanExcelValue(fileType)
			} else if strings.EqualFold(orderedH, "url") {
				row[idx] = cleanExcelValue(s.baseURL + key)
			}
		}

		if err := writer.Write(row); err != nil {
			return 0, false, 0, err
		}
	}

	total := len(s.totalKeys)
	repeat := repeatCount > 2
	return newCount, repeat, total, nil
}

func (s *Scanner) createExcel(csvFile string) (string, error) {
	file, err := os.Open(csvFile)
	if err != nil {
		return "", err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return "", err
	}

	if len(records) < 2 {
		return "", fmt.Errorf("CSV文件内容不足")
	}

	groups := make(map[string][][]string)
	headers := records[0]
	fileTypeIndex := -1
	for i, h := range headers {
		if h == "filetype" {
			fileTypeIndex = i
			break
		}
	}
	if fileTypeIndex == -1 {
		return "", fmt.Errorf("缺少filetype列")
	}

	for _, row := range records[1:] {
		if len(row) <= fileTypeIndex {
			continue
		}
		ft := row[fileTypeIndex]
		groups[ft] = append(groups[ft], row)
	}

	xf := xlsx.NewFile()
	usedNames := make(map[string]int)

	for ft, rows := range groups {
		baseName := s.sanitizeSheetName(ft)
		originalName := baseName

		for count := 1; ; count++ {
			if _, exists := usedNames[baseName]; !exists {
				usedNames[baseName] = 1
				break
			}
			suffix := fmt.Sprintf("(%d)", count)
			baseName = originalName
			if len(originalName)+len(suffix) > 31 {
				baseName = originalName[:31-len(suffix)]
			}
			baseName += suffix
		}

		sheet, err := xf.AddSheet(baseName)
		if err != nil {
			continue
		}

		headerRow := sheet.AddRow()
		for _, h := range headers {
			cell := headerRow.AddCell()
			cell.Value = cleanExcelValue(h)
		}

		for _, r := range rows {
			row := sheet.AddRow()
			for _, v := range r {
				cell := row.AddCell()
				cell.Value = cleanExcelValue(v)
			}
		}
	}

	outputFile := strings.TrimSuffix(csvFile, ".csv") + "_Type.xlsx"
	if err := xf.Save(outputFile); err != nil {
		return "", err
	}
	return outputFile, nil
}

func (s *Scanner) sanitizeSheetName(name string) string {
	reg := regexp.MustCompile(`[\\/?*[\]:()]`)
	name = reg.ReplaceAllString(name, "_")
	name = strings.TrimSpace(name)
	if name == "" {
		name = "unknown"
	}

	if len(name) > 0 {
		if unicode.IsDigit(rune(name[0])) {
			name = "S_" + name
		}
	}

	if len(name) > 31 {
		name = name[:31]
	}
	return strings.TrimRight(name, " ")
}

func cleanExcelValue(value string) string {
	value = strings.Map(func(r rune) rune {
		if r < 32 && r != '\t' && r != '\n' && r != '\r' {
			return -1
		}
		return r
	}, value)

	if len(value) > 32767 {
		return value[:32700] + "...[TRUNCATED]"
	}
	return value
}

func (s *Scanner) createClient() *http.Client {
	return proxy.GlobalProxyManager.GetHTTPClient()
}
