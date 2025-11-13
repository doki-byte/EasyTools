package request

import (
	util "EasyTools/app/controller/proxy"
	"context"
	"encoding/json"
	"fmt"
	"github.com/labstack/gommon/log"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"os"
	"path/filepath"
	"strconv"
)

type ProxyInfo struct {
	Key     string `json:"key"`
	Address string `json:"address"`
	Source  string `json:"source"`
	Kind    string `json:"kind"`
}

type ProxyFetcher interface {
	Fetch() ([]string, error)
	Name() string
}

type ProxyManager struct {
	fetchers   []ProxyFetcher
	allProxies []string
	proxies    []ProxyInfo
	ctx        context.Context
}

func NewProxyManager(fetchers []ProxyFetcher) *ProxyManager {
	return &ProxyManager{
		fetchers: fetchers,
	}
}

func (pm *ProxyManager) SetContext(ctx context.Context) {
	pm.ctx = ctx
}

func (pm *ProxyManager) RenderTable() ([]byte, error) {
	marshal, err := json.Marshal(pm.proxies)
	if err != nil {
		return nil, err
	}
	return marshal, nil
}

func (pm *ProxyManager) FetchAll() ([]string, error) {
	// 发射开始事件
	if pm.ctx != nil {
		runtime.EventsEmit(pm.ctx, "fetch_start", "开始获取代理数据...")
	}

	i := 1
	for _, fetcher := range pm.fetchers {
		sourceName := fetcher.Name()

		// 发射进度事件 - 开始获取某个源
		if pm.ctx != nil {
			progressMsg := map[string]interface{}{
				"message": "正在从 " + sourceName + " 获取代理数据...",
				"source":  sourceName,
			}
			runtime.EventsEmit(pm.ctx, "fetch_progress", progressMsg)
			fmt.Printf("发射进度事件: %s\n", progressMsg["message"])
		}

		proxies, err := fetcher.Fetch()
		if err != nil {
			// 发射错误事件
			if pm.ctx != nil {
				errorMsg := map[string]interface{}{
					"message": "从 " + sourceName + " 获取失败: " + err.Error(),
					"source":  sourceName,
				}
				runtime.EventsEmit(pm.ctx, "fetch_progress", errorMsg)
			}
			continue
		}

		// 发射进度事件 - 获取到数据
		if pm.ctx != nil && len(proxies) > 0 {
			progressMsg := map[string]interface{}{
				"message": fmt.Sprintf("从 %s 获取到 %d 个代理", sourceName, len(proxies)),
				"source":  sourceName,
				"proxies": proxies,
			}
			runtime.EventsEmit(pm.ctx, "fetch_progress", progressMsg)
			fmt.Printf("发射数据事件: 从 %s 获取到 %d 个代理\n", sourceName, len(proxies))
		}

		// 添加到数据集
		for _, proxy := range proxies {
			pi := &ProxyInfo{
				Key:     strconv.Itoa(i),
				Address: proxy,
				Source:  sourceName,
				Kind:    "socks5",
			}
			pm.proxies = append(pm.proxies, *pi)
			i++
		}

		pm.allProxies = append(pm.allProxies, proxies...)
	}

	// 写入文件
	baseDir := util.GetAppBaseDir()
	path := filepath.Join(baseDir, "proxy_success.txt")

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Error("无法打开文件: %v", err)
	}
	defer file.Close()

	proxySet := make(map[string]struct{})
	for _, proxy := range pm.allProxies {
		proxySet[proxy] = struct{}{}
	}

	for proxy := range proxySet {
		_, err := file.WriteString(proxy + "\n")
		if err != nil {
			log.Error("写入失败: %v", err)
		}
	}

	// 发射完成事件
	if pm.ctx != nil {
		completeMsg := "代理获取完成，共获取 " + strconv.Itoa(len(pm.allProxies)) + " 个代理"
		runtime.EventsEmit(pm.ctx, "fetch_complete", completeMsg)
		fmt.Printf("发射完成事件: %s\n", completeMsg)
	}

	return pm.allProxies, nil
}
