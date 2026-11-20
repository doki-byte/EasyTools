package skyshadow

import (
	"EasyTools/app/controller/system"
	"bufio"
	"bytes"
	"debug/pe"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
)

var (
	microsoftDlls = make(map[string]bool)
	lang          string
	// toolsDir 为运行时工具所在目录。对于嵌入式版本，会指向 %TEMP%\EasyTools\skyshadowTools。
	toolsDir = "SkyShadowTools"
	// dataRoot 为 SkyShadow 持久化数据目录（例如 EasyToolsFiles/file/skyshadow）
	dataRoot string
	// workRoot 为 SkyShadow 运行期临时工作目录（位于 %TEMP%\EasyTools\skyshadowWork，运行结束后自动清理）
	workRoot  string
	verbose   bool
	debug     bool
	toolNames = []string{
		"dumpbin.exe",
		"link.exe",
		"mspdbcore.dll",
		"sigcheck64.exe",
		"tbbmalloc.dll",
	}
	once sync.Once

	// 全局状态互斥锁：lang/verbose/debug/dataRoot/workRoot/microsoftDlls 均为包级可变状态，
	// 并发调用 Run 会互相覆写并触发 concurrent map writes 崩溃，统一串行化。
	stateMu sync.Mutex

	// 用于控制扫描取消（简单实现：全局单任务，标记后尽快中止）
	cancelFlag int32
)

// DllInfo 表示单个 DLL 及其可调用的函数信息
type DllInfo struct {
	Name      string   `json:"name"`
	Functions []string `json:"functions"`
}

// PayloadMeta 表示一个白程序（Payload 目录）及其详细信息
type PayloadMeta struct {
	Header            string    `json:"header"`            // 目录名/头部信息，例如："x64 1234.00KB 有数字签名 xxx.exe"
	ExeName           string    `json:"exeName"`
	ExePath           string    `json:"exePath"`
	Bit               string    `json:"bit"`
	Size              string    `json:"size"`
	Signed            string    `json:"signed"`
	Dlls              []DllInfo `json:"dlls"`
	SelectedFunction  string    `json:"selectedFunction,omitempty"`  // 前端选择的函数名（兼容旧版本）
	SelectedFunctions []string  `json:"selectedFunctions,omitempty"` // 前端选择的函数名列表（支持多选），用于白加黑生成
}

func requestCancel() {
	atomic.StoreInt32(&cancelFlag, 1)
}

func resetCancel() {
	atomic.StoreInt32(&cancelFlag, 0)
}

func isCanceled() bool {
	return atomic.LoadInt32(&cancelFlag) == 1
}

//go:embed SkyShadowTools/*
var embeddedTools embed.FS

// pathInWork 将相对路径固定到 SkyShadow 工作目录下
// 如果未设置 workRoot，则退回为普通的相对路径。
func pathInWork(elem ...string) string {
	if workRoot == "" {
		return filepath.Join(elem...)
	}
	return filepath.Join(append([]string{workRoot}, elem...)...)
}

// initTools 确保所有嵌入的工具都已释放到磁盘上的 toolsDir
func initTools() error {
	var initErr error
	once.Do(func() {
		if err := os.MkdirAll(toolsDir, 0755); err != nil {
			initErr = fmt.Errorf("创建 tools 目录失败: %w", err)
			return
		}
		// 这里不再使用 go:embed，而是假定 Tools 目录下已经存在相应 exe，
		// 仅做存在性检查，方便在 UI 中给出友好报错。
		for _, name := range toolNames {
			dest := filepath.Join(toolsDir, name)
			if _, err := os.Stat(dest); os.IsNotExist(err) {
				initErr = fmt.Errorf("缺少必要工具 %s，请将其放置到 %s 目录", name, toolsDir)
				return
			}
		}
	})
	return initErr
}

func makeExportLine(fnName string) string {
	if lang == "go" {
		return fmt.Sprintf(`//export %s
func %s() C.int {
    C.MessageBoxA(nil, C.CString("%s called"), C.CString("DLL Hijack"), 0)
    return 0
}`, fnName, fnName, fnName)
	}
	return fmt.Sprintf(`extern "C" __declspec(dllexport) int %s() {
    MessageBoxA(NULL, "%s", "DLL Hijack", 0);
    return 0;
}`, fnName, fnName)
}

