package util

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed bai.exe
//go:embed zhengshu.exe
var assets embed.FS

// ExtractBaiJiaHeiFile 解压白加黑白文件
func ExtractBaiJiaHeiFile(targetDir string) error {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	targetPath := filepath.Join(targetDir, "bai.exe")
	baiJiaHeiFile, err := assets.ReadFile("bai.exe")
	if err != nil {
		return fmt.Errorf("读取内嵌白文件失败: %w", err)
	}
	if err := os.WriteFile(targetPath, baiJiaHeiFile, 0755); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}
	fmt.Printf("文件已解压到: %s\n", targetPath)
	return nil
}

// GetEmbeddedDonorCert 返回内嵌 donor 证书 PE 字节
func GetEmbeddedDonorCert() ([]byte, error) {
	data, err := assets.ReadFile("zhengshu.exe")
	if err != nil {
		return nil, fmt.Errorf("内嵌 donor 证书不存在: %w", err)
	}
	return data, nil
}
