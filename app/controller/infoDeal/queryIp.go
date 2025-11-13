package infoDeal

import (
	"EasyTools/app/controller/proxy"
	"EasyTools/app/controller/system"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/xuri/excelize/v2"
)

// IP查询相关结构体
type IPQueryRequest struct {
	Targets []string `json:"targets"` // 改为数组，支持批量查询
}

type IPQueryResponse struct {
	Success  bool   `json:"success"`
	Original string `json:"original,omitempty"` // 添加原始输入
	IP       string `json:"ip,omitempty"`
	Location string `json:"location,omitempty"`
	Error    string `json:"error,omitempty"`
}

type IPExportRequest struct {
	Results   []IPResult `json:"results"`
	FileName  string     `json:"fileName,omitempty"`
	Timestamp string     `json:"timestamp,omitempty"`
}

type IPResult struct {
	Original string `json:"original"`
	IP       string `json:"ip"`
	Location string `json:"location"`
	Status   string `json:"status"`
}

// 修改批量查询结构体
type BatchQueryResult struct {
	Results   []IPQueryResponse `json:"results"`
	Progress  int               `json:"progress"`  // 进度百分比
	Completed int               `json:"completed"` // 已完成数量
	Total     int               `json:"total"`     // 总数量
}

// 查询任务结构
type QueryTask struct {
	Index  int
	Target string
	Result chan *IPQueryResponse
}

// 批量查询配置
type BatchQueryConfig struct {
	Concurrency int `json:"concurrency"` // 并发数
	Timeout     int `json:"timeout"`     // 超时时间（秒）
}

// QueryIp 控制器
type QueryIp struct {
	system.Base
}

// NewQueryIp 创建新的 QueryIp 控制器
func NewQueryIp() *QueryIp {
	return &QueryIp{}
}

// IP查询控制器方法 - 单条查询
func (i *QueryIp) QueryIPLocation(req IPQueryRequest) IPQueryResponse {
	if len(req.Targets) == 0 {
		return IPQueryResponse{
			Success: false,
			Error:   "请输入要查询的目标",
		}
	}

	// 只查询第一个目标
	target := req.Targets[0]

	// 提取主机名或IP
	host, err := extractHostFromInput(target)
	if err != nil {
		return IPQueryResponse{
			Success:  false,
			Original: target,
			Error:    fmt.Sprintf("输入格式错误: %v", err),
		}
	}

	// 解析域名获取IP
	ip, err := resolveHost(host)
	if err != nil {
		return IPQueryResponse{
			Success:  false,
			Original: target,
			Error:    fmt.Sprintf("域名解析失败: %v", err),
		}
	}

	// 查询IP归属地
	location, err := queryIPLocation(ip)
	if err != nil {
		return IPQueryResponse{
			Success:  false,
			Original: target,
			Error:    fmt.Sprintf("归属地查询失败: %v", err),
		}
	}

	return IPQueryResponse{
		Success:  true,
		Original: target,
		IP:       ip,
		Location: location,
	}
}

// 批量查询IP位置 - 多线程版本
func (i *QueryIp) BatchQueryIPLocation(req IPQueryRequest, config BatchQueryConfig) []IPQueryResponse {
	if len(req.Targets) == 0 {
		return []IPQueryResponse{}
	}

	// 设置默认配置
	if config.Concurrency <= 0 {
		config.Concurrency = 10 // 默认10个并发
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 // 默认30秒超时
	}

	// 创建任务通道
	taskChan := make(chan QueryTask, len(req.Targets))
	results := make([]IPQueryResponse, len(req.Targets))

	// 创建等待组
	var wg sync.WaitGroup

	// 启动工作协程
	for j := 0; j < config.Concurrency; j++ {
		wg.Add(1)
		go i.queryWorker(taskChan, &wg, config.Timeout)
	}

	// 发送任务
	go func() {
		for idx, target := range req.Targets {
			resultChan := make(chan *IPQueryResponse, 1)
			taskChan <- QueryTask{
				Index:  idx,
				Target: target,
				Result: resultChan,
			}

			// 等待结果或超时
			select {
			case result := <-resultChan:
				if result != nil {
					results[idx] = *result
				} else {
					results[idx] = IPQueryResponse{
						Success:  false,
						Original: target,
						Error:    "查询超时",
					}
				}
			case <-time.After(time.Duration(config.Timeout) * time.Second):
				results[idx] = IPQueryResponse{
					Success:  false,
					Original: target,
					Error:    "查询超时",
				}
			}
		}
		close(taskChan)
	}()

	// 等待所有工作协程完成
	wg.Wait()

	return results
}

