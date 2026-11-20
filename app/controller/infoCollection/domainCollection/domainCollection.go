package domainCollection

import (
	"EasyTools/app/controller/proxy"
	"EasyTools/app/controller/system"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// DomainCollection 域名收集器
type DomainCollection struct {
	system.Base
	tasks      map[string]*CollectionTask
	tasksMutex sync.RWMutex
	httpxMgr   *proxy.HTTPXManager
	portConfig *system.PortConfig // 端口配置
}

// CollectionTask 收集任务
type CollectionTask struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	RootDomain string             `json:"root_domain"`
	Mode       string             `json:"mode"`   // "dict" 或 "ai"
	Status     string             `json:"status"` // running, completed, stopped, error
	StartTime  time.Time          `json:"start_time"`
	EndTime    *time.Time         `json:"end_time,omitempty"`
	Config     *CollectionConfig  `json:"config"`
	Results    *CollectionResults `json:"results"`
	Logs       []LogEntry         `json:"logs"`
	cancelFunc context.CancelFunc
}

// CollectionConfig 收集配置
type CollectionConfig struct {
	// 字典模式配置
	DictFile    string   `json:"dict_file"`    // 字典文件路径
	DictContent []string `json:"dict_content"` // 字典内容（从文件读取）

	// AI模式配置
	AIProvider    string   `json:"ai_provider"`    // AI提供商：openai, deepseek, qianwen, custom
	AIAPIKey      string   `json:"ai_api_key"`     // AI API密钥
	AIAPIBase     string   `json:"ai_api_base"`    // AI API基础URL（自定义时使用）
	AIConfigID    string   `json:"ai_config_id"`   // AI配置ID（从系统配置中选择）
	SampleDomains []string `json:"sample_domains"` // 样本域名（用于AI学习）
	GenerateCount int      `json:"generate_count"` // AI生成数量

	// 空间测绘配置
	EnableSpaceMapping    bool     `json:"enable_space_mapping"`    // 是否启用空间测绘查询
	SpaceMappingPlatforms []string `json:"space_mapping_platforms"` // 空间测绘平台列表: fofa, hunter, quake

	// 通用配置
	Threads      int    `json:"threads"`       // 并发线程数
	Timeout      int    `json:"timeout"`       // DNS超时（秒）
	VerifyDNS    bool   `json:"verify_dns"`    // 是否验证DNS解析
	VerifyHTTP   bool   `json:"verify_http"`   // 是否验证HTTP访问
	DNSServer    string `json:"dns_server"`    // DNS服务器地址
	Wildcard     bool   `json:"wildcard"`      // 是否检测泛解析
	Ports        string `json:"ports"`         // 探测端口（逗号分隔）
	FilterIP     string `json:"filter_ip"`     // 过滤IP地址
	Fingerprint  bool   `json:"fingerprint"`   // 是否进行指纹识别
	POCDetection bool   `json:"poc_detection"` // 是否进行POC检测
	APIDetection bool   `json:"api_detection"` // 是否进行API探测
}

// CollectionResults 收集结果
type CollectionResults struct {
	TotalGenerated int          `json:"total_generated"` // 总生成数
	ValidDomains   []DomainInfo `json:"valid_domains"`   // 有效域名列表
	InvalidDomains []string     `json:"invalid_domains"` // 无效域名列表
}

// DomainInfo 域名信息
type DomainInfo struct {
	Domain     string   `json:"domain"`
	IPs        []string `json:"ips"`         // DNS解析的IP列表
	Status     string   `json:"status"`      // valid, invalid, timeout
	HTTPStatus int      `json:"http_status"` // HTTP状态码（如果验证了HTTP）
	HTTPTitle  string   `json:"http_title"`  // HTTP标题
	FoundTime  string   `json:"found_time"`  // 发现时间
	Source     string   `json:"source"`      // 域名来源：dict / ai / space-mapping / hybrid
	Platform   string   `json:"platform"`    // 空间测绘平台名称（如 fofa, hunter, quake 等，多个用逗号分隔）
	// 注意：指纹识别、接口探测、POC检测由前端调用相应模块完成，不在此处存储结果
}

// LogEntry 日志条目
type LogEntry struct {
	Time    string `json:"time"`
	Level   string `json:"level"` // info, warning, error
	Message string `json:"message"`
}

// NewDomainCollection 创建域名收集器
func NewDomainCollection() *DomainCollection {
	return &DomainCollection{
		tasks: make(map[string]*CollectionTask),
	}
}

// Startup 启动时设置上下文
func (dc *DomainCollection) Startup(ctx context.Context) {
	dc.Ctx = ctx
	// 初始化httpx管理器（如果存在）
	dc.httpxMgr = proxy.GetHTTPXManager()
	// 尝试从 System 获取端口配置
	if dc.Ctx != nil {
		// 通过 runtime 调用 System 的 GetPortConfig
		// 注意：这里需要 System 实例，可以通过全局方式获取
		// 暂时先不设置，在需要时通过调用获取
	}
}

// SetPortConfig 设置端口配置
func (dc *DomainCollection) SetPortConfig(config *system.PortConfig) {
	dc.portConfig = config
}

// getPort 获取服务器端口
func (dc *DomainCollection) getPort() int {
	if dc.portConfig != nil && dc.portConfig.MainPort > 0 {
		return dc.portConfig.MainPort
	}
	// 如果端口配置未设置，尝试从 System 获取（通过全局方式）
	// 注意：这里可能需要通过其他方式获取，暂时使用默认端口
	// 默认端口
	return 52867
}

// CreateTask 创建收集任务
func (dc *DomainCollection) CreateTask(name, rootDomain, mode string, configJSON string) (string, error) {
	dc.tasksMutex.Lock()
	defer dc.tasksMutex.Unlock()

	taskID := fmt.Sprintf("task_%d", time.Now().UnixNano())

	var config CollectionConfig
	if configJSON != "" {
		if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
			return "", fmt.Errorf("解析配置失败: %v", err)
		}
	}

	// 设置默认值
	if config.Threads <= 0 {
		config.Threads = 50
	}
	if config.Timeout <= 0 {
		config.Timeout = 5
	}
	if config.GenerateCount <= 0 {
		config.GenerateCount = 300 // 默认值改为300
	}

	task := &CollectionTask{
		ID:         taskID,
		Name:       name,
		RootDomain: rootDomain,
		Mode:       mode,
		Status:     "pending",
		StartTime:  time.Now(),
		Config:     &config,
		Results: &CollectionResults{
			ValidDomains:   []DomainInfo{},
			InvalidDomains: []string{},
		},
		Logs: []LogEntry{},
	}

	dc.tasks[taskID] = task
	return taskID, nil
}

// StartCollection 开始收集
func (dc *DomainCollection) StartCollection(taskID string) error {
	task, err := dc.GetTask(taskID)
	if err != nil {
		return err
	}

	if task.Status == "running" {
		return fmt.Errorf("任务已在运行中")
	}

	dc.tasksMutex.Lock()
	task.Status = "running"
	dc.tasksMutex.Unlock()

	dc.addLog(taskID, "info", fmt.Sprintf("开始域名收集，根域名: %s，模式: %s", task.RootDomain, task.Mode))

	// 在goroutine中执行收集
	go dc.runCollection(task)

	return nil
}

