//go:build linux || darwin

package controller

import (
	"bufio"
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var ansiRegex = regexp.MustCompile(`\x1B\[[0-9;]*[mKJH]`)

// 定义请求结构体来匹配前端传递的对象
type OpenFolderRequest struct {
	Path string `json:"path"`
}

type UnWxapp struct {
	Base
}

type OpenFolderInput struct {
	Path string `json:"path"`
}

func NewUnWxapp() *UnWxapp {
	return &UnWxapp{}
}

func (u *UnWxapp) InitCheck() bool {
	// 1. 验证node环境
	if _, err := exec.LookPath("node"); err != nil {
		log.Println("Node.js未安装或不在PATH中")
		return false
	}
	return true
}

func (u *UnWxapp) RunUnWxapp(_ struct{}, packages []string, appid string, format bool) (string, error) {
	log.Printf("反编译请求参数 - packages: %v, appid: %s, format: %t", packages, appid, format)

	baseDir := u.getAppPath()

	// 构建工具目录路径（相对于项目根目录）
	unwxappDir := filepath.Join(baseDir, "tools", "Unwxapp")
	nodeModulesPath := filepath.Join(unwxappDir, "node_modules")

	// 验证目录存在
	if _, err := os.Stat(unwxappDir); os.IsNotExist(err) {
		log.Printf("工具目录不存在: %s", unwxappDir)
		return "", fmt.Errorf("工具目录不存在")
	}

	// 2. 验证脚本路径
	scriptName := "index.js"
	fullScriptPath := filepath.Join(unwxappDir, scriptName)
	if _, err := os.Stat(fullScriptPath); os.IsNotExist(err) {
		log.Printf("脚本文件不存在: %s", fullScriptPath)
		return "", fmt.Errorf("脚本文件不存在")
	}

	args := []string{scriptName, "wx"}
	args = append(args, packages...)

	if appid != "" {
		args = append(args, "-i", appid)
	}
	if format {
		args = append(args, "-f")
	}

	log.Println("执行命令: node", strings.Join(args, " "))
	log.Println("工具目录:", unwxappDir)
	log.Println("Node模块路径:", nodeModulesPath)

	// 3. 执行命令并捕获输出
	cmd := exec.Command("node", args...)
	cmd.Dir = unwxappDir // 设置工作目录到工具目录

	// 设置环境变量 - 使用绝对路径
	cmd.Env = append(
		os.Environ(),
		"NODE_PATH="+nodeModulesPath,
	)

	// 创建管道捕获实时输出
	stdoutPipe, _ := cmd.StdoutPipe()
	stderrPipe, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		log.Println("命令启动失败:", err)
		return "", fmt.Errorf("命令启动失败")
	}

	// 实时读取输出
	var outputBuilder strings.Builder
	scanner := bufio.NewScanner(io.MultiReader(stdoutPipe, stderrPipe))
	for scanner.Scan() {
		line := scanner.Text()
		log.Println("NODE>", line)
		outputBuilder.WriteString(line + "\n")
	}

	// 等待命令完成
	if err := cmd.Wait(); err != nil {
		log.Println("命令执行失败:", err)
		return outputBuilder.String(), fmt.Errorf("命令执行失败: %v", err)
	}

	output := outputBuilder.String()
	cleanOutput := stripANSI(output)
	log.Println("命令执行成功，输出长度:", len(cleanOutput))
	return cleanOutput, nil
}

func (u *UnWxapp) SelectDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(u.ctx, runtime.OpenDialogOptions{
		Title: "选择解包目录",
	})
}

func stripANSI(input string) string {
	return ansiRegex.ReplaceAllString(input, "")
}
