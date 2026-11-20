package bypassAv

import (
	"EasyTools/app/bypass/shell/util"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// PassQvmRequest 绕过Qvm处理参数：原生 QVM 规避（随机化版本信息 + 清理静态特征），
// 可叠加图标处理（默认不更换图标；随机内置 / 指定图标均在本机纯 Go 完成，跨平台）。
type PassQvmRequest struct {
	IconMode string `json:"iconMode"` // none=不更换图标(默认) | random=随机内置图标 | spec=指定图标
	IconName string `json:"iconName"` // spec 模式下内置图标库文件名（如 access.ico）
	IconData string `json:"iconData"` // spec 模式下自定义图标 base64（.ico/.png，可带 data: 前缀）
}

func PassQvm(c *gin.Context) {
	var req PassQvmRequest
	if c.Request.Body != nil {
		body, _ := io.ReadAll(io.LimitReader(c.Request.Body, 64*1024))
		if len(strings.TrimSpace(string(body))) > 0 {
			_ = json.Unmarshal(body, &req)
		}
	}
	iconMode := strings.ToLower(strings.TrimSpace(req.IconMode))
	if iconMode == "" {
		iconMode = "none"
	}
	if iconMode != "none" && iconMode != "random" && iconMode != "spec" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "iconMode 仅支持 none / random / spec",
		})
		return
	}

	// 预先解析图标来源，避免逐个文件重复校验
	var specIcon []byte
	var libNames []string
	switch iconMode {
	case "random":
		names, err := util.ListBuiltinIconFiles()
		if err != nil || len(names) == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "内置图标库为空，无法随机更换图标",
			})
			return
		}
		libNames = names
	case "spec":
		if strings.TrimSpace(req.IconData) == "" && strings.TrimSpace(req.IconName) == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "请选择要应用的图标（内置库或自定义 .ico/.png）",
			})
			return
		}
		raw, err := util.ResolveIconBytes(&util.IconConfig{IconName: strings.TrimSpace(req.IconName), Data: req.IconData})
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
		specIcon = raw
	}

	currentPath, _ := os.Getwd()
	resultFolder := filepath.Join(currentPath, "EasyToolsFiles", "file")

	var wg sync.WaitGroup
	var processErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		processErr = processFolderQvm(resultFolder, iconMode, specIcon, libNames)
	}()
	wg.Wait()

	if processErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   fmt.Sprintf("处理文件失败: %s", processErr),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "PassQvm make successfully (native)",
		"status":  "success",
	})
}

// processFolderQvm 对输出目录下的所有 exe 执行 QVM 规避 + 可选图标替换
func processFolderQvm(resultFolder, iconMode string, specIcon []byte, libNames []string) error {
	if _, err := os.Stat(resultFolder); os.IsNotExist(err) {
		return fmt.Errorf("目录不存在: %s", resultFolder)
	}
	files, err := os.ReadDir(resultFolder)
	if err != nil {
		return fmt.Errorf("读取文件夹失败: %w", err)
	}
	handled := 0
	for _, file := range files {
		if filepath.Ext(file.Name()) != ".exe" {
			continue
		}
		exePath := filepath.Join(resultFolder, file.Name())
		if err := util.ApplyPassQvmNative(exePath); err != nil {
			fmt.Printf("[-] PassQvm 处理 %s 失败: %v\n", file.Name(), err)
			continue
		}
		if iconMode == "spec" && len(specIcon) > 0 {
			if err := util.ReplacePEIconFile(exePath, specIcon); err != nil {
				fmt.Printf("[-] 替换图标失败 %s: %v\n", file.Name(), err)
				continue
			}
		} else if iconMode == "random" {
			name := libNames[randomInt(len(libNames))]
			raw, rerr := util.ReadBuiltinIconFile(name)
			if rerr != nil {
				fmt.Printf("[-] 读取内置图标失败 %s: %v\n", name, rerr)
				continue
			}
			if err := util.ReplacePEIconFile(exePath, raw); err != nil {
				fmt.Printf("[-] 替换图标失败 %s: %v\n", file.Name(), err)
				continue
			}
		}
		handled++
		fmt.Printf("[+] PassQvm 处理成功: %s\n", file.Name())
	}
	if handled == 0 {
		return fmt.Errorf("EasyToolsFiles/file 下未找到可处理的 exe 文件")
	}
	return nil
}

func randomInt(n int) int {
	if n <= 0 {
		return 0
	}
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		return 0
	}
	return int(v.Int64())
}
