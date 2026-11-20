package bypassAv

import (
	"EasyTools/app/bypass/shell/util"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"
)

func AddSignthief(c *gin.Context) {
	currentPath, _ := os.Getwd()
	resultFolder := filepath.Join(currentPath, "EasyToolsFiles", "file")

	var wg sync.WaitGroup
	var lastErr error
	wg.Add(1)
	go func() {
		defer wg.Done()
		files, err := os.ReadDir(resultFolder)
		if err != nil {
			lastErr = err
			return
		}
		randomStrings := GenerateRandomStrings(5, 8)
		for _, file := range files {
			if filepath.Ext(file.Name()) != ".exe" {
				continue
			}
			exePath := filepath.Join(resultFolder, file.Name())
			outputName := randomStrings[1] + "_" + file.Name()
			outputPath := filepath.Join(resultFolder, outputName)

			if err := copyFileForSign(exePath, outputPath); err != nil {
				fmt.Printf("复制文件失败: %v\n", err)
				continue
			}
			if err := util.CopyCertificateFromEmbedded(outputPath); err != nil {
				fmt.Printf("签名复制失败 %s: %v\n", file.Name(), err)
				os.Remove(outputPath)
				continue
			}
			if err := os.Remove(exePath); err != nil {
				fmt.Printf("删除原文件失败: %s\n", err)
			}
			fmt.Printf("[+] 原生签名复制成功: %s\n", outputName)
		}
	}()
	wg.Wait()

	if lastErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   lastErr.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "所有文件处理完成 (native sig copy)",
		"status":  "success",
	})
}

func copyFileForSign(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o666)
}

// GenerateRandomStrings 生成随机字符串
func GenerateRandomStrings(count, length int) []string {
	letters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	result := make([]string, count)
	for i := 0; i < count; i++ {
		randomStr := make([]byte, length)
		for j := range randomStr {
			randomStr[j] = letters[rand.Intn(len(letters))]
		}
		result[i] = string(randomStr)
	}
	return result
}
