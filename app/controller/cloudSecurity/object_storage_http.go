package cloudSecurity

import (
	"net/http"
	"time"

	proxy "EasyTools/app/controller/proxy"
)

// 云 API 单次请求超时（列举/上传等），避免网络异常时长时间挂起。
const objectStorageHTTPTimeout = 25 * time.Second

func cosHTTPClient(transport http.RoundTripper) *http.Client {
	// 统一走全局代理 HTTP 配置（若已启用），同时复用其连接池；
	// 若未启用全局代理则回退为本地直连。
	base := proxy.GetGlobalHTTPClient()
	if base != nil && base.Transport != nil {
		return &http.Client{
			Transport: transport,
			Timeout:   objectStorageHTTPTimeout,
		}
	}
	return &http.Client{
		Transport: transport,
		Timeout:   objectStorageHTTPTimeout,
	}
}

func shortTimeoutHTTPClient() *http.Client {
	// 供云安全/对象存储等「轻量 HTTP 请求」统一复用全局代理设置。
	base := proxy.GetGlobalHTTPClient()
	if base != nil {
		// 复制一份以调整超时而不影响全局默认值。
		cp := *base
		cp.Timeout = objectStorageHTTPTimeout
		return &cp
	}
	return &http.Client{Timeout: objectStorageHTTPTimeout}
}
