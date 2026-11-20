package bypassAv

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
)

// OpenDir 打开指定目录。仅允许打开应用自身相关的目录（输出目录 / 临时目录），
// 拒绝任意路径，避免被恶意网页利用打开/执行任意文件。
func OpenDir(c *gin.Context) {
	// 检查授权
	if runtime.GOOS != "windows" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": "仅支持在 Windows 平台打开目录",
		})
		return
	}

	path := c.Query("path")
	if path == "" {
		currentDir, _ := os.Getwd()
		path = filepath.Join(currentDir, "EasyToolsFiles", "file")
	}

	// 确保路径是绝对路径
	if !filepath.IsAbs(path) {
		currentDir, _ := os.Getwd()
		path = filepath.Join(currentDir, path)
	}

	// 路径白名单：只允许应用输出目录与其子目录、以及本应用临时目录
	currentDir, _ := os.Getwd()
	allowedRoots := []string{
		filepath.Join(currentDir, "EasyToolsFiles", "file"),
		filepath.Join(os.TempDir(), "EasyTools"),
	}
	inside := false
	for _, root := range allowedRoots {
		rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(path))
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			inside = true
			break
		}
	}
	if !inside {
		c.JSON(http.StatusForbidden, gin.H{
			"message": "不允许打开该路径",
		})
		return
	}

	// 检查路径是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "路径不存在",
			"error":   err.Error(),
		})
		return
	}

	cmd := exec.Command("explorer", path)

	// 执行命令
	err := cmd.Start()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "无法打开文件夹",
			"error":   err.Error(),
		})
		return
	}

	// 返回成功消息
	c.JSON(http.StatusOK, gin.H{
		"message": "文件夹已打开",
	})
}
