package webscan

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"EasyTools/app/controller/proxy"
	"EasyTools/app/controller/system"
	"EasyTools/app/model"

	gonanoid "github.com/matoous/go-nanoid/v2"
	nuclei "github.com/projectdiscovery/nuclei/v3/lib"
	"github.com/projectdiscovery/nuclei/v3/pkg/output"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// PocController POC 扫描控制器
type PocController struct {
	ctx           context.Context
	scanners      map[string]*PocScanner // 支持多任务扫描
	mu            sync.Mutex
	taskStorePath string
	pocDB         *PocDatabase
	system.Base
}

// NewPocController 创建 POC 控制器实例
func NewPocController() *PocController {
	return &PocController{
		scanners: make(map[string]*PocScanner),
	}
}

// normalizePocName trims extension and normalizes separators to support nested folders.
func normalizePocName(name string) string {
	clean := strings.TrimSpace(name)
	clean = filepath.Clean(clean)
	// remove extension if user passes full filename
	clean = strings.TrimSuffix(clean, filepath.Ext(clean))
	// unify to slash for storage; filepath.FromSlash will convert when joining
	return filepath.ToSlash(clean)
}

// Startup 初始化上下文
func (p *PocController) Startup(ctx context.Context) {
	p.ctx = ctx
	p.taskStorePath = filepath.Join(p.GetAppPath(), "config", "poc_tasks.json")
	_ = os.MkdirAll(filepath.Dir(p.taskStorePath), 0755)
	p.pocDB = NewPocDatabase(ctx, p.Db())
	_ = p.pocDB.Initialize()
	
	// 启动时检查并恢复未完成的任务（状态为 running 或 paused 的任务）
	go func() {
		records, err := p.loadTaskRecords()
		if err == nil {
			for _, record := range records {
				// 如果任务状态是 running 或 paused，说明可能是程序异常退出导致的
				// 将这些任务标记为 paused，以便用户可以手动恢复
				if record.Status == "running" {
					// 将异常退出的运行中任务标记为 paused
					p.updateTaskStatus(record.TaskID, "paused", record.Progress, record.Completed, record.Total)
					p.sendLog("info", fmt.Sprintf("检测到未完成的任务 %s，已标记为暂停状态，可手动恢复", record.TaskID))
				}
			}
		}
	}()
	
	// 启动时加载 POC 配置并初始化标签
	go func() {
		configCtrl := NewPocConfigController()
		configCtrl.Startup(ctx)
		config, err := configCtrl.GetPocConfig()
		if err == nil {
			// 异步加载标签，避免阻塞启动
			if len(config.TemplateFolders) > 0 {
				ReloadPocTags(ctx, config.TemplateFolders)
			}
		}
	}()
}

// sendLog 发送日志事件
func (p *PocController) sendLog(level, message string) {
	if p.ctx == nil {
		return
	}
	// 同步写入系统日志文件
	p.Log(fmt.Sprintf("[漏洞扫描][%s] %s", strings.ToUpper(level), message))
	eventData := map[string]interface{}{
		"level":   level,
		"message": message,
		"time":    time.Now().Format("15:04:05"),
		"source":  "漏洞扫描",
	}
	runtime.EventsEmit(p.ctx, "log_event", eventData)
}

// PocTaskRecord 用于记录扫描任务及其配置
type PocTaskRecord struct {
	TaskID      string        `json:"task_id"`
	Name        string        `json:"name"`
	Targets     []string      `json:"targets"`
	Vuln        int           `json:"vuln"`
	Config      PocScanConfig `json:"config"`
	CreatedAt   string        `json:"created_at"`
	Status      string        `json:"status"`         // 任务状态: running, completed, stopped
	Progress    int           `json:"progress"`       // 进度百分比 0-100
	Completed   int           `json:"completed"`      // 已完成目标数
	Total       int           `json:"total"`          // 总目标数
	ScannedURLs []string      `json:"scanned_urls"`   // 已扫描的 URL 列表（用于恢复时跳过）
	Results     []model.VulnerabilityInfo `json:"results,omitempty"` // 任务结果（用于密码爆破等直接保存结果的任务）
}

// getTaskStorePath 返回任务存储路径
func (p *PocController) getTaskStorePath() string {
	if p.taskStorePath != "" {
		return p.taskStorePath
	}
	p.taskStorePath = filepath.Join(p.GetAppPath(), "config", "poc_tasks.json")
	return p.taskStorePath
}

// loadTaskRecords 读取任务记录
func (p *PocController) loadTaskRecords() ([]PocTaskRecord, error) {
	path := p.getTaskStorePath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return []PocTaskRecord{}, nil
	}
	if err != nil {
		return nil, err
	}
	var records []PocTaskRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, err
	}
	return records, nil
}

// persistTaskRecords 持久化任务记录
func (p *PocController) persistTaskRecords(records []PocTaskRecord) error {
	path := p.getTaskStorePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// saveTaskRecord 保存/更新单个任务记录
func (p *PocController) saveTaskRecord(record PocTaskRecord) error {
	records, err := p.loadTaskRecords()
	if err != nil {
		return err
	}
	updated := false
	for i, r := range records {
		if r.TaskID == record.TaskID {
			records[i] = record
			updated = true
			break
		}
	}
	if !updated {
		records = append(records, record)
	}
	// 按时间倒序
	sort.Slice(records, func(i, j int) bool {
		return records[i].CreatedAt > records[j].CreatedAt
	})
	return p.persistTaskRecords(records)
}

// incrementTaskVuln 增加任务漏洞计数
func (p *PocController) incrementTaskVuln(taskId string) {
	records, err := p.loadTaskRecords()
	if err != nil {
		return
	}
	changed := false
	for i, r := range records {
		if r.TaskID == taskId {
			records[i].Vuln++
			changed = true
			break
		}
	}
	if changed {
		_ = p.persistTaskRecords(records)
	}
}

// GetTaskStatus 获取任务状态和进度
func (p *PocController) GetTaskStatus(taskId string) (map[string]interface{}, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 先检查是否正在运行
	if scanner, exists := p.scanners[taskId]; exists {
		stats := scanner.GetStats()
		percentage := int(stats.ProgressRatio * 100)
		if percentage > 100 {
			percentage = 100
		}
		status := "running"
		if scanner.IsPaused() {
			status = "paused"
		}
		return map[string]interface{}{
			"task_id":    taskId,
			"status":     status,
			"progress":   percentage,
			"completed":  stats.Completed,
			"total":      stats.TotalTargets,
			"vuln_count": stats.VulnCount,
		}, nil
	}

	// 从任务记录中获取
	records, err := p.loadTaskRecords()
	if err != nil {
		return nil, err
	}
	for _, r := range records {
		if r.TaskID == taskId {
			return map[string]interface{}{
				"task_id":    r.TaskID,
				"status":     r.Status,
				"progress":   r.Progress,
				"completed":  r.Completed,
				"total":      r.Total,
				"vuln_count": r.Vuln,
			}, nil
		}
	}

	return nil, fmt.Errorf("任务 %s 不存在", taskId)
}

// GetRunningTasks 获取所有正在运行的任务
func (p *PocController) GetRunningTasks() []map[string]interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()

	var runningTasks []map[string]interface{}
	for taskId, scanner := range p.scanners {
		stats := scanner.GetStats()
		percentage := int(stats.ProgressRatio * 100)
		if percentage > 100 {
			percentage = 100
		}
		runningTasks = append(runningTasks, map[string]interface{}{
			"task_id":    taskId,
			"status":     "running",
			"progress":   percentage,
			"completed":  stats.Completed,
			"total":      stats.TotalTargets,
			"vuln_count": stats.VulnCount,
		})
	}
	return runningTasks
}

