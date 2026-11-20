package hostscan

import (
	"context"
	"fmt"
	"sync"
	"time"

	"EasyTools/app/controller/system"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// HostscanController 主机扫描控制器
type HostscanController struct {
	ctx     context.Context
	scanner *HostScanner
	mu      sync.Mutex
	system.Base
}

// HostscanResultWithTab 事件负载：扫描结果 + 前端标签页ID
type HostscanResultWithTab struct {
	HostscanResult
	TabId string `json:"tabId"`
}

// HostscanProgressEvent 事件负载：进度信息 + 前端标签页ID
type HostscanProgressEvent struct {
	Stats HostScanStats `json:"stats"`
	TabId string        `json:"tabId"`
}

// HostscanAliveCheckWithTab 事件负载：存活检测结果 + 前端标签页ID
type HostscanAliveCheckWithTab struct {
	AliveCheckResult
	TabId string `json:"tabId"`
}

type HostscanNoticeWithTab struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Host    string `json:"host,omitempty"`
	TabId   string `json:"tabId"`
}

func NewHostscanController() *HostscanController {
	return &HostscanController{}
}

func (p *HostscanController) Startup(ctx context.Context) {
	p.ctx = ctx
}

// 统一日志事件发送：通过 log_event 推到前端全局日志
func (p *HostscanController) sendLog(level, message string) {
	if p.ctx == nil {
		return
	}
	eventData := map[string]interface{}{
		"level":   level,
		"message": message,
		"time":    time.Now().Format("15:04:05"),
		"source":  "主机扫描",
	}
	runtime.EventsEmit(p.ctx, "log_event", eventData)
}

// StartHostScan 启动主机扫描
func (p *HostscanController) StartHostScan(configJSON string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.scanner != nil {
		p.scanner.Stop()
		p.scanner = nil
	}

	scanner, err := NewHostScannerFromJSON(configJSON)
	if err != nil {
		return err
	}
	p.scanner = scanner

	// 启动扫描时写入全局日志
	stats := scanner.GetStats()
	p.sendLog("info", fmt.Sprintf("主机扫描任务启动：预计扫描 %d 个端口目标组合", stats.TotalPorts))

	go p.runScan(scanner)
	return nil
}

// StopHostScan 停止扫描
func (p *HostscanController) StopHostScan() error {
	p.mu.Lock()
	scanner := p.scanner
	if scanner != nil {
		// 先停止扫描器
		scanner.Stop()
		// 立即设置为nil，确保后续检查能检测到停止
		p.scanner = nil
		p.sendLog("info", "主机扫描任务已手动停止")
	}
	p.mu.Unlock()
	return nil
}

