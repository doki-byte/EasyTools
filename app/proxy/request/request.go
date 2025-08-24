package request

import (
	util "EasyTools/app/proxy"
	"encoding/json"
	"github.com/labstack/gommon/log"
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
}

func NewProxyManager(fetchers []ProxyFetcher) *ProxyManager {
	return &ProxyManager{
		fetchers: fetchers,
	}
}

func (pm *ProxyManager) RenderTable() ([]byte, error) {
	marshal, err := json.Marshal(pm.proxies)
	if err != nil {
		return nil, err
	}
	return marshal, nil
}

func (pm *ProxyManager) FetchAll() ([]string, error) {
	i := 1
	for _, fetcher := range pm.fetchers {
		proxies, err := fetcher.Fetch()
		if err != nil {
			continue
		}

		for _, proxy := range proxies {
			pi := &ProxyInfo{
				Key:     strconv.Itoa(i),
				Address: proxy,
				Source:  fetcher.Name(),
				Kind:    "socks5",
			}
			pm.proxies = append(pm.proxies, *pi)
			i++
		}

		pm.allProxies = append(pm.allProxies, proxies...)
	}

	baseDir := util.GetAppBaseDir()
	path := filepath.Join(baseDir, "proxy_success.txt")

	// 检查文件是否已存在，如果不存在则创建
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		log.Error("无法打开文件: %v", err)
	}
	defer file.Close()

	// 使用 map 去重代理列表
	proxySet := make(map[string]struct{})
	for _, proxy := range pm.allProxies {
		proxySet[proxy] = struct{}{}
	}

	// 写入去重后的代理地址，并保证每行一个代理
	for proxy := range proxySet {
		_, err := file.WriteString(proxy + "\n")
		if err != nil {
			log.Error("写入失败: %v", err)
		}
	}

	return pm.allProxies, nil
}