// deleteTaskRecord 删除任务记录
func (p *PocController) deleteTaskRecord(taskId string) error {
	records, err := p.loadTaskRecords()
	if err != nil {
		return err
	}
	filtered := records[:0]
	for _, r := range records {
		if r.TaskID != taskId {
			filtered = append(filtered, r)
		}
	}
	return p.persistTaskRecords(filtered)
}

// parseAppendFolders 解析追加的 POC 目录
func parseAppendFolders(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ';' || r == ','
	})
	var folders []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			folders = append(folders, p)
		}
	}
	return folders
}

// mergeFolders 去重合并目录
func mergeFolders(base []string, extra []string) []string {
	m := make(map[string]struct{})
	for _, b := range base {
		if b == "" {
			continue
		}
		m[b] = struct{}{}
	}
	for _, e := range extra {
		if e == "" {
			continue
		}
		m[e] = struct{}{}
	}
	var result []string
	for k := range m {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}

// StartPocScan 启动 POC 扫描（支持多任务并发）
func (p *PocController) StartPocScan(configJSON string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 解析配置
	var config PocScanConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return "", fmt.Errorf("配置解析失败: %v", err)
	}
	// 合并追加的 POC 目录
	appendFolders := parseAppendFolders(config.AppendPoc)

	// 如果配置中没有指定模板文件夹，从配置文件读取
	if len(config.TemplateFolders) == 0 {
		configCtrl := NewPocConfigController()
		configCtrl.Startup(p.ctx)
		pocConfig, err := configCtrl.GetPocConfig()
		if err == nil && len(pocConfig.TemplateFolders) > 0 {
			config.TemplateFolders = pocConfig.TemplateFolders
		}
		// 如果配置中有默认请求头，也使用它们
		if config.CustomHeaders == "" && pocConfig.DefaultHeaders != "" {
			config.CustomHeaders = pocConfig.DefaultHeaders
		}
	}

	// 如果配置中没有指定代理，从全局代理管理器获取
	if config.Proxy == "" && proxy.GlobalProxyManager != nil {
		globalProxy := proxy.GlobalProxyManager.GetGlobalProxy()
		if globalProxy.Enabled && globalProxy.Config != nil {
			proxyConfig := globalProxy.Config
			if proxyConfig.Type != "" && proxyConfig.Host != "" && proxyConfig.Port != "" {
				// 格式化代理URL
				scheme := strings.ToLower(proxyConfig.Type)
				proxyURL := fmt.Sprintf("%s://%s:%s", scheme, proxyConfig.Host, proxyConfig.Port)

				// 如果有认证信息，添加到URL中
				if proxyConfig.Auth != nil && proxyConfig.Auth.Username != "" {
					username := url.QueryEscape(proxyConfig.Auth.Username)
					password := url.QueryEscape(proxyConfig.Auth.Password)
					proxyURL = fmt.Sprintf("%s://%s:%s@%s:%s", scheme, username, password, proxyConfig.Host, proxyConfig.Port)
				}

				config.Proxy = proxyURL
			}
		}
	}
	// 将追加目录并入模板目录
	if len(appendFolders) > 0 {
		config.TemplateFolders = mergeFolders(config.TemplateFolders, appendFolders)
	}

	// 生成任务 ID
	taskId, _ := gonanoid.New()
	taskName := strings.TrimSpace(config.TaskName)
	if taskName == "" {
		if len(taskId) >= 6 {
			taskName = fmt.Sprintf("Task-%s", taskId[:6])
		} else {
			taskName = "Task"
		}
	}

	// 创建扫描器
	scanner := NewPocScanner(p.ctx, taskId, config)
	p.scanners[taskId] = scanner

	// 启动扫描
	go p.runScan(scanner, taskId)

	// 落盘任务
	_ = p.saveTaskRecord(PocTaskRecord{
		TaskID:    taskId,
		Name:      taskName,
		Targets:   config.Targets,
		Vuln:      0,
		Config:    config,
		CreatedAt: time.Now().Format(time.RFC3339),
		Status:    "running",
		Progress:  0,
		Completed: 0,
		Total:     len(config.Targets),
	})

	p.sendLog("info", fmt.Sprintf("漏洞扫描任务启动：%s (ID %s)，目标数量 %d", taskName, taskId, len(config.Targets)))

	return taskId, nil
}

// StopPocScan 停止指定任务扫描（如果taskId为空则停止所有任务）
func (p *PocController) StopPocScan(taskId string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if taskId == "" {
		// 停止所有任务
		for id, scanner := range p.scanners {
			// 先获取已扫描的 URL 列表和统计信息（在停止前）
			scannedURLs := scanner.GetScannedURLs()
			stats := scanner.GetStats()
			percentage := int(stats.ProgressRatio * 100)
			if percentage > 100 {
				percentage = 100
			}
			// 停止扫描器
			scanner.Stop()
			// 更新任务状态
			p.updateTaskStatusWithScannedURLs(id, "stopped", percentage, stats.Completed, stats.TotalTargets, scannedURLs)
		}
		p.scanners = make(map[string]*PocScanner)
		
		// 同时更新所有不在内存中的任务状态为 stopped
		records, err := p.loadTaskRecords()
		if err == nil {
			for _, r := range records {
				if r.Status == "running" || r.Status == "paused" {
					p.updateTaskStatusWithScannedURLs(r.TaskID, "stopped", r.Progress, r.Completed, r.Total, r.ScannedURLs)
				}
			}
		}
		
		p.sendLog("info", "所有漏洞扫描任务已停止")
	} else {
		// 停止指定任务
		if scanner, exists := p.scanners[taskId]; exists {
			// 任务在内存中，先获取已扫描的 URL 列表和统计信息（在停止前）
			scannedURLs := scanner.GetScannedURLs()
			stats := scanner.GetStats()
			percentage := int(stats.ProgressRatio * 100)
			if percentage > 100 {
				percentage = 100
			}
			// 停止扫描器
			scanner.Stop()
			delete(p.scanners, taskId)
			// 更新任务状态
			p.updateTaskStatusWithScannedURLs(taskId, "stopped", percentage, stats.Completed, stats.TotalTargets, scannedURLs)
			p.sendLog("info", fmt.Sprintf("漏洞扫描任务 %s 已停止", taskId))
		} else {
			// 任务不在内存中（可能已暂停），从持久化记录中更新状态
			records, err := p.loadTaskRecords()
			if err != nil {
				return fmt.Errorf("任务 %s 不存在: %v", taskId, err)
			}
			var taskRecord *PocTaskRecord
			for _, r := range records {
				if r.TaskID == taskId {
					taskRecord = &r
					break
				}
			}
			if taskRecord == nil {
				return fmt.Errorf("任务 %s 不存在", taskId)
			}
			// 更新任务状态为停止（保存已扫描的 URL）
			p.updateTaskStatusWithScannedURLs(taskId, "stopped", taskRecord.Progress, taskRecord.Completed, taskRecord.Total, taskRecord.ScannedURLs)
			p.sendLog("info", fmt.Sprintf("漏洞扫描任务 %s 已标记为停止", taskId))
		}
	}
	
	// 发送停止事件
	runtime.EventsEmit(p.ctx, "pocScanStopped", map[string]interface{}{
		"task_id": taskId,
	})
	
	return nil
}

