package bypassAv

import (
	"EasyTools/app/bypass/shell/util"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"
)

func RemoveUpx(c *gin.Context) {
	currentPath, _ := os.Getwd()
	resultFolder := filepath.Join(currentPath, "EasyToolsFiles", "file")

	if err := os.MkdirAll(resultFolder, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("创建目录失败: %s", err),
		})
		return
	}

	var wg sync.WaitGroup
	var results []*util.AVSignatureResult
	var processErr error

	wg.Add(1)
	go func() {
		defer wg.Done()
		files, err := os.ReadDir(resultFolder)
		if err != nil {
			processErr = err
			return
		}
		for _, file := range files {
			ext := filepath.Ext(file.Name())
			if ext != ".exe" && ext != "" {
				continue
			}
			if ext == "" && !util.IsPEFile(filepath.Join(resultFolder, file.Name())) {
				continue
			}
			fpath := filepath.Join(resultFolder, file.Name())
			result, err := util.ScanAndEvadeAVSignatures(fpath)
			if err != nil {
				fmt.Printf("[-] 特征规避失败 %s: %v\n", file.Name(), err)
				continue
			}
			results = append(results, result)
			fmt.Printf("[+] 特征规避完成 %s: 发现 %d, 已处理 %d\n", file.Name(), len(result.Found), len(result.Evaded))
		}
	}()
	wg.Wait()

	if processErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   fmt.Sprintf("处理失败: %s", processErr),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "RemoveUpx/特征规避处理成功 (native)",
		"status":  "success",
		"results": results,
	})
}
