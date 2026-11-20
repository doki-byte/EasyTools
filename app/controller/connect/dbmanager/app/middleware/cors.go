package middleware

import (
	"EasyTools/app/controller/localguard"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Cors 解决跨域及预检 301/重定向问题。
// 安全防护：仅放行本地回环来源（127.0.0.1/localhost）与 Wails 协议，
// 拒绝恶意网页的跨域调用与 DNS rebinding（Host 校验）。
func Cors(port int) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !localguard.Check(c.Request, port) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"success": false,
				"error":   "拒绝访问：非法来源",
			})
			return
		}

		headers := c.Writer.Header()
		origin := c.Request.Header.Get("Origin")
		if origin != "" && (localguard.IsAllowedOrigin(origin) || localguard.ValidateRequestToken(c.Request)) {
			headers.Set("Access-Control-Allow-Origin", origin)
		} else if origin == "" {
			headers.Set("Access-Control-Allow-Origin", "*")
		} else {
			headers.Set("Access-Control-Allow-Origin", "null")
		}
		headers.Set("Access-Control-Allow-Credentials", "true")
		headers.Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		// X-Auth-Token：前端 main.js 统一注入的本地服务 CSRF Token。
		// 必须出现在 Allow-Headers 中，否则浏览器预检直接失败，axios 报 Network Error。
		headers.Set("Access-Control-Allow-Headers", "Authorization,Content-Type,Time,Origin,Accept,X-Requested-With,X-Auth-Token")
		headers.Set("Access-Control-Max-Age", "600")

		// 预检请求直接返回
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
