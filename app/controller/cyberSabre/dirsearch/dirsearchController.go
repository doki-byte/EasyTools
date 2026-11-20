package dirsearch

import (
	"EasyTools/app/controller/system"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// DirsearchController 目录扫描控制器
type DirsearchController struct {
	ctx      context.Context
	scanners map[*Scanner]bool // 支持多个scanner（多URL扫描）
	mu       sync.Mutex
	system.Base
}

// NewDirsearchController 创建目录扫描控制器
func NewDirsearchController() *DirsearchController {
	return &DirsearchController{
		scanners: make(map[*Scanner]bool),
	}
}

// Startup 启动时设置上下文
func (d *DirsearchController) Startup(ctx context.Context) {
	d.ctx = ctx
}

// 统一日志事件发送：通过 log_event 推到前端全局日志
func (d *DirsearchController) sendLog(level, message string) {
	if d.ctx == nil {
		return
	}
	eventData := map[string]interface{}{
		"level":   level,
		"message": message,
		"time":    time.Now().Format("15:04:05"),
		"source":  "目录扫描",
	}
	runtime.EventsEmit(d.ctx, "log_event", eventData)
}

// StartDirectoryScan 开始目录扫描（支持多URL）
func (d *DirsearchController) StartDirectoryScan(configJSON string) error {
	d.mu.Lock()

	// 如果已有扫描在运行，先停止所有
	for scanner := range d.scanners {
		scanner.Stop()
	}
	d.scanners = make(map[*Scanner]bool)
	d.mu.Unlock()

	// 解析配置
	var config ScanConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return err
	}

	// 处理多URL扫描
	targets := []string{}
	if len(config.Targets) > 0 {
		targets = config.Targets
	} else if config.Target != "" {
		targets = []string{config.Target}
	}

	if len(targets) == 0 {
		return fmt.Errorf("至少需要提供一个目标URL")
	}

	d.sendLog("info", fmt.Sprintf("目录扫描任务启动，共 %d 个目标", len(targets)))

	// 为每个URL创建扫描任务
	for _, target := range targets {
		target = strings.TrimSpace(target)
		if target == "" {
			continue
		}

		// 为当前target创建配置
		targetConfig := config
		targetConfig.Target = target
		targetConfig.Targets = nil

		configBytes, err := json.Marshal(targetConfig)
		if err != nil {
			continue
		}

		// 创建扫描器
		scanner, err := NewScannerFromJSON(string(configBytes))
		if err != nil {
			d.sendLog("error", fmt.Sprintf("创建目录扫描器失败 [%s]: %v", target, err))
			continue
		}

		// 注册scanner
		d.mu.Lock()
		d.scanners[scanner] = true
		d.mu.Unlock()

		// 启动扫描（每个URL独立扫描）
		d.sendLog("info", fmt.Sprintf("开始扫描目标: %s", target))
		go d.runScan(scanner)
	}

	return nil
}

// StopDirectoryScan 停止目录扫描
func (d *DirsearchController) StopDirectoryScan() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// 停止所有正在运行的scanner
	for scanner := range d.scanners {
		scanner.Stop()
	}
	// 清空scanner映射，runScan会在完成后自动清理
	d.scanners = make(map[*Scanner]bool)

	return nil
}

