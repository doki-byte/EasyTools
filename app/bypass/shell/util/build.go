package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BuildLoaderFile 编译生成免杀产物。
// CheckGoBuild 内部会并发编译多个随机命名的变体，单个变体失败只打印不返回错误；
// 此处通过"产物输出目录是否新增文件"做兜底校验，避免全部编译失败时上层误报成功。
// 全部失败时返回编译器真实报错（原实现只返回"未生成产物"，GUI 无法定位原因）。
// keepConsole=true 时保留控制台子系统（命令行工具如 mimikatz 需输出结果）。
func BuildLoaderFile(goFileName, exeFileName, sharuan, buildWay, loadWay string, dllName string, skyShadowPayloadHeader string, arch string, targetOS string, keepConsole bool) error {
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("获取当前目录失败: %v", err)
	}
	outputDir := filepath.Join(currentDir, "EasyToolsFiles", "file")
	before := listFileSet(outputDir)

	buildErr := CheckGoBuild(goFileName, exeFileName, sharuan, buildWay, loadWay, dllName, skyShadowPayloadHeader, arch, targetOS, keepConsole)

	after := listFileSet(outputDir)
	if len(after) <= len(before) {
		if buildErr != nil {
			return buildErr
		}
		return fmt.Errorf("编译未生成任何产物，请检查 go/gcc/garble 环境是否完整（或产物被安全软件拦截）")
	}
	if buildErr != nil {
		// 只要有一个变体成功即可交付；失败原因仅记录，便于排查
		fmt.Printf("[!] 部分编译变体失败（已有成功产物）: %v\n", buildErr)
	}
	return nil
}

// listFileSet 返回目录下所有普通文件名集合（小写）
func listFileSet(dir string) map[string]bool {
	set := make(map[string]bool)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return set
	}
	for _, e := range entries {
		if e.Type().IsRegular() {
			set[strings.ToLower(e.Name())] = true
		}
	}
	return set
}
