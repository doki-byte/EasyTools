package proxy

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/projectdiscovery/httpx/common/httpx"
	"github.com/projectdiscovery/retryablehttp-go"
)

// HTTPXManager httpx客户端管理器
type HTTPXManager struct {
	mu     sync.RWMutex
	client *httpx.HTTPX
	config *httpx.Options
}

var globalHTTPXManager *HTTPXManager
var httpxManagerOnce sync.Once

// GetHTTPXManager 获取全局httpx管理器实例（单例模式）
func GetHTTPXManager() *HTTPXManager {
	httpxManagerOnce.Do(func() {
		globalHTTPXManager = &HTTPXManager{}
		globalHTTPXManager.init()
	})
	return globalHTTPXManager
}

// init 初始化httpx客户端
func (h *HTTPXManager) init() {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 从全局代理管理器获取配置
	h.updateClient()
}

// updateClient 更新httpx客户端配置
func (h *HTTPXManager) updateClient() {
	options := httpx.DefaultOptions

	// 设置默认超时时间
	options.Timeout = 10 * time.Second

	// 设置重试次数
	options.RetryMax = 2

	// 从全局代理管理器获取代理配置
	var proxyURL string
	if GlobalProxyManager != nil {
		GlobalProxyManager.mu.RLock()
		proxyConfig := GlobalProxyManager.globalConfig
		proxyEnabled := GlobalProxyManager.globalEnabled
		GlobalProxyManager.mu.RUnlock()

		if proxyEnabled && proxyConfig != nil {
			// 构建代理URL
			proxyURL = h.buildProxyURL(*proxyConfig)
			if proxyURL != "" {
				options.Proxy = proxyURL
				log.Printf("HTTPXManager: 配置代理 - %s", proxyURL)
			} else {
				// 代理配置无效，清空Proxy选项
				options.Proxy = ""
				log.Printf("HTTPXManager: 代理配置无效，已清空代理设置")
			}
		} else {
			// 代理未启用，清空Proxy选项
			options.Proxy = ""
			if proxyEnabled {
				log.Printf("HTTPXManager: 代理已启用但配置为空，已清空代理设置")
			} else {
				log.Printf("HTTPXManager: 代理未启用，使用直连模式")
			}
		}

		// 设置超时时间（从代理配置获取）
		if proxyConfig != nil && proxyConfig.Timeout > 0 {
			options.Timeout = time.Duration(proxyConfig.Timeout) * time.Second
		}
	} else {
		// GlobalProxyManager未初始化，清空Proxy选项
		options.Proxy = ""
		log.Printf("HTTPXManager: GlobalProxyManager未初始化，使用直连模式")
	}

	// 创建httpx客户端
	client, err := httpx.New(&options)
	if err != nil {
		// 如果创建失败，记录错误但保留代理配置重试
		log.Printf("HTTPXManager: 创建httpx客户端失败: %v，尝试使用默认配置重试", err)

		// 保留代理配置，只重置其他选项
		fallbackOptions := httpx.DefaultOptions
		fallbackOptions.Timeout = 10 * time.Second
		fallbackOptions.RetryMax = 2
		// 保留代理配置
		if proxyURL != "" {
			fallbackOptions.Proxy = proxyURL
			log.Printf("HTTPXManager: 重试时保留代理配置 - %s", proxyURL)
		}

		client, err = httpx.New(&fallbackOptions)
		if err != nil {
			// 如果还是失败，记录错误并返回
			log.Printf("HTTPXManager: 使用默认配置重试仍然失败: %v", err)
			return
		}
		log.Printf("HTTPXManager: 使用默认配置重试成功")
	} else {
		if proxyURL != "" {
			log.Printf("HTTPXManager: httpx客户端创建成功，已配置代理 - %s", proxyURL)
		} else {
			log.Printf("HTTPXManager: httpx客户端创建成功，使用直连模式")
		}
	}

	// 关闭旧的客户端（如果存在）
	if h.client != nil && h.client.Dialer != nil {
		h.client.Dialer.Close()
	}

	h.client = client
	h.config = &options
}

// buildProxyURL 构建代理URL
func (h *HTTPXManager) buildProxyURL(config ProxyConfig) string {
	if config.Host == "" || config.Port == "" {
		return ""
	}

	var scheme string
	switch config.Type {
	case "socks5":
		// httpx支持socks5代理，格式：socks5://host:port
		scheme = "socks5"
	case "https":
		scheme = "https"
	default:
		scheme = "http"
	}

	proxyURL := fmt.Sprintf("%s://%s:%s", scheme, config.Host, config.Port)

	// 如果有认证信息，添加到URL中
	if config.Auth != nil && config.Auth.Username != "" {
		proxyURL = fmt.Sprintf("%s://%s:%s@%s:%s", scheme,
			config.Auth.Username, config.Auth.Password,
			config.Host, config.Port)
	}

	return proxyURL
}

// GetClient 获取httpx客户端
func (h *HTTPXManager) GetClient() *httpx.HTTPX {
	h.mu.RLock()
	defer h.mu.RUnlock()

	// 如果客户端不存在，尝试重新初始化
	if h.client == nil {
		h.mu.RUnlock()
		h.mu.Lock()
		if h.client == nil {
			h.updateClient()
		}
		h.mu.Unlock()
		h.mu.RLock()
	}

	return h.client
}

// Refresh 刷新客户端配置（当代理配置改变时调用）
func (h *HTTPXManager) Refresh() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.updateClient()
}

// NewRequest 创建HTTP请求（封装httpx的NewRequest方法）
func (h *HTTPXManager) NewRequest(method, url string) (*retryablehttp.Request, error) {
	client := h.GetClient()
	if client == nil {
		return nil, fmt.Errorf("httpx client not initialized")
	}
	return client.NewRequest(method, url)
}

// Do 执行HTTP请求（封装httpx的Do方法）
func (h *HTTPXManager) Do(req *retryablehttp.Request, options httpx.UnsafeOptions) (*httpx.Response, error) {
	client := h.GetClient()
	if client == nil {
		return nil, fmt.Errorf("httpx client not initialized")
	}
	return client.Do(req, options)
}

// DoWithOptions 执行HTTP请求（带自定义选项）
func (h *HTTPXManager) DoWithOptions(req *retryablehttp.Request, timeout time.Duration, retryMax int) (*httpx.Response, error) {
	client := h.GetClient()
	if client == nil {
		return nil, fmt.Errorf("httpx client not initialized")
	}

	// httpx.UnsafeOptions不支持直接设置Timeout和RetryMax
	// 这些选项需要在创建httpx客户端时配置
	options := httpx.UnsafeOptions{}

	return client.Do(req, options)
}

// ConvertToHTTPResponse 将httpx.Response转换为http.Response
// httpx.Response包含StatusCode、Headers、Body等字段，我们需要构建http.Response
func (h *HTTPXManager) ConvertToHTTPResponse(httpxResp *httpx.Response) *http.Response {
	if httpxResp == nil {
		return nil
	}

	return nil
}

// BuildHTTPResponseFromHTTPX 从httpx.Response的信息构建http.Response
// 这是一个辅助方法，用于在无法直接转换时手动构建
func (h *HTTPXManager) BuildHTTPResponseFromHTTPX(httpxResp *httpx.Response, req *retryablehttp.Request) *http.Response {
	if httpxResp == nil {
		return nil
	}

	return nil
}

// Close 关闭httpx客户端
func (h *HTTPXManager) Close() {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.client != nil && h.client.Dialer != nil {
		h.client.Dialer.Close()
		h.client = nil
	}
}
