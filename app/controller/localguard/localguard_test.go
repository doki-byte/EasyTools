package localguard

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// TestWailsOriginScenarios 模拟 wails（wails://wails 页面、Origin 序列化为 null）
// 与恶意网页（evil.com / sandbox iframe null origin）场景。
func TestWailsOriginScenarios(t *testing.T) {
	gin.SetMode(gin.TestMode)
	port := 52867
	InitToken()
	tok := GetToken()

	r := gin.New()
	r.Use(GinGuard(port))
	r.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://127.0.0.1:" + "52867", "http://localhost:52867", "http://[::1]:52867"},
		AllowOriginWithContextFunc: func(c *gin.Context, origin string) bool {
			if IsAllowedOrigin(origin) {
				return true
			}
			return ValidateRequestToken(c.Request)
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "X-Requested-With", "X-Auth-Token"},
		AllowCredentials: true,
	}))
	r.GET("/modelist", func(c *gin.Context) { c.JSON(200, gin.H{"ok": true}) })

	cases := []struct {
		name   string
		origin string
		host   string
		token  string // header X-Auth-Token
		query  string // ?token=
		wantOK bool
	}{
		{"vite localhost 34115", "http://localhost:34115", "127.0.0.1:52867", "", "", true},
		{"vite 127.0.0.1 34115", "http://127.0.0.1:34115", "127.0.0.1:52867", "", "", true},
		{"wails vhost http", "http://wails.localhost:34115", "127.0.0.1:52867", "", "", true},
		{"wails vhost https", "https://wails.localhost", "127.0.0.1:52867", "", "", true},
		{"wails scheme", "wails://wails", "127.0.0.1:52867", "", "", true},
		{"wails null origin + header token", "null", "127.0.0.1:52867", tok, "", true},
		{"wails null origin + query token", "null", "127.0.0.1:52867", "", tok, true},
		{"null origin no token (sandbox iframe)", "null", "127.0.0.1:52867", "", "", false},
		{"null origin wrong token", "null", "127.0.0.1:52867", "wrong", "", false},
		{"no origin (curl)", "", "127.0.0.1:52867", "", "", true},
		{"evil.com", "http://evil.com", "127.0.0.1:52867", "", "", false},
		{"dns rebinding (evil host + token)", "http://evil.com", "evil.com:52867", tok, "", false},
	}
	for _, tc := range cases {
		path := "/modelist?targetOS=windows"
		if tc.query != "" {
			path += "&token=" + tc.query
		}
		req := httptest.NewRequest("GET", path, nil)
		req.Host = tc.host
		if tc.origin != "" {
			req.Header.Set("Origin", tc.origin)
		}
		if tc.token != "" {
			req.Header.Set("X-Auth-Token", tc.token)
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		pass := w.Code == http.StatusOK
		if pass != tc.wantOK {
			t.Errorf("%s: 期望 %v 得到 %d body=%s", tc.name, tc.wantOK, w.Code, w.Body.String())
		} else {
			t.Logf("%-34s origin=%-22s => %d %v", tc.name, tc.origin, w.Code, pass)
		}
	}
}
