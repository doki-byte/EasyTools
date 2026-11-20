package infoCollection

import (
	"EasyTools/app/controller/proxy"
	"EasyTools/app/controller/system"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// SpaceMapping 空间测绘控制器
type SpaceMapping struct {
	system.Base
}

// 用于存储正在进行的查询的context和cancel函数
var queryContexts = make(map[string]context.CancelFunc)
var queryContextsMutex = make(chan struct{}, 1) // 简单的互斥锁

// NewSpaceMapping 创建新的空间测绘控制器
func NewSpaceMapping() *SpaceMapping {
	return &SpaceMapping{}
}

// QueryRequest 查询请求
type QueryRequest struct {
	Query     string                 `json:"query"`
	Platforms []string               `json:"platforms"`
	Settings  map[string]interface{} `json:"settings,omitempty"` // 允许前端传入当前设置，优先使用
	APIKeys   map[string]string      `json:"apiKeys,omitempty"`  // 允许前端传入当前API Keys，优先使用
	Enabled   map[string]bool        `json:"enabled,omitempty"`  // 允许前端传入当前启用状态，优先使用
	QueryID   string                 `json:"queryId,omitempty"`  // 查询ID，用于取消查询
}

// QueryResponse 查询响应
type QueryResponse struct {
	Success    bool                    `json:"success"`
	Message    string                  `json:"message,omitempty"`
	Data       map[string][]ResultItem `json:"data,omitempty"`
	Aggregated *AggregatedData         `json:"aggregated,omitempty"` // 聚合数据
}

// ResultItem 结果项
type ResultItem struct {
	IP       string `json:"ip"`
	Port     string `json:"port"`
	Title    string `json:"title"`
	Domain   string `json:"domain"`
	URL      string `json:"url"`
	Platform string `json:"platform,omitempty"` // 数据来源平台
}

// AggregatedData 聚合数据
type AggregatedData struct {
	Total     int                    `json:"total"`     // 总记录数（去重前）
	Unique    int                    `json:"unique"`    // 去重后记录数
	Platforms map[string]int         `json:"platforms"` // 各平台数据统计
	Results   []AggregatedResultItem `json:"results"`   // 聚合后的结果
}

// AggregatedResultItem 聚合结果项
type AggregatedResultItem struct {
	IP        string   `json:"ip"`
	Port      string   `json:"port"`
	Title     string   `json:"title"`
	Domain    string   `json:"domain"`
	URL       string   `json:"url"`
	Platforms []string `json:"platforms"` // 数据来源平台列表
	Count     int      `json:"count"`     // 在多少个平台出现
}

// ConfigRequest 配置请求
type ConfigRequest struct {
	APIKeys  map[string]string      `json:"apiKeys"`
	Enabled  map[string]bool        `json:"enabled"`
	Settings map[string]interface{} `json:"settings"`
}

// ConfigResponse 配置响应
type ConfigResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// Query 执行空间测绘查询
func (sm *SpaceMapping) Query(c *gin.Context) {
	var req QueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, QueryResponse{
			Success: false,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	if req.Query == "" {
		c.JSON(http.StatusBadRequest, QueryResponse{
			Success: false,
			Message: "查询内容不能为空",
		})
		return
	}

	if len(req.Platforms) == 0 {
		c.JSON(http.StatusBadRequest, QueryResponse{
			Success: false,
			Message: "请至少选择一个平台",
		})
		return
	}

	// 加载配置
	config, err := loadConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, QueryResponse{
			Success: false,
			Message: "加载配置失败: " + err.Error(),
		})
		return
	}

	// 确定使用的Settings：优先使用前端传入的，否则使用配置文件中的
	var settings map[string]interface{}
	if len(req.Settings) > 0 {
		settings = req.Settings
	} else {
		settings = config.Settings
	}

	// 确定使用的APIKeys和Enabled：优先使用前端传入的，否则使用配置文件中的
	apiKeys := config.APIKeys
	if len(req.APIKeys) > 0 {
		apiKeys = req.APIKeys
	}
	enabled := config.Enabled
	if len(req.Enabled) > 0 {
		enabled = req.Enabled
	}

	// 创建查询context，支持取消
	queryID := req.QueryID
	if queryID == "" {
		// 如果没有提供queryID，生成一个
		queryID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// 存储cancel函数，用于取消查询
	queryContextsMutex <- struct{}{}
	queryContexts[queryID] = cancel
	<-queryContextsMutex

	// 查询完成后清理
	defer func() {
		queryContextsMutex <- struct{}{}
		delete(queryContexts, queryID)
		<-queryContextsMutex
	}()

	// 并发查询各个平台
	results := make(map[string][]ResultItem)
	resultChan := make(chan platformResult, len(req.Platforms))

	for _, platform := range req.Platforms {
		go func(p string) {
			// 优先使用前端传入的配置，否则使用配置文件中的
			platformConfig, exists := apiKeys[p]
			platformEnabled, enabledExists := enabled[p]
			if !exists || !enabledExists || !platformEnabled || platformConfig == "" {
				resultChan <- platformResult{
					platform: p,
					results:  []ResultItem{},
					err:      fmt.Errorf("平台 %s 未配置或未启用", p),
				}
				return
			}

			items, err := queryPlatformWithContext(ctx, p, req.Query, platformConfig, settings)
			// 如果context被取消，err可能是nil（因为返回了已查询到的结果）
			// 我们需要检查context状态
			if err != nil && err != context.Canceled {
				// 只有真正的错误才返回错误
				resultChan <- platformResult{
					platform: p,
					results:  []ResultItem{},
					err:      err,
				}
			} else {
				// context取消或正常完成，返回已查询到的结果
				resultChan <- platformResult{
					platform: p,
					results:  items,
					err:      nil, // 即使被取消，也返回nil错误，因为我们已经有了部分结果
				}
			}
		}(platform)
	}

	// 收集结果和错误信息
	collectedCount := 0
	errorMessages := make([]string, 0) // 收集所有平台的错误信息
	for collectedCount < len(req.Platforms) {
		select {
		case result := <-resultChan:
			collectedCount++
			if result.err != nil {
				// 如果是取消错误，返回已查询到的结果
				if result.err == context.Canceled {
					// 查询被取消，但继续收集已查询到的结果
					// 继续收集其他goroutine的结果
					continue
				}
				errorMsg := fmt.Sprintf("平台 %s 查询失败: %v", result.platform, result.err)
				fmt.Printf("%s\n", errorMsg)
				// 收集错误信息
				errorMessages = append(errorMessages, errorMsg)
				// 即使查询失败，也返回空结果，而不是不返回该平台的数据
				// 这样前端可以知道该平台查询失败，而不是显示"未找到结果"
				results[result.platform] = []ResultItem{}
			} else {
				// 为每个结果项添加平台标识
				for j := range result.results {
					result.results[j].Platform = result.platform
				}
				results[result.platform] = result.results
			}
		case <-ctx.Done():
			// 查询被取消，但继续收集已查询到的结果
			// 等待一小段时间，让所有goroutine有时间发送结果
			// 使用非阻塞方式继续收集结果
			timeout := time.NewTimer(2 * time.Second)
			defer timeout.Stop()
			for collectedCount < len(req.Platforms) {
				select {
				case result := <-resultChan:
					collectedCount++
					if result.err != nil && result.err != context.Canceled {
						errorMsg := fmt.Sprintf("平台 %s 查询失败: %v", result.platform, result.err)
						fmt.Printf("%s\n", errorMsg)
						// 收集错误信息
						errorMessages = append(errorMessages, errorMsg)
						results[result.platform] = []ResultItem{}
					} else {
						// 为每个结果项添加平台标识
						for j := range result.results {
							result.results[j].Platform = result.platform
						}
						results[result.platform] = result.results
					}
				case <-timeout.C:
					// 超时，停止等待
					goto done
				}
			}
			goto done
		}
	}
done:

	// 生成聚合数据
	aggregated := aggregateResults(results)

	// 构建响应消息
	responseMessage := queryID // 默认返回queryID
	// 如果有错误信息，将错误信息添加到响应中
	if len(errorMessages) > 0 {
		// 如果所有平台都失败了，设置 success 为 false
		allFailed := len(errorMessages) == len(req.Platforms)
		// 组合所有错误信息
		errorMsg := strings.Join(errorMessages, "; ")
		if allFailed {
			// 所有平台都失败，返回错误响应
			c.JSON(http.StatusOK, QueryResponse{
				Success: false,
				Message: errorMsg,
				Data:    results,
			})
			return
		} else {
			// 部分平台失败，仍然返回成功，但在message中包含错误信息
			responseMessage = errorMsg
		}
	}

	c.JSON(http.StatusOK, QueryResponse{
		Success:    true,
		Data:       results,
		Aggregated: aggregated,
		Message:    responseMessage, // 返回queryID或错误信息
	})
}

// CancelQueryRequest 取消查询请求
type CancelQueryRequest struct {
	QueryID string `json:"queryId"`
}

// CancelQuery 取消查询
func (sm *SpaceMapping) CancelQuery(c *gin.Context) {
	var req CancelQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, QueryResponse{
			Success: false,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	if req.QueryID == "" {
		c.JSON(http.StatusBadRequest, QueryResponse{
			Success: false,
			Message: "查询ID不能为空",
		})
		return
	}

	// 查找并取消查询
	queryContextsMutex <- struct{}{}
	cancel, exists := queryContexts[req.QueryID]
	<-queryContextsMutex

	if !exists {
		c.JSON(http.StatusOK, QueryResponse{
			Success: false,
			Message: "查询不存在或已完成",
		})
		return
	}

	// 取消查询
	cancel()

	c.JSON(http.StatusOK, QueryResponse{
		Success: true,
		Message: "查询已取消",
	})
}

type platformResult struct {
	platform string
	results  []ResultItem
	err      error
}

// queryPlatform 查询单个平台（兼容旧接口）
func queryPlatform(platform, query, apiKey string, settings map[string]interface{}) ([]ResultItem, error) {
	return queryPlatformWithContext(context.Background(), platform, query, apiKey, settings)
}

// queryPlatformWithContext 查询单个平台（支持context取消）
func queryPlatformWithContext(ctx context.Context, platform, query, apiKey string, settings map[string]interface{}) ([]ResultItem, error) {
	switch platform {
	case "fofa":
		return queryFofaWithContext(ctx, query, apiKey, settings)
	case "otherfofa":
		return queryFofaRedWithContext(ctx, query, apiKey, settings)
	case "0zone":
		return query0ZoneWithContext(ctx, query, apiKey, settings)
	case "zone0":
		return query0ZoneWithContext(ctx, query, apiKey, settings)
	case "hunter":
		return queryHunterWithContext(ctx, query, apiKey, settings)
	case "quake":
		return queryQuakeWithContext(ctx, query, apiKey, settings)
	default:
		return nil, fmt.Errorf("不支持的平台: %s", platform)
	}
}

// query0Zone 查询 0.zone（兼容旧接口）
func query0Zone(query, apiKey string, settings map[string]interface{}) ([]ResultItem, error) {
	return query0ZoneWithContext(context.Background(), query, apiKey, settings)
}

// query0ZoneWithContext 查询 0.zone（支持context取消）
func query0ZoneWithContext(ctx context.Context, query, apiKey string, settings map[string]interface{}) ([]ResultItem, error) {
	baseURL := "https://0.zone/api/data/"
	maxPages := getMaxPages(settings)
	pageSize := getPageSize(settings)
	delay := getDelay(settings)
	client := getHTTPClient(settings)
	requestTimeout := getTimeout(settings)

	allResults := make([]ResultItem, 0)

	for page := 1; page <= maxPages; page++ {
		select {
		case <-ctx.Done():
			return allResults, nil
		default:
		}

		payload := map[string]interface{}{
			"query":       query,
			"query_type":  "site",
			"page":        page,
			"pagesize":    pageSize,
			"zone_key_id": apiKey,
		}
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("构建0.zone第%d页请求失败: %v", page, err)
		}

		requestCtx, requestCancel := context.WithTimeout(ctx, requestTimeout)
		defer requestCancel()
		req, err := http.NewRequestWithContext(requestCtx, "POST", baseURL, strings.NewReader(string(jsonData)))
		if err != nil {
			if ctx.Err() == context.Canceled {
				return allResults, nil
			}
			return nil, fmt.Errorf("创建0.zone第%d页请求失败: %v", page, err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			if ctx.Err() == context.Canceled {
				return allResults, nil
			}
			errStr := strings.ToLower(err.Error())
			if strings.Contains(errStr, "timeout") ||
				strings.Contains(errStr, "deadline exceeded") ||
				strings.Contains(errStr, "context deadline") ||
				strings.Contains(errStr, "i/o timeout") {
				return allResults, nil
			}
			return nil, fmt.Errorf("请求0.zone第%d页失败: %v", page, err)
		}

		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("读取0.zone第%d页响应失败: %v", page, err)
		}

		var zoneResp struct {
			Code    int                      `json:"code"`
			Message string                   `json:"message"`
			Data    []map[string]interface{} `json:"data"`
		}
		if err := json.Unmarshal(body, &zoneResp); err != nil {
			return nil, fmt.Errorf("解析0.zone第%d页响应失败: %v", page, err)
		}
		// 根据官方示例，0.zone 成功时 code 为 0 而不是 200
		if zoneResp.Code != 0 {
			return nil, fmt.Errorf("0.zone API错误（第%d页）: %s", page, zoneResp.Message)
		}

		for _, item := range zoneResp.Data {
			ip := toString(item["ip"])
			port := toString(item["port"])
			title := toString(item["title"])
			domain := toString(item["url"])
			if domain == "" {
				domain = toString(item["host"])
			}
			if domain == "" {
				domain = toString(item["domain"])
			}
			url := toString(item["url"])
			if url == "" {
				if port == "443" {
					url = fmt.Sprintf("https://%s:%s", ip, port)
				} else {
					url = fmt.Sprintf("http://%s:%s", ip, port)
				}
			}
			allResults = append(allResults, ResultItem{
				IP:     ip,
				Port:   port,
				Title:  title,
				Domain: domain,
				URL:    url,
			})
		}

		if len(zoneResp.Data) == 0 || len(zoneResp.Data) < pageSize {
			break
		}
		if page < maxPages {
			select {
			case <-ctx.Done():
				return allResults, nil
			case <-time.After(delay):
			}
		}
	}

	return allResults, nil
}

func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return strings.TrimSpace(val)
	case float64:
		return fmt.Sprintf("%.0f", val)
	case int:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	default:
		return ""
	}
}

