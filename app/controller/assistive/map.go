package assistive

import (
	"EasyTools/app/controller/proxy"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// proxyHTTPGET 通用 GET 代理，供第三方地图/地理 API 使用，解决浏览器 CORS
func (a *Assistive) proxyHTTPGET(c *gin.Context) {
	url := c.Query("url")
	if url == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "url参数不能为空",
		})
		return
	}

	client := proxy.GetGlobalHTTPClient()
	client.Timeout = 30 * time.Second

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "创建请求失败: " + err.Error(),
		})
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", "http://wails.localhost/")

	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "请求失败: " + err.Error(),
		})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "读取响应失败: " + err.Error(),
		})
		return
	}

	var jsonData interface{}
	if err := json.Unmarshal(body, &jsonData); err != nil {
		c.Data(resp.StatusCode, resp.Header.Get("Content-Type"), body)
		return
	}

	c.JSON(resp.StatusCode, jsonData)
}

// ProxyBaiduMap 代理百度地图API请求，解决CORS问题
func (a *Assistive) ProxyBaiduMap(c *gin.Context) {
	a.proxyHTTPGET(c)
}

// ProxyGoogleMap 代理 Google Maps Web Service（如 Geocoding JSON）请求，解决 CORS 问题
func (a *Assistive) ProxyGoogleMap(c *gin.Context) {
	a.proxyHTTPGET(c)
}