// runCollection 执行收集
func (dc *DomainCollection) runCollection(task *CollectionTask) {
	// 确保任务状态正确
	dc.tasksMutex.Lock()
	task.Status = "running"
	dc.tasksMutex.Unlock()

	dc.addLog(task.ID, "info", "=== 域名收集任务开始 ===")
	dc.addLog(task.ID, "info", fmt.Sprintf("任务ID: %s", task.ID))
	dc.addLog(task.ID, "info", fmt.Sprintf("根域名: %s", task.RootDomain))
	dc.addLog(task.ID, "info", fmt.Sprintf("收集模式: %s", task.Mode))
	dc.addLog(task.ID, "info", fmt.Sprintf("空间测绘配置: 启用=%v, 平台=%v", task.Config.EnableSpaceMapping, task.Config.SpaceMappingPlatforms))

	defer func() {
		if r := recover(); r != nil {
			dc.addLog(task.ID, "error", fmt.Sprintf("收集异常: %v", r))
			dc.tasksMutex.Lock()
			task.Status = "error"
			dc.tasksMutex.Unlock()
		}
		now := time.Now()
		dc.tasksMutex.Lock()
		task.EndTime = &now
		dc.tasksMutex.Unlock()
		dc.addLog(task.ID, "info", "=== 域名收集任务结束 ===")
	}()

	ctx, cancel := context.WithCancel(context.Background())
	dc.tasksMutex.Lock()
	task.cancelFunc = cancel
	dc.tasksMutex.Unlock()
	defer cancel()

	var domains []string
	// 记录每个域名的来源，便于前端展示
	domainSource := make(map[string]string)
	// 记录每个域名来自哪个空间测绘平台（key: 域名, value: 平台名称，多个用逗号分隔）
	domainPlatform := make(map[string]string)
	var aiGeneratedDict []string // AI生成的字典（用于追加到字典文件）

	// 根据模式生成域名
	if task.Mode == "dict" {
		// 如果启用了泛解析检测，先检测根域名是否泛解析
		if task.Config.Wildcard {
			dc.addLog(task.ID, "info", "开始检测泛解析...")
			isWildcard := dc.checkWildcard(task.RootDomain, task.Config.DNSServer, task.Config.Timeout)
			if isWildcard {
				dc.addLog(task.ID, "warning", fmt.Sprintf("根域名 %s 可能存在泛解析，结果可能不准确", task.RootDomain))
			} else {
				dc.addLog(task.ID, "info", "未检测到泛解析")
			}
		}
		domains = dc.generateDomainsFromDict(task)
		// 字典模式生成的域名来源统一标记为 dict
		for _, d := range domains {
			domainSource[d] = "dict"
		}
	} else if task.Mode == "ai" {
		// AI模式：先检查空间测绘key（如果启用）
		if task.Config.EnableSpaceMapping && len(task.Config.SpaceMappingPlatforms) > 0 {
			dc.addLog(task.ID, "info", fmt.Sprintf("检查空间测绘配置，已选择平台: %v", task.Config.SpaceMappingPlatforms))
			if err := dc.checkSpaceMappingKeys(task); err != nil {
				// 空间测绘key检查失败，记录警告但不阻止任务继续
				dc.addLog(task.ID, "warning", fmt.Sprintf("空间测绘Key检查失败: %v，将跳过空间测绘功能，继续使用AI生成", err))
				// 不设置错误状态，继续执行
			} else {
				dc.addLog(task.ID, "info", "空间测绘Key检查通过")

				// 自动调用空间测绘获取信息
				dc.addLog(task.ID, "info", "开始从空间测绘平台获取信息...")
				spaceMappingInfo, err := dc.querySpaceMappingForAI(task)
				if err != nil {
					dc.addLog(task.ID, "warning", fmt.Sprintf("空间测绘查询失败: %v，继续使用AI生成", err))
				} else if len(spaceMappingInfo) > 0 {
					// 使用空间测绘信息增强AI提示词
					dc.addLog(task.ID, "info", fmt.Sprintf("从空间测绘平台获取了 %d 条信息，用于AI分析", len(spaceMappingInfo)))
					// 将空间测绘信息添加到样本域名中
					for _, info := range spaceMappingInfo {
						if info.Domain != "" && !contains(task.Config.SampleDomains, info.Domain) {
							task.Config.SampleDomains = append(task.Config.SampleDomains, info.Domain)
						}
					}
				}
			}
		} else {
			// 即使没有启用空间测绘，也检查是否有配置的key（用于提示）
			dc.checkSpaceMappingKeysForInfo(task)

			// 检查是否有样本域名
			if len(task.Config.SampleDomains) == 0 {
				dc.addLog(task.ID, "warning", "AI模式未启用空间测绘且没有提供样本域名，AI可能无法生成有效的域名")
				dc.addLog(task.ID, "info", "建议：1) 启用空间测绘自动获取样本域名，或 2) 在配置中手动输入已有的子域名作为样本")
			} else {
				dc.addLog(task.ID, "info", fmt.Sprintf("使用 %d 个手动输入的样本域名进行AI分析", len(task.Config.SampleDomains)))
			}
		}

		// 验证是否有足够的输入数据
		hasSpaceMappingData := task.Config.EnableSpaceMapping && len(task.Config.SpaceMappingPlatforms) > 0
		hasSampleDomains := len(task.Config.SampleDomains) > 0

		if !hasSpaceMappingData && !hasSampleDomains {
			dc.addLog(task.ID, "error", "AI模式需要至少以下之一：1) 启用空间测绘并选择平台，或 2) 提供样本域名")
			task.Status = "error"
			return
		}

		var err error
		domains, err = dc.generateDomainsFromAI(task, ctx)
		if err != nil {
			dc.addLog(task.ID, "error", fmt.Sprintf("AI生成域名失败: %v", err))
			task.Status = "error"
			return
		}
		// AI 模式下生成的域名来源统一标记为 ai
		for _, d := range domains {
			domainSource[d] = "ai"
		}
		// 提取AI生成的字典前缀（用于后续追加）
		aiGeneratedDict = dc.extractDictPrefixes(domains, task.RootDomain)
	} else if task.Mode == "hybrid" {
		// 混合模式：AI生成字典 + 字典扫描
		dc.addLog(task.ID, "info", "开始混合模式：AI生成字典规则...")

		// 1. 先用AI生成字典前缀
		aiDomains, err := dc.generateDomainsFromAI(task, ctx)
		if err != nil {
			dc.addLog(task.ID, "warning", fmt.Sprintf("AI生成字典失败: %v，继续使用字典模式", err))
		} else {
			aiGeneratedDict = dc.extractDictPrefixes(aiDomains, task.RootDomain)
			dc.addLog(task.ID, "info", fmt.Sprintf("AI生成了 %d 个字典前缀", len(aiGeneratedDict)))

			// 追加AI生成的字典到配置中
			if len(task.Config.DictContent) == 0 {
				task.Config.DictContent = aiGeneratedDict
			} else {
				// 合并并去重
				dictMap := make(map[string]bool)
				for _, word := range task.Config.DictContent {
					dictMap[word] = true
				}
				for _, word := range aiGeneratedDict {
					if !dictMap[word] {
						task.Config.DictContent = append(task.Config.DictContent, word)
						dictMap[word] = true
					}
				}
			}
		}

		// 2. 使用合并后的字典进行扫描
		domains = dc.generateDomainsFromDict(task)
		dc.addLog(task.ID, "info", fmt.Sprintf("字典模式生成了 %d 个候选域名", len(domains)))
		// 混合模式下的域名统一标记为 hybrid，便于前端识别
		for _, d := range domains {
			domainSource[d] = "hybrid"
		}
	} else {
		dc.addLog(task.ID, "error", fmt.Sprintf("未知模式: %s", task.Mode))
		task.Status = "error"
		return
	}

	// 如果启用了空间测绘（非AI模式），从空间测绘平台获取域名
	dc.addLog(task.ID, "info", fmt.Sprintf("检查空间测绘条件: EnableSpaceMapping=%v, Platforms=%v (len=%d), Mode=%s",
		task.Config.EnableSpaceMapping, task.Config.SpaceMappingPlatforms, len(task.Config.SpaceMappingPlatforms), task.Mode))

	if task.Config.EnableSpaceMapping && len(task.Config.SpaceMappingPlatforms) > 0 && task.Mode != "ai" {
		// 检查空间测绘key（但不阻止查询）
		dc.addLog(task.ID, "info", fmt.Sprintf("检查空间测绘配置，已选择平台: %v", task.Config.SpaceMappingPlatforms))
		if err := dc.checkSpaceMappingKeys(task); err != nil {
			// 空间测绘key检查失败，记录警告但继续尝试查询（可能只是端口问题）
			dc.addLog(task.ID, "warning", fmt.Sprintf("空间测绘Key检查失败: %v，将尝试继续查询（可能是端口配置问题）", err))
		} else {
			dc.addLog(task.ID, "info", "空间测绘Key检查通过")
		}

		// 无论key检查是否通过，都尝试查询（因为可能只是端口配置问题）
		dc.addLog(task.ID, "info", "开始从空间测绘平台查询域名...")
		spaceMappingDomains, spaceMappingPlatformMap, err := dc.querySpaceMappingWithPlatform(task)
		if err != nil {
			dc.addLog(task.ID, "warning", fmt.Sprintf("空间测绘查询失败: %v", err))
			dc.addLog(task.ID, "info", "空间测绘查询失败，将继续使用字典模式")
		} else {
			dc.addLog(task.ID, "info", fmt.Sprintf("空间测绘查询成功，获取到 %d 个域名", len(spaceMappingDomains)))
			if len(spaceMappingDomains) > 0 {
				// 显示前几个域名作为示例
				sampleCount := 5
				if len(spaceMappingDomains) < sampleCount {
					sampleCount = len(spaceMappingDomains)
				}
				dc.addLog(task.ID, "info", fmt.Sprintf("空间测绘域名示例（前%d个）: %v", sampleCount, spaceMappingDomains[:sampleCount]))
			}

			// 合并空间测绘获取的域名，去重
			domainMap := make(map[string]bool)
			for _, d := range domains {
				domainMap[d] = true
			}

			newDomainCount := 0
			existingDomainCount := 0
			for _, d := range spaceMappingDomains {
				if !domainMap[d] {
					// 新域名，添加到列表
					domains = append(domains, d)
					domainMap[d] = true
					// 从空间测绘获取的域名标记为 space-mapping
					domainSource[d] = "space-mapping"
					// 记录平台名称
					if platform, ok := spaceMappingPlatformMap[d]; ok {
						domainPlatform[d] = platform
					}
					newDomainCount++
				} else {
					// 如果域名已存在（在字典中），同时标记多个来源
					existingSource := domainSource[d]
					if existingSource == "" {
						domainSource[d] = "space-mapping"
					} else if existingSource != "space-mapping" && !strings.Contains(existingSource, "space-mapping") {
						// 如果已有其他来源（如dict），则组合显示
						domainSource[d] = existingSource + ",space-mapping"
					}
					// 合并平台信息
					if platform, ok := spaceMappingPlatformMap[d]; ok {
						existingPlatform := domainPlatform[d]
						if existingPlatform == "" {
							domainPlatform[d] = platform
						} else if !strings.Contains(existingPlatform, platform) {
							domainPlatform[d] = existingPlatform + "," + platform
						}
					}
					existingDomainCount++
				}
			}
			dc.addLog(task.ID, "info", fmt.Sprintf("空间测绘域名处理完成：新增 %d 个，与字典重复 %d 个，合并后共 %d 个候选域名", newDomainCount, existingDomainCount, len(domains)))
		}
	}

	task.Results.TotalGenerated = len(domains)
	dc.addLog(task.ID, "info", fmt.Sprintf("生成 %d 个候选域名", len(domains)))

	// 验证域名
	if task.Config.VerifyDNS || task.Config.VerifyHTTP {
		dc.verifyDomains(task, domains, domainSource, domainPlatform, ctx)
	} else {
		// 不验证，直接添加到结果
		for _, domain := range domains {
			source := domainSource[domain]
			if source == "" {
				// 回退到模式推断
				switch task.Mode {
				case "ai":
					source = "ai"
				case "hybrid":
					source = "hybrid"
				default:
					source = "dict"
				}
			}
			platform := domainPlatform[domain]
			task.Results.ValidDomains = append(task.Results.ValidDomains, DomainInfo{
				Domain:    domain,
				Status:    "valid",
				FoundTime: time.Now().Format("2006-01-02 15:04:05"),
				Source:    source,
				Platform:  platform,
			})
		}
	}

	// 统一为已验证结果补充来源字段和平台信息
	if task.Results != nil && len(task.Results.ValidDomains) > 0 {
		for i, info := range task.Results.ValidDomains {
			updated := false
			if info.Source == "" {
				if src, ok := domainSource[info.Domain]; ok && src != "" {
					info.Source = src
					updated = true
				} else {
					// 回退到模式推断
					switch task.Mode {
					case "ai":
						info.Source = "ai"
					case "hybrid":
						info.Source = "hybrid"
					default:
						info.Source = "dict"
					}
					updated = true
				}
			}
			// 补充平台信息
			if info.Platform == "" {
				if platform, ok := domainPlatform[info.Domain]; ok && platform != "" {
					info.Platform = platform
					updated = true
				}
			}
			if updated {
				task.Results.ValidDomains[i] = info
			}
		}
	}

	// 如果AI生成了字典，追加到字典文件
	if len(aiGeneratedDict) > 0 && task.Config.DictFile != "" {
		dc.appendDictToFile(task.Config.DictFile, aiGeneratedDict)
		dc.addLog(task.ID, "info", fmt.Sprintf("已将 %d 个AI生成的字典前缀追加到字典文件", len(aiGeneratedDict)))
	}

	dc.tasksMutex.Lock()
	task.Status = "completed"
	dc.tasksMutex.Unlock()
	dc.addLog(task.ID, "info", fmt.Sprintf("收集完成：生成 %d 个，有效 %d 个，无效 %d 个",
		task.Results.TotalGenerated, len(task.Results.ValidDomains), len(task.Results.InvalidDomains)))

	// 保存任务到文件
	go func() {
		if err := dc.SaveTaskLog(task.ID); err != nil {
			dc.addLog(task.ID, "warning", fmt.Sprintf("保存任务日志失败: %v", err))
		}
	}()
}