// queryFofa 查询FOFA官方（兼容旧接口）
func queryFofa(query, apiKey string, settings map[string]interface{}) ([]ResultItem, error) {
	return queryFofaWithContext(context.Background(), query, apiKey, settings)
}

// queryFofaWithContext 查询FOFA官方（支持context取消）
func queryFofaWithContext(ctx context.Context, query, apiKey string, settings map[string]interface{}) ([]ResultItem, error) {
	// FOFA官方API: https://fofa.info/api
	email := ""
	key := apiKey

	// 如果API Key包含email:key格式
	if strings.Contains(apiKey, ":") {
		parts := strings.SplitN(apiKey, ":", 2)
		email = parts[0]
		key = parts[1]
	}

	baseURL := "https://fofa.info/api/v1/search/all"

	// 获取配置参数
	maxPages := getMaxPages(settings)
	pageSize := getPageSize(settings)
	delay := getDelay(settings)

	client := getHTTPClient(settings)
	// 获取每个请求的超时时间
	requestTimeout := getTimeout(settings)

	allResults := make([]ResultItem, 0)

	// 分页查询
	for page := 1; page <= maxPages; page++ {
		// 检查context是否被取消
		select {
		case <-ctx.Done():
			// 查询被取消，返回已查询到的结果
			return allResults, nil
		default:
		}

		params := fmt.Sprintf("email=%s&key=%s&qbase64=%s&page=%d&size=%d", email, key, base64Encode(query), page, pageSize)
		url := baseURL + "?" + params

		// 为每个请求创建独立的超时context
		requestCtx, requestCancel := context.WithTimeout(ctx, requestTimeout)
		defer requestCancel() // 确保总是取消超时context，避免泄漏
		req, err := http.NewRequestWithContext(requestCtx, "GET", url, nil)
		if err != nil {
			// 如果是context取消，返回已查询到的结果
			if ctx.Err() == context.Canceled {
				return allResults, nil
			}
			return nil, fmt.Errorf("创建第%d页请求失败: %v", page, err)
		}

		resp, err := client.Do(req)
		if err != nil {
			// 检查是否是context取消
			if ctx.Err() == context.Canceled {
				return allResults, nil
			}
			// 检查是否是超时错误（包括各种超时错误格式）
			errStr := strings.ToLower(err.Error())
			if strings.Contains(errStr, "timeout") ||
				strings.Contains(errStr, "deadline exceeded") ||
				strings.Contains(errStr, "context deadline") ||
				strings.Contains(errStr, "i/o timeout") {
				// 超时错误，返回已查询到的结果
				return allResults, nil
			}
			return nil, fmt.Errorf("请求第%d页失败: %v", page, err)
		}

		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("读取第%d页响应失败: %v", page, err)
		}

		var fofaResp struct {
			Error   bool       `json:"error"`
			ErrMsg  string     `json:"errmsg"`
			Size    int        `json:"size"`
			Results [][]string `json:"results"`
		}

		if err := json.Unmarshal(body, &fofaResp); err != nil {
			return nil, fmt.Errorf("解析第%d页响应失败: %v", page, err)
		}

		if fofaResp.Error {
			return nil, fmt.Errorf("FOFA API错误（第%d页）: %s", page, fofaResp.ErrMsg)
		}

		// 处理当前页结果
		for _, result := range fofaResp.Results {
			if len(result) >= 4 {
				item := ResultItem{
					IP:     result[1],
					Port:   result[2],
					Title:  result[3],
					Domain: result[0],
					URL:    fmt.Sprintf("http://%s:%s", result[1], result[2]),
				}
				allResults = append(allResults, item)
			}
		}

		// 如果当前页没有结果，停止查询
		if len(fofaResp.Results) == 0 {
			break
		}

		// 如果当前页返回的数据少于pageSize，说明没有更多数据了，停止查询
		// 注意：不依赖totalSize，因为API返回的Size可能受到账户权限限制（如免费账户1000条）
		// 我们应该继续查询直到达到maxPages或者没有更多数据
		if len(fofaResp.Results) < pageSize {
			// 当前页数据不足，说明已经是最后一页了
			break
		}

		// 如果不是最后一页，等待延迟时间（检查context）
		if page < maxPages && len(fofaResp.Results) > 0 {
			select {
			case <-ctx.Done():
				// 查询被取消，返回已查询到的结果
				return allResults, nil
			case <-time.After(delay):
				// 延迟完成，继续查询
			}
		}
	}

	return allResults, nil
}

