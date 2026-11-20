package util

import (
	shellutil "EasyTools/app/bypass/shell/util"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func BuildLoaderFile(goFileName, exeFileName string) error {
	// 获取临时目录中的工作目录
	workDir := filepath.Join(os.TempDir(), "EasyTools", "file")

	// 检查工作目录中是否存在 go.mod
	goModPath := filepath.Join(workDir, "go.mod")
	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		return fmt.Errorf("工作目录中缺少 go.mod 文件: %s", workDir)
	}

	// 检查 Go 文件是否存在
	if _, err := os.Stat(goFileName); os.IsNotExist(err) {
		return fmt.Errorf("Go 文件不存在: %s", goFileName)
	}

	// 确保输出目录存在
	outputDir := filepath.Dir(exeFileName)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %s", err)
	}

	// 获取 Go 文件相对于工作目录的相对路径
	relGoPath, err := filepath.Rel(workDir, goFileName)
	if err != nil {
		return fmt.Errorf("获取相对路径失败: %s", err)
	}

	fmt.Printf("工作目录: %s\n", workDir)
	fmt.Printf("Go 文件: %s\n", goFileName)
	fmt.Printf("相对路径: %s\n", relGoPath)
	fmt.Printf("输出文件: %s\n", exeFileName)

	// 构建命令 - 使用相对路径
	cmd := exec.Command("go", "build", "-o", exeFileName, "-ldflags", "-w -s", "-trimpath", relGoPath)
	cmd.Dir = workDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = hideWindowAttr()

	// 执行编译
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("编译失败: %s", err)
	}

	// 检查生成的可执行文件
	if _, err := os.Stat(exeFileName); os.IsNotExist(err) {
		return fmt.Errorf("编译成功但未生成可执行文件: %s", exeFileName)
	}

	// 可选：去除控制台窗口，直接调用内部函数
	if err := shellutil.CloseWindowsConsole(exeFileName); err != nil {
		fmt.Printf("[提示] 去除控制台窗口失败: %v\n", err)
	}

	return nil
}