// PausePocScan 暂停指定任务扫描
func (p *PocController) PausePocScan(taskId string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if taskId == "" {
		return fmt.Errorf("taskId 不能为空")
	}

	scanner, exists := p.scanners[taskId]
	if !exists {
		// 如果任务不在内存中，检查任务记录
		records, err := p.loadTaskRecords()
		if err != nil {
			return fmt.Errorf("任务 %s 不存在: %v", taskId, err)
		}
		for _, r := range records {
			if r.TaskID == taskId {
				if r.Status == "running" {
					// 任务记录显示正在运行，但内存中没有，可能是程序重启了
					// 直接更新状态为暂停
					p.updateTaskStatusWithScannedURLs(taskId, "paused", r.Progress, r.Completed, r.Total, r.ScannedURLs)
					runtime.EventsEmit(p.ctx, "pocScanPaused", map[string]interface{}{
						"task_id": taskId,
					})
					p.sendLog("info", fmt.Sprintf("漏洞扫描任务 %s 已标记为暂停", taskId))
					return nil
				}
				return fmt.Errorf("任务 %s 状态为 %s，无法暂停", taskId, r.Status)
			}
		}
		return fmt.Errorf("任务 %s 不存在或未运行", taskId)
	}

	scanner.Pause()
	stats := scanner.GetStats()
	percentage := int(stats.ProgressRatio * 100)
	if percentage > 100 {
		percentage = 100
	}
	
	// 获取已扫描的 URL 列表
	scannedURLs := scanner.GetScannedURLs()
	
	// 先更新任务状态到持久化存储（包含已扫描的 URL）
	// 这确保即使程序崩溃，状态也能恢复
	p.updateTaskStatusWithScannedURLs(taskId, "paused", percentage, stats.Completed, stats.TotalTargets, scannedURLs)
	
	// 暂停后，删除内存中的扫描器，这样恢复时可以从任务记录中重新创建
	delete(p.scanners, taskId)
	
	runtime.EventsEmit(p.ctx, "pocScanPaused", map[string]interface{}{
		"task_id": taskId,
	})
	p.sendLog("info", fmt.Sprintf("漏洞扫描任务 %s 已暂停，已扫描 %d 个目标", taskId, len(scannedURLs)))
	return nil
}

// ResumePocScan 恢复指定任务扫描
func (p *PocController) ResumePocScan(taskId string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if taskId == "" {
		return fmt.Errorf("taskId 不能为空")
	}

	scanner, exists := p.scanners[taskId]
	if !exists {
		// 如果任务不在运行中，尝试从任务记录中恢复（支持程序重启后恢复）
		records, err := p.loadTaskRecords()
		if err != nil {
			return fmt.Errorf("任务 %s 不存在: %v", taskId, err)
		}
		var taskRecord *PocTaskRecord
		for _, r := range records {
			if r.TaskID == taskId {
				// 允许恢复暂停、停止或运行中的任务（如果不在内存中）
				// 也允许恢复已完成的任务，可能是误标记为完成，或者用户想重新扫描
				if r.Status == "paused" || r.Status == "stopped" || r.Status == "running" || r.Status == "completed" {
					taskRecord = &r
					break
				}
			}
		}
		if taskRecord == nil {
			// 调试信息：查找所有记录的状态
			var statuses []string
			for _, r := range records {
				if r.TaskID == taskId {
					statuses = append(statuses, r.Status)
				}
			}
			if len(statuses) > 0 {
				return fmt.Errorf("任务 %s 状态为 %v，无法恢复", taskId, statuses)
			}
			return fmt.Errorf("任务 %s 不存在或无法恢复", taskId)
		}
		
		// 如果任务已完成，但仍然可以恢复（可能还有未扫描的 URL）
		if taskRecord.Status == "completed" {
			// 检查是否还有未扫描的 URL
			scannedURLs := taskRecord.ScannedURLs
			if scannedURLs == nil {
				scannedURLs = []string{}
			}
			scannedMap := make(map[string]bool)
			for _, url := range scannedURLs {
				scannedMap[url] = true
			}
			unscannedCount := 0
			for _, target := range taskRecord.Config.Targets {
				if !scannedMap[target] {
					unscannedCount++
				}
			}
			// 如果没有未扫描的 URL，说明确实已完成
			if unscannedCount == 0 {
				return fmt.Errorf("任务 %s 已完成，所有目标都已扫描完毕", taskId)
			}
		}

		// 从未扫描的 URL 继续扫描
		scannedURLs := taskRecord.ScannedURLs
		if scannedURLs == nil {
			scannedURLs = []string{}
		}
		
		// 创建新的配置，只包含未扫描的 URL
		unscannedTargets := []string{}
		scannedMap := make(map[string]bool)
		for _, url := range scannedURLs {
			scannedMap[url] = true
		}
		for _, target := range taskRecord.Config.Targets {
			if !scannedMap[target] {
				unscannedTargets = append(unscannedTargets, target)
			}
		}
		
		if len(unscannedTargets) == 0 {
			// 所有目标都已扫描完成，更新状态为完成
			p.updateTaskStatusWithScannedURLs(taskId, "completed", 100, taskRecord.Total, taskRecord.Total, scannedURLs)
			return fmt.Errorf("任务 %s 所有目标已完成扫描", taskId)
		}
		
		// 更新配置中的目标列表为未扫描的 URL
		taskRecord.Config.Targets = unscannedTargets
		
		// 重新创建扫描器（传入已扫描的 URL，用于进度计算）
		scanner = NewPocScannerWithScannedURLs(p.ctx, taskId, taskRecord.Config, scannedURLs)
		p.scanners[taskId] = scanner
		
		// 获取扫描器的新统计信息（用于更新任务记录）
		stats := scanner.GetStats()
		percentage := int(stats.ProgressRatio * 100)
		if percentage > 100 {
			percentage = 100
		}
		
		go p.runScan(scanner, taskId)
		
		// 更新任务状态为运行中（立即持久化，使用扫描器的新统计信息）
		p.updateTaskStatusWithScannedURLs(taskId, "running", percentage, stats.Completed, stats.TotalTargets, scannedURLs)
		
		runtime.EventsEmit(p.ctx, "pocScanResumed", map[string]interface{}{
			"task_id": taskId,
		})
		p.sendLog("info", fmt.Sprintf("漏洞扫描任务 %s 已恢复，剩余 %d 个目标待扫描", taskId, len(unscannedTargets)))
		return nil
	}

	// 如果任务正在运行，恢复它
	scanner.Resume()
	stats := scanner.GetStats()
	percentage := int(stats.ProgressRatio * 100)
	if percentage > 100 {
		percentage = 100
	}
	// 获取已扫描的 URL 列表并更新状态
	scannedURLs := scanner.GetScannedURLs()
	p.updateTaskStatusWithScannedURLs(taskId, "running", percentage, stats.Completed, stats.TotalTargets, scannedURLs)
	runtime.EventsEmit(p.ctx, "pocScanResumed", map[string]interface{}{
		"task_id": taskId,
	})
	p.sendLog("info", fmt.Sprintf("漏洞扫描任务 %s 已恢复", taskId))
	return nil
}