// extractDictPrefixes 从域名列表中提取字典前缀
func (dc *DomainCollection) extractDictPrefixes(domains []string, rootDomain string) []string {
	prefixes := make(map[string]bool)
	for _, domain := range domains {
		// 移除根域名，提取前缀
		if strings.HasSuffix(domain, "."+rootDomain) {
			prefix := strings.TrimSuffix(domain, "."+rootDomain)
			prefix = strings.Trim(prefix, ".")
			if prefix != "" {
				prefixes[prefix] = true
			}
		} else if domain == rootDomain {
			// 根域名本身，跳过
			continue
		}
	}

	result := make([]string, 0, len(prefixes))
	for prefix := range prefixes {
		result = append(result, prefix)
	}
	return result
}

// appendDictToFile 追加字典到文件
func (dc *DomainCollection) appendDictToFile(filePath string, newWords []string) error {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// 读取现有内容，避免重复
	existingWords := make(map[string]bool)
	if existing, err := dc.loadDictFile(filePath); err == nil {
		for _, word := range existing {
			existingWords[word] = true
		}
	}

	// 追加新词
	for _, word := range newWords {
		word = strings.TrimSpace(word)
		if word != "" && !existingWords[word] {
			file.WriteString(word + "\n")
			existingWords[word] = true
		}
	}

	return nil
}

// generateDomainsFromDict 从字典生成域名
func (dc *DomainCollection) generateDomainsFromDict(task *CollectionTask) []string {
	var dictWords []string

	// 从文件读取字典
	if task.Config.DictFile != "" {
		// 如果DictFile是文件名（不包含路径），则从domain目录查找
		dictFilePath := task.Config.DictFile
		if !filepath.IsAbs(dictFilePath) && !strings.Contains(dictFilePath, string(filepath.Separator)) {
			// 是文件名，从domain目录查找
			baseDir := system.GetAppBaseDir()
			dictDir := filepath.Join(baseDir, "tools", "dict", "domain")
			dictFilePath = filepath.Join(dictDir, dictFilePath)
		}

		words, err := dc.loadDictFile(dictFilePath)
		if err != nil {
			dc.addLog(task.ID, "warning", fmt.Sprintf("读取字典文件失败: %v，使用配置的字典内容", err))
		} else {
			dictWords = words
			dc.addLog(task.ID, "info", fmt.Sprintf("从字典文件加载了 %d 个词条", len(dictWords)))
		}
	}

	// 如果文件读取失败，使用配置的字典内容
	if len(dictWords) == 0 && len(task.Config.DictContent) > 0 {
		dictWords = task.Config.DictContent
		dc.addLog(task.ID, "info", fmt.Sprintf("使用配置的字典内容，共 %d 个词条", len(dictWords)))
	}

	// 如果还是没有，使用默认字典
	if len(dictWords) == 0 {
		dictWords = dc.getDefaultDict()
		dc.addLog(task.ID, "info", "使用默认字典")
	}

	// 生成域名
	var domains []string
	rootDomain := strings.ToLower(strings.TrimSpace(task.RootDomain))

	if rootDomain == "" {
		dc.addLog(task.ID, "error", "根域名为空，无法生成域名")
		return domains
	}

	dc.addLog(task.ID, "info", fmt.Sprintf("开始从 %d 个字典词条生成域名，根域名: %s", len(dictWords), rootDomain))

	domainMap := make(map[string]bool) // 用于去重
	for _, word := range dictWords {
		word = strings.TrimSpace(word)
		if word == "" {
			continue
		}
		// 生成子域名：word.rootdomain
		subdomain := fmt.Sprintf("%s.%s", word, rootDomain)
		// 去重
		if !domainMap[subdomain] {
			domains = append(domains, subdomain)
			domainMap[subdomain] = true
		}
	}

	dc.addLog(task.ID, "info", fmt.Sprintf("字典模式生成域名完成，共生成 %d 个唯一域名", len(domains)))
	if len(domains) > 0 {
		// 显示前几个域名作为示例
		sampleCount := 10
		if len(domains) < sampleCount {
			sampleCount = len(domains)
		}
		dc.addLog(task.ID, "info", fmt.Sprintf("示例域名（前%d个）: %v", sampleCount, domains[:sampleCount]))

		// 特别检查用户提到的域名
		testDomains := []string{
			fmt.Sprintf("www.%s", rootDomain),
			fmt.Sprintf("ip.%s", rootDomain),
			fmt.Sprintf("b.%s", rootDomain),
		}
		for _, testDomain := range testDomains {
			found := false
			for _, d := range domains {
				if d == testDomain {
					found = true
					break
				}
			}
			if found {
				dc.addLog(task.ID, "info", fmt.Sprintf("✓ 已生成测试域名: %s", testDomain))
			} else {
				dc.addLog(task.ID, "warning", fmt.Sprintf("✗ 未生成测试域名: %s", testDomain))
			}
		}
	} else {
		dc.addLog(task.ID, "error", "警告：未生成任何域名！请检查字典文件或配置")
	}

	return domains
}

// generateDomainsFromAI 使用AI生成域名
func (dc *DomainCollection) generateDomainsFromAI(task *CollectionTask, ctx context.Context) ([]string, error) {
	dc.addLog(task.ID, "info", "开始调用AI生成域名规则...")

	// 构建AI提示词
	prompt := dc.buildAIPrompt(task)

	// 调用AI API
	aiDomains, err := dc.callAIAPI(task, prompt, ctx)
	if err != nil {
		return nil, err
	}

	// 解析AI返回的域名
	domains := dc.parseAIResponse(aiDomains, task.RootDomain)

	dc.addLog(task.ID, "info", fmt.Sprintf("AI生成了 %d 个域名规则", len(domains)))
	return domains, nil
}

