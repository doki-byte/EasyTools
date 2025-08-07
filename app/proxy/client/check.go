package proxy

import (
	"fmt"
	"github.com/imroc/req/v3"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"os"
	runtime2 "runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

func (p *Proxy) CheckDatasets() Response {
	p.Debug("workers: ", p.config.CoroutineCount)
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

	// 获取配置文件路径
	optSys := runtime2.GOOS
	path := ""
	if optSys == "windows" {
		path = "EasyToolsFiles" + "\\proxy_success.txt"
	} else {
		path = "EasyToolsFiles" + "/proxy_success.txt"
	}

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
	p.Debug("msg: ", msg)
	runtime.EventsEmit(p.ctx, "is_ready", p.config.LiveProxies)
	runtime.EventsEmit(p.ctx, "log_update", fmt.Sprintf("[INF] %s ", msg))

	return p.startListening()
}

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
		p.Error("不可用： %s, 错误: %v\n", proxyIP, err)
		runtime.EventsEmit(p.ctx, "log_update", fmt.Sprintf("[ERR] %s <-- : --> %v", proxyIP, err))
		return false
	}

	if strings.Contains(resp.String(), "当前 IP") {
		p.Error("可用： %s\n", proxyIP, "resp: ", resp.String())
		runtime.EventsEmit(p.ctx, "log_update", fmt.Sprintf("[INF] 有效值 %s ", resp.String()))
		return true
	}

	p.Error("不可用： %s\n", proxyIP)
	runtime.EventsEmit(p.ctx, "log_update", fmt.Sprintf("[WAR] 不稳定 %s -- %v", proxyIP, err))
	return false
}
