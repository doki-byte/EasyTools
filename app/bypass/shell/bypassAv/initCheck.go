package bypassAv

import (
	"EasyTools/app/bypass/shell/util"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// 环境检测
func InitCheckHandler(c *gin.Context) {
	// 检查授权
	err := util.InitGoModule()
	if err != nil {
		// 同时返回 error 字段：前端统一读 error.response.data.error 展示真实原因，
		// 只给 message 时前端拿不到内容，只能显示"Request failed with status code 400"。
		msg := fmt.Sprintf("环境初始化失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   msg,
			"message": msg,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "环境初始化检测成功",
		"path":    "",
	})
}
