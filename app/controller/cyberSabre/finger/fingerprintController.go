package finger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"EasyTools/app/controller/system"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// FingerprintController 指纹识别控制器
type FingerprintController struct {
	ctx     context.Context
	scanner *FingerprintScanner
	mu      sync.Mutex
	system.Base
}

// FingerprintResultWithTab 事件负载：指纹结果 + 前端标签页ID
type FingerprintResultWithTab struct {
	FingerprintResult
	TabId string `json:"tabId"`
}

// FingerprintProgressEvent 事件负载：进度信息 + 前端标签页ID
type FingerprintProgressEvent struct {
	Percentage int                    `json:"percentage"`
	Stats      map[string]interface{} `json:"stats"`
	TabId      string                 `json:"tabId"`
}

// NewFingerprintController 创建控制器实例
func NewFingerprintController() *FingerprintController {
	return &FingerprintController{}
}

// Startup 初始化上下文
func (f *FingerprintController) Startup(ctx context.Context) {
	f.ctx = ctx
}

// 统一日志事件发送：通过 log_event 推到前端全局日志
func (f *FingerprintController) sendLog(level, message string) {
	if f.ctx == nil {
		return
	}
	eventData := map[string]interface{}{
		"level":   level,
		"message": message,
		"time":    time.Now().Format("15:04:05"),
		"source":  "指纹识别",
	}
	runtime.EventsEmit(f.ctx, "log_event", eventData)
}

// StartFingerprintScan 启动指纹识别
func (f *FingerprintController) StartFingerprintScan(configJSON string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.scanner != nil {
		f.scanner.Stop()
		f.scanner = nil
	}

	scanner, err := NewFingerprintScannerFromJSON(configJSON, f.GetAppPath())
	if err != nil {
		return err
	}

	f.scanner = scanner
	go f.runScan(scanner)

	// 写入全局日志
	stats := scanner.GetStats()
	f.sendLog("info", fmt.Sprintf("指纹识别任务启动：目标数量 %d", stats.TotalTargets))

	return nil
}

// StopFingerprintScan 停止扫描
func (f *FingerprintController) StopFingerprintScan() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.scanner != nil {
		f.scanner.Stop()
		f.scanner = nil
		f.sendLog("info", "指纹识别任务已手动停止")
	}
	return nil
}

// ListCustomFingerRules 返回用户自定义指纹列表
func (f *FingerprintController) ListCustomFingerRules() ([]fingerRule, error) {
	appPath := f.GetAppPath()
	rules := loadCustomFingerRules(appPath)
	return rules, nil
}

// AddCustomFingerRule 新增用户指纹，写入应用数据目录 config/finger_custom.json
func (f *FingerprintController) AddCustomFingerRule(rule fingerRule) error {
	appPath := f.GetAppPath()
	if appPath == "" {
		return fmt.Errorf("无法获取应用数据目录")
	}
	if err := addCustomFingerRule(appPath, rule); err != nil {
		return err
	}

	// 写入成功后，向全局日志输出提示
	f.sendLog("info", fmt.Sprintf("已新增指纹：%s (%s/%s)", rule.CMS, rule.Method, rule.Location))
	return nil
}

// ClearCustomFingerRules 清空用户自定义指纹（保留内置指纹）
func (f *FingerprintController) ClearCustomFingerRules() error {
	appPath := f.GetAppPath()
	path := filepath.Join(appPath, "config", customFingerFile)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	f.sendLog("info", "已清空自定义指纹规则")
	return nil
}

func (f *FingerprintController) runScan(scanner *FingerprintScanner) {
	results, errs := scanner.Start()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for result := range results {
			// 携带 tabId，方便前端精确路由到对应的标签页
			payload := FingerprintResultWithTab{
				FingerprintResult: result,
				TabId:             scanner.tabId,
			}
			runtime.EventsEmit(f.ctx, "fingerprintResult", payload)

			// 重要结果写入全局日志（info 级）
			// 使用 FingerprintResult 当前结构体中的字段进行记录
			f.sendLog("info", fmt.Sprintf(
				"指纹识别结果：URL %s，状态码: %d，标题: %s，CMS: %s，技术栈: %v",
				result.URL,
				result.Status,
				result.Title,
				result.CMS,
				result.Technologies,
			))
		}
	}()

	go func() {
		defer wg.Done()
		for err := range errs {
			runtime.EventsEmit(f.ctx, "fingerprintError", map[string]interface{}{
				"message": err.Error(),
				"tabId":   scanner.tabId,
			})

			// 同步错误到全局日志
			f.sendLog("error", fmt.Sprintf("指纹识别错误: %v", err))
		}
	}()

		// 优化：使用500ms间隔，提供更流畅的进度显示
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-scanner.done:
				wg.Wait()
				// 发送最终统计
				finalStats := scanner.GetStats()
				runtime.EventsEmit(f.ctx, "fingerprintProgress", FingerprintProgressEvent{
					Percentage: int(finalStats.ProgressPercent),
					Stats: map[string]interface{}{
						"totalTargets":     finalStats.TotalTargets,
						"completed":        finalStats.Completed,
						"success":          finalStats.Success,
						"failed":           finalStats.Failed,
						"targetsPerSecond": finalStats.TargetsPerSecond,
						"progressPercent":  finalStats.ProgressPercent,
						"elapsedSeconds":   finalStats.ElapsedSeconds,
						"estimatedSeconds": finalStats.EstimatedSeconds,
					},
					TabId: scanner.tabId,
				})
				
				// 携带 tabId，方便前端精确路由到对应的标签页
				runtime.EventsEmit(f.ctx, "fingerprintComplete", map[string]interface{}{
					"tabId": scanner.tabId,
					"stats": finalStats,
				})

				// 识别完成写入全局日志
				f.sendLog("info", fmt.Sprintf(
					"指纹识别完成：总目标 %d，已完成 %d，成功 %d，失败 %d，平均速度 %.2f 目标/秒，用时: %.2f 秒",
					finalStats.TotalTargets,
					finalStats.Completed,
					finalStats.Success,
					finalStats.Failed,
					finalStats.TargetsPerSecond,
					finalStats.ElapsedSeconds,
				))
				f.mu.Lock()
				if f.scanner == scanner {
					f.scanner = nil
				}
				f.mu.Unlock()
				return
			case <-ticker.C:
				// 检查是否已停止
				f.mu.Lock()
				shouldStop := f.scanner != scanner || f.scanner == nil
				currentScanner := f.scanner
				f.mu.Unlock()

				if shouldStop {
					scanner.Stop()
					wg.Wait()
					f.mu.Lock()
					if f.scanner == scanner {
						f.scanner = nil
					}
					f.mu.Unlock()
					f.sendLog("info", "指纹识别任务已停止")
					return
				}

				// 只有在扫描器仍然活跃时才发送进度
				if currentScanner == scanner {
					stats := scanner.GetStats()
					percentage := int(stats.ProgressPercent)
					payload := FingerprintProgressEvent{
						Percentage: percentage,
						Stats: map[string]interface{}{
							"totalTargets":     stats.TotalTargets,
							"completed":        stats.Completed,
							"success":          stats.Success,
							"failed":           stats.Failed,
							"targetsPerSecond": stats.TargetsPerSecond,
							"progressPercent":  stats.ProgressPercent,
							"elapsedSeconds":   stats.ElapsedSeconds,
							"estimatedSeconds": stats.EstimatedSeconds,
						},
						TabId: scanner.tabId,
					}
					runtime.EventsEmit(f.ctx, "fingerprintProgress", payload)
				}
		case <-f.ctx.Done():
			wg.Wait()
			return
		}
	}
}