func (p *HostscanController) runScan(scanner *HostScanner) {
	// 存活检测在独立 goroutine 中完成；此前在 scan 启动前读 GetAliveCheckResult 几乎总为 nil，导致前端收不到结果
	go func() {
		<-scanner.AlivePhaseDone()
		if aliveResult := scanner.GetAliveCheckResult(); aliveResult != nil {
			payload := HostscanAliveCheckWithTab{
				AliveCheckResult: *aliveResult,
				TabId:            scanner.tabId,
			}
			runtime.EventsEmit(p.ctx, "hostscanAliveCheck", payload)
			if aliveResult.TotalIPs > 0 && aliveResult.AliveIPs == 0 {
				p.sendLog("warning", "未检测到存活 IP（禁 ping 且探测端口均未响应时会如此）。可关闭「IP存活检测」直接扫端口，或在存活检测中改用 TCP/HTTP。")
			} else {
				p.sendLog("info", fmt.Sprintf(
					"IP存活检测完成：总IP数 %d，存活 %d，不存活 %d，域名 %d",
					aliveResult.TotalIPs,
					aliveResult.AliveIPs,
					aliveResult.DeadIPs,
					aliveResult.DomainCount,
				))
			}
		}
	}()

	results, errs := scanner.Start()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for result := range results {
			// 携带 tabId，方便前端精确路由到对应的标签页
			payload := HostscanResultWithTab{
				HostscanResult: result,
				TabId:          scanner.tabId,
			}
			runtime.EventsEmit(p.ctx, "hostscanResult", payload)

			// 仅对开放端口写入全局日志，避免日志量过大
			if result.Open {
				p.sendLog("info", fmt.Sprintf(
					"发现开放主机端口: %s:%d (%s)",
					result.Target,
					result.Port,
					result.Service,
				))
			}
		}
	}()

	go func() {
		defer wg.Done()
		for err := range errs {
			runtime.EventsEmit(p.ctx, "hostscanError", map[string]interface{}{
				"message": err.Error(),
				"tabId":   scanner.tabId,
			})

			// 同步错误到全局日志
			p.sendLog("error", fmt.Sprintf("主机扫描错误: %v", err))
		}
	}()

	// 优化：使用更频繁的进度更新（500ms），提供更流畅的体验
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-scanner.done:
			wg.Wait()
			for _, msg := range scanner.PopNotices() {
				runtime.EventsEmit(p.ctx, "hostscanNotice", HostscanNoticeWithTab{
					Type:    msg.Type,
					Message: msg.Message,
					Host:    msg.Host,
					TabId:   scanner.tabId,
				})
				p.sendLog("warning", msg.Message)
			}
			// 发送最终统计
			finalStats := scanner.GetStats()
			runtime.EventsEmit(p.ctx, "hostscanProgress", HostscanProgressEvent{
				Stats: finalStats,
				TabId: scanner.tabId,
			})
			
			// 携带 tabId，方便前端精确路由到对应的标签页
			runtime.EventsEmit(p.ctx, "hostscanComplete", map[string]interface{}{
				"tabId": scanner.tabId,
				"stats": finalStats,
			})

			// 扫描完成写入全局日志
			p.sendLog("info", fmt.Sprintf(
				"主机扫描完成，总端口: %d，已检查: %d，开放: %d，关闭/过滤: %d，平均速度: %.2f 个端口/秒，用时: %.2f 秒",
				finalStats.TotalPorts,
				finalStats.Checked,
				finalStats.OpenCount,
				finalStats.ClosedOrFiltered,
				finalStats.PortsPerSecond,
				finalStats.ElapsedSeconds,
			))

			p.mu.Lock()
			if p.scanner == scanner {
				p.scanner = nil
			}
			p.mu.Unlock()
			return
		case <-ticker.C:
			// 检查是否已停止
			p.mu.Lock()
			shouldStop := p.scanner != scanner || p.scanner == nil
			currentScanner := p.scanner
			p.mu.Unlock()
			
			if shouldStop {
				// 确保停止扫描器
				scanner.Stop()
				// 等待所有goroutine完成
				wg.Wait()
				// 清理状态
				p.mu.Lock()
				if p.scanner == scanner {
					p.scanner = nil
				}
				p.mu.Unlock()
				p.sendLog("info", "主机扫描任务已停止")
				return
			}

			// 只有在扫描器仍然活跃时才发送进度
			if currentScanner == scanner {
				stats := scanner.GetStats()
				// 携带 tabId，方便前端精确路由到对应的标签页
				payload := HostscanProgressEvent{
					Stats: stats,
					TabId: scanner.tabId,
				}
				runtime.EventsEmit(p.ctx, "hostscanProgress", payload)
				for _, msg := range scanner.PopNotices() {
					runtime.EventsEmit(p.ctx, "hostscanNotice", HostscanNoticeWithTab{
						Type:    msg.Type,
						Message: msg.Message,
						Host:    msg.Host,
						TabId:   scanner.tabId,
					})
					p.sendLog("warning", msg.Message)
				}
			}
		case <-p.ctx.Done():
			scanner.Stop()
			wg.Wait()
			p.sendLog("warning", "主机扫描因上下文取消而终止")
			return
		}
	}
}