// detectBitness 通过解析 PE 头，返回 "x64"、"x86" 或 "unknown"
func detectBitness(path string) (string, error) {
	f, err := pe.Open(path)
	if err != nil {
		return "unknown", err
	}
	defer f.Close()

	switch f.FileHeader.Machine {
	case pe.IMAGE_FILE_MACHINE_AMD64:
		return "x64", nil
	case pe.IMAGE_FILE_MACHINE_I386:
		return "x86", nil
	default:
		return "unknown", nil
	}
}

func getPayload(path, exeName string) {
	exeFullPath := filepath.Join(path, exeName)
	log.Printf("处理 EXE: %s", exeFullPath)

	fileInfo, err := os.Stat(exeFullPath)
	if err != nil {
		log.Printf("错误: 无法访问文件 - %v", err)
		return
	}

	exeSize := fileInfo.Size()

	dumpbinPath := filepath.Join(toolsDir, "dumpbin.exe")
	if _, err := os.Stat(dumpbinPath); os.IsNotExist(err) {
		log.Printf("错误: 缺少工具 - dumpbin.exe 不在 %s 目录", toolsDir)
		return
	}

	cmd := exec.Command(dumpbinPath, "/imports", exeFullPath)
	cmd.SysProcAttr = hideWindowAttr()
	importsOutput, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("dumpbin 错误: %v", err)
		return
	}

	if debug {
		debugPath := pathInWork("debug", exeName+".dumpbin.txt")
		_ = os.MkdirAll(filepath.Dir(debugPath), 0755)
		if err := os.WriteFile(debugPath, importsOutput, 0644); err == nil {
			log.Printf("保存 dumpbin 输出到: %s", debugPath)
		}
	}

	hijackableDLLs := make(map[string][]string)
	outputStr := string(importsOutput)

	dllPattern := regexp.MustCompile(`(?im)^\s+(\S+\.dll)\s*$`)

	log.Println("开始分析 DLL 导入")

	currentDLL := ""
	currentIsMicrosoft := false

	for _, line := range strings.Split(outputStr, "\n") {
		if dllMatch := dllPattern.FindStringSubmatch(line); len(dllMatch) > 1 {
			dllName := dllMatch[1]
			dllLower := strings.ToLower(dllName)

			currentDLL = dllName
			currentIsMicrosoft = microsoftDlls[dllLower]
			continue
		}

		if currentDLL != "" && !currentIsMicrosoft {
			funcPattern := regexp.MustCompile(`^\s+\d+\s+(\w+)\s*$`)
			if funcMatch := funcPattern.FindStringSubmatch(line); len(funcMatch) > 1 {
				fnName := funcMatch[1]
				hijackableDLLs[currentDLL] = append(hijackableDLLs[currentDLL], fnName)
			}
		}
	}

	if len(hijackableDLLs) == 0 {
		log.Printf("没有找到可劫持的 DLL: %s", exeName)
		return
	}

	log.Println("Found hijackable EXE:", exeFullPath)

	bit := "unknown"
	signed := "无数字签名"

	sigcheckPath := filepath.Join(toolsDir, "sigcheck64.exe")
	if _, err := os.Stat(sigcheckPath); err == nil {
		cmd = exec.Command(sigcheckPath, "-accepteula", "-nobanner", exeFullPath)
		cmd.SysProcAttr = hideWindowAttr()
		sigcheckOutput, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("sigcheck 错误: %v", err)
		}
		sigStr := string(sigcheckOutput)

		if strings.Contains(sigStr, "64-bit PE") ||
			strings.Contains(sigStr, "x64") ||
			strings.Contains(sigStr, "AMD64") {
			bit = "x64"
		} else if strings.Contains(sigStr, "32-bit PE") ||
			strings.Contains(sigStr, "x86") ||
			strings.Contains(sigStr, "Intel") {
			bit = "x86"
		}

		signed = "无数字签名"
		if strings.Contains(sigStr, "Signing date:") ||
			strings.Contains(sigStr, "Publisher:") ||
			strings.Contains(sigStr, "Verified: Signed") {
			signed = "有数字签名"
		}

		bitness, err := detectBitness(exeFullPath)
		if err == nil && bitness != "unknown" {
			bit = bitness
		}
	}

	header := fmt.Sprintf("%s %s %s %s", bit, fmt.Sprintf("%.2fMB", float64(exeSize)/1024/1024), signed, exeName)

	// Payload 目录放在持久化目录下：EasyToolsFiles/file/skyshadow/Payload/...
	basePayloadRoot := "Payload"
	if dataRoot != "" {
		basePayloadRoot = filepath.Join(dataRoot, "Payload")
	}
	outDir := filepath.Join(basePayloadRoot, header)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		log.Printf("创建目录失败: %v", err)
		return
	}

	payloadPath := filepath.Join(outDir, "payload.txt")
	payloadFile, err := os.Create(payloadPath)
	if err != nil {
		log.Printf("创建payload文件失败: %v", err)
		return
	}
	defer payloadFile.Close()

	_, _ = payloadFile.WriteString(header + "\n\n")

	for dllName, functions := range hijackableDLLs {
		_, _ = payloadFile.WriteString(dllName + "\n")
		for _, fn := range functions {
			_, _ = payloadFile.WriteString(makeExportLine(fn))
			_, _ = payloadFile.WriteString("\n\n")
		}
	}

	// 生成结构化的元数据文件，便于前端展示白程序和可调用函数列表
	var dlls []DllInfo
	for dllName, functions := range hijackableDLLs {
		dlls = append(dlls, DllInfo{
			Name:      dllName,
			Functions: functions,
		})
	}

	meta := PayloadMeta{
		Header:  header,
		ExeName: exeName,
		ExePath: exeFullPath,
		Bit:     bit,
		Size:    fmt.Sprintf("%.2fMB", float64(exeSize)/1024/1024),
		Signed:  signed,
		Dlls:    dlls,
	}

	metaBytes, err := json.MarshalIndent(meta, "", "  ")
	if err == nil {
		metaPath := filepath.Join(outDir, "meta.json")
		if err := os.WriteFile(metaPath, metaBytes, 0644); err != nil {
			log.Printf("写入元数据失败: %v", err)
		}
	} else {
		log.Printf("序列化元数据失败: %v", err)
	}

	dst := filepath.Join(outDir, exeName)
	if err := copyFile(exeFullPath, dst); err != nil {
		log.Printf("复制文件失败: %v", err)
	}

	log.Printf("Payload created: %s", outDir)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func loadMicrosoftDlls() {
	// 微软 DLL 列表等中间文件放在临时工作目录下，运行完自动清理
	dllFile := pathInWork("微软 DLL.txt")
	if _, err := os.Stat(dllFile); err == nil {
		file, err := os.Open(dllFile)
		if err != nil {
			log.Printf("打开文件失败: %v", err)
			return
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			dll := strings.ToLower(scanner.Text())
			microsoftDlls[dll] = true
		}
		return
	}

	log.Println("扫描系统目录获取微软DLL列表...")

	scanSystemDir := func(path string) {
		if _, err := os.Stat(path); err == nil {
			scan(path, ".dll")
		}
	}

	scanSystemDir("C:\\Windows\\System32")
	scanSystemDir("C:\\Windows\\SysWOW64")
	scanSystemDir("C:\\Windows\\System")
	scanSystemDir("C:\\Windows\\WinSxS")

	var dllList []string
	for dll := range microsoftDlls {
		dllList = append(dllList, dll)
	}

	_ = os.WriteFile(dllFile, []byte(strings.Join(dllList, "\n")), 0644)
}