// 查询工作协程 - 在 QueryIp 结构体上定义
func (i *QueryIp) queryWorker(tasks <-chan QueryTask, wg *sync.WaitGroup, timeout int) {
	defer wg.Done()

	for task := range tasks {
		// 设置超时上下文
		resultChan := make(chan *IPQueryResponse, 1)

		go func(t QueryTask) {
			defer close(resultChan)

			// 提取主机名或IP
			host, err := extractHostFromInput(t.Target)
			if err != nil {
				resultChan <- &IPQueryResponse{
					Success:  false,
					Original: t.Target,
					Error:    fmt.Sprintf("输入格式错误: %v", err),
				}
				return
			}

			// 解析域名获取IP
			ip, err := resolveHost(host)
			if err != nil {
				resultChan <- &IPQueryResponse{
					Success:  false,
					Original: t.Target,
					Error:    fmt.Sprintf("域名解析失败: %v", err),
				}
				return
			}

			// 查询IP归属地
			location, err := queryIPLocation(ip)
			if err != nil {
				resultChan <- &IPQueryResponse{
					Success:  false,
					Original: t.Target,
					Error:    fmt.Sprintf("归属地查询失败: %v", err),
				}
				return
			}

			resultChan <- &IPQueryResponse{
				Success:  true,
				Original: t.Target,
				IP:       ip,
				Location: location,
			}
		}(task)

		// 等待结果或超时
		select {
		case result := <-resultChan:
			if result != nil {
				task.Result <- result
			} else {
				task.Result <- &IPQueryResponse{
					Success:  false,
					Original: task.Target,
					Error:    "查询失败",
				}
			}
		case <-time.After(time.Duration(timeout) * time.Second):
			task.Result <- &IPQueryResponse{
				Success:  false,
				Original: task.Target,
				Error:    "查询超时",
			}
		}
	}
}

// 导出IP查询结果到Excel
func (i *QueryIp) ExportIPResults(req IPExportRequest) map[string]interface{} {
	// 创建新的Excel文件
	f := excelize.NewFile()

	// 设置文档属性
	f.SetDocProps(&excelize.DocProperties{
		Title:   "IP查询结果导出",
		Subject: "IP地址及归属地查询结果",
		Created: time.Now().Format("2006-01-02 15:04:05"),
	})

	// 创建主数据表
	sheetName := "IP查询结果"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("创建工作表失败: %v", err),
		}
	}
	f.SetActiveSheet(index)

	// 设置表头
	headers := []string{"序号", "原始输入", "IP地址", "IP归属地", "查询状态", "查询时间"}
	for col, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(col+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}

	// 填充数据
	for row, result := range req.Results {
		rowIndex := row + 2
		cells := []interface{}{
			row + 1,                                  // 序号
			result.Original,                          // 原始输入
			result.IP,                                // IP地址
			result.Location,                          // IP归属地
			getStatusDisplayText(result.Status),      // 查询状态
			time.Now().Format("2006-01-02 15:04:05"), // 查询时间
		}

		for col, value := range cells {
			cell, _ := excelize.CoordinatesToCellName(col+1, rowIndex)
			f.SetCellValue(sheetName, cell, value)
		}
	}

	// 设置列宽
	f.SetColWidth(sheetName, "A", "A", 8)  // 序号
	f.SetColWidth(sheetName, "B", "B", 40) // 原始输入
	f.SetColWidth(sheetName, "C", "C", 16) // IP地址
	f.SetColWidth(sheetName, "D", "D", 40) // IP归属地
	f.SetColWidth(sheetName, "E", "E", 12) // 查询状态
	f.SetColWidth(sheetName, "F", "F", 20) // 查询时间

	// 删除默认的Sheet1
	f.DeleteSheet("Sheet1")

	// 保存到字节数组
	buffer := new(bytes.Buffer)
	if err := f.Write(buffer); err != nil {
		return map[string]interface{}{
			"success": false,
			"error":   fmt.Sprintf("写入缓冲区失败: %v", err),
		}
	}

	// 返回 base64 编码的数据
	return map[string]interface{}{
		"success": true,
		"data":    base64.StdEncoding.EncodeToString(buffer.Bytes()),
	}
}