// queryFofaRed 查询FOFA第三方（兼容旧接口）
func queryFofaRed(query, apiKey string, settings map[string]interface{}) ([]ResultItem, error) {
	return queryFofaRedWithContext(context.Background(), query, apiKey, settings)
}

// queryFofaRedWithContext 查询FOFA第三方（支持context取消）
func queryFofaRedWithContext(ctx context.Context, query, apiKey string, settings map[string]interface{}) ([]ResultItem, error) {
	// FOFA 第三方接口（默认使用 FOFA.RED）
	key := apiKey

	// 允许通过全局 Settings 配置 FOFA 第三方接口的 baseURL，例如:
	// https://fofa.red 或其他兼容 FOFA API 的第三方镜像
	baseURL := "https://fofa.red"
	if settings != nil {
		if v, ok := settings["fofaThirdBaseURL"]; ok {
			if s, ok2 := v.(string); ok2 && strings.TrimSpace(s) != "" {
				baseURL = strings.TrimSpace(s)
			}
		}
	}
	// 规范化 baseURL，移除结尾的斜杠，拼接 FOFA API 路径
	baseURL = strings.TrimRight(baseURL, "/") + "/api/v1/search/all"

	// 获取配置参数
	maxPages := getMaxPages(settings)
	pageSize := getPageSize(settings)
	delay := getDelay(settings)

	client := getHTTPClient(settings)
	// 获取每个请求的超时时间
	requestTimeout := getTimeout(settings)

	allResults := make([]ResultItem, 0)

	// 分页查询
	for page := 1; page <= maxPages; page++ {
		// 检查context是否被取消
		select {
		case <-ctx.Done():
			// 查询被取消，返回已查询到的结果
			return allResults, nil
		default:
		}

		// 使用url.Values来正确编码参数
		params := url.Values{}
		params.Set("key", key)
		params.Set("qbase64", base64Encode(query))
		params.Set("page", fmt.Sprintf("%d", page))
		params.Set("size", fmt.Sprintf("%d", pageSize))
		params.Set("fields", "host,ip,port,title")
		params.Set("full", "false")

		requestURL := baseURL + "?" + params.Encode()

		// 为每个请求创建独立的超时context
		requestCtx, requestCancel := context.WithTimeout(ctx, requestTimeout)
		defer requestCancel() // 确保总是取消超时context，避免泄漏
		req, err := http.NewRequestWithContext(requestCtx, "GET", requestURL, nil)
		if err != nil {
			// 如果是context取消，返回已查询到的结果
			if ctx.Err() == context.Canceled {
				return allResults, nil
			}
			return nil, fmt.Errorf("创建第%d页请求失败: %v", page, err)
		}

		resp, err := client.Do(req)
		if err != nil {
			// 检查是否是context取消
			if ctx.Err() == context.Canceled {
				return allResults, nil
			}
			// 检查是否是超时错误（包括各种超时错误格式）
			errStr := strings.ToLower(err.Error())
			if strings.Contains(errStr, "timeout") ||
				strings.Contains(errStr, "deadline exceeded") ||
				strings.Contains(errStr, "context deadline") ||
				strings.Contains(errStr, "i/o timeout") {
				// 超时错误，返回已查询到的结果
				return allResults, nil
			}
			return nil, fmt.Errorf("请求第%d页失败: %v", page, err)
		}

		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("读取第%d页响应失败: %v", page, err)
		}

		var fofaResp struct {
			Error   bool       `json:"error"`
			ErrMsg  string     `json:"errmsg"`
			Size    int        `json:"size"`
			Results [][]string `json:"results"`
		}

		if err := json.Unmarshal(body, &fofaResp); err != nil {
			return nil, fmt.Errorf("解析第%d页响应失败: %v", page, err)
		}

		if fofaResp.Error {
			return nil, fmt.Errorf("Fofa第三方API错误（第%d页）: %s", page, fofaResp.ErrMsg)
		}

		// 处理当前页结果
		for _, result := range fofaResp.Results {
			if len(result) >= 4 {
				host := result[0] // host可能是域名或完整URL
				ip := result[1]
				port := result[2]
				title := result[3]

				// 判断host是否是完整URL
				var url string
				if strings.HasPrefix(host, "http://") || strings.HasPrefix(host, "https://") {
					url = host
				} else {
					// 根据端口判断协议
					if port == "443" {
						url = fmt.Sprintf("https://%s:%s", ip, port)
					} else {
						url = fmt.Sprintf("http://%s:%s", ip, port)
					}
				}

				// 提取域名（如果host是URL，提取域名部分）
				domain := host
				if strings.HasPrefix(host, "http://") {
					domain = strings.TrimPrefix(host, "http://")
				} else if strings.HasPrefix(host, "https://") {
					domain = strings.TrimPrefix(host, "https://")
				}
				// 移除端口号
				if idx := strings.Index(domain, ":"); idx != -1 {
					domain = domain[:idx]
				}
				// 移除路径
				if idx := strings.Index(domain, "/"); idx != -1 {
					domain = domain[:idx]
				}

				item := ResultItem{
					IP:     ip,
					Port:   port,
					Title:  title,
					Domain: domain,
					URL:    url,
				}
				allResults = append(allResults, item)
			}
		}

		// 如果当前页没有结果，停止查询
		if len(fofaResp.Results) == 0 {
			break
		}

		// 如果当前页返回的数据少于pageSize，说明没有更多数据了，停止查询
		// 注意：不依赖totalSize，因为API返回的Size可能受到账户权限限制（如免费账户1000条）
		// 我们应该继续查询直到达到maxPages或者没有更多数据
		if len(fofaResp.Results) < pageSize {
			// 当前页数据不足，说明已经是最后一页了
			break
		}

		// 如果不是最后一页，等待延迟时间（检查context）
		if page < maxPages && len(fofaResp.Results) > 0 {
			select {
			case <-ctx.Done():
				// 查询被取消，返回已查询到的结果
				return allResults, nil
			case <-time.After(delay):
				// 延迟完成，继续查询
			}
		}
	}

	return allResults, nil
}