func scan(path, suffix string) {
	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		// 外部请求了取消，则尽快结束扫描
		if isCanceled() {
			if info != nil && info.IsDir() {
				return filepath.SkipDir
			}
			return filepath.SkipDir
		}

		if err != nil {
			return nil
		}

		if info.IsDir() {
			if strings.EqualFold(filePath, "C:\\Windows\\WinSxS") && !isAdmin() {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(filePath))
		if ext != suffix {
			return nil
		}

		fileName := filepath.Base(filePath)
		dir := filepath.Dir(filePath)

		if suffix == ".dll" {
			dllName := strings.ToLower(fileName)
			if _, exists := microsoftDlls[dllName]; !exists {
				microsoftDlls[dllName] = true
			}
		} else if suffix == ".exe" {
			getPayload(dir, fileName)
		}

		return nil
	})

	if err != nil && verbose {
		log.Printf("扫描错误: %v", err)
	}
}

func processPayloadFile(filename string) {
	content, err := os.ReadFile(filename)
	if err != nil {
		log.Printf("读取文件失败: %v", err)
		return
	}

	funcPattern := regexp.MustCompile(`extern "C" __declspec\(dllexport\) int (\w+)\(\)`)
	matches := funcPattern.FindAllSubmatch(content, -1)

	if len(matches) == 0 {
		log.Println("未找到函数")
		return
	}

	var code []string
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		fnName := string(match[1])
		code = append(code, makeExportLine(fnName))
	}

	// 语言导出模板需要保存在持久化目录 EasyToolsFiles/file/skyshadow 下，便于后续复用
	// 如果 dataRoot 未设置，则使用默认路径
	if dataRoot == "" {
		base := new(system.Base)
		appBaseDir := base.GetAppPath()
		dataRoot = filepath.Join(appBaseDir, "file", "skyshadow")
	}
	if err := os.MkdirAll(dataRoot, 0755); err != nil {
		log.Printf("创建导出目录失败: %v", err)
		return
	}
	outFile := filepath.Join(dataRoot, fmt.Sprintf("exports_%s.txt", lang))
	if err := os.WriteFile(outFile, []byte(strings.Join(code, "\n\n")), 0644); err != nil {
		log.Printf("写入导出文件失败: %v", err)
		return
	}
}