// buildAIPrompt 构建AI提示词
func (dc *DomainCollection) buildAIPrompt(task *CollectionTask) string {
	var prompt strings.Builder

	// 确保生成数量有效
	generateCount := task.Config.GenerateCount
	if generateCount <= 0 {
		generateCount = 300
	}

	prompt.WriteString("你是一位专业的网络安全研究员和域名收集专家，擅长分析目标域名的命名规律并生成高质量的子域名候选列表。\n\n")

	// 如果有样本域名，添加样本域名列表
	if len(task.Config.SampleDomains) > 0 {
		prompt.WriteString("## 已知子域名样本\n")
		prompt.WriteString("以下是已发现的子域名列表，请仔细分析它们的命名规律：\n\n")
		for i, domain := range task.Config.SampleDomains {
			// 提取子域名前缀
			prefix := strings.TrimSuffix(domain, "."+task.RootDomain)
			prefix = strings.TrimSuffix(prefix, task.RootDomain)
			if prefix != "" {
				prompt.WriteString(fmt.Sprintf("%d. %s (完整域名: %s)\n", i+1, prefix, domain))
			} else {
				prompt.WriteString(fmt.Sprintf("%d. %s\n", i+1, domain))
			}
		}
		prompt.WriteString("\n")
		prompt.WriteString("请分析这些样本域名的命名特点，包括：\n")
		prompt.WriteString("- 使用的单词类型（技术术语、业务词汇、缩写等）\n")
		prompt.WriteString("- 命名风格（全小写、驼峰、连字符等）\n")
		prompt.WriteString("- 常见模式（api、admin、test、dev、www等）\n")
		prompt.WriteString("- 业务相关词汇（mail、shop、blog、cdn等）\n\n")
	} else {
		prompt.WriteString("## 提示\n")
		prompt.WriteString("没有提供样本域名，请基于常见的子域名命名规律生成。\n\n")
	}

	prompt.WriteString("## 目标信息\n")
	prompt.WriteString(fmt.Sprintf("根域名：%s\n", task.RootDomain))
	prompt.WriteString(fmt.Sprintf("需要生成数量：%d 个子域名前缀（必须生成至少 %d 个，不能少于这个数量）\n\n", generateCount, generateCount))

	prompt.WriteString("## 任务要求\n")
	prompt.WriteString("请根据上述分析，生成可能的子域名前缀列表。要求如下：\n\n")
	prompt.WriteString(fmt.Sprintf("1. **严格遵循数量要求**：必须生成至少 %d 个子域名前缀，不能少于这个数量。如果生成的域名少于要求数量，请继续生成更多，直到达到或超过 %d 个。\n", generateCount, generateCount))
	prompt.WriteString("2. **遵循命名规律**：如果提供了样本域名，必须严格遵循其命名模式和风格\n")
	prompt.WriteString("3. **多样性**：生成的前缀应该多样化，包括但不限于：\n")
	prompt.WriteString("   - 技术相关：api, admin, test, dev, staging, prod, backup, db, redis, mysql等\n")
	prompt.WriteString("   - 业务相关：www, mail, shop, blog, news, forum, cdn, static, assets等\n")
	prompt.WriteString("   - 环境相关：internal, private, public, demo, beta, alpha等\n")
	prompt.WriteString("   - 服务相关：ftp, smtp, imap, vpn, proxy, gateway等\n")
	prompt.WriteString("   - 数字组合：01, 02, 1, 2, test1, test2等\n")
	prompt.WriteString("   - 年份日期：2024, 2023, 2022等\n")
	prompt.WriteString("4. **真实性**：生成的前缀应该是有意义的、符合实际使用场景的单词或组合\n")
	prompt.WriteString("5. **避免重复**：不要生成与样本域名完全相同的前缀\n")
	prompt.WriteString("6. **格式要求**：\n")
	prompt.WriteString("   - 只返回前缀，不要包含根域名\n")
	prompt.WriteString("   - 每行一个前缀\n")
	prompt.WriteString("   - 不要包含编号、符号或任何解释文字\n")
	prompt.WriteString("   - 使用小写字母\n")
	prompt.WriteString(fmt.Sprintf("   - 必须生成至少 %d 行前缀\n\n", generateCount))

	prompt.WriteString("## 输出格式示例\n")
	prompt.WriteString("```\n")
	prompt.WriteString("api\n")
	prompt.WriteString("admin\n")
	prompt.WriteString("test\n")
	prompt.WriteString("dev\n")
	prompt.WriteString("staging\n")
	prompt.WriteString("```\n\n")

	prompt.WriteString("现在请开始生成子域名前缀列表：\n")

	return prompt.String()
}

// callAIAPI 调用AI API
func (dc *DomainCollection) callAIAPI(task *CollectionTask, prompt string, ctx context.Context) (string, error) {
	// 如果配置了AI配置ID，从系统配置中加载
	if task.Config.AIConfigID != "" {
		aiConfig, err := dc.loadAIConfig(task.Config.AIConfigID)
		if err == nil && aiConfig != nil {
			// 使用系统配置中的设置
			task.Config.AIProvider = aiConfig.Provider
			task.Config.AIAPIKey = aiConfig.APIKey
			if aiConfig.APIBase != "" {
				task.Config.AIAPIBase = aiConfig.APIBase
			}
			// 如果系统配置中有模型名称，也使用它（但域名收集功能中模型名称由各API函数内部决定）
		}
	}

	provider := task.Config.AIProvider
	if provider == "" {
		provider = "openai"
	}

	switch provider {
	case "openai":
		return dc.callOpenAI(task, prompt, ctx)
	case "deepseek":
		return dc.callDeepSeekAPI(task, prompt, ctx)
	case "qianwen":
		return dc.callQianwenAPI(task, prompt, ctx)
	case "custom":
		return dc.callCustomAI(task, prompt, ctx)
	default:
		return "", fmt.Errorf("不支持的AI提供商: %s", provider)
	}
}

// AIConfig 内部AI配置结构（避免循环依赖）
type internalAIConfig struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Provider    string `json:"provider"`
	APIKey      string `json:"api_key"`
	APIBase     string `json:"api_base"`
	Model       string `json:"model"`
	Enabled     bool   `json:"enabled"`
	IsDefault   bool   `json:"is_default"`
	Description string `json:"description"`
}

// loadAIConfig 从系统配置加载AI配置
func (dc *DomainCollection) loadAIConfig(configID string) (*internalAIConfig, error) {
	acm := system.GetAIConfigManager()
	configJSON, err := acm.GetConfig(configID)
	if err != nil {
		return nil, err
	}

	var config internalAIConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, err
	}

	if !config.Enabled {
		return nil, fmt.Errorf("AI配置未启用")
	}

	return &config, nil
}

// callOpenAI 调用OpenAI API
func (dc *DomainCollection) callOpenAI(task *CollectionTask, prompt string, ctx context.Context) (string, error) {
	apiKey := task.Config.AIAPIKey
	if apiKey == "" {
		return "", fmt.Errorf("OpenAI API Key未配置")
	}

	apiURL := "https://api.openai.com/v1/chat/completions"
	if task.Config.AIAPIBase != "" {
		apiURL = task.Config.AIAPIBase
	}

	// 构建请求
	reqBody := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"max_tokens":  2000,
		"temperature": 0.7,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// 发送请求（使用代理配置）
	client := proxy.GlobalProxyManager.GetHTTPClient()
	client.Timeout = 60 * time.Second
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("AI API返回错误状态码: %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	// 提取响应内容
	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("AI API返回格式错误")
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("AI API返回格式错误")
	}

	message, ok := choice["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("AI API返回格式错误")
	}

	content, ok := message["content"].(string)
	if !ok {
		return "", fmt.Errorf("AI API返回格式错误")
	}

	return content, nil
}

