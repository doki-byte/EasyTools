//go:build darwin || linux

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// 新增：修复执行处理器
func genericExecuteHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExecutionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Command == "" {
		http.Error(w, "Command is required", http.StatusBadRequest)
		return
	}

	// 生成执行ID
	executionID := generateExecutionID()

	// 如果是 web 模式，创建 Web 终端会话
	if req.ExecutionMode == "web" {
		// 构建命令参数
		args := buildCommandArgs(req.Params, req.ToolParams)

		log.Printf("Creating web terminal session: executionID=%s, command=%s", executionID, req.Command)

		// 创建 Web 终端会话
		_, err := createWebTerminalSession(executionID, req.Command, args, req.WorkingDir)
		if err != nil {
			log.Printf("Failed to create web terminal session: %v", err)
			http.Error(w, "Failed to create web terminal session: "+err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("Web terminal session created successfully: executionID=%s", executionID)

		// 返回执行ID和 WebSocket URL
		response := ExecutionResponse{
			ID:        executionID,
			Status:    "started",
			Timestamp: time.Now().Format(time.RFC3339),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	// 构建命令参数
	args := buildCommandArgs(req.Params, req.ToolParams)

	// 创建执行上下文
	ctx, cancel := context.WithCancel(context.Background())

	var cmd *exec.Cmd

	// 检查是否需要使用 shell 执行复合命令
	needsShellExecution := needsShellExecution(req.Command, args, req.WorkingDir)

	if req.Terminal == 1 {
		// 真正的终端模式 - 使用 /K 保持窗口
		cmd = executeWithTerminal(ctx, req.Command, args, req.WorkingDir)
	} else if needsShellExecution {
		// 普通模式但需要shell - 使用 /C 执行后关闭
		cmd = executeWithTerminal(ctx, req.Command, args, req.WorkingDir)
	} else {
		// 无终端模式 - 普通命令执行
		cmd = exec.CommandContext(ctx, req.Command, args...)

		setupCommandEncoding(cmd, req.Encoding)

		// 设置工作目录
		if req.WorkingDir != "" {
			cmd.Dir = req.WorkingDir
		} else {
			// 默认使用当前工作目录
			cmd.Dir, _ = os.Getwd()
		}
	}

	// 创建执行上下文记录
	execCtx := &executionContext{
		id:         executionID,
		cmd:        cmd,
		cancel:     cancel,
		startTime:  time.Now(),
		status:     "running",
		outputChan: make(chan string, 1000),
		terminal:   req.Terminal,
		completed:  false,
		encoding:   req.Encoding,
	}

	// 保存到活跃执行记录
	executionMutex.Lock()
	activeExecutions[executionID] = execCtx
	executionMutex.Unlock()

	// 添加到历史记录
	executionHistory = append(executionHistory, ExecutionStatus{
		ID:        executionID,
		Command:   req.Command,
		Status:    "running",
		StartTime: execCtx.startTime,
	})

	// 限制历史记录大小
	if len(executionHistory) > maxHistorySize {
		executionHistory = executionHistory[len(executionHistory)-maxHistorySize:]
	}

	// 启动命令和输出收集
	if req.Terminal == 1 || needsShellExecution {
		// 终端模式或复合命令 - 使用 shell 执行
		if req.Terminal == 0 {
			// 普通模式下的复合命令，仍然收集输出
			if err := startNonTerminalExecution(executionID, cmd); err != nil {
				handleExecutionError(executionID, err)
				http.Error(w, "Failed to start command: "+err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			// 真正的终端模式 - 不收集输出
			go startTerminalExecution(executionID, cmd)
		}
	} else {
		// 无终端模式 - 正常收集输出
		if err := startNonTerminalExecution(executionID, cmd); err != nil {
			handleExecutionError(executionID, err)
			http.Error(w, "Failed to start command: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// 返回执行ID
	response := ExecutionResponse{
		ID:        executionID,
		Status:    "started",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// 新增：判断是否需要使用 shell 执行
func needsShellExecution(command string, args []string, workingDir string) bool {
	// 如果命令包含路径分隔符或者是相对路径，可能需要工作目录支持
	if strings.Contains(command, "\\") || strings.Contains(command, "/") {
		return true
	}

	// 如果指定了工作目录，并且命令不在系统PATH中，需要使用shell确保正确的工作目录
	if workingDir != "" {
		// 检查命令是否在系统PATH中
		if _, err := exec.LookPath(command); err != nil {
			return true
		}
	}

	// 检查参数中是否包含需要shell解析的特殊字符
	fullCommand := command + " " + strings.Join(args, " ")
	if strings.Contains(fullCommand, "&&") || strings.Contains(fullCommand, "||") ||
		strings.Contains(fullCommand, ">") || strings.Contains(fullCommand, "<") ||
		strings.Contains(fullCommand, "|") {
		return true
	}

	return false
}

// 根据操作系统执行终端命令
func executeWithTerminal(ctx context.Context, command string, args []string, workingDir string) *exec.Cmd {
	switch runtime.GOOS {
	case "windows":
		return executeWindowsTerminal(ctx, command, args, workingDir)
	case "darwin", "linux":
		return executeUnixTerminal(ctx, command, args, workingDir)
	default:
		// 默认使用Unix方式
		return executeUnixTerminal(ctx, command, args, workingDir)
	}
}

// Windows终端执行
func executeWindowsTerminal(ctx context.Context, command string, args []string, workingDir string) *exec.Cmd {
	// 拼接完整命令
	var fullCommand string
	if workingDir != "" {
		fullCommand = fmt.Sprintf("cd /D %s && %s %s", workingDir, command, strings.Join(args, " "))
	} else {
		fullCommand = fmt.Sprintf("%s %s", command, strings.Join(args, " "))
	}

	// 使用 cmd /K 保持终端打开，并在命令执行后暂停
	echoCmd := fmt.Sprintf(`echo "使用命令: %s"`, fullCommand)
	return exec.CommandContext(ctx, "cmd", "/K", echoCmd+"&&"+fullCommand+" && pause")
}

// Unix终端执行 (macOS, Linux)
func executeUnixTerminal(ctx context.Context, command string, args []string, workingDir string) *exec.Cmd {
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		if runtime.GOOS == "darwin" {
			shellPath = "/bin/zsh"
		} else {
			shellPath = "/bin/bash"
		}
	}

	// 拼接完整命令
	var fullCommand string
	if workingDir != "" {
		fullCommand = fmt.Sprintf("cd %s && %s %s", workingDir, command, strings.Join(args, " "))
	} else {
		fullCommand = fmt.Sprintf("%s %s", command, strings.Join(args, " "))
	}

	// 获取终端信息
	terminalApp, terminalArgs := getTerminalInfo(shellPath, filepath.Base(shellPath), fullCommand)
	if terminalApp != "" {
		return exec.CommandContext(ctx, terminalApp, terminalArgs...)
	}

	// 如果没有找到支持的终端，使用默认shell
	return exec.CommandContext(ctx, shellPath, "-c", fullCommand)
}

// 获取终端信息 (Unix系统)
func getTerminalInfo(shellPath, shellName, command string) (string, []string) {
	switch runtime.GOOS {
	case "darwin":
		// macOS
		return "osascript", []string{
			"-e", `tell application "Terminal"`,
			"-e", "activate",
			"-e", fmt.Sprintf(`do script "%s"`, command),
			"-e", "end tell",
		}
	case "linux":
		// Linux - 尝试常见的终端
		terminals := []struct {
			name string
			args []string
		}{
			{"gnome-terminal", []string{"--", shellPath, "-c", command}},
			{"konsole", []string{"-e", shellPath, "-c", command}},
			{"xterm", []string{"-e", shellPath, "-c", command}},
			{"terminator", []string{"-e", shellPath + " -c '" + command + "'"}},
		}

		for _, term := range terminals {
			if _, err := exec.LookPath(term.name); err == nil {
				return term.name, term.args
			}
		}
	}

	return "", nil
}

// 添加编码转换函数
func utf8ToGBK(text string) (string, error) {
	reader := transform.NewReader(bytes.NewReader([]byte(text)), simplifiedchinese.GBK.NewEncoder())
	d, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(d), nil
}

func gbkToUTF8(text string) (string, error) {
	reader := transform.NewReader(bytes.NewReader([]byte(text)), simplifiedchinese.GBK.NewDecoder())
	d, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(d), nil
}

// 修改命令执行环境
func setupCommandEncoding(cmd *exec.Cmd, encoding string) {
	env := os.Environ()

	// 设置编码环境变量
	switch strings.ToLower(encoding) {
	case "gbk", "gb2312":
		env = append(env, "LANG=zh_CN.GBK")
		env = append(env, "LC_ALL=zh_CN.GBK")
	default:
		env = append(env, "LANG=zh_CN.UTF-8")
		env = append(env, "LC_ALL=zh_CN.UTF-8")
	}

	cmd.Env = env
}