func isAdmin() bool {
	_, err := os.Open("\\\\.\\PHYSICALDRIVE0")
	return err == nil
}

// prepareSkyShadowTools 会在每次调用前，将源码目录下通过 go:embed 打包的 SkyShadowTools
// 释放到 %TEMP%\\EasyTools\\skyshadowTools 下，返回最终使用的 tools 目录
// 以及清理函数，用于调用结束后删除临时目录。
//
// 注意：这里不再依赖 EasyToolsFiles\\file\\skyshadow\\skyshadowTools 这个持久化目录，
// 只要在源码路径 app/bypass/skyshadow/SkyShadowTools 下放好需要的 EXE/DLL 即可。
func prepareSkyShadowTools() (string, func(), error) {
	// 目标目录：%TEMP%\\EasyTools\\skyshadowTools
	tempRoot := filepath.Join(os.TempDir(), "EasyTools")
	dstDir := filepath.Join(tempRoot, "skyshadowTools")

	// 每次调用前先清掉旧目录，避免残留影响
	_ = os.RemoveAll(dstDir)

	// 从 embed 中按文件名逐个写出
	// 注意：embed.FS 使用的是统一的 "/" 分隔符，不能用 filepath.Join 来构造路径，否则在 Windows 上会变成 "\" 导致找不到文件。
	for _, name := range toolNames {
		srcPath := "SkyShadowTools/" + name
		data, err := embeddedTools.ReadFile(srcPath)
		if err != nil {
			return "", func() {}, fmt.Errorf("读取内置工具 %s 失败: %w，请确认其存在于 SkyShadowTools 目录", name, err)
		}

		targetPath := filepath.Join(dstDir, name)
		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return "", func() {}, fmt.Errorf("创建工具目录失败: %w", err)
		}
		if err := os.WriteFile(targetPath, data, 0o755); err != nil {
			return "", func() {}, fmt.Errorf("写入工具文件 %s 失败: %w", targetPath, err)
		}
	}

	// 清理函数：只删除 SkyShadow 自己的工具目录，不删除整个 EasyTools 临时目录
	//（避免与其他模块（upload/make/binder）的共享工作目录互相破坏）
	cleanup := func() {
		_ = os.RemoveAll(dstDir)
	}
	return dstDir, cleanup, nil
}

// PrepareDumpbin 对外暴露 dumpbin.exe 的准备逻辑，供其他模块（例如 dllproxy）复用。
// 内部复用 SkyShadow 的嵌入资源和释放逻辑，返回 dumpbin.exe 的绝对路径和清理函数。
func PrepareDumpbin() (string, func(), error) {
	toolsDir, cleanup, err := prepareSkyShadowTools()
	if err != nil {
		return "", func() {}, fmt.Errorf("准备 SkyShadowTools 失败：%w", err)
	}

	dumpbinPath := filepath.Join(toolsDir, "dumpbin.exe")
	if _, err := os.Stat(dumpbinPath); err != nil {
		// 如果 dumpbin 不存在，立即清理临时目录并返回错误
		cleanup()
		return "", func() {}, fmt.Errorf("dumpbin.exe 不存在：%w", err)
	}

	return dumpbinPath, cleanup, nil
}

// copyDir 递归复制目录
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(targetPath, 0755)
		}

		// 普通文件直接复制
		if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
			return err
		}
		return copyFile(path, targetPath)
	})
}

// Options 用于控制 SkyShadow 扫描行为
type Options struct {
	Lang    string
	Verbose bool
	Debug   bool
}