// 获取状态显示文本
func getStatusDisplayText(status string) string {
	switch status {
	case "success":
		return "成功"
	case "error":
		return "失败"
	case "pending":
		return "待查询"
	default:
		return status
	}
}

// 提取主机名或IP
func extractHostFromInput(input string) (string, error) {
	input = strings.TrimSpace(input)

	// 如果是纯IP地址
	if isIPAddress(input) {
		return input, nil
	}

	// 如果是URL格式，提取主机名
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		parsedURL, err := url.Parse(input)
		if err != nil {
			return "", fmt.Errorf("URL解析失败")
		}
		hostname := parsedURL.Hostname()
		if hostname == "" {
			return "", fmt.Errorf("无法从URL中提取主机名")
		}
		return hostname, nil
	}

	// 如果是域名格式（包含点）
	if strings.Contains(input, ".") && !strings.Contains(input, " ") {
		// 移除可能的路经部分
		parts := strings.Split(input, "/")
		host := parts[0]

		// 验证主机名格式
		if isValidHostname(host) {
			return host, nil
		}
	}

	return "", fmt.Errorf("无法识别的输入格式: %s", input)
}

// 检查是否为有效的域名
func isValidHostname(hostname string) bool {
	if len(hostname) > 253 {
		return false
	}

	// 简单的域名验证
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`, hostname)
	return matched
}

// 检查是否为IP地址
func isIPAddress(input string) bool {
	return net.ParseIP(input) != nil
}

// 解析域名获取IP
func resolveHost(host string) (string, error) {
	// 如果已经是IP地址，直接返回
	if isIPAddress(host) {
		return host, nil
	}

	// 解析域名
	ips, err := net.LookupIP(host)
	if err != nil {
		return "", err
	}

	// 优先返回IPv4地址
	for _, ip := range ips {
		if ip.To4() != nil {
			return ip.String(), nil
		}
	}

	// 如果没有IPv4，返回第一个IPv6
	if len(ips) > 0 {
		return ips[0].String(), nil
	}

	return "", fmt.Errorf("未找到IP地址")
}

// 查询IP归属地 - 使用多个备用接口
func queryIPLocation(ip string) (string, error) {
	// 接口列表，按优先级排序
	apis := []struct {
		name  string
		url   string
		parse func([]byte) (string, error)
	}{
		{
			name:  "ipapi.co",
			url:   fmt.Sprintf("https://ipapi.co/%s/json/", ip),
			parse: parseIPAPIResponse,
		},
		{
			name:  "ip-api.com",
			url:   fmt.Sprintf("http://ip-api.com/json/%s?lang=zh-CN", ip),
			parse: parseIPAPIComResponse,
		},
	}

	// 尝试每个接口
	var lastError error
	for _, api := range apis {
		location, err := queryWithAPI(api.url, api.parse)
		if err == nil && location != "" && location != "未知" {
			return location, nil
		}
		if err != nil {
			lastError = err
		}
	}

	if lastError != nil {
		return "", lastError
	}
	return "", fmt.Errorf("所有查询接口均失败")
}

// 使用具体API查询
func queryWithAPI(apiURL string, parseFunc func([]byte) (string, error)) (string, error) {
	client := proxy.GlobalProxyManager.GetHTTPClient()

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", err
	}

	// 设置通用的请求头
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return parseFunc(body)
}

// 解析 ipapi.co 响应
func parseIPAPIResponse(body []byte) (string, error) {
	var result struct {
		Country string `json:"country_name"`
		Region  string `json:"region"`
		City    string `json:"city"`
		ISP     string `json:"org"`
		Error   bool   `json:"error"`
		Reason  string `json:"reason"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if result.Error {
		return "", fmt.Errorf(result.Reason)
	}

	location := []string{}
	if result.Country != "" {
		location = append(location, result.Country)
	}
	if result.Region != "" {
		location = append(location, result.Region)
	}
	if result.City != "" {
		location = append(location, result.City)
	}
	if result.ISP != "" {
		location = append(location, result.ISP)
	}

	if len(location) == 0 {
		return "未知", nil
	}

	return strings.Join(location, ", "), nil
}

