package bypassAv

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
)

type FileSizeRequestData struct {
	TargetSizeMB string `json:"size"`
}

func ChangeFileSize(c *gin.Context) {
	// 检查授权
	var requestData FileSizeRequestData
	// 解析 JSON 请求体
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid JSON format",
		})
		return
	}

	if requestData.TargetSizeMB == "" {
		requestData.TargetSizeMB = "18"
	}

	// 定义文件路径
	inputFilePath := filepath.Join(os.TempDir(), "EasyTools", "file", "sourceFile.exe")
	currentPath, _ := os.Getwd()
	currentFilePath := filepath.Join(currentPath, "EasyToolsFiles", "file")

	// 确保目标目录存在
	if err := os.MkdirAll(currentFilePath, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   fmt.Sprintf("创建目标目录失败: %s", err),
		})
		return
	}

	// 检查源文件是否存在
	if _, err := os.Stat(inputFilePath); os.IsNotExist(err) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "源文件不存在: " + inputFilePath,
		})
		return
	}

	var wg sync.WaitGroup
	var processErr error
	var finalFilePath string

	wg.Add(1)

	// 异步处理任务
	go func() {
		defer wg.Done()

		// 读取源文件内容
		sourceData, err := os.ReadFile(inputFilePath)
		if err != nil {
			processErr = fmt.Errorf("读取源文件时出错: %s", err)
			return
		}

		// 计算目标文件的大小（单位：字节）
		IntTargetSizeMB, err := strconv.Atoi(requestData.TargetSizeMB)
		if err != nil {
			processErr = fmt.Errorf("转换错误: %s", err)
			return
		}
		// 限制填充上限，防止恶意超大目标导致内存/磁盘耗尽
		const maxSizeMB = 200
		if IntTargetSizeMB < 1 || IntTargetSizeMB > maxSizeMB {
			processErr = fmt.Errorf("目标大小必须在 1-%d MB 之间", maxSizeMB)
			return
		}
		targetSize := int64(IntTargetSizeMB) * 1024 * 1024

		// 获取源文件的当前大小
		currentSize := int64(len(sourceData))

		// 如果目标文件大小小于当前文件大小，输出提示并退出
		if targetSize <= currentSize {
			processErr = fmt.Errorf("目标文件大小(%d MB)小于或等于源文件大小(%.2f MB)，程序不会做填充操作",
				IntTargetSizeMB, float64(currentSize)/1024/1024)
			return
		}

		// 生成输出文件名
		outputFileName := "newSizeFile_" + requestData.TargetSizeMB + ".exe"
		finalFilePath = filepath.Join(currentFilePath, outputFileName)

		// 创建目标文件
		newFile, err := os.Create(finalFilePath)
		if err != nil {
			processErr = fmt.Errorf("创建目标文件时出错: %s", err)
			return
		}
		defer newFile.Close()

		// 写入源文件数据
		_, err = newFile.Write(sourceData)
		if err != nil {
			processErr = fmt.Errorf("写入源文件数据时出错: %s", err)
			return
		}

		// 分块写入零字节填充，避免一次性大内存分配
		const chunkSize = 1 << 20 // 1MB
		zeroChunk := make([]byte, chunkSize)
		written := currentSize
		for written < targetSize {
			n := int64(chunkSize)
			if written+n > targetSize {
				n = targetSize - written
			}
			if _, err := newFile.Write(zeroChunk[:n]); err != nil {
				processErr = fmt.Errorf("写入填充数据时出错: %s", err)
				return
			}
			written += n
		}

		// 获取新文件的大小
		fi, err := newFile.Stat()
		if err != nil {
			processErr = fmt.Errorf("获取新文件大小时出错: %s", err)
			return
		}

		// 输出日志信息
		fmt.Printf("源文件大小: %.2f MB\n", float64(currentSize)/1024/1024)
		fmt.Printf("需要填充 %.2f MB\n", float64(targetSize-currentSize)/1024/1024)
		fmt.Printf("新文件大小: %.2f MB\n", float64(fi.Size())/1024/1024)

		// 只清理本次请求的源文件，避免删除其他模块共享的临时工作目录
		defer os.Remove(inputFilePath)
	}()

	// 等待异步任务完成
	wg.Wait()

	// 检查处理过程中是否有错误
	if processErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   fmt.Sprintf("修改文件大小失败: %s", processErr),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"message":   "文件大小修改成功",
		"status":    "success",
		"file_path": finalFilePath,
		"size_mb":   requestData.TargetSizeMB,
	})
}