// Run 作为库函数供后端调用
// target 可以是目录（扫描 EXE）或 Payload 文本文件
//
//	output:  简单状态字符串（如 "ok"），当前版本不捕获详细日志输出
//	payloadDir: 生成的 Payload 根目录路径（相对路径："Payload"）
//	payloads: Payload 下的子目录名称列表
//	appBaseDir: 应用主目录（即 EasyToolsFiles 跟目录），用于定位 SkyShadowTools
func Run(target string, opts Options, appBaseDir string) (output string, payloadDir string, payloads []string, details []PayloadMeta, err error) {
	// 串行化全局状态访问（单任务设计），避免并发 map 写崩溃与状态串台
	stateMu.Lock()
	defer stateMu.Unlock()

	if opts.Lang == "" {
		lang = "c"
	} else {
		lang = opts.Lang
	}
	verbose = opts.Verbose
	debug = opts.Debug

	// 重置取消标记
	resetCancel()

	// 捕获日志到缓冲区，并写入日志文件（不再输出到终端），便于前端实时拉取
	var logBuf bytes.Buffer
	var logWriter io.Writer = &logBuf

	// 设置 SkyShadow 持久化目录（白程序、Payload 等）：EasyToolsFiles/file/skyshadow
	if appBaseDir != "" {
		dataRoot = filepath.Join(appBaseDir, "file", "skyshadow")
		if err := os.MkdirAll(dataRoot, 0755); err != nil {
			return "", "", nil, nil, fmt.Errorf("初始化 SkyShadow 数据目录失败：%w", err)
		}

		// 初始化日志文件 writer：EasyToolsFiles/file/skyshadow/skyshadow.log
		logFilePath := filepath.Join(dataRoot, "skyshadow.log")
		logFile, ferr := os.Create(logFilePath)
		if ferr == nil {
			defer logFile.Close()
			logWriter = io.MultiWriter(&logBuf, logFile)
		}

		// 设置临时工作目录：%TEMP%\EasyTools\skyshadowWork（中间文件：微软 DLL.txt、debug、exports_*.txt 等）
		tempWorkRoot := filepath.Join(os.TempDir(), "EasyTools", "skyshadowWork")
		_ = os.RemoveAll(tempWorkRoot)
		if err := os.MkdirAll(tempWorkRoot, 0755); err != nil {
			return "", "", nil, nil, fmt.Errorf("初始化 SkyShadow 临时工作目录失败：%w", err)
		}
		workRoot = tempWorkRoot
	} else {
		dataRoot = ""
		workRoot = ""
	}

	log.SetOutput(logWriter)
	log.SetFlags(log.LstdFlags)

	// 每次调用前，将内置 SkyShadowTools 释放到临时目录，并在结束后清理
	if tmpToolsDir, cleanup, perr := prepareSkyShadowTools(); perr != nil {
		return "", "", nil, nil, fmt.Errorf("准备 SkyShadowTools 失败：%w", perr)
	} else {
		toolsDir = tmpToolsDir
		defer cleanup()
	}

	if err := initTools(); err != nil {
		return "", "", nil, nil, fmt.Errorf("工具初始化失败：%w", err)
	}

	loadMicrosoftDlls()

	info, err := os.Stat(target)
	if err != nil {
		return "", "", nil, nil, fmt.Errorf("路径无效: %w", err)
	}

	if info.IsDir() {
		log.Println("开始扫描目录:", target)
		scan(target, ".exe")
	} else {
		log.Println("处理Payload文件:", target)
		processPayloadFile(target)
	}

	// Payload 根目录在持久化目录下
	payloadRoot := "Payload"
	if dataRoot != "" {
		payloadRoot = filepath.Join(dataRoot, "Payload")
	}
	if st, err := os.Stat(payloadRoot); err == nil && st.IsDir() {
		entries, _ := os.ReadDir(payloadRoot)
		for _, e := range entries {
			if e.IsDir() {
				name := e.Name()
				payloads = append(payloads, name)

				// 读取每个 Payload 目录下的 meta.json，聚合为详情列表
				metaPath := filepath.Join(payloadRoot, name, "meta.json")
				metaBytes, err := os.ReadFile(metaPath)
				if err != nil {
					continue
				}
				var meta PayloadMeta
				if err := json.Unmarshal(metaBytes, &meta); err != nil {
					continue
				}
				details = append(details, meta)
			}
		}
	}

	// 返回给上层的目录相对于 EasyToolsFiles：file/skyshadow/Payload
	payloadDir = filepath.Join("file", "skyshadow", "Payload")

	// 如果在执行过程中被取消，给出明确提示
	if isCanceled() {
		output = "扫描已被用户取消。\n\n" + logBuf.String()
	} else {
		output = logBuf.String()
	}

	return output, payloadDir, payloads, details, nil
}
