package encode

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// 载荷「静态落地」处理。
//
// 历史问题：upload 接口曾把原始载荷（Donut PIC / CS·MSF / EasyShell stageless shellcode）
// 明文写到 %TEMP%\EasyTools\file\payload.bin。这类原始 shellcode 本身就是各大安全软件的
// 静态特征命中项，落盘即被实时监控扫描，轻则文件被隔离/锁定，导致后续生成阶段
// 「打开文件失败」而报"shellcode处理失败"，重则直接触发告警；
// 对免杀工具而言，明文载荷落盘本身也是操作痕迹。
//
// 现在改为：载荷一律先做随机密钥混淆再落盘（payload.bin），密钥单独存放（payload.key），
// 生成阶段读入内存后还原。磁盘上不再出现可被静态识别的原始 shellcode。

const (
	// PayloadBinName 混淆后的载荷文件名（保持旧名，兼容既有流程）。
	PayloadBinName = "payload.bin"
	// PayloadKeyName 载荷混淆密钥文件名；存在该文件即表示 payload.bin 为混淆态。
	PayloadKeyName = "payload.key"
	// payloadAtRestKeyLen 混淆密钥长度。
	payloadAtRestKeyLen = 32
)

// obfuscateAtRest 对载荷做可逆混淆：随机密钥 XOR + 写入位置扰动。
// XOR 自逆，故同一函数既用于加密也用于解密。
func obfuscateAtRest(data, key []byte) []byte {
	out := make([]byte, len(data))
	kl := len(key)
	for i := range data {
		out[i] = data[i] ^ key[i%kl] ^ byte(i>>8)
	}
	return out
}

// ObfuscatePayloadAtRest 生成载荷的混淆态与随机密钥。
func ObfuscatePayloadAtRest(data []byte) (blob []byte, key []byte, err error) {
	key = make([]byte, payloadAtRestKeyLen)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, nil, fmt.Errorf("生成载荷混淆密钥失败: %w", err)
	}
	return obfuscateAtRest(data, key), key, nil
}

// DeobfuscatePayloadAtRest 还原混淆态载荷。
func DeobfuscatePayloadAtRest(blob, key []byte) ([]byte, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("载荷混淆密钥为空")
	}
	return obfuscateAtRest(blob, key), nil
}

// WritePayloadAtRest 将载荷混淆后写入 dir/payload.bin，密钥写入 dir/payload.key。
// 先写载荷再写密钥：密钥文件的存在即代表载荷已完整落盘且为混淆态。
func WritePayloadAtRest(dir string, data []byte) (string, error) {
	blob, key, err := ObfuscatePayloadAtRest(data)
	if err != nil {
		return "", err
	}
	binPath := filepath.Join(dir, PayloadBinName)
	if err := os.WriteFile(binPath, blob, 0644); err != nil {
		return "", fmt.Errorf("保存载荷失败: %w", err)
	}
	keyPath := filepath.Join(dir, PayloadKeyName)
	if err := os.WriteFile(keyPath, key, 0600); err != nil {
		// 密钥写失败则删除载荷，避免留下无法还原的混淆数据
		_ = os.Remove(binPath)
		return "", fmt.Errorf("保存载荷密钥失败: %w", err)
	}
	return binPath, nil
}

// ReadPayloadAtRest 读取载荷：存在同名 payload.key 时按混淆态还原，
// 否则按旧流程当作明文载荷返回（兼容历史遗留的工作目录）。
func ReadPayloadAtRest(binPath string) ([]byte, error) {
	blob, err := OpenBinToByte(binPath)
	if err != nil {
		return nil, err
	}
	keyPath := filepath.Join(filepath.Dir(binPath), PayloadKeyName)
	key, keyErr := os.ReadFile(keyPath)
	if keyErr != nil || len(key) == 0 {
		// 无密钥文件：旧流程遗留的明文载荷
		return blob, nil
	}
	return DeobfuscatePayloadAtRest(blob, key)
}

// ReadPayloadAtRestRetry 带重试读取载荷。
// 安全软件的实时监控会在文件刚落盘时短暂持有句柄，导致瞬时「拒绝访问/文件被占用」，
// 这类失败重试即可恢复，不应直接判定为处理失败。
func ReadPayloadAtRestRetry(binPath string, attempts int, delay time.Duration) ([]byte, error) {
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for i := 0; i < attempts; i++ {
		data, err := ReadPayloadAtRest(binPath)
		if err == nil {
			return data, nil
		}
		lastErr = err
		if i < attempts-1 {
			time.Sleep(delay)
		}
	}
	return nil, lastErr
}