// callDeepSeekAPI 调用 DeepSeek API
func (dc *DomainCollection) callDeepSeekAPI(task *CollectionTask, prompt string, ctx context.Context) (string, error) {
	apiKey := task.Config.AIAPIKey
	if apiKey == "" {
		return "", fmt.Errorf("DeepSeek API Key未配置")
	}

	apiURL := "https://api.deepseek.com/v1/chat/completions"
	if task.Config.AIAPIBase != "" {
		apiURL = task.Config.AIAPIBase
	}

	// 从系统配置中获取模型名称（如果配置了）
	modelName := "deepseek-chat" // DeepSeek 默认模型
	if task.Config.AIConfigID != "" {
		aiConfig, err := dc.loadAIConfig(task.Config.AIConfigID)
		if err == nil && aiConfig != nil && aiConfig.Model != "" {
			modelName = aiConfig.Model
		}
	}

	// 构建请求（DeepSeek API 格式与 OpenAI 兼容）
	reqBody := map[string]interface{}{
		"model": modelName,
		"messages": []map[string]string{
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"max_tokens":  2000,
		"temperature": 0.7,
		"stream":      false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// 发送请求（使用代理配置）
	client := proxy.GlobalProxyManager.GetHTTPClient()
	client.Timeout = 60 * time.Second
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("DeepSeek API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	// 提取响应内容（DeepSeek API 格式与 OpenAI 兼容）
	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("DeepSeek API返回格式错误")
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("DeepSeek API返回格式错误")
	}

	message, ok := choice["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("DeepSeek API返回格式错误")
	}

	content, ok := message["content"].(string)
	if !ok {
		return "", fmt.Errorf("DeepSeek API返回格式错误")
	}

	return content, nil
}

// callQianwenAPI 调用阿里云千机百炼 API
func (dc *DomainCollection) callQianwenAPI(task *CollectionTask, prompt string, ctx context.Context) (string, error) {
	apiKey := task.Config.AIAPIKey
	if apiKey == "" {
		return "", fmt.Errorf("阿里云千机百炼 API Key未配置")
	}

	apiURL := "https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation"
	if task.Config.AIAPIBase != "" {
		apiURL = task.Config.AIAPIBase
	}

	// 从系统配置中获取模型名称（如果配置了）
	modelName := "qwen-turbo" // 默认模型
	if task.Config.AIConfigID != "" {
		aiConfig, err := dc.loadAIConfig(task.Config.AIConfigID)
		if err == nil && aiConfig != nil && aiConfig.Model != "" {
			modelName = aiConfig.Model
		}
	}

	// 构建请求（阿里云千机百炼 API 格式）
	reqBody := map[string]interface{}{
		"model": modelName,
		"input": map[string]interface{}{
			"messages": []map[string]string{
				{
					"role":    "user",
					"content": prompt,
				},
			},
		},
		"parameters": map[string]interface{}{
			"max_tokens":  2000,
			"temperature": 0.7,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// 发送请求（使用代理配置）
	client := proxy.GlobalProxyManager.GetHTTPClient()
	client.Timeout = 60 * time.Second
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("阿里云千机百炼 API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	// 提取响应内容（阿里云千机百炼 API 格式）
	output, ok := result["output"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("阿里云千机百炼 API返回格式错误: 缺少output字段")
	}

	choices, ok := output["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("阿里云千机百炼 API返回格式错误: 缺少choices字段")
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("阿里云千机百炼 API返回格式错误")
	}

	message, ok := choice["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("阿里云千机百炼 API返回格式错误")
	}

	content, ok := message["content"].(string)
	if !ok {
		return "", fmt.Errorf("阿里云千机百炼 API返回格式错误")
	}

	return content, nil
}

// callCustomAI 调用自定义AI API（兼容OpenAI格式）
func (dc *DomainCollection) callCustomAI(task *CollectionTask, prompt string, ctx context.Context) (string, error) {
	if task.Config.AIAPIBase == "" {
		return "", fmt.Errorf("自定义AI API Base URL未配置")
	}

	// 使用与OpenAI相同的格式
	return dc.callOpenAI(task, prompt, ctx)
}

// parseAIResponse 解析AI响应
func (dc *DomainCollection) parseAIResponse(response string, rootDomain string) []string {
	var domains []string
	lines := strings.Split(response, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 移除可能的编号、符号等
		line = regexp.MustCompile(`^\d+[\.\)]\s*`).ReplaceAllString(line, "")
		line = strings.TrimSpace(line)

		// 如果包含根域名，提取前缀
		if strings.Contains(line, rootDomain) {
			parts := strings.Split(line, rootDomain)
			if len(parts) > 0 {
				prefix := strings.Trim(strings.TrimSpace(parts[0]), ".")
				if prefix != "" {
					domains = append(domains, fmt.Sprintf("%s.%s", prefix, rootDomain))
				}
			}
		} else if !strings.Contains(line, ".") {
			// 如果不包含点，可能是前缀
			domains = append(domains, fmt.Sprintf("%s.%s", line, rootDomain))
		}
	}

	return domains
}

// verifyDomains 验证域名
func (dc *DomainCollection) verifyDomains(task *CollectionTask, domains []string, domainSource map[string]string, domainPlatform map[string]string, ctx context.Context) {
	dc.addLog(task.ID, "info", fmt.Sprintf("开始验证 %d 个域名...", len(domains)))
	dnsServerInfo := task.Config.DNSServer
	if dnsServerInfo == "" {
		dnsServerInfo = "系统默认DNS"
	}
	dc.addLog(task.ID, "info", fmt.Sprintf("验证配置 - DNS验证: %v, HTTP验证: %v, DNS服务器: %s, 超时: %ds",
		task.Config.VerifyDNS, task.Config.VerifyHTTP, dnsServerInfo, task.Config.Timeout))

	semaphore := make(chan struct{}, task.Config.Threads)
	var wg sync.WaitGroup
	verifiedCount := 0
	var verifiedMutex sync.Mutex

	for _, domain := range domains {
		// 检查是否已取消
		select {
		case <-ctx.Done():
			dc.addLog(task.ID, "info", "域名验证已取消")
			return
		default:
		}

		wg.Add(1)
		go func(d string) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 再次检查是否已取消
			select {
			case <-ctx.Done():
				return
			default:
			}

			domainInfo := dc.verifySingleDomain(task, d, domainSource, domainPlatform)

			// 检查是否已取消（在添加结果前）
			select {
			case <-ctx.Done():
				return
			default:
			}

			if domainInfo.Status == "valid" {
				dc.tasksMutex.Lock()
				task.Results.ValidDomains = append(task.Results.ValidDomains, domainInfo)
				dc.tasksMutex.Unlock()
				dc.addLog(task.ID, "info", fmt.Sprintf("发现有效域名: %s", d))
			} else {
				dc.tasksMutex.Lock()
				task.Results.InvalidDomains = append(task.Results.InvalidDomains, d)
				dc.tasksMutex.Unlock()
			}

			// 更新已验证计数
			verifiedMutex.Lock()
			verifiedCount++
			// 每验证10%或每100个域名记录一次进度
			if verifiedCount%100 == 0 || verifiedCount == len(domains) || (len(domains) > 0 && verifiedCount*10 >= len(domains)*9) {
				percentage := (verifiedCount * 100) / len(domains)
				dc.addLog(task.ID, "info", fmt.Sprintf("验证进度: %d/%d (%d%%)", verifiedCount, len(domains), percentage))
			}
			verifiedMutex.Unlock()
		}(domain)
	}

	wg.Wait()

	// 检查是否在等待过程中被取消
	select {
	case <-ctx.Done():
		dc.addLog(task.ID, "info", "域名验证已停止")
		return
	default:
	}

	dc.addLog(task.ID, "info", fmt.Sprintf("域名验证完成: 有效 %d 个，无效 %d 个",
		len(task.Results.ValidDomains), len(task.Results.InvalidDomains)))
}

// verifySingleDomain 验证单个域名
func (dc *DomainCollection) verifySingleDomain(task *CollectionTask, domain string, domainSource map[string]string, domainPlatform map[string]string) DomainInfo {
	// 优先从domainSource map中获取来源，如果没有则根据任务模式推断
	source := domainSource[domain]
	if source == "" {
		switch task.Mode {
		case "ai":
			source = "ai"
		case "hybrid":
			source = "hybrid"
		default:
			source = "dict"
		}
	}

	// 获取平台名称
	platform := domainPlatform[domain]

	info := DomainInfo{
		Domain:    domain,
		Status:    "invalid",
		FoundTime: time.Now().Format("2006-01-02 15:04:05"),
		Source:    source,
		Platform:  platform,
	}

	// DNS验证
	if task.Config.VerifyDNS {
		ips, err := dc.resolveDNS(domain, task.Config.Timeout, task.Config.DNSServer)
		if err != nil {
			// DNS解析失败，记录详细错误日志
			dc.addLog(task.ID, "warning", fmt.Sprintf("域名 %s DNS解析失败: %v", domain, err))
			return info
		}

		if len(ips) == 0 {
			return info
		}

		// 过滤IP（如果配置了）
		if task.Config.FilterIP != "" {
			filteredIPs := []string{}
			for _, ip := range ips {
				if ip != task.Config.FilterIP {
					filteredIPs = append(filteredIPs, ip)
				}
			}
			if len(filteredIPs) == 0 {
				// 所有IP都被过滤，视为无效
				return info
			}
			ips = filteredIPs
		}

		info.IPs = ips
		info.Status = "valid"
	} else {
		// 如果没有启用DNS验证，直接标记为有效
		info.Status = "valid"
	}

	// HTTP验证（始终开启，用于获取HTTP状态码和标题）
	// 配置项中的fingerprint、api_detection、poc_detection只控制是否自动推送到相应模块
	if info.Status == "valid" {
		statusCode, title, _ := dc.verifyHTTP(domain, task.Config.Timeout, task.Config.Ports)
		info.HTTPStatus = statusCode
		info.HTTPTitle = title

		// 如果HTTP验证成功，根据配置项决定是否自动推送到相应模块
		if statusCode > 0 {
			if task.Config.Fingerprint {
				dc.addLog(task.ID, "info", fmt.Sprintf("域名 %s HTTP验证成功，将自动推送到指纹识别模块", domain))
			}
			if task.Config.APIDetection {
				dc.addLog(task.ID, "info", fmt.Sprintf("域名 %s HTTP验证成功，将自动推送到浏览器扫描模块进行接口探测", domain))
			}
			if task.Config.POCDetection {
				dc.addLog(task.ID, "info", fmt.Sprintf("域名 %s HTTP验证成功，将自动推送到漏洞扫描模块进行POC检测", domain))
			}
		}
	}

	return info
}

// resolveDNS 解析DNS
func (dc *DomainCollection) resolveDNS(domain string, timeout int, dnsServer string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	var ips []string

	// 优先使用用户配置的 DNS 服务器
	if dnsServer != "" {
		dialer := &net.Dialer{
			Timeout: time.Duration(timeout) * time.Second,
		}
		r := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				// 使用配置的DNS服务器
				return dialer.DialContext(ctx, network, net.JoinHostPort(dnsServer, "53"))
			},
		}

		addrs, err := r.LookupHost(ctx, domain)
		if err == nil && len(addrs) > 0 {
			ips = append(ips, addrs...)
			return ips, nil
		}

		// 如果自定义 DNS 解析失败，回退到系统 DNS（不记录日志，避免日志过多）
	}

	// 使用系统默认DNS作为回退
	r := &net.Resolver{}
	addrs, err := r.LookupHost(ctx, domain)
	if err != nil {
		return nil, err
	}

	ips = append(ips, addrs...)
	return ips, nil
}

// verifyHTTP 验证HTTP访问，返回状态码、标题和可访问的URL
func (dc *DomainCollection) verifyHTTP(domain string, timeout int, ports string) (int, string, string) {
	// 构建URL列表
	urls := []string{}

	// 如果配置了端口，使用配置的端口
	if ports != "" {
		portList := strings.Split(ports, ",")
		for _, port := range portList {
			port = strings.TrimSpace(port)
			if port != "" {
				urls = append(urls, fmt.Sprintf("https://%s:%s", domain, port))
				urls = append(urls, fmt.Sprintf("http://%s:%s", domain, port))
			}
		}
	}

	// 如果没有配置端口或端口列表为空，使用默认端口
	if len(urls) == 0 {
		urls = []string{
			fmt.Sprintf("https://%s", domain),
			fmt.Sprintf("http://%s", domain),
		}
	}

	// 使用代理配置
	client := proxy.GlobalProxyManager.GetHTTPClient()
	client.Timeout = time.Duration(timeout) * time.Second

	for _, u := range urls {
		resp, err := client.Get(u)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		// 读取标题（简化版，只读取前500字节）
		title := ""
		if resp.StatusCode == http.StatusOK {
			buf := make([]byte, 500)
			n, _ := resp.Body.Read(buf)
			if n > 0 {
				content := string(buf[:n])
				// 简单提取title标签
				titleMatch := regexp.MustCompile(`<title[^>]*>([^<]+)</title>`).FindStringSubmatch(content)
				if len(titleMatch) > 1 {
					title = strings.TrimSpace(titleMatch[1])
				}
			}
		}

		return resp.StatusCode, title, u
	}

	return 0, "", ""
}

// loadDictFile 加载字典文件
func (dc *DomainCollection) loadDictFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var words []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			words = append(words, line)
		}
	}

	return words, scanner.Err()
}