// runScan 运行扫描
func (p *PocController) runScan(scanner *PocScanner, taskId string) {
	results, errs := scanner.Start()

	var wg sync.WaitGroup
	wg.Add(2)

	// 处理扫描结果
	go func() {
		defer wg.Done()
		for result := range results {
			runtime.EventsEmit(p.ctx, "pocScanResult", result)
			p.sendLog("success", fmt.Sprintf("发现漏洞: [%s] %s - %s", result.Severity, result.VulnName, result.VulnURL))
			if p.pocDB != nil {
				if err := p.pocDB.AddPocResult(result); err != nil {
					p.sendLog("error", fmt.Sprintf("保存漏洞结果失败: %v", err))
				} else {
					p.incrementTaskVuln(result.TaskId)
				}
			}
		}
	}()

	// 处理错误
	go func() {
		defer wg.Done()
		for err := range errs {
			runtime.EventsEmit(p.ctx, "pocScanError", map[string]interface{}{
				"task_id": taskId,
				"message": err.Error(),
			})
			p.sendLog("error", fmt.Sprintf("扫描错误: %v", err))
		}
	}()

	// 进度更新
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	// 创建一个停止检查 channel，定期检查任务是否被停止
	stopCheckTicker := time.NewTicker(500 * time.Millisecond)
	defer stopCheckTicker.Stop()

	for {
		select {
		case <-scanner.done:
			wg.Wait()
			// 检查任务是否已被暂停或停止，如果是则不应该更新为 completed
			p.mu.Lock()
			_, stillInMap := p.scanners[taskId]
			p.mu.Unlock()
			
			// 如果扫描器已不在 map 中（可能已被暂停删除），也不更新
			if !stillInMap {
				return
			}
			
			// 检查任务记录中的状态（不在锁内调用，避免死锁）
			records, err := p.loadTaskRecords()
			currentStatus := ""
			if err == nil {
				for _, r := range records {
					if r.TaskID == taskId {
						currentStatus = r.Status
						break
					}
				}
			}
			
			// 如果任务已被暂停或停止，不更新为 completed
			if currentStatus == "paused" || currentStatus == "stopped" {
				p.mu.Lock()
				if _, exists := p.scanners[taskId]; exists {
					delete(p.scanners, taskId)
				}
				p.mu.Unlock()
				return
			}
			
			stats := scanner.GetStats()
			percentage := int(stats.ProgressRatio * 100)
			if percentage > 100 {
				percentage = 100
			}
			// 获取已扫描的 URL 列表并更新任务状态为完成（立即持久化）
			scannedURLs := scanner.GetScannedURLs()
			p.updateTaskStatusWithScannedURLs(taskId, "completed", percentage, stats.Completed, stats.TotalTargets, scannedURLs)
			runtime.EventsEmit(p.ctx, "pocScanComplete", map[string]interface{}{
				"task_id": taskId,
			})
			p.sendLog("info", fmt.Sprintf(
				"漏洞扫描完成 [%s]：总目标 %d，已完成 %d，发现漏洞 %d",
				taskId,
				stats.TotalTargets,
				stats.Completed,
				stats.VulnCount,
			))
			p.mu.Lock()
			delete(p.scanners, taskId)
			p.mu.Unlock()
			return
		case <-ticker.C:
			// 检查任务是否还在内存中，如果不在则停止更新
			p.mu.Lock()
			_, stillInMap := p.scanners[taskId]
			p.mu.Unlock()
			if !stillInMap {
				// 任务已被暂停或删除，停止更新
				continue
			}
			
			stats := scanner.GetStats()
			percentage := int(stats.ProgressRatio * 100)
			if percentage > 100 {
				percentage = 100
			}
			// 获取已扫描的 URL 列表并更新任务进度（立即持久化）
			scannedURLs := scanner.GetScannedURLs()
			p.updateTaskStatusWithScannedURLs(taskId, "running", percentage, stats.Completed, stats.TotalTargets, scannedURLs)
			runtime.EventsEmit(p.ctx, "pocScanProgress", map[string]interface{}{
				"task_id":    taskId,
				"percentage": percentage,
				"stats": map[string]interface{}{
					"totalTargets": stats.TotalTargets,
					"completed":    stats.Completed,
					"vulnCount":    stats.VulnCount,
				},
			})
		case <-stopCheckTicker.C:
			// 定期检查任务状态，如果被停止则退出
			records, err := p.loadTaskRecords()
			if err == nil {
				for _, r := range records {
					if r.TaskID == taskId {
						if r.Status == "stopped" {
							// 任务已被停止，清理并退出
							p.mu.Lock()
							if _, exists := p.scanners[taskId]; exists {
								delete(p.scanners, taskId)
							}
							p.mu.Unlock()
							// 停止扫描器
							scanner.Stop()
							return
						}
						break
					}
				}
			}
		case <-p.ctx.Done():
			wg.Wait()
			p.mu.Lock()
			delete(p.scanners, taskId)
			p.mu.Unlock()
			return
		}
	}
}

// updateTaskStatus 更新任务状态和进度
func (p *PocController) updateTaskStatus(taskId, status string, progress, completed, total int) {
	p.updateTaskStatusWithScannedURLs(taskId, status, progress, completed, total, nil)
}

// updateTaskStatusWithScannedURLs 更新任务状态和进度（包含已扫描的 URL）
// 所有状态变更都会立即持久化到文件，确保程序重启后能恢复
func (p *PocController) updateTaskStatusWithScannedURLs(taskId, status string, progress, completed, total int, scannedURLs []string) {
	records, err := p.loadTaskRecords()
	if err != nil {
		p.sendLog("error", fmt.Sprintf("更新任务状态失败，无法读取任务记录: %v", err))
		return
	}
	
	found := false
	for i, r := range records {
		if r.TaskID == taskId {
			found = true
			// 如果当前状态是 paused 或 stopped，且新状态是 completed，则不更新状态
			// 这是为了防止 runScan 中的 goroutine 在任务被暂停后覆盖状态
			if (r.Status == "paused" || r.Status == "stopped") && status == "completed" {
				// 只更新进度和完成数，不更新状态
				records[i].Progress = progress
				records[i].Completed = completed
				if total > 0 {
					records[i].Total = total
				}
				if scannedURLs != nil {
					records[i].ScannedURLs = scannedURLs
				}
			} else {
				// 正常更新状态
				records[i].Status = status
				records[i].Progress = progress
				records[i].Completed = completed
				if total > 0 {
					records[i].Total = total
				}
				if scannedURLs != nil {
					records[i].ScannedURLs = scannedURLs
				}
			}
			// 立即持久化到文件，确保状态不丢失
			if err := p.persistTaskRecords(records); err != nil {
				p.sendLog("error", fmt.Sprintf("持久化任务状态失败: %v", err))
			}
			break
		}
	}
	
	if !found {
		// 如果任务记录不存在，记录警告（但不创建新记录，因为任务应该已经存在）
		p.sendLog("warning", fmt.Sprintf("更新任务状态时未找到任务记录: %s", taskId))
	}
}

// ListScanTasks 返回已保存的扫描任务
func (p *PocController) ListScanTasks() ([]PocTaskRecord, error) {
	return p.loadTaskRecords()
}

// DeleteScanTask 删除指定扫描任务（包含结果）
func (p *PocController) DeleteScanTask(taskId string) error {
	if taskId == "" {
		return fmt.Errorf("taskId 不能为空")
	}
	if err := p.deleteTaskRecord(taskId); err != nil {
		return err
	}
	if p.pocDB != nil {
		_ = p.pocDB.RemovePocResultsByTaskId(taskId)
	}
	p.sendLog("info", fmt.Sprintf("已删除漏洞扫描任务 %s", taskId))
	return nil
}

// ClearScanTasks 清除所有漏洞扫描任务及其结果
func (p *PocController) ClearScanTasks() error {
	// 清空任务记录文件
	if err := p.persistTaskRecords([]PocTaskRecord{}); err != nil {
		return err
	}

	// 清空数据库中的漏洞结果
	if p.pocDB != nil {
		if err := p.pocDB.ClearAllPocResults(); err != nil {
			return err
		}
	}

	p.sendLog("info", "已清除所有漏洞扫描任务及结果")
	return nil
}

// GetScanResults 获取任务的扫描结果
func (p *PocController) GetScanResults(taskId string) ([]model.VulnerabilityInfo, error) {
	if taskId == "" {
		return nil, fmt.Errorf("taskId 不能为空")
	}
	
	// 对于密码爆破任务，从任务记录的Results字段读取结果
	// 使用实际的taskId，而不是统一为 hostscan-crack-unified
	if taskId == "hostscan-crack-unified" || 
	   (len(taskId) >= len("hostscan-crack-") && taskId[:len("hostscan-crack-")] == "hostscan-crack-") ||
	   (len(taskId) >= len("crack-") && taskId[:len("crack-")] == "crack-") {
		records, err := p.loadTaskRecords()
		if err != nil {
			return nil, err
		}
		// 使用实际的taskId查找任务记录，而不是统一使用 hostscan-crack-unified
		for _, r := range records {
			if r.TaskID == taskId {
				if r.Results != nil {
					return r.Results, nil
				}
				return []model.VulnerabilityInfo{}, nil
			}
		}
		return []model.VulnerabilityInfo{}, nil
	}
	
	// 其他任务从数据库读取结果
	if p.pocDB == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	return p.pocDB.RetrievePocResults(taskId)
}