// GetDictList 获取字典文件列表
func (d *DirsearchController) GetDictList() ([]map[string]interface{}, error) {
	dictDir := filepath.Join(d.GetAppPath(), "tools", "dict", "dirsearch")

	// 如果目录不存在，返回空列表
	if _, err := os.Stat(dictDir); os.IsNotExist(err) {
		return []map[string]interface{}{}, nil
	}

	var dictList []map[string]interface{}

	err := filepath.Walk(dictDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只处理文件，跳过目录
		if !info.IsDir() {
			dictList = append(dictList, map[string]interface{}{
				"name": info.Name(),
				"size": info.Size(),
				"path": path,
			})
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return dictList, nil
}

// runScan 运行扫描并发送事件
func (d *DirsearchController) runScan(scanner *Scanner) {
	results, errors := scanner.Start()

	// 统计信息
	var totalRequests int64
	var success int64
	var failed int64
	var found int64

	// 处理结果
	// 使用计数器限制日志发送频率，避免日志过多导致前端卡顿
	var logCounter int64
	go func() {
		for result := range results {
			found++
			totalRequests++
			if result.Status >= 200 && result.Status < 400 {
				success++
			} else {
				failed++
			}

			// 发送结果事件，携带 tabId 方便前端精确路由到对应的标签页
			payload := map[string]interface{}{
				"url":               result.URL,
				"status":            result.Status,
				"size":              result.Size,
				"responsePacketSize": result.ResponsePacketSize,
				"time":              result.Time,
				"title":             result.Title,
				"headers":           result.Headers,
				"content":           result.Content,
				"request":           result.Request,
				"found_by":          result.FoundBy,
				"requestPacket":     result.RequestPacket,
				"responsePacket":    result.ResponsePacket,
				"tabId":             scanner.tabId,
			}
			runtime.EventsEmit(d.ctx, "dirsearchResult", payload)

			// 将重要结果写入全局日志（限制频率：每10个成功结果才记录一次，避免日志过于庞大导致前端卡顿）
			if result.Status >= 200 && result.Status < 400 {
				logCounter++
				// 只记录每10个结果中的第一个，或者状态码为200的重要结果
				if logCounter%10 == 1 || result.Status == 200 {
					d.sendLog("info", fmt.Sprintf("[%d] %s (大小: %d, 耗时: %dms)",
						result.Status, result.URL, result.Size, result.Time))
				}
			}
		}
	}()

	// 处理错误
	go func() {
		for err := range errors {
			failed++
			totalRequests++
			runtime.EventsEmit(d.ctx, "dirsearchError", map[string]interface{}{
				"message": err.Error(),
				"tabId":   scanner.tabId,
			})

			// 同步到全局日志
			d.sendLog("error", fmt.Sprintf("目录扫描错误: %v", err))
		}
	}()

	// 定期发送进度更新（优化：使用500ms间隔，提供更流畅的进度显示）
	var lastLogPercentage int = -1
	var lastLogTime time.Time = time.Now()
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond) // 优化：500ms更新一次，提供流畅体验
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if scanner == nil {
					return
				}

				stats := scanner.GetStats()
				percentage := int(stats.ProgressPercent)

				runtime.EventsEmit(d.ctx, "dirsearchProgress", map[string]interface{}{
					"percentage": percentage,
					"stats": map[string]interface{}{
						"totalRequests":     stats.TotalRequests,
						"plannedTotal":      stats.PlannedTotal,
						"success":           stats.Success,
						"failed":            stats.Failed,
						"found":             stats.Found,
						"requestsPerSecond": stats.RequestsPerSecond,
						"progressPercent":   stats.ProgressPercent,
						"elapsedSeconds":    stats.ElapsedSeconds,
						"estimatedSeconds":  stats.EstimatedSeconds,
					},
					"tabId": scanner.tabId,
				})

				// 周期性将概括性进度写入全局日志（debug 级别，降低频率避免前端卡顿）
				// 只在进度有明显变化时（每10%）或每30秒记录一次
				now := time.Now()
				shouldLog := false
				if percentage != lastLogPercentage && percentage%10 == 0 {
					// 进度每10%记录一次
					shouldLog = true
					lastLogPercentage = percentage
				} else if now.Sub(lastLogTime) >= 30*time.Second {
					// 每30秒至少记录一次
					shouldLog = true
					lastLogTime = now
				}

				if shouldLog {
					d.sendLog("debug", fmt.Sprintf(
						"目录扫描进度: %d%%，总请求: %d，成功: %d，失败: %d，发现: %d，速度: %.2f req/s",
						percentage,
						stats.TotalRequests,
						stats.Success,
						stats.Failed,
						stats.Found,
						stats.RequestsPerSecond,
					))
				}
			case <-scanner.ctx.Done():
				return
			}
		}
	}()

	// 等待扫描完成
	<-scanner.done

	// 发送完成事件，携带 tabId
	runtime.EventsEmit(d.ctx, "dirsearchComplete", map[string]interface{}{
		"tabId": scanner.tabId,
	})

	// 写入全局日志
	stats := scanner.GetStats()
	d.sendLog("info", fmt.Sprintf(
		"目录扫描完成，总请求: %d，成功: %d，失败: %d，发现: %d，平均速度: %.2f req/s",
		stats.TotalRequests,
		stats.Success,
		stats.Failed,
		stats.Found,
		stats.RequestsPerSecond,
	))

	// 从scanners映射中移除
	d.mu.Lock()
	delete(d.scanners, scanner)
	d.mu.Unlock()
}
