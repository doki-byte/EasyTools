package cloudSecurity

import (
	"EasyTools/app/controller/proxy"
	"EasyTools/app/controller/system"
	"context"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// OssBucker 控制器
type OssBucker struct {
	system.Base
}

// NewOssBucker 创建新的 OssBucker 控制器
func NewOssBucker() *OssBucker {
	return &OssBucker{}
}

// 统一日志事件发送：通过 log_event 推到前端全局日志
func (i *OssBucker) sendLog(level, message string) {
	ctx := i.GetCtx()
	if ctx == nil {
		return
	}
	eventData := map[string]interface{}{
		"level":   level,
		"message": message,
		"time":    time.Now().Format("15:04:05"),
		"source":  "云上安全/OSS",
	}
	runtime.EventsEmit(ctx, "log_event", eventData)
}

// OSS存储桶遍历
func (i *OssBucker) DealOssList(ossURL string) (string, error) {
	i.sendLog("info", fmt.Sprintf("开始 OSS 文件遍历: %s", ossURL))

	s := NewScanner(ossURL)
	excelPath, err := s.Process()
	if err != nil {
		i.sendLog("error", fmt.Sprintf("OSS 文件遍历失败: %s (%v)", ossURL, err))
		return "", fmt.Errorf("处理失败：%v", err)
	}
	i.sendLog("info", fmt.Sprintf("OSS 文件遍历完成: %s，结果文件: %s", ossURL, excelPath))
	return excelPath, nil
}

func (i *OssBucker) StartVulnScan(urlStr string, configJSON string) map[string]interface{} {
	i.sendLog("info", fmt.Sprintf("开始 OSS 漏洞扫描: %s", urlStr))

	scanner := NewScanner(urlStr)

	var config VulnScanConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		i.sendLog("error", fmt.Sprintf("OSS 漏洞扫描配置解析失败: %s (%v)", urlStr, err))
		return map[string]interface{}{
			"success": false,
			"error":   "配置解析失败: " + err.Error(),
		}
	}

	results, err := scanner.VulnScan(configJSON)
	if err != nil {
		i.sendLog("error", fmt.Sprintf("OSS 漏洞扫描执行失败: %s (%v)", urlStr, err))
		return map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
	}

	summary := fmt.Sprintf("OSS 漏洞扫描完成: %s，共检测到 %d 条结果", urlStr, len(results))
	i.sendLog("info", summary)

	return map[string]interface{}{
		"success":       true,
		"results":       results,
		"cloudProvider": scanner.baseURL, // 返回检测到的云服务商信息
	}
}

func (i *OssBucker) DetectCloudProvider(urlStr string) map[string]interface{} {
	detector := &CloudDetector{}
	cloudProvider := detector.DetectCloudProvider(urlStr)

	if cloudProvider == "unknown" {
		return map[string]interface{}{
			"success": false,
			"error":   "无法识别的云服务商URL格式",
		}
	}

	bucket, region := detector.ExtractBucketInfo(urlStr, cloudProvider)

	result := map[string]interface{}{
		"success":       true,
		"cloudProvider": cloudProvider,
		"bucket":        bucket,
		"region":        region,
	}

	// 特殊处理腾讯云APPID
	if cloudProvider == "tencent" && strings.Contains(bucket, "-") {
		parts := strings.Split(bucket, "-")
		if len(parts) > 1 {
			result["appid"] = parts[len(parts)-1]
			result["bucket"] = strings.Join(parts[:len(parts)-1], "-")
		}
	}

	// 特殊处理Azure账户名
	if cloudProvider == "azure" {
		// Azure URL格式: https://account.blob.core.windows.net/container
		if re := regexp.MustCompile(`https?://([^.]+)\.blob\.core\.windows\.net/([^/]+)`); re.MatchString(urlStr) {
			matches := re.FindStringSubmatch(urlStr)
			if len(matches) == 3 {
				result["account"] = matches[1]
				result["bucket"] = matches[2]
			}
		}
	}

	return result
}

// ReadOssListCSV 读取OSS文件遍历生成的CSV文件
func (i *OssBucker) ReadOssListCSV(csvPath string) map[string]interface{} {
	if csvPath == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "CSV文件路径为空",
		}
	}

	// 检查文件是否存在
	if _, err := os.Stat(csvPath); os.IsNotExist(err) {
		return map[string]interface{}{
			"success": false,
			"error":   "CSV文件不存在: " + csvPath,
		}
	}

	// 打开CSV文件
	file, err := os.Open(csvPath)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   "打开CSV文件失败: " + err.Error(),
		}
	}
	defer file.Close()

	// 读取CSV内容
	reader := csv.NewReader(file)
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	var headers []string
	var rows [][]string

	// 读取表头
	headerRow, err := reader.Read()
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   "读取CSV表头失败: " + err.Error(),
		}
	}
	headers = headerRow

	// 读取数据行（限制最多10000行，避免内存过大）
	maxRows := 10000
	rowCount := 0
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue // 跳过有问题的行
		}
		if rowCount >= maxRows {
			break
		}
		rows = append(rows, row)
		rowCount++
	}

	return map[string]interface{}{
		"success": true,
		"headers": headers,
		"rows":    rows,
		"total":   rowCount,
	}
}

// GetKkfilePreview 通过代理获取kkfile预览内容
func (i *OssBucker) GetKkfilePreview(fileURL string, previewURL string) map[string]interface{} {
	if fileURL == "" {
		return map[string]interface{}{
			"success": false,
			"error":   "文件URL为空",
		}
	}

	if previewURL == "" {
		// 官方演示站默认需携带 key=000
		previewURL = "https://file.kkview.cn/onlinePreview?url={base64(url)}&key=000"
	}

	// kkFileView 3.x：url 参数 = encodeURIComponent(Base64(文件下载地址))，见 https://www.kkview.cn/zh-cn/docs/usage.html
	encodedURL := base64.StdEncoding.EncodeToString([]byte(fileURL))
	urlParam := url.QueryEscape(encodedURL)
	previewURL = strings.ReplaceAll(previewURL, "{base64(url)}", urlParam)

	// 使用代理客户端获取预览内容
	client := proxy.GetGlobalHTTPClient()
	if client == nil {
		// 如果代理客户端不可用，创建默认客户端
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, previewURL, nil)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   "创建请求失败: " + err.Error(),
		}
	}

	// 设置请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   "请求失败: " + err.Error(),
		}
	}
	defer resp.Body.Close()

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   "读取响应失败: " + err.Error(),
		}
	}

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("请求失败，状态码: %d", resp.StatusCode),
			"content": string(body),
		}
	}

	return map[string]interface{}{
		"success": true,
		"content": string(body),
		"url":     previewURL,
	}
}