// queryHunter 查询Hunter（兼容旧接口）
func queryHunter(query, apiKey string, settings map[string]interface{}) ([]ResultItem, error) {
	return queryHunterWithContext(context.Background(), query, apiKey, settings)
}

// queryHunterWithContext 查询Hunter（支持context取消）
func queryHunterWithContext(ctx context.Context, query, apiKey string, settings map[string]interface{}) ([]ResultItem, error) {
	// Hunter API: https://hunter.qianxin.com/home/helpCenter?r=5-1-3
	baseURL := "https://hunter.qianxin.com/openApi/search"

	// 获取配置参数
	maxPages := getMaxPages(settings)
	pageSize := getPageSize(settings)
	delay := getDelay(settings)

	// Hunter API限制每页最多100条
	if pageSize > 100 {
		pageSize = 100
	}

	client := getHTTPClient(settings)
	// 获取每个请求的超时时间
	requestTimeout := getTimeout(settings)

	allResults := make([]ResultItem, 0)

	// 分页查询
	for page := 1; page <= maxPages; page++ {
		// 检查context是否被取消
		select {
		case <-ctx.Done():
			// 查询被取消，返回已查询到的结果
			return allResults, nil
		default:
		}

		params := map[string]string{
			"api-key":   apiKey,
			"search":    query,
			"page":      fmt.Sprintf("%d", page),
			"page_size": fmt.Sprintf("%d", pageSize),
		}

		url := baseURL + "?"
		for k, v := range params {
			url += fmt.Sprintf("%s=%s&", k, v)
		}

		// 为每个请求创建独立的超时context
		requestCtx, requestCancel := context.WithTimeout(ctx, requestTimeout)
		defer requestCancel() // 确保总是取消超时context，避免泄漏
		req, err := http.NewRequestWithContext(requestCtx, "GET", url, nil)
		if err != nil {
			// 如果是context取消，返回已查询到的结果
			if ctx.Err() == context.Canceled {
				return allResults, nil
			}
			return nil, fmt.Errorf("创建第%d页请求失败: %v", page, err)
		}

		resp, err := client.Do(req)
		if err != nil {
			// 检查是否是context取消
			if ctx.Err() == context.Canceled {
				return allResults, nil
			}
			// 检查是否是超时错误（包括各种超时错误格式）
			errStr := strings.ToLower(err.Error())
			if strings.Contains(errStr, "timeout") ||
				strings.Contains(errStr, "deadline exceeded") ||
				strings.Contains(errStr, "context deadline") ||
				strings.Contains(errStr, "i/o timeout") {
				// 超时错误，返回已查询到的结果
				return allResults, nil
			}
			return nil, fmt.Errorf("请求第%d页失败: %v", page, err)
		}

		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("读取第%d页响应失败: %v", page, err)
		}

		var hunterResp struct {
			Code int `json:"code"`
			Data struct {
				Total int `json:"total"`
				Arr   []struct {
					IP     string `json:"ip"`
					Port   int    `json:"port"`
					Title  string `json:"web_title"`
					Domain string `json:"domain"`
					URL    string `json:"url"`
				} `json:"arr"`
			} `json:"data"`
			Message string `json:"message"`
		}

		if err := json.Unmarshal(body, &hunterResp); err != nil {
			return nil, fmt.Errorf("解析第%d页响应失败: %v", page, err)
		}

		if hunterResp.Code != 200 {
			return nil, fmt.Errorf("Hunter API错误（第%d页）: %s", page, hunterResp.Message)
		}

		// 处理当前页结果
		for _, item := range hunterResp.Data.Arr {
			allResults = append(allResults, ResultItem{
				IP:     item.IP,
				Port:   fmt.Sprintf("%d", item.Port),
				Title:  item.Title,
				Domain: item.Domain,
				URL:    item.URL,
			})
		}

		// 如果当前页没有结果，停止查询
		if len(hunterResp.Data.Arr) == 0 {
			break
		}

		// 如果当前页返回的数据少于pageSize，说明没有更多数据了，停止查询
		// 注意：不依赖totalCount，因为API返回的Total可能受到账户权限限制
		if len(hunterResp.Data.Arr) < pageSize {
			// 当前页数据不足，说明已经是最后一页了
			break
		}

		// 如果不是最后一页，等待延迟时间
		if page < maxPages && len(hunterResp.Data.Arr) > 0 {
			time.Sleep(delay)
		}
	}

	return allResults, nil
}

