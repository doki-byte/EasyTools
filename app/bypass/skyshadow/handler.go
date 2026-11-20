package skyshadow

import (
	"EasyTools/app/bypass/shell/util"
	"EasyTools/app/controller/system"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

// ScanRequest HTTP 请求参数
type ScanRequest struct {
	Target  string `json:"target"`
	Lang    string `json:"lang"`
	Verbose bool   `json:"verbose"`
	Debug   bool   `json:"debug"`
}

// CancelRequest HTTP 请求参数（预留，可扩展多任务 ID）
type CancelRequest struct {
	// 目前仅支持全局单任务取消，保留字段便于以后扩展
	JobID string `json:"jobId"`
}

// RunScanHandler HTTP 接口：封装 SkyShadow-main 的 Go 代码
func RunScanHandler(c *gin.Context) {
	// 检查授权
	if runtime.GOOS != "windows" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "SkyShadow 仅支持在 Windows 平台运行",
		})
		return
	}

	var req ScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "请求参数错误",
		})
		return
	}

	target := strings.TrimSpace(req.Target)
	if target == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "target 不能为空",
		})
		return
	}

	// 规范语言参数
	lang := strings.ToLower(strings.TrimSpace(req.Lang))
	if lang == "" {
		lang = "go"
	}
	if lang != "go" && lang != "c" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "lang 仅支持 go 或 c",
		})
		return
	}

	// 目标路径转换为绝对路径（相对应用根目录）
	base := new(system.Base)
	baseDir := base.GetAppPath()
	if !filepath.IsAbs(target) {
		target = filepath.Join(baseDir, target)
	}

	// 安全限制：扫描目标必须位于 skyshadow 数据目录内，防止任意目录全盘扫描/EXE 收集
	skyshadowRoot := filepath.Join(baseDir, "file", "skyshadow")
	absTarget, absErr := filepath.Abs(target)
	if absErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "目标路径无效",
		})
		return
	}
	if rel, relErr := filepath.Rel(filepath.Clean(skyshadowRoot), filepath.Clean(absTarget)); relErr != nil ||
		rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "target 必须在 skyshadow 数据目录内",
		})
		return
	}

	opts := Options{
		Lang:    lang,
		Verbose: req.Verbose,
		Debug:   req.Debug,
	}

	out, payloadDir, payloads, details, err := Run(target, opts, baseDir)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   err.Error(),
			"output":  out,
		})
		return
	}

	// SkyShadow 返回的 Payload 目录是相对路径，这里转换为绝对路径方便前端使用
	absPayloadDir := payloadDir
	if !filepath.IsAbs(absPayloadDir) {
		absPayloadDir = filepath.Join(baseDir, payloadDir)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"output":         out,
		"payloads":       payloads,
		"payloadDir":     absPayloadDir,
		"payloadDetails": details,
	})
}

// 用一个简单标记，避免在没有正在运行任务时重复取消
var cancelRequested int32

// CancelScanHandler 提供前端"取消扫描"接口
func CancelScanHandler(c *gin.Context) {
	// 检查授权
	if runtime.GOOS != "windows" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "SkyShadow 仅支持在 Windows 平台运行",
		})
		return
	}

	var req CancelRequest
	_ = c.ShouldBindJSON(&req) // 暂时忽略错误，仅作兼容

	// 标记已请求取消
	atomic.StoreInt32(&cancelRequested, 1)
	requestCancel()

	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

// LogsHandler 返回当前 SkyShadow 扫描日志内容，供前端轮询实现"准实时"刷新
func LogsHandler(c *gin.Context) {
	// 检查授权
	base := new(system.Base)
	baseDir := base.GetAppPath()

	logPath := filepath.Join(baseDir, "file", "skyshadow", "skyshadow.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "读取日志失败或暂无日志",
			"logs":    "",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"logs":    string(data),
	})
}

