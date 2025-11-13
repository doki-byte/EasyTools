package proxy

import (
	util "EasyTools/app/controller/proxy"
	"EasyTools/app/controller/proxy/xui" // 导入xui包
	"fmt"
	"github.com/imroc/req/v3"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

func (p *Proxy) CheckDatasets() Response {
	profile := p.GetProfile()
	p.Debug("当前Xui模式: %s", profile.Xui)

	// 根据profile.Xui判断使用哪种逻辑
	if profile.Xui == "1" || profile.Xui == "2" {
		return p.checkXUIProxies()
	} else {
		return p.checkRegularProxies()
	}
}

// 原来的代理检测逻辑
func (p *Proxy) checkRegularProxies() Response {
	availableProxies := make(chan string, p.config.CoroutineCount)
	var wg sync.WaitGroup

	checkedProxies := 0
	var mu sync.Mutex

	runtime.EventsEmit(p.ctx, "start_task", p.config.GetProfile())
	for _, ip := range p.config.LiveProxyLists {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			if p.checkProxy(ip) {
				availableProxies <- ip
			}

			mu.Lock()
			checkedProxies++
			mu.Unlock()

			progress := float64(checkedProxies) / float64(p.config.AllProxies)
			progressStr := fmt.Sprintf("%.2f", progress)
			runtime.EventsEmit(p.ctx, "task_progress", progressStr)
		}(ip)
	}

	go func() {
		wg.Wait()
		close(availableProxies)
	}()

	var availableProxiesList []string
	for proxy := range availableProxies {
		availableProxiesList = append(availableProxiesList, proxy)
	}

	p.config.SetLiveProxies(availableProxiesList)

	baseDir := util.GetAppBaseDir()
	path := filepath.Join(baseDir, "proxy_success.txt")

	// 使用 map 去重代理列表
	proxySet := make(map[string]struct{})
	for _, proxy := range availableProxiesList {
		proxySet[proxy] = struct{}{}
	}

	if len(proxySet) != 0 {
		// 检查文件是否已存在，如果不存在则创建
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		if err != nil {
			p.Error("无法打开文件: %v", err)
		}
		defer file.Close()

		// 写入去重后的代理地址，并保证每行一个代理
		for proxy := range proxySet {
			_, err := file.WriteString(proxy + "\n")
			if err != nil {
				p.Error("写入失败: %v", err)
			}
		}
	}

	msg := fmt.Sprintf("共有 %d 条有效数据", p.config.LiveProxies)
	runtime.EventsEmit(p.ctx, "is_ready", p.config.LiveProxies)
	runtime.EventsEmit(p.ctx, "log_update", fmt.Sprintf("[INF] %s ", msg))

	return p.startListening()
}

// xui检测逻辑
func (p *Proxy) checkXUIProxies() Response {
	// 创建事件发射函数
	eventEmitter := func(event string, data ...interface{}) {
		if event == "log_update" {
			runtime.EventsEmit(p.ctx, "log_update", data...)
		} else if event == "task_progress" {
			runtime.EventsEmit(p.ctx, "task_progress", data...)
		}
	}

	// 将字符串模式转换为整数
	xuiMode := p.config.Xui

	// 创建xui扫描器，传入模式参数
	scanner := xui.NewXUIScanner(p.ctx, xuiMode, eventEmitter)

	// 发射开始任务事件
	runtime.EventsEmit(p.ctx, "start_task", p.config.GetProfile())

	// 同步执行扫描
	scanner.ScanURLs(p.config.LiveProxyLists, p.config.CoroutineCount)

	// 扫描完成后发射完成事件
	runtime.EventsEmit(p.ctx, "is_ready", []string{})

	// 根据模式显示不同的完成消息
	var completionMsg string
	if xuiMode == "1" {
		completionMsg = "[INF] x-ui面板扫描任务完成(不添加HTTP代理),结果保存在3x-ui可用代理集合.xlsx中,请手动打开查看"
	} else if xuiMode == "2" {
		completionMsg = "[INF] x-ui面板扫描任务完成(已添加HTTP代理),结果保存在3x-ui可用代理集合.xlsx中,请手动打开查看"
	} else {
		completionMsg = "[INF] x-ui面板扫描任务完成,结果保存在3x-ui可用代理集合.xlsx中,请手动打开查看"
	}

	runtime.EventsEmit(p.ctx, "log_update", completionMsg)

	// 对于xui模式，我们不启动监听，直接返回成功
	var resultMsg string
	if xuiMode == "1" {
		resultMsg = "x-ui扫描完成(不添加HTTP代理),结果保存在3x-ui可用代理集合.xlsx中,请手动打开查看"
	} else if xuiMode == "2" {
		resultMsg = "x-ui扫描完成(已添加HTTP代理),结果保存在3x-ui可用代理集合.xlsx中,请手动打开查看"
	} else {
		resultMsg = "x-ui扫描完成,结果保存在3x-ui可用代理集合.xlsx中,请手动打开查看"
	}

	return p.successResponse(resultMsg, map[string]interface{}{
		"mode":    "xui",
		"xuiMode": xuiMode,
		"message": "节点信息已保存到Excel文件",
	})
}

// 原有的checkProxy方法保持不变
func (p *Proxy) checkProxy(proxyIP string) bool {
	client := req.C()
	client.SetProxyURL(fmt.Sprintf("socks5://%s", proxyIP))
	timeout, err := strconv.Atoi(p.config.Timeout)
	if err != nil {
		p.Debug("Invalid timeout value: %v", err)
	}
	client.SetTimeout(time.Duration(timeout) * time.Second)
	resp, err := client.R().Get("http://myip.ipip.net")
	if err != nil {
		runtime.EventsEmit(p.ctx, "log_update", fmt.Sprintf("[ERR] %s <-- : --> %v", proxyIP, err))
		return false
	}

	if strings.Contains(resp.String(), "当前 IP") {
		runtime.EventsEmit(p.ctx, "log_update", fmt.Sprintf("[INF] 有效值 %s ", resp.String()))
		return true
	}

	runtime.EventsEmit(p.ctx, "log_update", fmt.Sprintf("[WAR] 不稳定 %s -- %v", proxyIP, err))
	return false
}
