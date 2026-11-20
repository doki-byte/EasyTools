package util

import (
	"EasyTools/app/controller/system"
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func generateRandomSuffix(existingFiles map[string]bool) string {
	// 使用更安全的随机生成方式（crypto/rand）
	bytes := make([]byte, 4) // 生成8个字节的随机数
	_, err := rand.Read(bytes)
	if err != nil {
		fmt.Println("Failed to generate random bytes:", err)
		return ""
	}
	// 将字节转换为十六进制字符串
	suffix := hex.EncodeToString(bytes)

	// 确保文件名唯一，避免重复
	for existingFiles[suffix] {
		_, err := rand.Read(bytes)
		if err != nil {
			fmt.Println("Failed to regenerate random bytes:", err)
			return ""
		}
		suffix = hex.EncodeToString(bytes)
	}
	return suffix
}

// generateBuildInfo 生成随机的构建信息，用于注入到-ldflags
func generateBuildInfo() string {
	// 生成随机版本号
	versionMajor, _ := rand.Int(rand.Reader, big.NewInt(100))
	versionMinor, _ := rand.Int(rand.Reader, big.NewInt(100))
	versionPatch, _ := rand.Int(rand.Reader, big.NewInt(1000))
	version := fmt.Sprintf("%s.%s.%s", versionMajor.String(), versionMinor.String(), versionPatch.String())

	// 生成随机构建ID
	buildIDBytes := make([]byte, 16)
	rand.Read(buildIDBytes)
	buildID := hex.EncodeToString(buildIDBytes)

	// 生成随机字符串标识
	randomStrBytes := make([]byte, 12)
	rand.Read(randomStrBytes)
	randomStr := hex.EncodeToString(randomStrBytes)

	// 当前时间戳
	timestamp := time.Now().Unix()

	// 构建-ldflags字符串，使用-X来设置变量值
	// 注意：这些变量需要在源代码中定义才能被注入
	// -buildid= 清空 Go 工具链写入的构建 ID（Go_BuildID 静态特征的源头）
	ldflags := fmt.Sprintf(`-buildid= -X main.BuildVersion=%s -X main.BuildID=%s -X main.BuildTime=%d -X main.BuildTag=%s`,
		version, buildID, timestamp, randomStr)

	return ldflags
}

// CheckGoBuild 编译免杀产物。
// 返回 error：把编译器的真实报错回传给调用方（原实现只打印到标准输出，
// GUI 用户看不到任何原因，只能看到笼统的"生成出错"）。
func CheckGoBuild(inputFile, outputFile, sharuan, buildWay, loadWay string, dllName string, skyShadowPayloadHeader string, arch string, targetOS string, keepConsole bool) error {
	targetOS = NormalizeTargetOS(targetOS)
	binExt := BinaryExtension(targetOS)
	// 获取临时目录中的工作目录
	workDir := filepath.Join(os.TempDir(), "EasyTools", "file")

	// 保存当前目录
	originalDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("获取当前目录失败: %s\n", err)
		return fmt.Errorf("获取当前目录失败: %v", err)
	}

	// 切换到临时目录中的工作目录
	if err := os.Chdir(workDir); err != nil {
		fmt.Printf("切换工作目录失败: %s\n", err)
		return fmt.Errorf("切换工作目录失败: %v", err)
	}

	// 确保函数结束时切换回原目录
	defer func() {
		os.Chdir(originalDir)
	}()

	// 规范化 outputFile 参数
	outputFile = filepath.Base(outputFile)
	outputFile = strings.TrimSuffix(outputFile, ".exe")
	outputFile = strings.TrimSuffix(outputFile, binExt)

	// 目标目录
	absOriginDir := filepath.Join(originalDir, "EasyToolsFiles", "file")

	// 确保目标目录存在
	if err := os.MkdirAll(absOriginDir, 0755); err != nil {
		fmt.Printf("创建目标目录失败: %v\n", err)
		return fmt.Errorf("创建输出目录失败: %v", err)
	}

	// 检查输入文件是否存在
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		fmt.Printf("输入文件不存在: %s\n", inputFile)
		return fmt.Errorf("待编译源码不存在: %s", inputFile)
	}

	// 源码规模决定编译并发度与缓存策略：
	// 大载荷会被内联成数十 MB 的 []byte 字面量源码，并行多份编译会瞬间吃光内存，
	// 在普通配置机器上直接 OOM 失败（这正是"大载荷必失败、小载荷正常"的根因）。
	srcSize := int64(0)
	if st, statErr := os.Stat(inputFile); statErr == nil {
		srcSize = st.Size()
	}

	// Compilation options
	// 全部变体默认携带 -trimpath（消除本机 GOPATH/用户名/模块缓存路径等静态特征），
	// 不再产出 -race 变体（竞态运行时体积大、特征面大且 loader 场景无任何收益）。
	// 保留一组无额外裁剪的变体兜底兼容，但同样带 -trimpath。
	flags := [][]string{
		{"-trimpath"},
		{"-trimpath", "-ldflags=-w"},
		{"-trimpath", "-ldflags=-s"},
		{"-trimpath", "-ldflags=-w -s"},
	}

	// 跨平台适配：非 Windows 目标（linux/darwin）交叉编译时无法使用 -race
	// （-race 依赖 cgo 与目标平台运行时，纯 Go 交叉编译会直接失败），统一过滤。
	if NormalizeTargetOS(targetOS) != "windows" {
		var filtered [][]string
		for _, f := range flags {
			skip := false
			for _, flag := range f {
				if flag == "-race" {
					skip = true
					break
				}
			}
			if !skip {
				filtered = append(filtered, f)
			}
		}
		flags = filtered
	}

	garbleFlags := [][]string{
		{"-seed=random"}, {"-tiny"}, {"-literals"},
		{"-seed=random", "-tiny"}, {"-seed=random", "-literals"},
		{"-seed=random", "-tiny", "-literals"},
	}

	// 如果 sharuan == "360"，移除某些编译选项
	if sharuan == "360" {
		var filteredFlags [][]string
		for _, f := range flags {
			skip := false
			for _, flag := range f {
				if flag == "-race" || flag == "-ldflags=-w" || flag == "-race -ldflags=-w -s" {
					skip = true
					break
				}
			}
			if !skip {
				filteredFlags = append(filteredFlags, f)
			}
		}
		flags = filteredFlags
	}

	// 检查可执行文件路径
	goBinary, garbleBinary := "go", "garble"
	if buildWay == "garble" {
		if path, err := exec.LookPath("garble"); err == nil {
			garbleBinary = path
		} else {
			fmt.Println("Garble binary not found in PATH.")
			return fmt.Errorf("未找到 garble（已选择 garble 编译方式），请安装 garble 或改用 go 编译方式")
		}
	} else {
		if path, err := exec.LookPath("go"); err == nil {
			goBinary = path
		} else {
			fmt.Println("Go binary not found in PATH.")
			return fmt.Errorf("未找到 go 编译器，请确认已安装 Go 并加入 PATH")
		}
	}

	// -------- 白加黑（DLL）模式：使用手动指定的架构 --------
	if loadWay == "白加黑" {
		// 确定DLL文件名：如果提供了dllName则使用，否则使用默认的host.dll
		dllFileName := "iphlpapi.dll"
		if dllName != "" {
			dllFileName = filepath.Base(dllName) // 安全：仅取文件名，防止路径穿越
			if !strings.HasSuffix(strings.ToLower(dllFileName), ".dll") {
				dllFileName = dllFileName + ".dll"
			}
		}

		dllFileHeaderName := strings.ReplaceAll(dllFileName, "dll", "h")
		outputDllPath := filepath.Join(originalDir, "EasyToolsFiles", "file", dllFileName)

		// 生成动态构建信息
		buildInfo := generateBuildInfo()

		// 根据 SkyShadow payload header 检测架构并设置 mingw
		env := os.Environ()
		env = append(env, "GOOS=windows")
		env = append(env, "CGO_ENABLED=1")

		var bitness string
		// 架构规范化：amd64/x64 → x64，386/x86/i386 → x86（兼容新旧写法）
		if NormalizeGoArch(arch) == "386" {
			bitness = "x86"
			fmt.Printf("使用指定的架构: x86 (386)\n")
		} else {
			bitness = "x64"
			fmt.Printf("使用指定的架构: x64 (amd64)\n")
		}

		var ccPath, newPath string
		if bitness == "x86" {
			env = append(env, "GOARCH=386")
			mingwInfo := FindMinGW("mingw32")
			if mingwInfo == nil {
				fmt.Printf("警告: 未找到 MinGW32 交叉编译器，将尝试使用系统默认编译器（Linux 可安装 gcc-mingw-w64-i686，macOS 可 brew install mingw-w64）\n")
			} else {
				ccPath = mingwInfo.CC
				newPath = mingwInfo.PATH
				env = append(env, "CC="+ccPath)
				env = append(env, "PATH="+newPath)
				fmt.Printf("使用 MinGW32 交叉编译器: %s\n", ccPath)
			}
		} else {
			env = append(env, "GOARCH=amd64")
			mingwInfo := FindMinGW("mingw64")
			if mingwInfo == nil {
				fmt.Printf("警告: 未找到 MinGW64 交叉编译器，将尝试使用系统默认编译器（Linux 可安装 gcc-mingw-w64-x86-64，macOS 可 brew install mingw-w64）\n")
			} else {
				ccPath = mingwInfo.CC
				newPath = mingwInfo.PATH
				env = append(env, "CC="+ccPath)
				env = append(env, "PATH="+newPath)
				fmt.Printf("使用 MinGW64 交叉编译器: %s\n", ccPath)
			}
		}

		// 构建命令参数，合并原有的-ldflags和新的构建信息
		args := []string{
			"build",
			"-buildmode=c-shared",
			"--trimpath",
			"-ldflags=-s -w " + buildInfo,
			"-o", outputDllPath,
			inputFile,
		}

		cmd := exec.Command("go", args...)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		cmd.SysProcAttr = hideWindowAttr()
		cmd.Env = env
		fmt.Printf("正在构建 DLL: %s (架构: %s)...\n", dllFileName, bitness)

		// 执行命令
		err = cmd.Run()
		if err != nil {
			fmt.Printf("构建失败: %v\n", err)
			return fmt.Errorf("构建 DLL 失败: %v", err)
		}

		fmt.Printf("DLL 构建成功: %s\n", outputDllPath)

		// 删除生成的.h文件
		headerPath := filepath.Join(originalDir, "EasyToolsFiles", "file", dllFileHeaderName)
		os.Remove(headerPath)

		// 如果提供了SkyShadow白程序信息，将DLL复制到对应的白程序目录
		if skyShadowPayloadHeader != "" {
			base := new(system.Base)
			baseDir := base.GetAppPath()
			payloadRoot := filepath.Join(baseDir, "file", "skyshadow", "Payload")
			payloadDir := filepath.Join(payloadRoot, skyShadowPayloadHeader)

			// 确保目标目录存在
			if err := os.MkdirAll(payloadDir, 0755); err != nil {
				fmt.Printf("创建白程序目录失败: %v\n", err)
				return fmt.Errorf("创建白程序目录失败: %v", err)
			}

			// 复制DLL到白程序目录
			destDllPath := filepath.Join(payloadDir, dllFileName)
			if err := copyFile(outputDllPath, destDllPath); err != nil {
				fmt.Printf("复制DLL到白程序目录失败: %v\n", err)
				return fmt.Errorf("复制 DLL 到白程序目录失败: %v", err)
			}

			fmt.Printf("DLL 已复制到白程序目录: %s\n", destDllPath)
		}

		return nil
	}

	// -------- 普通模式（一体 / 远程 / 分离）：根据前端传入的 arch 设置目标架构 --------
	// 架构规范化：amd64/x64 → amd64，386/x86/i386 → 386（兼容新旧写法）
	goArch := NormalizeGoArch(arch)

	// 使用 sync.Map 确保唯一文件名（并发安全）
	var generatedFiles sync.Map
	var wg sync.WaitGroup
	errChan := make(chan string, len(flags)*len(garbleFlags))

	// GOCACHE 使用隔离目录；GOMODCACHE 不再重定向到临时空目录，
	// 直接复用全局模块缓存，避免每次生成都联网重复下载 golang.org/x/sys 等依赖。
	//
	// 注意：该目录曾"每次编译后整体删除"，导致 garble 每次生成都要从零重建整个
	// 标准库与依赖（单次生成动辄数分钟、内存占用成倍放大，大载荷直接失败）。
	// 现在改为跨次复用，仅在体积超限时清理。
	sharedCacheDir := filepath.Join(os.TempDir(), "EasyTools", "file", "shared_garble_cache")

	// 创建隔离编译缓存目录
	os.MkdirAll(sharedCacheDir, 0755)

	// 记录已经执行的编译选项
	var executedOptions sync.Map

	// 控制并发数量：并发度按源码规模自适应。
	// 载荷会被内联为 []byte 字面量，源码体积约为载荷的 5 倍（OTP 为 10 倍），
	// 例如 8.8MB 载荷 → ~44MB 源码。多份此类编译并行会在普通机器上直接 OOM，
	// 表现为"大载荷必然失败、小载荷正常"。这里按规模退化为串行/低并发。
	concurrencyLimit := 5
	switch {
	case srcSize > 24<<20:
		concurrencyLimit = 1
	case srcSize > 8<<20:
		concurrencyLimit = 2
	case srcSize > 2<<20:
		concurrencyLimit = 3
	}
	// garble 自身额外做混淆，开销远大于 go build，进一步限制并发
	if buildWay == "garble" && concurrencyLimit > 2 {
		concurrencyLimit = 2
	}
	fmt.Printf("编译并发度: %d（源码 %d 字节）\n", concurrencyLimit, srcSize)
	sem := make(chan struct{}, concurrencyLimit)

	// 编译不同版本
	for _, additionalFlags := range flags {
		if buildWay == "garble" {
			for _, garbleFlagSet := range garbleFlags {
				wg.Add(1)
				go func(additionalFlags, garbleFlagSet []string) {
					defer wg.Done()

					// 生成唯一文件名
					var output string
					for {
						suffix := generateRandomSuffix(nil)
						output = fmt.Sprintf("%s_%s%s", outputFile, suffix, binExt)
						if _, exists := generatedFiles.LoadOrStore(output, true); !exists {
							break
						}
					}

					// 检查是否已经执行过该编译选项
					optionKey := fmt.Sprintf("%v%v", additionalFlags, garbleFlagSet)
					if _, exists := executedOptions.LoadOrStore(optionKey, true); exists {
						return
					}

					// 启动 Goroutine 进行编译
					sem <- struct{}{}
					defer func() { <-sem }()

					// 构建完整输出路径
					outputPath := filepath.Join(absOriginDir, output)
					//fmt.Printf("编译输出路径: %s\n", outputPath)

					// 生成动态构建信息
					buildInfo := generateBuildInfo()

					// 处理-ldflags：如果additionalFlags中包含-ldflags，需要合并构建信息
					var mergedFlags []string
					ldflagsFound := false
					for _, flag := range additionalFlags {
						if strings.HasPrefix(flag, "-ldflags=") {
							// 合并原有的-ldflags和新的构建信息
							existingLdflags := strings.TrimPrefix(flag, "-ldflags=")
							// 确保正确合并，使用空格分隔多个-X参数
							mergedLdflags := existingLdflags
							if existingLdflags != "" {
								mergedLdflags += " "
							}
							mergedLdflags += buildInfo
							mergedFlags = append(mergedFlags, "-ldflags="+mergedLdflags)
							ldflagsFound = true
						} else {
							mergedFlags = append(mergedFlags, flag)
						}
					}
					// 如果没有-ldflags，添加一个
					if !ldflagsFound {
						mergedFlags = append(mergedFlags, "-ldflags="+buildInfo)
					}

					cmdArgs := append(garbleFlagSet, "build", "-o", outputPath)
					cmdArgs = append(cmdArgs, mergedFlags...)
					cmdArgs = append(cmdArgs, inputFile)

					cmd := exec.Command(garbleBinary, cmdArgs...)
					cmd.Stderr = os.Stderr
					cmd.Stdout = os.Stdout
					cmd.SysProcAttr = hideWindowAttr()

					// 使用隔离编译缓存目录 + 附加国内代理链（仅首次缺缓存时下载依赖用） + 强制目标架构
					// GARBLE_CACHE 必须显式指定：garble 默认把自身缓存写到「当前工作目录/build」，
					// 会污染程序安装目录（生成一堆 trim.txt / 缓存文件）。
					baseEnv := append(os.Environ(),
						"GOCACHE="+sharedCacheDir,
						"GARBLE_CACHE="+filepath.Join(sharedCacheDir, "garble"),
					)
					baseEnv = append(baseEnv, goProxyExtra()...)
					// 非 Windows 目标交叉编译时禁用 cgo（-race 已过滤，纯 Go 可直接编译）
					if NormalizeTargetOS(targetOS) != "windows" {
						baseEnv = append(baseEnv, "CGO_ENABLED=0")
					}
					cmd.Env = append(baseEnv,
						"GOOS="+targetOS,
						"GOARCH="+goArch,
					)

					if buildOut, err := runWithCapture(cmd); err != nil {
						errChan <- fmt.Sprintf("[-] Compilation failed for %s with Garble: %v | %s", output, err, shortOutput(buildOut))
					} else {
						fmt.Printf("[+] Compilation succeeded with Garble: %s\n", output)
						PostProcessBuiltBinary(outputPath, targetOS, keepConsole)
					}
				}(additionalFlags, garbleFlagSet)
			}
		} else {
			// Go 编译分支（修复类似问题）
			wg.Add(1)
			go func(additionalFlags []string) {
				defer wg.Done()

				// 生成唯一文件名
				var output string
				for {
					suffix := generateRandomSuffix(nil)
					output = fmt.Sprintf("%s_%s%s", outputFile, suffix, binExt)
					if _, exists := generatedFiles.LoadOrStore(output, true); !exists {
						break
					}
				}

				// 检查是否已经执行过该编译选项
				optionKey := fmt.Sprintf("%v", additionalFlags)
				if _, exists := executedOptions.LoadOrStore(optionKey, true); exists {
					return
				}

				sem <- struct{}{}
				defer func() { <-sem }()

				// 构建完整输出路径
				outputPath := filepath.Join(absOriginDir, output)
				//fmt.Printf("编译输出路径: %s\n", outputPath)

				// 生成动态构建信息
				buildInfo := generateBuildInfo()

				// 处理-ldflags：如果additionalFlags中包含-ldflags，需要合并构建信息
				var mergedFlags []string
				ldflagsFound := false
				for _, flag := range additionalFlags {
					if strings.HasPrefix(flag, "-ldflags=") {
						// 合并原有的-ldflags和新的构建信息
						existingLdflags := strings.TrimPrefix(flag, "-ldflags=")
						// 确保正确合并，使用空格分隔多个-X参数
						mergedLdflags := existingLdflags
						if existingLdflags != "" {
							mergedLdflags += " "
						}
						mergedLdflags += buildInfo
						mergedFlags = append(mergedFlags, "-ldflags="+mergedLdflags)
						ldflagsFound = true
					} else {
						mergedFlags = append(mergedFlags, flag)
					}
				}
				// 如果没有-ldflags，添加一个
				if !ldflagsFound {
					mergedFlags = append(mergedFlags, "-ldflags="+buildInfo)
				}

				cmdArgs := append([]string{"build", "-o", outputPath}, mergedFlags...)
				cmdArgs = append(cmdArgs, inputFile)

				cmd := exec.Command(goBinary, cmdArgs...)
				cmd.Stderr = os.Stderr
				cmd.Stdout = os.Stdout
				cmd.SysProcAttr = hideWindowAttr()

				// 强制目标架构；非 Windows 目标交叉编译时禁用 cgo
				buildEnv := append(os.Environ(),
					"GOOS="+targetOS,
					"GOARCH="+goArch,
				)
				buildEnv = append(buildEnv, goProxyExtra()...)
				if NormalizeTargetOS(targetOS) != "windows" {
					buildEnv = append(buildEnv, "CGO_ENABLED=0")
				}
				cmd.Env = buildEnv

				if buildOut, err := runWithCapture(cmd); err != nil {
					errChan <- fmt.Sprintf("[-] Compilation failed for %s with Go: %v | %s", output, err, shortOutput(buildOut))
				} else {
					fmt.Printf("[+] Compilation succeeded with Go: %s\n", output)
					PostProcessBuiltBinary(outputPath, targetOS, keepConsole)
				}
			}(additionalFlags)
		}
	}

	// 等待所有 Goroutine 完成
	wg.Wait()
	close(errChan)

	// 收集编译错误：既打印到日志，也回传给调用方，便于 GUI 直接展示真实原因
	var compileErrs []string
	for errMsg := range errChan {
		fmt.Println(errMsg)
		compileErrs = append(compileErrs, strings.TrimSpace(errMsg))
	}

	// 隔离缓存跨次复用（garble 全量重建代价极高）；仅在其明显膨胀时清理一次。
	pruneGoCache(sharedCacheDir, 4<<30)

	if len(compileErrs) > 0 {
		// 只透出前两条，避免错误提示刷屏
		head := compileErrs
		if len(head) > 2 {
			head = head[:2]
		}
		msg := strings.Join(head, " | ")
		if len(compileErrs) > len(head) {
			msg += fmt.Sprintf("（共 %d 个编译变体失败）", len(compileErrs))
		}
		return fmt.Errorf("编译失败: %s", shortOutput(msg))
	}
	return nil
}

// runWithCapture 运行命令，同时把输出透传到标准输出并捕获一份，
// 便于把编译器/工具的真实报错带回给调用方（GUI 只看得到"exit status 1"时无从定位）。
func runWithCapture(cmd *exec.Cmd) (string, error) {
	var buf bytes.Buffer
	cmd.Stdout = io.MultiWriter(os.Stdout, &buf)
	cmd.Stderr = io.MultiWriter(os.Stderr, &buf)
	err := cmd.Run()
	return buf.String(), err
}

// pruneGoCache 编译缓存超限时清理，避免无限占用磁盘。
// 累计超过上限即提前结束遍历（缓存目录文件数可达十万级，全量统计会明显拖慢生成）。
func pruneGoCache(dir string, maxBytes int64) {
	total := int64(0)
	_ = filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		total += info.Size()
		if total > maxBytes {
			return filepath.SkipAll
		}
		return nil
	})
	if total <= maxBytes {
		return
	}
	fmt.Printf("编译缓存 %s 已超过 %.1fGB，清理中...\n", dir, float64(maxBytes)/(1<<30))
	if err := os.RemoveAll(dir); err != nil {
		fmt.Printf("清理编译缓存失败: %v\n", err)
	}
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