// queryQuake 查询Quake（兼容旧接口）
func queryQuake(query, apiKey string, settings map[string]interface{}) ([]ResultItem, error) {
	return queryQuakeWithContext(context.Background(), query, apiKey, settings)
}

// queryQuakeWithContext 查询Quake（支持context取消）
func queryQuakeWithContext(ctx context.Context, query, apiKey string, settings map[string]interface{}) ([]ResultItem, error) {
	// Quake API: https://quake.360.net/quake/#/help?anchor=API%E6%8E%A5%E5%8F%A3
	baseURL := "https://quake.360.net/api/v3/search/quake_service"

	// 获取配置参数
	maxPages := getMaxPages(settings)
	pageSize := getPageSize(settings)
	delay := getDelay(settings)

	// Quake API限制每页最多100条
	if pageSize > 100 {
		pageSize = 100
	}

	client := getHTTPClient(settings)
	// 获取每个请求的超时时间
	requestTimeout := getTimeout(settings)

	allResults := make([]ResultItem, 0)

	// 定义Quake数据结构
	type QuakeServiceItem struct {
		IP      string `json:"ip"`
		Port    int    `json:"port"`
		Service struct {
			HTTP struct {
				Title string `json:"title"`
			} `json:"http"`
		} `json:"service"`
		Domain string `json:"domain"`
	}

	type QuakeDataObject struct {
		Total int                `json:"total"`
		List  []QuakeServiceItem `json:"list"`
	}

	// 分页查询
	for page := 1; page <= maxPages; page++ {
		// 检查context是否被取消
		select {
		case <-ctx.Done():
			// 查询被取消，返回已查询到的结果
			return allResults, nil
		default:
		}

		start := (page - 1) * pageSize
		payload := map[string]interface{}{
			"query": query,
			"start": start,
			"size":  pageSize,
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("构建第%d页请求失败: %v", page, err)
		}

		// 为每个请求创建独立的超时context
		requestCtx, requestCancel := context.WithTimeout(ctx, requestTimeout)
		defer requestCancel() // 确保总是取消超时context，避免泄漏
		req, err := http.NewRequestWithContext(requestCtx, "POST", baseURL, strings.NewReader(string(jsonData)))
		if err != nil {
			// 如果是context取消，返回已查询到的结果
			if ctx.Err() == context.Canceled {
				return allResults, nil
			}
			return nil, fmt.Errorf("创建第%d页请求失败: %v", page, err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-QuakeToken", apiKey)

		resp, err := client.Do(req)
		if err != nil {
			// 检查是否是context取消
			if ctx.Err() == context.Canceled {
				return allResults, nil
			}
			// 检查是否是超时错误（包括各种超时错误格式）
			errStr := strings.ToLower(err.Error())
			if strings.Contains(errStr, "timeout") ||
				strings.Contains(errStr, "deadline exceeded") ||
				strings.Contains(errStr, "context deadline") ||
				strings.Contains(errStr, "i/o timeout") {
				// 超时错误，返回已查询到的结果
				return allResults, nil
			}
			return nil, fmt.Errorf("请求第%d页失败: %v", page, err)
		}

		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("读取第%d页响应失败: %v", page, err)
		}

		// 先解析基础结构
		var quakeRespBase struct {
			Code    int             `json:"code"`
			Message string          `json:"message"`
			Data    json.RawMessage `json:"data"` // 使用RawMessage延迟解析
		}

		if err := json.Unmarshal(body, &quakeRespBase); err != nil {
			return nil, fmt.Errorf("解析第%d页响应失败: %v, 响应内容: %s", page, err, string(body))
		}

		if quakeRespBase.Code != 0 {
			return nil, fmt.Errorf("Quake API错误（第%d页）: %s", page, quakeRespBase.Message)
		}

		var items []QuakeServiceItem

		// 尝试解析为对象格式（包含total和list）
		var dataObj QuakeDataObject
		if err := json.Unmarshal(quakeRespBase.Data, &dataObj); err == nil && len(dataObj.List) > 0 {
			// 成功解析为对象格式
			items = dataObj.List
		} else {
			// 尝试解析为数组格式
			if err := json.Unmarshal(quakeRespBase.Data, &items); err != nil {
				return nil, fmt.Errorf("解析第%d页data字段失败: %v, data内容: %s", page, err, string(quakeRespBase.Data))
			}
		}

		// 处理当前页结果
		for _, item := range items {
			url := fmt.Sprintf("http://%s:%d", item.IP, item.Port)
			if item.Service.HTTP.Title != "" {
				url = fmt.Sprintf("https://%s:%d", item.IP, item.Port)
			}
			allResults = append(allResults, ResultItem{
				IP:     item.IP,
				Port:   fmt.Sprintf("%d", item.Port),
				Title:  item.Service.HTTP.Title,
				Domain: item.Domain,
				URL:    url,
			})
		}

		// 如果当前页没有结果，停止查询
		if len(items) == 0 {
			break
		}

		// 如果当前页返回的数据少于pageSize，说明没有更多数据了，停止查询
		// 注意：不依赖totalCount，因为API返回的Total可能受到账户权限限制
		if len(items) < pageSize {
			// 当前页数据不足，说明已经是最后一页了
			break
		}

		// 如果不是最后一页，等待延迟时间
		if page < maxPages && len(items) > 0 {
			time.Sleep(delay)
		}
	}

	return allResults, nil
}

// aggregateResults 聚合多个平台的结果
func aggregateResults(results map[string][]ResultItem) *AggregatedData {
	// 使用 IP:Port 作为唯一键进行去重
	uniqueMap := make(map[string]*AggregatedResultItem)
	platformStats := make(map[string]int)
	totalCount := 0

	// 遍历所有平台的结果
	for platform, items := range results {
		platformCount := len(items)
		platformStats[platform] = platformCount
		totalCount += platformCount

		// 处理每个结果项
		for _, item := range items {
			key := fmt.Sprintf("%s:%s", item.IP, item.Port)

			if existing, exists := uniqueMap[key]; exists {
				// 如果已存在，更新信息
				// 如果标题为空，尝试使用新标题
				if existing.Title == "" && item.Title != "" {
					existing.Title = item.Title
				}
				// 如果域名为空，尝试使用新域名
				if existing.Domain == "" && item.Domain != "" {
					existing.Domain = item.Domain
				}
				// 如果URL为空，尝试使用新URL
				if existing.URL == "" && item.URL != "" {
					existing.URL = item.URL
				}
				// 添加平台到列表（如果不存在）
				platformExists := false
				for _, p := range existing.Platforms {
					if p == platform {
						platformExists = true
						break
					}
				}
				if !platformExists {
					existing.Platforms = append(existing.Platforms, platform)
					existing.Count++
				}
			} else {
				// 创建新项
				uniqueMap[key] = &AggregatedResultItem{
					IP:        item.IP,
					Port:      item.Port,
					Title:     item.Title,
					Domain:    item.Domain,
					URL:       item.URL,
					Platforms: []string{platform},
					Count:     1,
				}
			}
		}
	}

	// 转换为切片
	aggregatedResults := make([]AggregatedResultItem, 0, len(uniqueMap))
	for _, item := range uniqueMap {
		aggregatedResults = append(aggregatedResults, *item)
	}

	return &AggregatedData{
		Total:     totalCount,
		Unique:    len(uniqueMap),
		Platforms: platformStats,
		Results:   aggregatedResults,
	}
}

// GetConfig 获取配置
func (sm *SpaceMapping) GetConfig(c *gin.Context) {
	config, err := loadConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ConfigResponse{
			Success: false,
			Message: "加载配置失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ConfigResponse{
		Success: true,
		Data:    config,
	})
}

