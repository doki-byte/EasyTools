package bypassAv

import (
	"EasyTools/app/bypass/shell/util"
	"net/http"

	"github.com/gin-gonic/gin"
)

// IconList 内置图标库列表（文件名数组，供前端"增加图标"弹窗展示）
func IconList(c *gin.Context) {
	names, err := util.ListBuiltinIconFiles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"icons":   names,
	})
}

// IconFile 提供内置图标文件内容（image/x-icon），用于前端缩略图预览
func IconFile(c *gin.Context) {
	name := c.Param("name")
	if !util.ValidBuiltinIconName(name) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "非法的图标名",
		})
		return
	}
	data, err := util.ReadBuiltinIconFile(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   err.Error(),
		})
		return
	}
	c.Header("Cache-Control", "public, max-age=3600")
	c.Data(http.StatusOK, "image/x-icon", data)
}
