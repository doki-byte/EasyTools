package bypassAv

import (
	"EasyTools/app/bypass/shell/loader"
	"EasyTools/app/bypass/shell/util"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func ModeLists(c *gin.Context) {
	targetOS := util.NormalizeTargetOS(c.Query("targetOS"))

	var allLoader []string
	for k := range loader.Modules {
		allLoader = append(allLoader, k)
	}
	// loaderN 系列（规范命名的新模板）排在前面，保持列表稳定有序
	sort.Slice(allLoader, func(i, j int) bool {
		ni, nj := allLoader[i], allLoader[j]
		ri, isLi := isLoaderN(ni)
		rj, isLj := isLoaderN(nj)
		if isLi && isLj {
			return ri < rj
		}
		if isLi != isLj {
			return isLi
		}
		return ni < nj
	})
	filtered := util.FilterLoadersByTargetOS(allLoader, targetOS)

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"all_modes":   filtered,
		"target_os":   targetOS,
		"bai_jia_hei": util.LoaderBaiJiaHei, // 白加黑专用模板编号（前端特判用，避免前端硬编码模板名）
		"message":     fmt.Sprintf("当前版本一共%d种运行方式", len(filtered)),
	})
}

// isLoaderN 判断名称是否为 loaderN（N 为数字）规范命名，返回序号。
func isLoaderN(name string) (int, bool) {
	if !strings.HasPrefix(name, "loader") {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimPrefix(name, "loader"))
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

func PlatformList(c *gin.Context) {
	platforms := []map[string]string{
		{"value": "windows", "label": "Windows"},
		{"value": "linux", "label": "Linux"},
		{"value": "darwin", "label": "macOS"},
	}
	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"platforms": platforms,
	})
}

func AvList(c *gin.Context) {
	AvList := []string{"360", "wdf", "hr"}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"av_list": AvList,
	})
}

func EncodeWayList(c *gin.Context) {
	// 对象结构：value 传后端做编码判定，label 用于前端展示说明
	EncodeWayList := []map[string]string{
		{"value": "aes", "label": "AES 加密"},
		{"value": "xor", "label": "XOR 加密"},
		{"value": "chacha20", "label": "ChaCha20 加密"},
		{"value": "otp", "label": "OTP 一次一密（就地解码）"},
	}
	c.JSON(http.StatusOK, gin.H{
		"success":         true,
		"encode_way_list": EncodeWayList,
	})
}