// SaveConfig 保存配置
func (sm *SpaceMapping) SaveConfig(c *gin.Context) {
	var req ConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ConfigResponse{
			Success: false,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	if err := saveConfig(req); err != nil {
		c.JSON(http.StatusInternalServerError, ConfigResponse{
			Success: false,
			Message: "保存配置失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, ConfigResponse{
		Success: true,
		Message: "配置保存成功",
	})
}

// VerifyKeyRequest 验证Key请求
type VerifyKeyRequest struct {
	Platform string `json:"platform"`
	APIKey   string `json:"apiKey"`
	// Settings 允许前端在验证时直接传入当前界面的配置（例如 FOFA 第三方 BaseURL、代理等），
	// 这样用户无需先点击“保存”即可使用最新输入进行验证。
	Settings map[string]interface{} `json:"settings,omitempty"`
}

// VerifyKeyResponse 验证Key响应
type VerifyKeyResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Valid   bool   `json:"valid,omitempty"`
}

// VerifyKey 验证API Key是否有效
func (sm *SpaceMapping) VerifyKey(c *gin.Context) {
	var req VerifyKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, VerifyKeyResponse{
			Success: false,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	if req.Platform == "" || req.APIKey == "" {
		c.JSON(http.StatusBadRequest, VerifyKeyResponse{
			Success: false,
			Message: "平台和API Key不能为空",
		})
		return
	}

	// 默认使用全局配置中的 Settings；如果请求中携带了 Settings，则优先使用请求中的配置
	var settings map[string]interface{}
	if len(req.Settings) > 0 {
		settings = req.Settings
	} else {
		// 加载全局配置以便在验证时复用 Settings（例如 FOFA 第三方 baseURL）
		config, err := loadConfig()
		if err != nil {
			c.JSON(http.StatusInternalServerError, VerifyKeyResponse{
				Success: false,
				Message: "加载配置失败: " + err.Error(),
			})
			return
		}
		settings = config.Settings
	}

	// 使用测试查询验证Key
	valid, err := verifyAPIKey(req.Platform, req.APIKey, settings)
	if err != nil {
		c.JSON(http.StatusOK, VerifyKeyResponse{
			Success: true,
			Valid:   false,
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, VerifyKeyResponse{
		Success: true,
		Valid:   valid,
		Message: func() string {
			if valid {
				return "API Key有效"
			}
			return "API Key无效"
		}(),
	})
}

// verifyAPIKey 验证API Key
func verifyAPIKey(platform, apiKey string, configSettings map[string]interface{}) (bool, error) {
	// 使用平台兼容的简单测试查询来验证 Key，避免语法差异导致误判
	testQuery := getVerifyTestQuery(platform)
	settings := map[string]interface{}{}

	// 先拷贝一份全局配置中的 Settings，保证例如 FOFA 第三方 baseURL 生效
	for k, v := range configSettings {
		settings[k] = v
	}
	// 验证时使用轻量请求参数，并设置更稳妥的超时，减少“第一次校验失败”的冷启动误判
	timeoutSeconds := 20.0
	if cfgTimeout, ok := configSettings["timeout"].(float64); ok && cfgTimeout > timeoutSeconds {
		timeoutSeconds = cfgTimeout
	}
	settings["timeout"] = timeoutSeconds
	settings["enableProxy"] = false // 验证时不使用代理
	settings["maxPages"] = 1.0
	settings["pageSize"] = 1.0
	settings["delay"] = 0.0

	results, err := queryPlatform(platform, testQuery, apiKey, settings)
	if err != nil {
		// 检查是否是认证错误
		errMsg := strings.ToLower(err.Error())

		// 对于超时/网络抖动，提示重试而不是直接判定 key 无效
		transientErrorKeywords := []string{
			"timeout",
			"deadline exceeded",
			"context deadline",
			"i/o timeout",
			"connection reset",
			"temporary",
			"temporarily",
			"network",
		}
		for _, keyword := range transientErrorKeywords {
			if strings.Contains(errMsg, keyword) {
				return false, fmt.Errorf("验证超时或网络不稳定，请重试: %v", err)
			}
		}

		// 明确的认证失败关键词
		authErrorKeywords := []string{
			"unauthorized",
			"invalid",
			"authentication",
			"api key",
			"token",
			"认证",
			"授权",
			"forbidden",
			"access denied",
			"invalid key",
			"invalid token",
			"api key无效",
			"token无效",
		}

		for _, keyword := range authErrorKeywords {
			if strings.Contains(errMsg, keyword) {
				return false, fmt.Errorf("API Key无效: %v", err)
			}
		}

		// 对于FOFA，检查特定的错误消息
		if platform == "fofa" {
			if strings.Contains(errMsg, "fofa api错误") {
				// FOFA API返回的错误通常包含在errmsg中
				return false, fmt.Errorf("API Key无效: %v", err)
			}
		}

		// 对于 FOFA 第三方，检查特定的错误消息
		if platform == "otherfofa" {
			if strings.Contains(errMsg, "fofa api错误") {
				// FOFA 第三方 API 返回的错误通常包含在 errmsg 中
				return false, fmt.Errorf("API Key无效: %v", err)
			}
		}

		// 对于 0.zone，检查特定的错误消息（query0ZoneWithContext 中统一包装为 "0.zone API错误（第X页）: xxx"）
		if platform == "0zone" || platform == "zone0" {
			if strings.Contains(errMsg, "0.zone api错误") {
				return false, fmt.Errorf("API Key无效: %v", err)
			}
		}

		// 对于Hunter，检查code不为200的情况
		if platform == "hunter" {
			if strings.Contains(errMsg, "hunter api错误") {
				return false, fmt.Errorf("API Key无效: %v", err)
			}
		}

		// 其他错误可能是网络问题，但为了安全起见，返回false
		return false, fmt.Errorf("验证失败: %v", err)
	}

	// 如果查询成功（即使结果为空），说明Key有效
	// 某些平台可能返回空结果，但这是正常的
	_ = results
	return true, nil
}

func getVerifyTestQuery(platform string) string {
	switch platform {
	case "0zone":
		return "ip=8.8.8.8"
	case "zone0":
		return "ip=8.8.8.8"
	case "quake":
		return "ip:\"8.8.8.8\""
	default:
		return "ip=\"8.8.8.8\""
	}
}

// 辅助函数
func base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func getTimeout(settings map[string]interface{}) time.Duration {
	if timeout, ok := settings["timeout"].(float64); ok {
		return time.Duration(timeout) * time.Second
	}
	return 30 * time.Second
}

// getMaxPages 从settings获取最大页数，默认5
func getMaxPages(settings map[string]interface{}) int {
	if maxPages, ok := settings["maxPages"].(float64); ok {
		return int(maxPages)
	}
	return 5
}

// getPageSize 从settings获取每页数量，默认100，最大限制10000（FOFA API限制）
func getPageSize(settings map[string]interface{}) int {
	pageSize := 100
	if ps, ok := settings["pageSize"].(float64); ok {
		pageSize = int(ps)
	}
	// FOFA API通常限制每页最多10000条，但实际可能更少，这里设置上限为10000
	if pageSize > 10000 {
		pageSize = 10000
	}
	if pageSize < 10 {
		pageSize = 10
	}
	return pageSize
}

// getDelay 从settings获取请求延迟，默认1秒
func getDelay(settings map[string]interface{}) time.Duration {
	if delay, ok := settings["delay"].(float64); ok {
		return time.Duration(delay * float64(time.Second))
	}
	return 1 * time.Second
}

// getHTTPClient 获取HTTP客户端，根据配置决定是否使用代理
func getHTTPClient(settings map[string]interface{}) *http.Client {
	enableProxy := false
	if ep, ok := settings["enableProxy"].(bool); ok {
		enableProxy = ep
	}

	if enableProxy {
		// 使用全局代理管理器
		return proxy.GetGlobalHTTPClient()
	}

	// 不使用代理，创建普通客户端
	return &http.Client{
		Timeout: getTimeout(settings),
	}
}