// getDefaultDict 获取默认字典
func (dc *DomainCollection) getDefaultDict() []string {
	return []string{
		"www", "api", "admin", "test", "dev", "staging", "prod", "production",
		"mail", "email", "smtp", "pop", "imap", "ftp", "sftp", "ssh",
		"db", "database", "mysql", "postgres", "redis", "mongodb",
		"cdn", "static", "assets", "media", "images", "img", "css", "js",
		"blog", "news", "forum", "shop", "store", "pay", "payment",
		"vpn", "proxy", "gateway", "router", "firewall",
		"backup", "backups", "archive", "old", "legacy",
		"mobile", "m", "wap", "app", "apps", "application",
		"secure", "security", "auth", "login", "oauth", "sso",
		"monitor", "monitoring", "stats", "statistics", "analytics",
		"docs", "documentation", "wiki", "help", "support",
		"demo", "demo1", "demo2", "test1", "test2", "testing",
		"beta", "alpha", "preview", "staging1", "staging2",
		"internal", "private", "local", "localhost",
		"web", "web1", "web2", "server", "server1", "server2",
		"ns1", "ns2", "dns", "dns1", "dns2",
		"mx", "mx1", "mx2", "mail1", "mail2",
		"cpanel", "whm", "plesk", "directadmin",
		"phpmyadmin", "adminer", "webmin",
		"jenkins", "gitlab", "github", "bitbucket",
		"jira", "confluence", "atlassian",
		"zabbix", "nagios", "prometheus", "grafana",
		"kibana", "elasticsearch", "logstash",
		"kafka", "rabbitmq", "activemq",
		"docker", "kubernetes", "k8s",
		"aws", "azure", "gcp", "cloud",
		"v1", "v2", "v3", "api1", "api2", "api3",
		"old", "new", "new1", "new2",
		"2020", "2021", "2022", "2023", "2024",
	}
}

// addLog 添加日志
func (dc *DomainCollection) addLog(taskID, level, message string) {
	task, err := dc.GetTask(taskID)
	if err != nil {
		return
	}

	logEntry := LogEntry{
		Time:    time.Now().Format("15:04:05"),
		Level:   level,
		Message: message,
	}

	dc.tasksMutex.Lock()
	task.Logs = append(task.Logs, logEntry)
	logCount := len(task.Logs)
	// 限制日志数量，保留最近1000条
	if logCount > 500 {
		task.Logs = task.Logs[logCount-500:]
		logCount = 500
	}
	dc.tasksMutex.Unlock()

	// 发送日志事件到前端（模块内日志）
	if dc.Ctx != nil {
		eventData := map[string]interface{}{
			"task_id": taskID,
			"level":   level,
			"message": message,
			"time":    logEntry.Time,
		}
		runtime.EventsEmit(dc.Ctx, "domain_collection_log", eventData)
	}

	// 发送到全局日志系统（限制频率，避免卡死）
	// 只发送重要日志：error、warning，以及部分info（每10条info发送1条）
	if dc.Ctx != nil {
		shouldSendToGlobal := false
		if level == "error" || level == "warning" {
			// 错误和警告总是发送
			shouldSendToGlobal = true
		} else if level == "info" {
			// info日志限制频率：每10条info发送1条，或者消息较长时发送
			if logCount%10 == 0 || len(message) > 50 {
				shouldSendToGlobal = true
			}
		}

		if shouldSendToGlobal {
			globalLogData := map[string]interface{}{
				"message": fmt.Sprintf("[域名收集] %s", message),
				"level":   level,
				"time":    logEntry.Time,
				"source":  "信息收集/域名收集",
			}
			runtime.EventsEmit(dc.Ctx, "log_event", globalLogData)
		}
	}
}

// GetTask 获取任务（优先从内存，如果不存在则从文件加载）
func (dc *DomainCollection) GetTask(taskID string) (*CollectionTask, error) {
	// 先尝试从内存中获取
	dc.tasksMutex.RLock()
	task, ok := dc.tasks[taskID]
	dc.tasksMutex.RUnlock()

	if ok {
		return task, nil
	}

	// 如果内存中不存在，尝试从文件加载
	return dc.LoadTaskLog(taskID)
}

// StopCollection 停止收集
func (dc *DomainCollection) StopCollection(taskID string) error {
	task, err := dc.GetTask(taskID)
	if err != nil {
		return err
	}

	if task.cancelFunc != nil {
		task.cancelFunc()
	}

	task.Status = "stopped"
	dc.addLog(taskID, "info", "收集已停止")
	return nil
}

// DeleteTask 删除任务（包括内存中的任务和保存的文件）
func (dc *DomainCollection) DeleteTask(taskID string) error {
	dc.tasksMutex.Lock()
	task, exists := dc.tasks[taskID]
	if exists {
		if task.Status == "running" && task.cancelFunc != nil {
			task.cancelFunc()
		}
		delete(dc.tasks, taskID)
	}
	dc.tasksMutex.Unlock()

	// 删除任务日志文件（.json 和 .log）
	logDir := filepath.Join("EasyToolsFiles", "logs", "domain_collection")
	jsonFile := filepath.Join(logDir, fmt.Sprintf("%s.json", taskID))
	logFile := filepath.Join(logDir, fmt.Sprintf("%s.log", taskID))

	if _, err := os.Stat(jsonFile); err == nil {
		_ = os.Remove(jsonFile)
	}
	if _, err := os.Stat(logFile); err == nil {
		_ = os.Remove(logFile)
	}

	return nil
}