// EnsureCrackTaskRecord 确保密码爆破任务记录存在（用于主机扫描的密码爆破结果）
// resultJSON: 可选的密码爆破结果JSON字符串，如果提供则追加到任务记录的结果列表中
func (p *PocController) EnsureCrackTaskRecord(taskId string, resultJSON string) error {
	if taskId == "" {
		return fmt.Errorf("taskId 不能为空")
	}
	
	// 检查任务记录是否已存在
	records, err := p.loadTaskRecords()
	if err != nil {
		return err
	}
	
	// 解析结果JSON（如果有）
	var result *model.VulnerabilityInfo
	if resultJSON != "" {
		var vuln model.VulnerabilityInfo
		if err := json.Unmarshal([]byte(resultJSON), &vuln); err == nil {
			// 如果字段为空，设置默认值
			if vuln.VulnName == "" {
				if vuln.Protocol != "" {
					vuln.VulnName = fmt.Sprintf("%s弱口令", strings.ToUpper(vuln.Protocol))
				} else {
					vuln.VulnName = "未知漏洞"
				}
			}
			if vuln.Description == "" {
				if vuln.Protocol != "" {
					vuln.Description = fmt.Sprintf("%s弱口令漏洞", strings.ToUpper(vuln.Protocol))
				} else {
					vuln.Description = "未知漏洞"
				}
			}
			if vuln.Severity == "" {
				vuln.Severity = "HIGH"
			}
			// 验证 extract 字段（账号密码信息）
			if vuln.Extract == "" {
				p.sendLog("warning", fmt.Sprintf("密码爆破结果缺少 extract 字段（账号密码信息）: %s - %s", vuln.VulnName, vuln.VulnURL))
			} else {
				p.sendLog("info", fmt.Sprintf("成功解析密码爆破结果: %s - %s (协议: %s, 账号密码: %s)", vuln.VulnName, vuln.VulnURL, vuln.Protocol, vuln.Extract))
			}
			result = &vuln
		} else {
			p.sendLog("error", fmt.Sprintf("解析密码爆破结果JSON失败: %v, JSON内容: %s", err, resultJSON))
		}
	}
	
	// 查找最近的密码爆破任务（10分钟内创建的，用于同一批次聚合）
	now := time.Now()
	var recentTask *PocTaskRecord
	var recentTaskIndex int = -1
	timeWindow := 10 * time.Minute // 10分钟内的任务认为是同一批次
	
	for i, r := range records {
		// 检查是否是密码爆破任务（名称包含"密码爆破结果"或taskId以crack开头）
		if strings.Contains(r.Name, "密码爆破结果") || 
		   strings.HasPrefix(r.TaskID, "hostscan-crack-") || 
		   strings.HasPrefix(r.TaskID, "crack-") {
			// 解析创建时间
			if createdTime, err := time.Parse(time.RFC3339, r.CreatedAt); err == nil {
				// 如果任务在时间窗口内，使用这个任务
				if now.Sub(createdTime) <= timeWindow {
					recentTask = &records[i]
					recentTaskIndex = i
					break
				}
			}
		}
	}
	
	// 如果找到最近的任务，追加结果
	if recentTask != nil && recentTaskIndex >= 0 {
		// 设置正确的taskId
		if result != nil {
			result.TaskId = recentTask.TaskID
		}
		
		// 如果有新的结果，追加到结果列表
		if result != nil {
			// 初始化结果数组（如果为空）
			if records[recentTaskIndex].Results == nil {
				records[recentTaskIndex].Results = []model.VulnerabilityInfo{}
			}
			// 检查是否已存在相同的结果（避免重复）
			exists := false
			for _, existingResult := range records[recentTaskIndex].Results {
				if existingResult.VulnURL == result.VulnURL && 
				   existingResult.Extract == result.Extract && 
				   existingResult.TemplateID == result.TemplateID {
					exists = true
					break
				}
			}
			if !exists {
				records[recentTaskIndex].Results = append(records[recentTaskIndex].Results, *result)
				records[recentTaskIndex].Vuln = len(records[recentTaskIndex].Results)
				p.sendLog("info", fmt.Sprintf("已追加密码爆破结果到任务: %s (现有结果数: %d)", recentTask.TaskID, records[recentTaskIndex].Vuln))
			} else {
				// 如果已存在，更新漏洞数量
				records[recentTaskIndex].Vuln = len(records[recentTaskIndex].Results)
			}
		} else {
			// 如果没有提供新结果，只更新漏洞数量（基于现有结果数量）
			if records[recentTaskIndex].Results != nil {
				records[recentTaskIndex].Vuln = len(records[recentTaskIndex].Results)
			}
		}
		
		// 更新任务的更新时间
		records[recentTaskIndex].CreatedAt = now.Format(time.RFC3339)
		
		if err := p.persistTaskRecords(records); err != nil {
			return fmt.Errorf("更新密码爆破任务记录失败: %v", err)
		}
		return nil
	}
	
	// 如果没有找到最近的任务，创建新任务（使用时间戳作为批次标识）
	timestamp := now.Format("20060102-150405")
	finalTaskId := fmt.Sprintf("hostscan-crack-%s", timestamp)
	taskName := fmt.Sprintf("密码爆破结果-%s", timestamp)
	
	results := []model.VulnerabilityInfo{}
	if result != nil {
		result.TaskId = finalTaskId
		results = append(results, *result)
	}
	
	taskRecord := PocTaskRecord{
		TaskID:    finalTaskId,
		Name:      taskName,
		Targets:   []string{}, // 密码爆破任务没有固定的目标列表
		Vuln:      len(results),
		Config:    PocScanConfig{}, // 密码爆破任务没有扫描配置
		CreatedAt: now.Format(time.RFC3339),
		Status:    "completed", // 密码爆破任务直接标记为已完成
		Progress:  100,
		Completed: 1,
		Total:     1,
		Results:   results, // 保存结果到任务记录
	}
	
	if err := p.saveTaskRecord(taskRecord); err != nil {
		return fmt.Errorf("创建密码爆破任务记录失败: %v", err)
	}
	p.sendLog("info", fmt.Sprintf("已创建新的密码爆破任务记录: %s (名称: %s)", finalTaskId, taskName))
	
	return nil
}

// SaveAppendPoc 保存追加的 POC 目录
func (p *PocController) SaveAppendPoc(value string) error {
	path := filepath.Join(p.GetAppPath(), "config", "poc_append.json")
	payload := map[string]string{
		"append_poc": strings.TrimSpace(value),
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}
	p.sendLog("info", "追加 POC 路径已保存")
	return nil
}

// GetAppendPoc 读取保存的追加 POC 目录
func (p *PocController) GetAppendPoc() (string, error) {
	path := filepath.Join(p.GetAppPath(), "config", "poc_append.json")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	obj := map[string]string{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return "", err
	}
	return obj["append_poc"], nil
}

// GetFingerPocMap 获取指纹到 POC 的映射关系
func (p *PocController) GetFingerPocMap() map[string][]string {
	return GetFingerPocMap()
}

// ReloadPocTags 重新加载 POC 标签
func (p *PocController) ReloadPocTags(templateFolders []string) error {
	return ReloadPocTags(p.ctx, templateFolders)
}

