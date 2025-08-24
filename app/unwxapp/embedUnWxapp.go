package unwxapp

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// 使用最全面的嵌入模式
//
//go:embed all:node_modules
//go:embed index.js parser-928e23b1.js traverse-252284fd.js
var assets embed.FS

// 获取应用基础目录
func getAppBaseDir() string {
	// 如果是 macOS，使用应用支持目录
	if runtime.GOOS == "darwin" {
		appName := "EasyTools"
		homeDir, err := os.UserHomeDir()
		if err != nil {
			panic("获取用户主目录失败: " + err.Error())
		}
		return filepath.Join(homeDir, "Library", "Application Support", appName)
	}

	// 其他系统使用当前目录下的 EasyToolsFiles
	currentPath, err := os.Getwd()
	if err != nil {
		panic("获取当前路径失败: " + err.Error())
	}
	return filepath.Join(currentPath, "EasyToolsFiles")
}

// 提取并写入文件到目标目录（保留目录结构）
func ExtractResource(resourcePath, targetDir string) (string, error) {
	// 标准化路径分隔符为 /
	resourcePath = strings.ReplaceAll(resourcePath, "\\", "/")

	// 构建完整输出路径
	outputPath := filepath.Join(targetDir, resourcePath)

	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	// 读取嵌入的资源文件
	data, err := assets.ReadFile(resourcePath)
	if err != nil {
		return "", fmt.Errorf("无法读取资源文件 %s: %w", resourcePath, err)
	}

	// 写入到目标路径
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return "", fmt.Errorf("无法写入文件 %s: %w", outputPath, err)
	}

	return outputPath, nil
}

// 解压所有嵌入资源
func ExtractAllResources() error {
	targetDir := filepath.Join(getAppBaseDir(), "tools", "Unwxapp")

	// 创建目标根目录
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	// 打印嵌入文件列表用于调试
	//fmt.Println("嵌入文件列表:")
	err := fs.WalkDir(assets, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && path != "." {
			//fmt.Printf("  - %s\n", path)
		}
		return nil
	})
	if err != nil {
		fmt.Printf("遍历嵌入文件错误: %v\n", err)
	}

	// 提取所有嵌入文件
	var fileCount int
	err = fs.WalkDir(assets, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录和根目录本身
		if d.IsDir() || path == "." {
			return nil
		}

		// 修复路径格式
		relPath := strings.TrimPrefix(path, "./")
		if relPath == "" {
			return nil
		}

		// 提取文件
		_, err = ExtractResource(relPath, targetDir)
		if err != nil {
			// 尝试创建符号链接作为后备方案
			if strings.Contains(err.Error(), "cannot find") {
				return tryCreateSymlink(relPath, targetDir)
			}
			return fmt.Errorf("处理文件 %s 失败: %w", path, err)
		}

		// 记录提取的文件
		fileCount++
		//fmt.Printf("已提取: %s -> %s\n", path, output)
		return nil
	})

	if err != nil {
		return fmt.Errorf("遍历嵌入文件失败: %w", err)
	}

	fmt.Printf("成功提取 %d 个文件到: %s\n", fileCount, targetDir)

	return nil
}

// 尝试创建符号链接
func tryCreateSymlink(relPath, targetDir string) error {
	fullPath := filepath.Join(targetDir, relPath)
	dir := filepath.Dir(fullPath)
	filename := filepath.Base(fullPath)

	// 处理以下划线开头的特殊文件
	if strings.HasPrefix(filename, "_") {
		// 尝试查找不带下划线的版本
		normalName := strings.TrimPrefix(filename, "_")
		normalPath := filepath.Join(dir, normalName)

		if _, err := os.Stat(normalPath); err == nil {
			// 创建符号链接
			if err := os.Symlink(normalName, fullPath); err == nil {
				//fmt.Printf("已创建符号链接: %s -> %s\n", relPath, normalName)
				return nil
			}
		}
	}

	return fmt.Errorf("无法创建符号链接: %s", relPath)
}