// ClearAllTasks 清空所有任务
func (dc *DomainCollection) ClearAllTasks() error {
	dc.tasksMutex.Lock()
	defer dc.tasksMutex.Unlock()

	// 停止所有运行中的任务
	for _, task := range dc.tasks {
		if task.Status == "running" && task.cancelFunc != nil {
			task.cancelFunc()
		}
	}

	// 删除所有已保存的任务文件（包括未在内存中的历史任务）
	logDir := filepath.Join("EasyToolsFiles", "logs", "domain_collection")
	entries, err := os.ReadDir(logDir)
	if err == nil {
		for _, entry := range entries {
			name := entry.Name()
			if strings.HasSuffix(name, ".json") || strings.HasSuffix(name, ".log") {
				_ = os.Remove(filepath.Join(logDir, name))
			}
		}
	}

	// 清空内存中的任务
	dc.tasks = make(map[string]*CollectionTask)
	return nil
}

// SaveTaskLog 保存任务日志到文件
func (dc *DomainCollection) SaveTaskLog(taskID string) error {
	task, err := dc.GetTask(taskID)
	if err != nil {
		return err
	}

	logDir := filepath.Join("EasyToolsFiles", "logs", "domain_collection")
	_ = os.MkdirAll(logDir, 0755)

	// 保存任务JSON文件
	jsonFile := filepath.Join(logDir, fmt.Sprintf("%s.json", taskID))
	taskData, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化任务失败: %v", err)
	}
	if err := os.WriteFile(jsonFile, taskData, 0644); err != nil {
		return fmt.Errorf("保存任务文件失败: %v", err)
	}

	// 保存日志文件（每行一个JSON）
	logFile := filepath.Join(logDir, fmt.Sprintf("%s.log", taskID))
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("创建日志文件失败: %v", err)
	}
	defer file.Close()

	for _, logEntry := range task.Logs {
		logData, err := json.Marshal(logEntry)
		if err != nil {
			continue
		}
		_, _ = file.Write(logData)
		_, _ = file.WriteString("\n")
	}

	return nil
}

// LoadTaskLog 从文件加载任务数据
func (dc *DomainCollection) LoadTaskLog(taskID string) (*CollectionTask, error) {
	logDir := filepath.Join("EasyToolsFiles", "logs", "domain_collection")
	jsonFile := filepath.Join(logDir, fmt.Sprintf("%s.json", taskID))

	// 如果文件不存在，返回错误
	if _, err := os.Stat(jsonFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("任务文件不存在: %s", taskID)
	}

	// 读取任务文件
	data, err := os.ReadFile(jsonFile)
	if err != nil {
		return nil, fmt.Errorf("读取任务文件失败: %v", err)
	}

	var task CollectionTask
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, fmt.Errorf("解析任务文件失败: %v", err)
	}

	return &task, nil
}

// GetAllSavedTasks 获取所有已保存的任务（从文件系统读取）
func (dc *DomainCollection) GetAllSavedTasks() ([]*CollectionTask, error) {
	logDir := filepath.Join("EasyToolsFiles", "logs", "domain_collection")

	var savedTasks []*CollectionTask

	if _, err := os.Stat(logDir); err == nil {
		files, err := os.ReadDir(logDir)
		if err == nil {
			for _, file := range files {
				if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
					continue
				}

				taskID := strings.TrimSuffix(file.Name(), ".json")

				// 如果内存中已有该任务，跳过（内存中的优先）
				dc.tasksMutex.RLock()
				_, exists := dc.tasks[taskID]
				dc.tasksMutex.RUnlock()
				if exists {
					continue
				}

				// 从文件加载任务
				task, err := dc.LoadTaskLog(taskID)
				if err != nil {
					// 如果加载失败，跳过该任务
					continue
				}

				savedTasks = append(savedTasks, task)
			}
		}
	}

	return savedTasks, nil
}

// GetDictList 获取字典文件列表
func (dc *DomainCollection) GetDictList() ([]string, error) {
	baseDir := system.GetAppBaseDir()
	dictDir := filepath.Join(baseDir, "tools", "dict", "domain")

	var dictFiles []string

	// 优先读取domain目录下的字典文件
	if entries, err := os.ReadDir(dictDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".txt") {
				dictFiles = append(dictFiles, filepath.Join(dictDir, entry.Name()))
			}
		}
	}

	// 如果domain目录不存在或为空，尝试读取通用字典目录
	if len(dictFiles) == 0 {
		commonDictDir := filepath.Join(baseDir, "tools", "dict")
		if entries, err := os.ReadDir(commonDictDir); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".txt") {
					fullPath := filepath.Join(commonDictDir, entry.Name())
					dictFiles = append(dictFiles, fullPath)
				}
			}
		}
	}

	return dictFiles, nil
}

// LoadDictFile 加载字典文件内容
func (dc *DomainCollection) LoadDictFile(filePath string) ([]string, error) {
	return dc.loadDictFile(filePath)
}