// 解析 ip-api.com 响应
func parseIPAPIComResponse(body []byte) (string, error) {
	var result struct {
		Country    string `json:"country"`
		RegionName string `json:"regionName"`
		City       string `json:"city"`
		ISP        string `json:"isp"`
		Status     string `json:"status"`
		Message    string `json:"message"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if result.Status != "success" {
		return "", fmt.Errorf(result.Message)
	}

	location := []string{}
	if result.Country != "" {
		location = append(location, result.Country)
	}
	if result.RegionName != "" {
		location = append(location, result.RegionName)
	}
	if result.City != "" {
		location = append(location, result.City)
	}
	if result.ISP != "" {
		location = append(location, result.ISP)
	}

	if len(location) == 0 {
		return "未知", nil
	}

	return strings.Join(location, ", "), nil
}

// 修改批量查询方法，支持进度回调
func (i *QueryIp) BatchQueryIPLocationWithProgress(req IPQueryRequest, config BatchQueryConfig, progressCallback func(int, int)) []IPQueryResponse {
	if len(req.Targets) == 0 {
		return []IPQueryResponse{}
	}

	// 设置默认配置
	if config.Concurrency <= 0 {
		config.Concurrency = 10
	}
	if config.Timeout <= 0 {
		config.Timeout = 30
	}

	total := len(req.Targets)
	results := make([]IPQueryResponse, total)
	completed := 0

	// 创建任务通道
	taskChan := make(chan QueryTask, total)

	// 创建等待组
	var wg sync.WaitGroup

	// 启动工作协程
	for j := 0; j < config.Concurrency; j++ {
		wg.Add(1)
		go i.queryWorkerWithProgress(taskChan, &wg, config.Timeout, &completed, total, progressCallback)
	}

	// 发送任务
	go func() {
		for idx, target := range req.Targets {
			resultChan := make(chan *IPQueryResponse, 1)
			taskChan <- QueryTask{
				Index:  idx,
				Target: target,
				Result: resultChan,
			}

			// 等待结果
			select {
			case result := <-resultChan:
				if result != nil {
					results[idx] = *result
				}
			case <-time.After(time.Duration(config.Timeout) * time.Second):
				results[idx] = IPQueryResponse{
					Success:  false,
					Original: target,
					Error:    "查询超时",
				}
			}
		}
		close(taskChan)
	}()

	// 等待所有工作协程完成
	wg.Wait()

	return results
}

// 修改工作协程，支持进度更新
func (i *QueryIp) queryWorkerWithProgress(tasks <-chan QueryTask, wg *sync.WaitGroup, timeout int, completed *int, total int, progressCallback func(int, int)) {
	defer wg.Done()

	for task := range tasks {
		resultChan := make(chan *IPQueryResponse, 1)

		go func(t QueryTask) {
			defer close(resultChan)

			host, err := extractHostFromInput(t.Target)
			if err != nil {
				resultChan <- &IPQueryResponse{
					Success:  false,
					Original: t.Target,
					Error:    fmt.Sprintf("输入格式错误: %v", err),
				}
				return
			}

			ip, err := resolveHost(host)
			if err != nil {
				resultChan <- &IPQueryResponse{
					Success:  false,
					Original: t.Target,
					Error:    fmt.Sprintf("域名解析失败: %v", err),
				}
				return
			}

			location, err := queryIPLocation(ip)
			if err != nil {
				resultChan <- &IPQueryResponse{
					Success:  false,
					Original: t.Target,
					Error:    fmt.Sprintf("归属地查询失败: %v", err),
				}
				return
			}

			resultChan <- &IPQueryResponse{
				Success:  true,
				Original: t.Target,
				IP:       ip,
				Location: location,
			}
		}(task)

		select {
		case result := <-resultChan:
			if result != nil {
				task.Result <- result
			}
			// 更新进度
			*completed++
			if progressCallback != nil {
				progress := int(float64(*completed) / float64(total) * 100)
				progressCallback(*completed, progress)
			}
		case <-time.After(time.Duration(timeout) * time.Second):
			task.Result <- &IPQueryResponse{
				Success:  false,
				Original: task.Target,
				Error:    "查询超时",
			}
			// 更新进度
			*completed++
			if progressCallback != nil {
				progress := int(float64(*completed) / float64(total) * 100)
				progressCallback(*completed, progress)
			}
		}
	}
}