// PayloadsHandler 返回当前所有 Payload 目录及其详情，供前端实时刷新表格
func PayloadsHandler(c *gin.Context) {
	// 检查授权
	base := new(system.Base)
	baseDir := base.GetAppPath()

	payloadRoot := filepath.Join(baseDir, "file", "skyshadow", "Payload")

	var payloads []string
	var details []PayloadMeta

	if st, err := os.Stat(payloadRoot); err == nil && st.IsDir() {
		entries, _ := os.ReadDir(payloadRoot)
		for _, e := range entries {
			if e.IsDir() {
				name := e.Name()
				payloads = append(payloads, name)

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

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"payloads":       payloads,
		"payloadDetails": details,
		"payloadDir":     payloadRoot,
	})
}

// GenerateTestDllRequest 生成测试 DLL 的请求参数
type GenerateTestDllRequest struct {
	DllName string `json:"dllName"` // 要生成的 DLL 名称，例如 "test.dll"
	Lang    string `json:"lang"`    // 语言：go 或 c
}

// GenerateTestDllsRequest 批量生成测试 DLL 的请求参数
type GenerateTestDllsRequest struct {
	Payloads []PayloadMeta `json:"payloads"` // Payload 列表
	Lang     string        `json:"lang"`     // 语言：go 或 c
}

// GenerateDllResult 单个 DLL 生成结果
type GenerateDllResult struct {
	DllName string `json:"dllName"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// GenerateTestDllHandler 生成测试 messagebox DLL 的接口
func GenerateTestDllHandler(c *gin.Context) {
	// 检查授权
	if runtime.GOOS != "windows" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "SkyShadow 仅支持在 Windows 平台运行",
		})
		return
	}

	var req GenerateTestDllRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "请求参数错误",
		})
		return
	}

	dllName := strings.TrimSpace(req.DllName)
	if dllName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "dllName 不能为空",
		})
		return
	}

	// 规范语言参数
	lang := strings.ToLower(strings.TrimSpace(req.Lang))
	if lang == "" {
		lang = "go"
	}
	if lang != "go" && lang != "c" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "lang 仅支持 go 或 c",
		})
		return
	}

	// 确保 DLL 名称以 .dll 结尾
	dllName = sanitizeDllName(dllName)
	if dllName == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   "非法的 DLL 名称",
		})
		return
	}

	// 1. 环境检测
	if err := util.InitGoModule(); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   fmt.Sprintf("环境检测失败: %v", err),
		})
		return
	}

	// 2. 生成代码文件（在 temp 目录下）
	tempDir := filepath.Join(os.TempDir(), "EasyTools", "file")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   fmt.Sprintf("创建临时目录失败: %v", err),
		})
		return
	}

	var codeFile string
	var codeContent string

	if lang == "go" {
		codeFile = filepath.Join(tempDir, "test_dll.go")
		codeContent = fmt.Sprintf(`package main

/*
#include <windows.h>
*/
import "C"
import (
	"unsafe"
)

//export TestFunction
func TestFunction() C.int {
	message := C.CString("Test DLL loaded successfully!")
	title := C.CString("DLL Hijack Test")
	C.MessageBoxA(nil, message, title, 0)
	C.free(unsafe.Pointer(message))
	C.free(unsafe.Pointer(title))
	return 0
}

func main() {}
`)
	} else {
		codeFile = filepath.Join(tempDir, "test_dll.cpp")
		codeContent = `#include <windows.h>

extern "C" {
__declspec(dllexport) int TestFunction() {
    MessageBoxA(NULL, "Test DLL loaded successfully!", "DLL Hijack Test", 0);
    return 0;
}
}
`
	}

	if err := os.WriteFile(codeFile, []byte(codeContent), 0644); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   fmt.Sprintf("写入代码文件失败: %v", err),
		})
		return
	}

	outputDllPath := filepath.Join(tempDir, dllName)

	// 编译 DLL
	var cmd *exec.Cmd
	if lang == "go" {
		// Go 使用 c-shared 模式
		args := []string{
			"build",
			"-buildmode=c-shared",
			"--trimpath",
			"-ldflags=-s -w",
			"-o", outputDllPath,
			codeFile,
		}
		cmd = exec.Command("go", args...)
		cmd.Dir = tempDir
	} else {
		// C/C++ 使用 g++ 编译，保持 C 接口导出
		args := []string{
			"-shared",
			"-s", // 去掉调试信息，避免泄露本机信息
			"-o", outputDllPath,
			codeFile,
		}
		cmd = exec.Command("g++", args...)
		cmd.Dir = tempDir
	}

	cmd.SysProcAttr = hideWindowAttr()
	output, err := cmd.CombinedOutput()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   fmt.Sprintf("编译 DLL 失败: %v\n输出: %s", err, string(output)),
		})
		return
	}

	// 删除生成的 .h 文件（Go 编译会生成）
	if lang == "go" {
		headerFile := strings.TrimSuffix(outputDllPath, ".dll") + ".h"
		os.Remove(headerFile)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": fmt.Sprintf("测试 DLL 生成成功: %s", outputDllPath),
		"dllPath": outputDllPath,
	})
}

// GenerateTestDllsHandler 批量生成测试 DLL 的接口
func GenerateTestDllsHandler(c *gin.Context) {
	// 检查授权
	if runtime.GOOS != "windows" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "SkyShadow 仅支持在 Windows 平台运行",
		})
		return
	}

	var req GenerateTestDllsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "请求参数错误",
		})
		return
	}

	if len(req.Payloads) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "payloads 不能为空",
		})
		return
	}

	// 规范语言参数
	lang := strings.ToLower(strings.TrimSpace(req.Lang))
	if lang == "" {
		lang = "go"
	}
	if lang != "go" && lang != "c" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "lang 仅支持 go 或 c",
		})
		return
	}

	base := new(system.Base)
	baseDir := base.GetAppPath()

	// 1. 环境检测
	if err := util.InitGoModule(); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   fmt.Sprintf("环境检测失败: %v", err),
		})
		return
	}

	// 2. 准备临时目录
	tempDir := filepath.Join(os.TempDir(), "EasyTools", "file")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   fmt.Sprintf("创建临时目录失败: %v", err),
		})
		return
	}

	// 4. 批量生成 DLL
	// 先收集所有 DLL 信息，合并相同 DLL 的函数列表，
	// 同时记录每个 DLL 对应出现在哪些 Payload 目录（Header）下
	dllMap := make(map[string]map[string]bool)        // DLL 名称 -> 函数名集合
	dllPayloadMap := make(map[string]map[string]bool) // DLL 名称 -> Payload Header 集合
	dllBitnessMap := make(map[string]string)          // DLL 名称 -> 架构 (x86/x64)，取第一个 payload 的架构
	for _, payload := range req.Payloads {
		bit := payload.Bit // 获取 payload 的架构信息
		for _, dllInfo := range payload.Dlls {
			dllName := sanitizeDllName(dllInfo.Name)
			if dllName == "" {
				continue // 非法 DLL 名直接忽略
			}
			if dllMap[dllName] == nil {
				dllMap[dllName] = make(map[string]bool)
			}
			if dllPayloadMap[dllName] == nil {
				dllPayloadMap[dllName] = make(map[string]bool)
			}
			// 记录架构信息（如果还没有记录，使用当前 payload 的架构）
			if dllBitnessMap[dllName] == "" {
				dllBitnessMap[dllName] = bit
			}
			for _, fn := range dllInfo.Functions {
				fn = sanitizeFuncName(fn)
				if fn == "" {
					continue // 非法函数名直接忽略，防止生成源码注入
				}
				dllMap[dllName][fn] = true
			}
			// 记录该 DLL 出现在当前 Payload(Header) 中，后面用于把生成的 DLL 分发到各自的目录下
			if payload.Header != "" {
				dllPayloadMap[dllName][payload.Header] = true
			}
		}
	}

	var results []GenerateDllResult
	successCount := 0
	failCount := 0

	// Payload 根目录：EasyToolsFiles/file/skyshadow/Payload
	payloadRoot := filepath.Join(baseDir, "file", "skyshadow", "Payload")

	// 为每个唯一的 DLL 生成代码
	for dllName, funcSet := range dllMap {
		// 将函数集合转换为列表
		var functions []string
		for fn := range funcSet {
			functions = append(functions, fn)
		}

		if len(functions) == 0 {
			results = append(results, GenerateDllResult{
				DllName: dllName,
				Success: false,
				Message: "没有可导出的函数",
			})
			failCount++
			continue
		}

		// 生成代码文件
		var codeFile string
		var codeContent string

		if lang == "go" {
			codeFile = filepath.Join(tempDir, fmt.Sprintf("%s_%s.go", strings.TrimSuffix(dllName, ".dll"), generateRandomSuffix()))
			codeContent = generateGoDllCode(dllName, functions)
		} else {
			codeFile = filepath.Join(tempDir, fmt.Sprintf("%s_%s.cpp", strings.TrimSuffix(dllName, ".dll"), generateRandomSuffix()))
			codeContent = generateCDllCode(dllName, functions)
		}

		if err := os.WriteFile(codeFile, []byte(codeContent), 0644); err != nil {
			results = append(results, GenerateDllResult{
				DllName: dllName,
				Success: false,
				Message: fmt.Sprintf("写入代码文件失败: %v", err),
			})
			failCount++
			continue
		}

		// 在公共输出目录中先生成一份 DLL，后面再复制到各自的 Payload 目录下
		outputDllPath := filepath.Join(tempDir, dllName)

		// 编译 DLL（生成到公共目录）
		var cmd *exec.Cmd
		env := os.Environ()

		if lang == "go" {
			// 获取该 DLL 对应的架构信息
			bitness := dllBitnessMap[dllName]
			if bitness == "" {
				bitness = "x64" // 默认使用 x64
			}

			// 根据架构设置编译环境
			env = append(env, "GOOS=windows")
			env = append(env, "CGO_ENABLED=1")

			var ccPath, newPath string
			if bitness == "x86" {
				env = append(env, "GOARCH=386")
				mingw32Path := util.FindMinGWPath("mingw32")
				if mingw32Path == "" {
					results = append(results, GenerateDllResult{
						DllName: dllName,
						Success: false,
						Message: "未找到 MinGW32 编译器，请安装 MinGW32",
					})
					failCount++
					continue
				}
				ccPath = filepath.Join(mingw32Path, "bin", "gcc.exe")
				newPath = filepath.Join(mingw32Path, "bin") + ";" + os.Getenv("PATH")
			} else {
				env = append(env, "GOARCH=amd64")
				mingw64Path := util.FindMinGWPath("mingw64")
				if mingw64Path == "" {
					results = append(results, GenerateDllResult{
						DllName: dllName,
						Success: false,
						Message: "未找到 MinGW64 编译器，请安装 MinGW64",
					})
					failCount++
					continue
				}
				ccPath = filepath.Join(mingw64Path, "bin", "gcc.exe")
				newPath = filepath.Join(mingw64Path, "bin") + ";" + os.Getenv("PATH")
			}

			// 检查编译器是否存在
			if _, err := os.Stat(ccPath); os.IsNotExist(err) {
				results = append(results, GenerateDllResult{
					DllName: dllName,
					Success: false,
					Message: fmt.Sprintf("编译器不存在: %s", ccPath),
				})
				failCount++
				continue
			}

			env = append(env, "CC="+ccPath)
			env = append(env, "PATH="+newPath)

			args := []string{
				"build",
				"-buildmode=c-shared",
				"--trimpath",
				"-ldflags=-s -w",
				"-o", outputDllPath,
				codeFile,
			}
			cmd = exec.Command("go", args...)
			cmd.Dir = tempDir
			cmd.Env = env
		} else {
			args := []string{
				"-shared",
				"-s", // 去掉调试信息，避免泄露本机信息
				"-o", outputDllPath,
				codeFile,
			}
			cmd = exec.Command("g++", args...)
			cmd.Dir = tempDir
		}

		cmd.SysProcAttr = hideWindowAttr()
		output, err := cmd.CombinedOutput()
		if err != nil {
			results = append(results, GenerateDllResult{
				DllName: dllName,
				Success: false,
				Message: fmt.Sprintf("编译失败: %v", string(output)),
			})
			failCount++
			continue
		}

		// 删除生成的 .h 文件（Go 编译会生成）
		if lang == "go" {
			headerFile := strings.TrimSuffix(outputDllPath, ".dll") + ".h"
			_ = os.Remove(headerFile)
		}

		// 将生成好的 DLL 分发到各自的 Payload 目录中：
		// EasyToolsFiles/file/skyshadow/Payload/<Header>/<dllName>
		payloadHeaders := dllPayloadMap[dllName]
		if len(payloadHeaders) == 0 {
			// 理论上不会发生，但为了健壮性保留
			results = append(results, GenerateDllResult{
				DllName: dllName,
				Success: true,
				Message: fmt.Sprintf("生成成功(未找到对应 Payload，保留在公共目录): %s", outputDllPath),
			})
			successCount++
			continue
		}

		distributeOK := true
		for header := range payloadHeaders {
			if header == "" {
				continue
			}
			// 目录段安全校验：只允许普通目录名（防路径穿越写任意位置）
			if header != filepath.Base(header) || strings.ContainsAny(header, `/\`) {
				continue
			}
			payloadDir := filepath.Join(payloadRoot, header)
			if err := os.MkdirAll(payloadDir, 0755); err != nil {
				distributeOK = false
				results = append(results, GenerateDllResult{
					DllName: dllName,
					Success: false,
					Message: fmt.Sprintf("创建 Payload 目录失败: %v", err),
				})
				continue
			}

			targetDllPath := filepath.Join(payloadDir, dllName)

			// 复制新生成的 DLL 到 Payload 目录（如存在则直接覆盖）
			if err := copyFile(outputDllPath, targetDllPath); err != nil {
				distributeOK = false
				results = append(results, GenerateDllResult{
					DllName: dllName,
					Success: false,
					Message: fmt.Sprintf("复制 DLL 到 Payload 目录失败(%s): %v", header, err),
				})
				continue
			}
		}

		if distributeOK {
			results = append(results, GenerateDllResult{
				DllName: dllName,
				Success: true,
				Message: fmt.Sprintln("生成并分发成功，请点击下方打开目录查看生成的 DLL"),
			})
			successCount++
		} else {
			// 分发过程中如果有失败，但至少编译成功了，这里视情况可以算部分成功。
			// 当前逻辑：如果所有 Payload 都失败则在上面的循环中已经记录失败。
		}
	}

	util.DelDirPath(filepath.Join(os.TempDir(), "EasyTools"))

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"successCount": successCount,
		"failCount":    failCount,
		"details":      results,
	})
}

// generateGoDllCode 生成 Go DLL 代码
func generateGoDllCode(dllName string, functions []string) string {
	var exportFuncs []string
	for _, fn := range functions {
		exportFuncs = append(exportFuncs, fmt.Sprintf(`//export %s
func %s() C.int {
	message := C.CString("%s called")
	title := C.CString("DLL Hijack Test")
	C.MessageBoxA(nil, message, title, 0)
	C.free(unsafe.Pointer(message))
	C.free(unsafe.Pointer(title))
	return 0
}`, fn, fn, fn))
	}

	return fmt.Sprintf(`package main

/*
#include <windows.h>
*/
import "C"
import (
	"unsafe"
)

%s

func main() {}
`, strings.Join(exportFuncs, "\n\n"))
}

// generateCDllCode 生成 C DLL 代码
func generateCDllCode(dllName string, functions []string) string {
	var exportFuncs []string
	for _, fn := range functions {
		exportFuncs = append(exportFuncs, fmt.Sprintf(`__declspec(dllexport) int %s() {
	MessageBoxA(NULL, "%s called", "DLL Hijack Test", 0);
	return 0;
}`, fn, fn))
	}

	return fmt.Sprintf(`#include <windows.h>

extern "C" {

%s
}
`, strings.Join(exportFuncs, "\n\n"))
}

// generateRandomSuffix 生成随机后缀
func generateRandomSuffix() string {
	// 使用时间戳和进程ID生成唯一后缀
	return fmt.Sprintf("%d_%d", time.Now().UnixNano(), os.Getpid())
}

// sanitizeDllName 校验 DLL 文件名：只允许普通文件名 + .dll 后缀（防路径穿越/任意路径写入）。
func sanitizeDllName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return ""
	}
	if filepath.Base(name) != name || strings.ContainsAny(name, `/\`) {
		return ""
	}
	if !strings.HasSuffix(strings.ToLower(name), ".dll") {
		return ""
	}
	if len(name) > 128 {
		return ""
	}
	return name
}

// sanitizeFuncName 校验导出函数名：只允许合法标识符（防生成源码注入）。
func sanitizeFuncName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if len(name) > 64 {
		return ""
	}
	for i, r := range name {
		isLetter := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_'
		isDigit := r >= '0' && r <= '9'
		if i == 0 && !isLetter {
			return ""
		}
		if !isLetter && !isDigit {
			return ""
		}
	}
	return name
}