// GetPocConfig 获取 POC 配置（委托给配置控制器）
func (p *PocController) GetPocConfigJSON() (string, error) {
	configCtrl := NewPocConfigController()
	configCtrl.Startup(p.ctx)
	config, err := configCtrl.GetPocConfig()
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ListPocs 返回 POC 列表（文件名及路径），便于前端展示管理
// 关联指纹标签通过 GetFingerPocMap 异步加载，避免阻塞列表首屏渲染
func (p *PocController) ListPocs() ([]map[string]interface{}, error) {
	configCtrl := NewPocConfigController()
	configCtrl.Startup(p.ctx)
	cfg, err := configCtrl.GetPocConfig()
	if err != nil {
		return nil, err
	}
	if len(cfg.TemplateFolders) == 0 {
		return []map[string]interface{}{}, nil
	}

	result := []map[string]interface{}{}
	for _, folder := range cfg.TemplateFolders {
		filepath.WalkDir(folder, func(path string, d os.DirEntry, err error) error {
			if err != nil || d == nil || d.IsDir() {
				return nil
			}
			if strings.HasSuffix(d.Name(), ".yaml") {
				relPath, _ := filepath.Rel(folder, path)
				relPath = filepath.ToSlash(strings.TrimSuffix(relPath, filepath.Ext(relPath)))
				result = append(result, map[string]interface{}{
					// name 保留相对路径（不含扩展名），前端可据此还原层级结构
					"name": relPath,
					"path": path,
				})
			}
			return nil
		})
	}
	return result, nil
}

// ReadPoc 读取单个 POC 内容
func (p *PocController) ReadPoc(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("POC 名称不能为空")
	}
	normalized := normalizePocName(name)
	configCtrl := NewPocConfigController()
	configCtrl.Startup(p.ctx)
	cfg, err := configCtrl.GetPocConfig()
	if err != nil {
		return "", err
	}
	for _, folder := range cfg.TemplateFolders {
		full := filepath.Join(folder, filepath.FromSlash(normalized+".yaml"))
		if _, err := os.Stat(full); err == nil {
			data, err := os.ReadFile(full)
			if err != nil {
				return "", fmt.Errorf("读取失败: %v", err)
			}
			return string(data), nil
		}
	}
	return "", fmt.Errorf("未找到 POC：%s", name)
}

// SavePoc 保存/覆盖 POC 文件（写入第一个模板目录），返回保存的文件路径
func (p *PocController) SavePoc(name string, content string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("POC 名称不能为空")
	}
	if content == "" {
		return "", fmt.Errorf("POC 内容不能为空")
	}
	normalized := normalizePocName(name)
	configCtrl := NewPocConfigController()
	configCtrl.Startup(p.ctx)
	cfg, err := configCtrl.GetPocConfig()
	if err != nil {
		return "", err
	}
	if len(cfg.TemplateFolders) == 0 {
		return "", fmt.Errorf("尚未配置 POC 模板目录")
	}
	targetDir := cfg.TemplateFolders[0]
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("创建目录失败: %v", err)
	}
	full := filepath.Join(targetDir, filepath.FromSlash(normalized+".yaml"))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		return "", fmt.Errorf("创建目录失败: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("写入失败: %v", err)
	}
	// 异步重新加载标签，避免阻塞
	go func() {
		ReloadPocTags(p.ctx, cfg.TemplateFolders)
	}()
	return full, nil
}

// TestPoc 测试POC，返回测试结果
func (p *PocController) TestPoc(targetURL string, pocContent string, testConfigJSON string) (map[string]interface{}, error) {
	if targetURL == "" {
		return nil, fmt.Errorf("测试URL不能为空")
	}
	if pocContent == "" {
		return nil, fmt.Errorf("POC内容不能为空")
	}

	// 解析测试配置
	var testConfig struct {
		Threads   int `json:"threads"`
		Timeout   int `json:"timeout"`
		RateLimit int `json:"rate_limit"`
		Retries   int `json:"retries"`
	}

	// 获取扫描配置作为默认值
	configCtrl := NewPocConfigController()
	configCtrl.Startup(p.ctx)
	scanConfig, err := configCtrl.GetPocConfig()
	if err != nil {
		scanConfig = configCtrl.getDefaultConfig()
	}

	// 设置默认值
	testConfig.Threads = scanConfig.Threads
	testConfig.Timeout = scanConfig.Timeout
	testConfig.RateLimit = scanConfig.RateLimit
	testConfig.Retries = scanConfig.Retries

	// 如果提供了配置JSON，解析它
	if testConfigJSON != "" {
		if err := json.Unmarshal([]byte(testConfigJSON), &testConfig); err != nil {
			// 解析失败时使用默认值
		}
	}

	// 创建临时文件保存POC内容
	tmpDir := filepath.Join(p.GetAppPath(), "temp", "poc_test")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return nil, fmt.Errorf("创建临时目录失败: %v", err)
	}

	// 生成临时文件名
	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("test_%d.yaml", time.Now().UnixNano()))
	defer func() {
		// 清理临时文件
		os.Remove(tmpFile)
	}()

	// 写入POC内容到临时文件
	if err := os.WriteFile(tmpFile, []byte(pocContent), 0644); err != nil {
		return nil, fmt.Errorf("写入临时文件失败: %v", err)
	}

	// 使用nuclei引擎测试
	result := map[string]interface{}{
		"matched":      false,
		"message":      "",
		"request":      "",
		"response":     "",
		"responseTime": "",
		"extracted":    []string{},
	}

	// 从全局代理配置构建代理字符串（如果已启用）
	var proxyURL string
	if proxy.GlobalProxyManager != nil {
		globalProxy := proxy.GlobalProxyManager.GetGlobalProxy()
		if globalProxy.Enabled && globalProxy.Config != nil {
			proxyConfig := globalProxy.Config
			if proxyConfig.Type != "" && proxyConfig.Host != "" && proxyConfig.Port != "" {
				scheme := strings.ToLower(proxyConfig.Type)
				// 带认证信息的代理
				if proxyConfig.Auth != nil && proxyConfig.Auth.Username != "" {
					username := url.QueryEscape(proxyConfig.Auth.Username)
					password := url.QueryEscape(proxyConfig.Auth.Password)
					proxyURL = fmt.Sprintf("%s://%s:%s@%s:%s", scheme, username, password, proxyConfig.Host, proxyConfig.Port)
				} else {
					proxyURL = fmt.Sprintf("%s://%s:%s", scheme, proxyConfig.Host, proxyConfig.Port)
				}
			}
		}
	}

	// 构建nuclei选项
	options := []nuclei.NucleiSDKOptions{
		nuclei.DisableUpdateCheck(),
		nuclei.WithTemplatesOrWorkflows(nuclei.TemplateSources{
			Templates: []string{tmpFile},
		}),
	}

	// 添加网络配置
	if testConfig.Timeout > 0 {
		options = append(options, nuclei.WithNetworkConfig(nuclei.NetworkConfig{
			Timeout: testConfig.Timeout,
			Retries: testConfig.Retries,
		}))
	}

	// 添加代理配置（走全局代理）
	if proxyURL != "" {
		options = append(options, nuclei.WithProxy([]string{proxyURL}, false))
	}

	// 添加速率限制
	if testConfig.RateLimit > 0 {
		options = append(options, nuclei.WithGlobalRateLimitCtx(p.ctx, testConfig.RateLimit, time.Second))
	}

	// 使用nuclei引擎测试
	ne, err := nuclei.NewNucleiEngineCtx(p.ctx, options...)
	if err != nil {
		result["message"] = fmt.Sprintf("初始化nuclei引擎失败: %v", err)
		return result, nil
	}
	defer ne.Close()

	// 加载目标
	ne.LoadTargets([]string{targetURL}, false)

	// 执行测试并收集结果
	var matched bool
	var lastRequest, lastResponse, responseTime string
	var extracted []string

	err = ne.ExecuteWithCallback(func(event *output.ResultEvent) {
		matched = true
		lastRequest = showRequest(event)
		lastResponse = showResponse(event)
		responseTime = limitDecimalPlaces(event.ResponseTime)
		extracted = event.ExtractedResults
	})

	if err != nil {
		result["message"] = fmt.Sprintf("执行测试失败: %v", err)
		return result, nil
	}

	result["matched"] = matched
	if matched {
		result["message"] = "POC匹配成功！"
		result["request"] = lastRequest
		result["response"] = lastResponse
		result["responseTime"] = responseTime
		result["extracted"] = extracted
	} else {
		result["message"] = "POC未匹配，目标可能不存在该漏洞"
	}

	return result, nil
}