// querySpaceMappingWithPlatform 从空间测绘平台查询域名，并返回域名和平台的映射关系
func (dc *DomainCollection) querySpaceMappingWithPlatform(task *CollectionTask) ([]string, map[string]string, error) {
	// 根据平台构建不同的查询语句
	// FOFA/Hunter/Quake都支持 domain="xxx" 语法
	query := fmt.Sprintf(`domain="%s"`, task.RootDomain)

	// 获取服务器端口（优先使用已设置的配置，否则使用默认值）
	port := dc.getPort()
	if dc.portConfig == nil || dc.portConfig.MainPort == 0 {
		dc.addLog(task.ID, "warning", fmt.Sprintf("端口配置未设置，使用默认端口 %d。建议在启动时通过 SetPortConfig 设置端口配置", port))
	} else {
		dc.addLog(task.ID, "info", fmt.Sprintf("使用端口 %d 进行空间测绘查询", port))
	}

	// 调用本地API
	apiURL := fmt.Sprintf("http://127.0.0.1:%d/api/info-collection/space-mapping/query", port)

	// 构建请求
	reqBody := map[string]interface{}{
		"query":     query,
		"platforms": task.Config.SpaceMappingPlatforms,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, nil, fmt.Errorf("构建请求失败: %v", err)
	}

	// 发送HTTP请求（使用代理配置）
	client := proxy.GlobalProxyManager.GetHTTPClient()
	client.Timeout = time.Second * 30

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 解析响应
	var apiResp struct {
		Success bool                                `json:"success"`
		Message string                              `json:"message"`
		Data    map[string][]SpaceMappingResultItem `json:"data"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, nil, fmt.Errorf("解析响应失败: %v, 响应内容: %s", err, string(body))
	}

	if !apiResp.Success {
		return nil, nil, fmt.Errorf("API返回错误: %s", apiResp.Message)
	}

	// 提取域名，并记录每个域名来自哪个平台
	domainMap := make(map[string]bool)
	domainPlatformMap := make(map[string]string) // key: 域名, value: 平台名称（多个用逗号分隔）

	rootDomainLower := strings.ToLower(strings.TrimSpace(task.RootDomain))
	dc.addLog(task.ID, "info", fmt.Sprintf("开始提取域名，根域名: %s (小写: %s)", task.RootDomain, rootDomainLower))

	totalItems := 0
	matchedCount := 0
	unmatchedCount := 0

	for platform, platformResults := range apiResp.Data {
		// 格式化平台名称
		platformName := dc.formatPlatformName(platform)
		dc.addLog(task.ID, "info", fmt.Sprintf("处理平台 %s (%s)，共 %d 条结果", platform, platformName, len(platformResults)))

		for _, item := range platformResults {
			if item.Domain != "" {
				totalItems++
				// 提取域名，去除协议前缀
				domain := item.Domain
				originalDomain := domain
				domain = strings.TrimPrefix(domain, "http://")
				domain = strings.TrimPrefix(domain, "https://")
				// 提取主域名部分
				parts := strings.Split(domain, "/")
				if len(parts) > 0 {
					domain = parts[0]
				}
				// 提取域名（去除端口）
				parts = strings.Split(domain, ":")
				domain = parts[0]
				domain = strings.ToLower(strings.TrimSpace(domain))

				// 只保留属于根域名的子域名
				// 支持多种匹配方式：
				// 1. 完全匹配根域名
				// 2. 以 .根域名 结尾（子域名，如 subdomain.rootdomain.com）
				if domain == rootDomainLower || strings.HasSuffix(domain, "."+rootDomainLower) {
					matchedCount++
					domainMap[domain] = true
					// 记录平台信息（如果域名在多个平台出现，用逗号分隔）
					if existingPlatform, exists := domainPlatformMap[domain]; exists {
						if !strings.Contains(existingPlatform, platformName) {
							domainPlatformMap[domain] = existingPlatform + "," + platformName
						}
					} else {
						domainPlatformMap[domain] = platformName
					}
					if matchedCount <= 5 {
						dc.addLog(task.ID, "info", fmt.Sprintf("匹配域名: %s (原始: %s, 平台: %s)", domain, originalDomain, platformName))
					}
				} else {
					unmatchedCount++
				}
			}
		}
	}

	dc.addLog(task.ID, "info", fmt.Sprintf("域名提取完成：总结果数 %d，匹配 %d 个，未匹配 %d 个", totalItems, matchedCount, unmatchedCount))

	// 转换为切片
	domains := make([]string, 0, len(domainMap))
	for domain := range domainMap {
		domains = append(domains, domain)
	}

	return domains, domainPlatformMap, nil
}

// querySpaceMapping 从空间测绘平台查询域名（保持向后兼容）
func (dc *DomainCollection) querySpaceMapping(task *CollectionTask) ([]string, error) {
	domains, _, err := dc.querySpaceMappingWithPlatform(task)
	return domains, err
}

// formatPlatformName 格式化平台名称
func (dc *DomainCollection) formatPlatformName(platform string) string {
	platformMap := map[string]string{
		"fofa":      "FOFA",
		"otherfofa": "FOFA第三方",
		"hunter":    "Hunter",
		"quake":     "Quake",
	}
	if name, ok := platformMap[platform]; ok {
		return name
	}
	return platform
}

// SpaceMappingResultItem 空间测绘结果项
type SpaceMappingResultItem struct {
	IP     string `json:"ip"`
	Port   string `json:"port"`
	Title  string `json:"title"`
	Domain string `json:"domain"`
	URL    string `json:"url"`
}

// SpaceMappingInfoForAI 用于AI的空间测绘信息
type SpaceMappingInfoForAI struct {
	Domain string `json:"domain"`
	IP     string `json:"ip"`
	Port   string `json:"port"`
	Title  string `json:"title"`
	URL    string `json:"url"`
}

// checkSpaceMappingKeys 检查空间测绘key是否有效
func (dc *DomainCollection) checkSpaceMappingKeys(task *CollectionTask) error {
	if len(task.Config.SpaceMappingPlatforms) == 0 {
		return nil
	}

	// 获取服务器端口（从启动时创建的端口配置）
	port := dc.getPort()
	apiURL := fmt.Sprintf("http://127.0.0.1:%d/api/info-collection/config", port)

	client := proxy.GlobalProxyManager.GetHTTPClient()
	client.Timeout = time.Second * 10

	resp, err := client.Get(apiURL)
	if err != nil {
		return fmt.Errorf("获取空间测绘配置失败（端口: %d）: %v", port, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取配置响应失败: %v", err)
	}

	var configResp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			APIKeys map[string]string `json:"apiKeys"`
			Enabled map[string]bool   `json:"enabled"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &configResp); err != nil {
		return fmt.Errorf("解析配置失败: %v", err)
	}

	if !configResp.Success {
		return fmt.Errorf("获取配置失败: %s", configResp.Message)
	}

	// 检查每个选中的平台
	var invalidPlatforms []string
	for _, platform := range task.Config.SpaceMappingPlatforms {
		apiKey, exists := configResp.Data.APIKeys[platform]
		if !exists || apiKey == "" {
			invalidPlatforms = append(invalidPlatforms, platform)
			continue
		}

		if !configResp.Data.Enabled[platform] {
			invalidPlatforms = append(invalidPlatforms, platform)
			continue
		}

		// 验证key是否有效
		verifyURL := fmt.Sprintf("http://127.0.0.1:%d/api/info-collection/space-mapping/verify-key", port)
		verifyReq := map[string]interface{}{
			"platform": platform,
			"apiKey":   apiKey,
		}

		verifyJSON, _ := json.Marshal(verifyReq)
		verifyResp, err := client.Post(verifyURL, "application/json", strings.NewReader(string(verifyJSON)))
		if err != nil {
			invalidPlatforms = append(invalidPlatforms, platform)
			continue
		}
		defer verifyResp.Body.Close()

		var verifyResult struct {
			Success bool   `json:"success"`
			Valid   bool   `json:"valid"`
			Message string `json:"message"`
		}

		verifyBody, _ := io.ReadAll(verifyResp.Body)
		if err := json.Unmarshal(verifyBody, &verifyResult); err != nil {
			invalidPlatforms = append(invalidPlatforms, platform)
			continue
		}

		if !verifyResult.Valid {
			invalidPlatforms = append(invalidPlatforms, platform)
		}
	}

	if len(invalidPlatforms) > 0 {
		return fmt.Errorf("以下空间测绘平台的Key无效或未配置: %s", strings.Join(invalidPlatforms, ", "))
	}

	return nil
}

// checkSpaceMappingKeysForInfo 检查空间测绘key（仅用于信息提示，不阻止执行）
func (dc *DomainCollection) checkSpaceMappingKeysForInfo(task *CollectionTask) {
	port := dc.getPort()
	apiURL := fmt.Sprintf("http://127.0.0.1:%d/api/info-collection/config", port)

	client := proxy.GlobalProxyManager.GetHTTPClient()
	client.Timeout = time.Second * 5

	resp, err := client.Get(apiURL)
	if err != nil {
		dc.addLog(task.ID, "info", "无法检查空间测绘配置，建议在配置API中设置空间测绘Key以启用自动查询功能")
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var configResp struct {
		Success bool `json:"success"`
		Data    struct {
			APIKeys map[string]string `json:"apiKeys"`
			Enabled map[string]bool   `json:"enabled"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &configResp); err != nil || !configResp.Success {
		dc.addLog(task.ID, "info", "空间测绘Key未配置，建议在配置API中设置以启用自动查询功能")
		return
	}

	// 检查是否有任何有效的key
	hasValidKey := false
	for platform, key := range configResp.Data.APIKeys {
		if key != "" && configResp.Data.Enabled[platform] {
			hasValidKey = true
			break
		}
	}

	if !hasValidKey {
		dc.addLog(task.ID, "info", "空间测绘Key未配置或未启用，建议在配置API中设置以启用自动查询功能")
	}
}

// querySpaceMappingForAI 从空间测绘平台查询信息用于AI分析
func (dc *DomainCollection) querySpaceMappingForAI(task *CollectionTask) ([]SpaceMappingInfoForAI, error) {
	// 根据平台构建不同的查询语句
	// FOFA/Hunter/Quake都支持 domain="xxx" 语法
	query := fmt.Sprintf(`domain="%s"`, task.RootDomain)

	port := dc.getPort()
	apiURL := fmt.Sprintf("http://127.0.0.1:%d/api/info-collection/space-mapping/query", port)

	reqBody := map[string]interface{}{
		"query":     query,
		"platforms": task.Config.SpaceMappingPlatforms,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("构建请求失败: %v", err)
	}

	client := proxy.GlobalProxyManager.GetHTTPClient()
	client.Timeout = time.Second * 30

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	var apiResp struct {
		Success bool                                `json:"success"`
		Message string                              `json:"message"`
		Data    map[string][]SpaceMappingResultItem `json:"data"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("API返回错误: %s", apiResp.Message)
	}

	// 提取信息用于AI分析
	var infoList []SpaceMappingInfoForAI
	domainMap := make(map[string]bool)

	for _, platformResults := range apiResp.Data {
		for _, item := range platformResults {
			if item.Domain != "" {
				// 提取域名，去除协议前缀
				domain := item.Domain
				domain = strings.TrimPrefix(domain, "http://")
				domain = strings.TrimPrefix(domain, "https://")
				parts := strings.Split(domain, "/")
				if len(parts) > 0 {
					domain = parts[0]
				}
				parts = strings.Split(domain, ":")
				domain = parts[0]

				// 只保留属于根域名的子域名，且去重
				if (strings.HasSuffix(domain, "."+task.RootDomain) || domain == task.RootDomain) && !domainMap[domain] {
					infoList = append(infoList, SpaceMappingInfoForAI{
						Domain: domain,
						IP:     item.IP,
						Port:   item.Port,
						Title:  item.Title,
						URL:    item.URL,
					})
					domainMap[domain] = true
				}
			}
		}
	}

	return infoList, nil
}

// contains 检查字符串切片是否包含指定字符串
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// checkWildcard 检测域名是否存在泛解析
func (dc *DomainCollection) checkWildcard(rootDomain, dnsServer string, timeout int) bool {
	// 生成一个随机的不可能存在的子域名
	randomSubdomain := fmt.Sprintf("nonexistent-%d.%s", time.Now().UnixNano(), rootDomain)

	// 尝试解析这个随机子域名
	ips, err := dc.resolveDNS(randomSubdomain, timeout, dnsServer)
	if err != nil {
		// 解析失败，可能不是泛解析
		return false
	}

	// 如果随机子域名也能解析出IP，说明可能存在泛解析
	// 进一步验证：检查根域名和随机子域名的IP是否相同
	rootIPs, err := dc.resolveDNS(rootDomain, timeout, dnsServer)
	if err != nil {
		return false
	}

	// 如果随机子域名的IP和根域名的IP有交集，很可能是泛解析
	ipMap := make(map[string]bool)
	for _, ip := range rootIPs {
		ipMap[ip] = true
	}

	for _, ip := range ips {
		if ipMap[ip] {
			return true
		}
	}

	return false
}
