// Package localguard 本地 HTTP 服务访问防护：
// 所有绑定 127.0.0.1 的本地服务统一校验 Host、Origin 与进程级 CSRF Token。
//
// 防护规则：
//   - Host 必须为回环地址（127.0.0.1 / localhost / [::1]），阻断 DNS rebinding；
//   - Origin 在本地白名单内（回环 http(s) / wails://）或为空 → 直接放行；
//   - Origin 存在但不在白名单（如 Wails 自定义协议下浏览器序列化为 "null"）→
//     必须携带有效的 X-Auth-Token（请求头）或 ?token= 查询参数（WebSocket），
//     Token 由本进程随机生成、经 Wails 绑定仅注入应用前端，恶意网页无法获取。
package localguard

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// hostAllowed 校验请求 Host 是否为回环地址。
// 端口不做强制要求（允许不带端口的回环 Host），核心防线是主机名必须为回环。
func hostAllowed(r *http.Request) bool {
	host := r.Host
	if host == "" {
		return false
	}
	hostname := host
	if h, _, err := net.SplitHostPort(host); err == nil {
		hostname = h
	}
	hostname = strings.Trim(hostname, "[]")
	switch strings.ToLower(hostname) {
	case "127.0.0.1", "localhost", "::1", "0:0:0:0:0:0:0:1":
		return true
	default:
		return false
	}
}

// originAllowed 校验 Origin 是否允许：
// 允许 127.0.0.1 / localhost / [::1] / wails.localhost 任意端口、wails:// 自定义协议；
// 拒绝 null 及其它一切来源。
// （Wails 各版本/平台的页面 Origin 形态不同：wails://wails、http(s)://wails.localhost:<port>、
// 或 "null"，统一在此放行合法形态；wails.localhost 为 Wails 保留虚拟主机名，公网无法解析，
// 恶意网页无法伪造该 Origin。）
func originAllowed(origin string) bool {
	if origin == "" {
		// 无 Origin：同源请求 / 非浏览器客户端（curl、本地程序），放行
		return true
	}
	// 浏览器 file:// / sandbox iframe 会发送字面量 "null"，单独处理（见 Check：需 Token）
	if strings.EqualFold(origin, "null") {
		return false
	}
	lower := strings.ToLower(origin)
	if strings.HasPrefix(lower, "wails://") {
		return true
	}
	// 仅允许 http/https 且主机为回环地址或 Wails 虚拟主机
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		return false
	}
	rest := lower
	for _, scheme := range []string{"http://", "https://"} {
		if strings.HasPrefix(rest, scheme) {
			rest = strings.TrimPrefix(rest, scheme)
			break
		}
	}
	hostPart := rest
	if idx := strings.IndexAny(rest, "/?"); idx >= 0 {
		hostPart = rest[:idx]
	}
	if h, _, err := net.SplitHostPort(hostPart); err == nil {
		hostPart = h
	}
	hostPart = strings.Trim(hostPart, "[]")
	switch strings.ToLower(hostPart) {
	case "127.0.0.1", "localhost", "::1", "0:0:0:0:0:0:0:1", "wails.localhost":
		return true
	default:
		return false
	}
}

// IsAllowedOrigin 供 CORS 中间件使用：与请求防护共用同一套 Origin 白名单逻辑。
func IsAllowedOrigin(origin string) bool {
	return originAllowed(origin)
}

// ---------------------------------------------------------------------------
// 进程级 CSRF Token：仅经 Wails 绑定注入应用前端，恶意网页无法获取。
// ---------------------------------------------------------------------------

var (
	tokenMu sync.RWMutex
	csrfToken string
)

// InitToken 生成并设置进程级 CSRF Token（已设置则原样返回，幂等）。
func InitToken() string {
	tokenMu.Lock()
	defer tokenMu.Unlock()
	if csrfToken == "" {
		b := make([]byte, 32)
		_, _ = rand.Read(b)
		csrfToken = hex.EncodeToString(b)
	}
	return csrfToken
}

// GetToken 返回当前进程的 CSRF Token（未初始化则生成）。
func GetToken() string {
	return InitToken()
}

// tokenMatches 恒时比较请求携带的 Token 与进程 Token（头或查询参数）。
func tokenMatches(got string) bool {
	tokenMu.RLock()
	defer tokenMu.RUnlock()
	if csrfToken == "" || got == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(csrfToken)) == 1
}

// ValidateRequestToken 校验请求头 X-Auth-Token 或查询参数 token。
func ValidateRequestToken(r *http.Request) bool {
	if tokenMatches(r.Header.Get("X-Auth-Token")) {
		return true
	}
	return tokenMatches(r.URL.Query().Get("token"))
}

// Check 对单个请求执行 Host + Origin + CSRF Token 校验。
func Check(r *http.Request, port int) bool {
	if !hostAllowed(r) {
		return false
	}
	// CORS 预检（OPTIONS）由浏览器自动发出，按规范不允许携带自定义请求头，
	// 因此 X-Auth-Token 必然缺失。若在此按 Origin 拒绝，预检就会拿到无 CORS 头的
	// 403，浏览器直接判定跨域失败，前端 axios 只会看到 "Network Error"
	// （典型现象：简练助手 SSH/FTP/数据库连接全部报 network error）。
	// 预检本身不执行任何业务逻辑，放行只做 Host 校验即可；
	// 真正的来源校验仍由随后到达的实际请求（GET/POST/...）完成。
	if r.Method == http.MethodOptions {
		return true
	}
	origin := r.Header.Get("Origin")
	if originAllowed(origin) {
		return true
	}
	// Origin 存在但不在白名单（如 Wails 协议下序列化为 "null"）→ 必须携带有效 Token
	return ValidateRequestToken(r)
}

// GinGuard 生成 gin 中间件（用于主服务与 ssh/ftp/dbmanager 子服务）。
func GinGuard(port int) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !Check(c.Request, port) {
			// 拒绝时记录原因，便于定位本地前端异常请求（不会打印敏感内容）
			origin := c.Request.Header.Get("Origin")
			fmt.Printf("[localguard] 拒绝访问 %s %s host=%q origin=%q\n",
				c.Request.Method, c.Request.URL.Path, c.Request.Host, origin)
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "拒绝访问：非法来源",
			})
			return
		}
		c.Next()
	}
}

// HTTPGuard 生成 net/http 中间件（用于 CLI 执行器服务）。
func HTTPGuard(port int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !Check(r, port) {
				origin := r.Header.Get("Origin")
				fmt.Printf("[localguard] 拒绝访问 %s %s host=%q origin=%q\n",
					r.Method, r.URL.Path, r.Host, origin)
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"success":false,"error":"拒绝访问：非法来源"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// WebSocketOriginCheck 生成 WebSocket 握手 Origin 校验函数
// （浏览器 WebSocket 无法自定义请求头，Token 通过 URL 查询参数携带）。
func WebSocketOriginCheck(port int) func(r *http.Request) bool {
	return func(r *http.Request) bool {
		return Check(r, port)
	}
}