// DeletePoc 删除 POC 文件（在配置的模板目录中查找并删除）
func (p *PocController) DeletePoc(name string) error {
	if name == "" {
		return fmt.Errorf("POC 名称不能为空")
	}
	normalized := normalizePocName(name)
	configCtrl := NewPocConfigController()
	configCtrl.Startup(p.ctx)
	cfg, err := configCtrl.GetPocConfig()
	if err != nil {
		return err
	}
	found := false
	for _, folder := range cfg.TemplateFolders {
		full := filepath.Join(folder, filepath.FromSlash(normalized+".yaml"))
		if _, err := os.Stat(full); err == nil {
			if err := os.Remove(full); err != nil {
				return fmt.Errorf("删除失败: %v", err)
			}
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("未找到 POC：%s", name)
	}
	// 异步重新加载标签，避免阻塞删除操作
	go func() {
		ReloadPocTags(p.ctx, cfg.TemplateFolders)
	}()
	return nil
}

// PocScanConfig POC 扫描配置
type PocScanConfig struct {
	Targets               []string `json:"targets"`                  // 扫描目标
	TaskName              string   `json:"task_name"`                // 任务名称
	AppendPoc             string   `json:"append_poc"`               // 追加 POC 目录
	Tags                  []string `json:"tags"`                     // 指纹标签
	CustomTags            []string `json:"custom_tags"`              // 自定义标签
	TemplateFile          []string `json:"template_file"`            // 指定的 POC 文件
	TemplateFolders       []string `json:"template_folders"`         // POC 模板文件夹
	CustomHeaders         string   `json:"custom_headers"`           // 自定义请求头
	Proxy                 string   `json:"proxy"`                    // 代理地址
	SkipNucleiWithoutTags bool     `json:"skip_nuclei_without_tags"` // 没有标签时是否跳过
	ThreadSafe            bool     `json:"thread_safe"`              // 是否使用线程安全模式
	Threads               int      `json:"threads"`                  // 扫描线程数
	Timeout               int      `json:"timeout"`                  // 超时时间（秒）
	RateLimit             int      `json:"rate_limit"`               // 速率限制（每秒请求数）
	Retries               int      `json:"retries"`                  // 重试次数
}

// PocScanner POC 扫描器
type PocScanner struct {
	ctx         context.Context
	taskId      string
	config      PocScanConfig
	done        chan struct{}
	stop        chan struct{}
	pause       chan struct{}
	resume      chan struct{}
	results     chan model.VulnerabilityInfo
	errors      chan error
	stats       *ScanStats
	statsMu     sync.Mutex
	paused      bool
	pauseMu     sync.Mutex
	scannedURLs map[string]bool // 已扫描的 URL 集合，用于快速查找
	scannedMu   sync.Mutex      // 保护 scannedURLs 的互斥锁
	urlPocCount map[string]int  // 每个URL对应的POC数量
	urlPocMu    sync.Mutex      // 保护 urlPocCount 的互斥锁
}

// ScanStats 扫描统计
type ScanStats struct {
	TotalTargets  int
	Completed     int
	VulnCount     int
	ProgressRatio float64
}

// NewPocScanner 创建 POC 扫描器
func NewPocScanner(ctx context.Context, taskId string, config PocScanConfig) *PocScanner {
	return &PocScanner{
		ctx:         ctx,
		taskId:      taskId,
		config:      config,
		done:        make(chan struct{}),
		stop:        make(chan struct{}),
		pause:       make(chan struct{}),
		resume:      make(chan struct{}),
		results:     make(chan model.VulnerabilityInfo, 100),
		errors:      make(chan error, 100),
		stats: &ScanStats{
			TotalTargets: len(config.Targets),
		},
		paused:      false,
		scannedURLs: make(map[string]bool),
		urlPocCount: make(map[string]int),
	}
}

// NewPocScannerWithScannedURLs 创建 POC 扫描器（支持从未扫描的 URL 继续）
func NewPocScannerWithScannedURLs(ctx context.Context, taskId string, config PocScanConfig, scannedURLs []string) *PocScanner {
	scanner := NewPocScanner(ctx, taskId, config)
	scanner.scannedMu.Lock()
	for _, url := range scannedURLs {
		scanner.scannedURLs[url] = true
	}
	scannedCount := len(scanner.scannedURLs)
	scanner.scannedMu.Unlock()
	// 更新总数：已扫描数 + 当前未扫描数
	scanner.statsMu.Lock()
	scanner.stats.TotalTargets = scannedCount + len(config.Targets)
	scanner.stats.Completed = scannedCount // 已完成数初始化为已扫描数
	scanner.statsMu.Unlock()
	return scanner
}

// Start 启动扫描
func (s *PocScanner) Start() (chan model.VulnerabilityInfo, chan error) {
	go s.run()
	return s.results, s.errors
}

// Stop 停止扫描
func (s *PocScanner) Stop() {
	close(s.stop)
}

// Pause 暂停扫描
func (s *PocScanner) Pause() {
	s.pauseMu.Lock()
	defer s.pauseMu.Unlock()
	if !s.paused {
		s.paused = true
		select {
		case s.pause <- struct{}{}:
		default:
		}
	}
}

// Resume 恢复扫描
func (s *PocScanner) Resume() {
	s.pauseMu.Lock()
	defer s.pauseMu.Unlock()
	if s.paused {
		s.paused = false
		select {
		case s.resume <- struct{}{}:
		default:
		}
	}
}

// IsPaused 检查是否已暂停
func (s *PocScanner) IsPaused() bool {
	s.pauseMu.Lock()
	defer s.pauseMu.Unlock()
	return s.paused
}

// GetStats 获取统计信息
func (s *PocScanner) GetStats() *ScanStats {
	s.statsMu.Lock()
	defer s.statsMu.Unlock()
	copied := *s.stats
	return &copied
}

// MarkURLAsScanned 标记 URL 为已扫描
func (s *PocScanner) MarkURLAsScanned(url string) {
	s.scannedMu.Lock()
	defer s.scannedMu.Unlock()
	s.scannedURLs[url] = true
}

// IsURLScanned 检查 URL 是否已扫描
func (s *PocScanner) IsURLScanned(url string) bool {
	s.scannedMu.Lock()
	defer s.scannedMu.Unlock()
	return s.scannedURLs[url]
}

// GetScannedURLs 获取已扫描的 URL 列表
func (s *PocScanner) GetScannedURLs() []string {
	s.scannedMu.Lock()
	defer s.scannedMu.Unlock()
	result := make([]string, 0, len(s.scannedURLs))
	for url := range s.scannedURLs {
		result = append(result, url)
	}
	return result
}

// run 运行扫描逻辑
func (s *PocScanner) run() {
	defer close(s.done)
	defer close(s.results)
	defer close(s.errors)

	// 获取扫描配置
	configCtrl := NewPocConfigController()
	configCtrl.Startup(s.ctx)
	scanConfig, err := configCtrl.GetPocConfig()
	if err != nil {
		scanConfig = configCtrl.getDefaultConfig()
	}

	// 如果配置中没有指定代理，从全局代理管理器获取
	proxyURL := s.config.Proxy
	if proxyURL == "" && proxy.GlobalProxyManager != nil {
		globalProxy := proxy.GlobalProxyManager.GetGlobalProxy()
		if globalProxy.Enabled && globalProxy.Config != nil {
			proxyConfig := globalProxy.Config
			if proxyConfig.Type != "" && proxyConfig.Host != "" && proxyConfig.Port != "" {
				// 格式化代理URL
				scheme := strings.ToLower(proxyConfig.Type)
				proxyURL = fmt.Sprintf("%s://%s:%s", scheme, proxyConfig.Host, proxyConfig.Port)

				// 如果有认证信息，添加到URL中
				if proxyConfig.Auth != nil && proxyConfig.Auth.Username != "" {
					username := url.QueryEscape(proxyConfig.Auth.Username)
					password := url.QueryEscape(proxyConfig.Auth.Password)
					proxyURL = fmt.Sprintf("%s://%s:%s@%s:%s", scheme, username, password, proxyConfig.Host, proxyConfig.Port)
				}
				// 记录日志，确认代理已应用
				sendLog(s.ctx, "info", fmt.Sprintf("[漏洞扫描] 已应用全局代理: %s", scheme+"://"+proxyConfig.Host+":"+proxyConfig.Port))
			}
		}
	}
	if proxyURL != "" {
		// 隐藏密码的代理URL用于日志
		maskedProxyURL := proxyURL
		if strings.Contains(maskedProxyURL, "@") {
			// 如果有认证信息，隐藏密码部分
			parts := strings.Split(maskedProxyURL, "@")
			if len(parts) == 2 {
				authPart := parts[0]
				if strings.Contains(authPart, ":") {
					authParts := strings.Split(authPart, ":")
					if len(authParts) >= 3 {
						maskedProxyURL = fmt.Sprintf("%s:%s:***@%s", authParts[0], authParts[1], parts[1])
					}
				}
			}
		}
		sendLog(s.ctx, "info", fmt.Sprintf("[漏洞扫描] 使用代理: %s", maskedProxyURL))
	} else {
		sendLog(s.ctx, "warning", "[漏洞扫描] 未配置代理，将使用直连")
	}

	// 构建 Nuclei 选项列表
	var allOptions []model.NucleiOption
	totalPocCount := 0 // 计算总POC数量（所有URL的POC数量之和）
	s.urlPocMu.Lock()
	for _, target := range s.config.Targets {
		option := model.NucleiOption{
			SkipNucleiWithoutTags: s.config.SkipNucleiWithoutTags,
			URL:                   target,
			Tags:                  s.config.Tags,
			CustomTags:            s.config.CustomTags,
			TemplateFile:          s.config.TemplateFile,
			TemplateFolders:       s.config.TemplateFolders,
			CustomHeaders:         s.config.CustomHeaders,
			Proxy:                 proxyURL,
			Threads:               s.config.Threads,
			Timeout:               s.config.Timeout,
			RateLimit:             s.config.RateLimit,
			Retries:               s.config.Retries,
		}
		// 如果配置中没有设置，使用扫描配置中的默认值
		if option.Threads <= 0 {
			option.Threads = scanConfig.Threads
		}
		if option.Timeout <= 0 {
			option.Timeout = scanConfig.Timeout
		}
		if option.RateLimit <= 0 {
			option.RateLimit = scanConfig.RateLimit
		}
		if option.Retries < 0 {
			option.Retries = scanConfig.Retries
		}
		allOptions = append(allOptions, option)
		
		// 计算该URL对应的POC数量并记录
		templatesToUse := resolveTemplatesForRun(option)
		pocCount := len(templatesToUse)
		if pocCount == 0 {
			// 如果该URL没有POC，设置为1，避免进度计算错误
			pocCount = 1
		}
		s.urlPocCount[target] = pocCount
		totalPocCount += pocCount
	}
	s.urlPocMu.Unlock()
	
	// 初始化总任务数为URL数量（不再使用URL × POC）
	urlCount := len(s.config.Targets)
	if urlCount == 0 {
		urlCount = 1 // 避免除零
	}
	s.statsMu.Lock()
	s.stats.TotalTargets = urlCount
	s.statsMu.Unlock()

	// 创建控制上下文
	ctrlCtx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	// 监听停止信号
	stopDone := make(chan struct{})
	go func() {
		select {
		case <-s.stop:
			cancel()
			close(stopDone)
		case <-s.ctx.Done():
			cancel()
			close(stopDone)
		}
	}()

	// 监听暂停/恢复信号
	pauseDone := make(chan struct{})
	go func() {
		defer close(pauseDone)
		for {
			select {
			case <-s.pause:
				// 收到暂停信号，取消上下文以停止扫描
				cancel()
				sendLog(s.ctx, "info", fmt.Sprintf("[漏洞扫描] 任务 %s 已暂停", s.taskId))
				// 等待恢复信号
				<-s.resume
				sendLog(s.ctx, "info", fmt.Sprintf("[漏洞扫描] 任务 %s 已恢复", s.taskId))
				// 重新创建上下文（但 Nuclei 引擎已经停止，无法恢复）
				// 注意：由于 Nuclei 引擎已经启动，取消 ctrlCtx 后无法恢复
				// 实际的暂停需要在 callnuclei.go 中检查 ctrlCtx.Err()
				ctrlCtx, cancel = context.WithCancel(s.ctx)
			case <-stopDone:
				return
			case <-s.ctx.Done():
				return
			}
		}
	}()

	// 启动 Nuclei 引擎（使用自定义回调来收集结果）
	progressCb := func(completed, total int, url string) {
		// 检查是否暂停，如果暂停则不更新进度
		if s.IsPaused() {
			return
		}
		// 标记 URL 为已扫描
		if url != "" {
			s.MarkURLAsScanned(url)
		}
		s.updateProgress(completed, total)
	}

	if s.config.ThreadSafe {
		NewThreadSafeNucleiEngineWithCallback(s.ctx, ctrlCtx, s.taskId, allOptions, progressCb, func(vuln model.VulnerabilityInfo) {
			// 检查是否暂停或停止
			if s.IsPaused() {
				return
			}
			select {
			case s.results <- vuln:
				s.incrementVulnCount()
			case <-s.stop:
				return
			default:
				// 如果 channel 已满，跳过
			}
		})
	} else {
		NewNucleiEngineWithCallback(s.ctx, ctrlCtx, s.taskId, allOptions, progressCb, func(vuln model.VulnerabilityInfo) {
			// 检查是否暂停或停止
			if s.IsPaused() {
				return
			}
			select {
			case s.results <- vuln:
				s.incrementVulnCount()
			case <-s.stop:
				return
			default:
				// 如果 channel 已满，跳过
			}
		})
	}

	// 进度由Nuclei引擎的progressCb回调函数更新，不需要在这里强制设置为100%
}

func (s *PocScanner) incrementVulnCount() {
	s.statsMu.Lock()
	s.stats.VulnCount++
	s.statsMu.Unlock()
}

func (s *PocScanner) updateProgress(completed, total int) {
	s.statsMu.Lock()
	
	// TotalTargets 已经在 run() 函数中初始化为URL数量
	// 如果TotalTargets还没有设置（可能是旧代码路径），使用URL数量作为备选
	if s.stats.TotalTargets == 0 {
		s.stats.TotalTargets = len(s.config.Targets)
		if s.stats.TotalTargets == 0 {
			s.stats.TotalTargets = 1 // 避免除零
		}
	}
	
	// 统计已扫描的URL数量（不再使用URL × POC）
	s.scannedMu.Lock()
	completedURLCount := len(s.scannedURLs)
	s.scannedMu.Unlock()
	
	// 更新已完成数
	s.stats.Completed = completedURLCount
	
	// 计算进度比例
	if s.stats.TotalTargets > 0 {
		s.stats.ProgressRatio = float64(s.stats.Completed) / float64(s.stats.TotalTargets)
		// 确保进度不超过100%
		if s.stats.ProgressRatio > 1.0 {
			s.stats.ProgressRatio = 1.0
		}
	} else {
		s.stats.ProgressRatio = 0
	}
	s.statsMu.Unlock()
}
