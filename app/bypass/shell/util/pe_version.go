package util

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math/big"
	"os"
	"strings"
	"unicode/utf16"
)

// SetPEVersionStrings 原生修改 PE 版本信息字符串（跨平台，不依赖 rcedit/ResourceHacker）
func SetPEVersionStrings(exePath string, fields map[string]string) error {
	data, err := os.ReadFile(exePath)
	if err != nil {
		return fmt.Errorf("读取 PE 失败: %w", err)
	}
	if !bytesIsPE(data) {
		return fmt.Errorf("非 PE 文件")
	}

	modified := false
	for key, value := range fields {
		// StringFileInfo 中 Key 为 UTF-16LE
		keyUTF16 := utf16LEBytes(key)
		valUTF16 := utf16LEBytes(value)
		if replaced := replaceUTF16InPE(data, keyUTF16, valUTF16); replaced {
			modified = true
		}
		// 也尝试直接替换 Value 字段（CompanyName 等作为 key 后跟 value）
		if replaced := replaceUTF16ValueAfterKey(data, keyUTF16, valUTF16); replaced {
			modified = true
		}
	}

	if !modified {
		// 若无版本资源，写入 UTF-16 明文占位（部分杀软仍检测）
		for key, value := range fields {
			old := utf16LEBytes(key)
			newVal := utf16LEBytes(value)
			if len(newVal) <= len(old)+64 {
				data = injectUTF16NearMarker(data, []byte("StringFileInfo"), old, newVal)
				modified = true
			}
		}
	}

	if err := os.WriteFile(exePath, data, 0o666); err != nil {
		return err
	}
	return fixPEChecksum(exePath)
}

// RandomizePEVersionInfo 随机化 PE 版本信息（PassQvm 原生实现）
func RandomizePEVersionInfo(exePath string) error {
	rnd := func(n int) string {
		letters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
		out := make([]byte, n)
		for i := range out {
			idx, _ := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
			out[i] = letters[idx.Int64()]
		}
		return string(out)
	}
	ver := func() string {
		a, _ := rand.Int(rand.Reader, big.NewInt(50))
		b, _ := rand.Int(rand.Reader, big.NewInt(50))
		c, _ := rand.Int(rand.Reader, big.NewInt(50))
		d, _ := rand.Int(rand.Reader, big.NewInt(50))
		return fmt.Sprintf("%d.%d.%d.%d", a.Int64(), b.Int64(), c.Int64(), d.Int64())
	}
	s := rnd(8)
	fields := map[string]string{
		"FileDescription":  "Application " + s,
		"CompanyName":      "Software " + rnd(6),
		"LegalCopyright":   "(C) " + rnd(10),
		"ProductName":      rnd(8) + " App",
		"FileVersion":      ver(),
		"ProductVersion":   ver(),
		"OriginalFilename": s + ".exe",
		"InternalName":     s,
	}
	return SetPEVersionStrings(exePath, fields)
}

func utf16LEBytes(s string) []byte {
	u := utf16.Encode([]rune(s))
	b := make([]byte, len(u)*2)
	for i, r := range u {
		binary.LittleEndian.PutUint16(b[i*2:], r)
	}
	return b
}

func bytesIsPE(data []byte) bool {
	if len(data) < 64 || data[0] != 'M' || data[1] != 'Z' {
		return false
	}
	peOff := binary.LittleEndian.Uint32(data[0x3C:])
	return int(peOff)+4 <= len(data) && string(data[peOff:peOff+4]) == "PE\x00\x00"
}

func replaceUTF16InPE(data []byte, keyUTF16, valUTF16 []byte) bool {
	// 在 key 之后寻找 value 并等长替换
	idx := 0
	for {
		pos := bytesIndex(data[idx:], keyUTF16)
		if pos < 0 {
			return false
		}
		abs := idx + pos + len(keyUTF16)
		// 跳过后续 00 填充
		for abs < len(data)-2 && data[abs] == 0 && data[abs+1] == 0 {
			abs += 2
		}
		if abs+len(valUTF16) <= len(data) {
			copy(data[abs:abs+len(valUTF16)], valUTF16)
			// 截断填充
			for j := abs + len(valUTF16); j < abs+len(valUTF16)+20 && j+1 < len(data); j += 2 {
				if data[j] == 0 && data[j+1] == 0 {
					break
				}
			}
			return true
		}
		idx = idx + pos + len(keyUTF16)
	}
}

func replaceUTF16ValueAfterKey(data, keyUTF16, valUTF16 []byte) bool {
	return replaceUTF16InPE(data, keyUTF16, valUTF16)
}

func injectUTF16NearMarker(data, marker, keyUTF16, valUTF16 []byte) []byte {
	pos := bytesIndex(data, marker)
	if pos < 0 {
		return data
	}
	// 在 marker 后搜索可写零区域写入 key+value
	region := pos
	if region+512 > len(data) {
		region = len(data) - 512
	}
	slot := region + 256
	if slot+len(keyUTF16)+len(valUTF16)+4 < len(data) {
		copy(data[slot:], keyUTF16)
		slot += len(keyUTF16)
		data[slot] = 0
		data[slot+1] = 0
		slot += 2
		copy(data[slot:], valUTF16)
	}
	return data
}

func bytesIndex(data, sub []byte) int {
	for i := 0; i+len(sub) <= len(data); i++ {
		if string(data[i:i+len(sub)]) == string(sub) {
			return i
		}
	}
	return -1
}

// ApplyPassQvmNative 原生 PassQvm：随机化版本信息（Mac/Linux/Windows 均可）
func ApplyPassQvmNative(exePath string) error {
	if _, err := os.Stat(exePath); err != nil {
		return fmt.Errorf("文件不存在: %w", err)
	}
	if err := RandomizePEVersionInfo(exePath); err != nil {
		return err
	}
	// 额外混淆 .text 段外的明文 Company/Product 字符串（ASCII 兜底）
	data, err := os.ReadFile(exePath)
	if err != nil {
		return err
	}
	for _, kw := range []string{"Tencent", "Go build", "EasyTools", "garble"} {
		data = scrubASCIIKeywordAll(data, []byte(kw))
	}
	if err := os.WriteFile(exePath, data, 0o666); err != nil {
		return err
	}
	return fixPEChecksum(exePath)
}

func scrubASCIIKeywordAll(data, keyword []byte) []byte {
	for i := 0; i+len(keyword) <= len(data); i++ {
		if strings.EqualFold(string(data[i:i+len(keyword)]), string(keyword)) {
			for j := i; j < i+len(keyword); j++ {
				data[j] = randomAlphaNum()
			}
		}
	}
	return data
}
