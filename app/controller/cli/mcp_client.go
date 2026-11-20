package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// ---------------------------
// HTTP handlers for frontend
// ---------------------------

// MCP 工具列表：GET /cli/mcp/tools
// 直接返回主进程中已加载的工具列表
func mcpListToolsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !IsMCPEndpointEnabled() {
		writeMCPEndpointDisabled(w)
		return
	}

	// 确保工具配置已加载
	toolsMutex.RLock()
	toolCount := len(tools)
	toolsMutex.RUnlock()

	if toolCount == 0 {
		loadToolsConfig()
	}

	// 直接从主进程的工具列表构建 MCP 格式的工具列表
	mcpTools := make([]map[string]interface{}, 0)

	toolsMutex.RLock()
	currentTools := make([]ToolConfig, len(tools))
	copy(currentTools, tools)
	toolsMutex.RUnlock()

	for _, t := range currentTools {
		mcpTools = append(mcpTools, map[string]interface{}{
			"name":         normalizeToolName(t.ID),
			"description":  t.Description,
			"input_schema": buildToolSchema(t),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tools": mcpTools,
	})
}

type mcpCallHTTPReq struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
	Stream    bool                   `json:"stream,omitempty"`
}

// MCP 调用（一次性）：POST /cli/mcp/call
// 直接在主进程中执行，无需启动子进程
func mcpCallToolHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !IsMCPEndpointEnabled() {
		writeMCPEndpointDisabled(w)
		return
	}

	var req mcpCallHTTPReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if req.Arguments == nil {
		req.Arguments = map[string]interface{}{}
	}

	// 查找工具配置
	toolCfg, ok := findToolByName(req.Name)
	if !ok {
		http.Error(w, fmt.Sprintf("tool not found: %s", req.Name), http.StatusBadRequest)
		return
	}

	// 直接在主进程中执行工具
	output, err := executeTool(toolCfg, req.Arguments)
	if err != nil {
		http.Error(w, "Execute tool failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"content": []map[string]string{
			{
				"type": "text",
				"text": output,
			},
		},
	})
}

// MCP 调用（实时流 SSE）：POST /cli/mcp/call/stream
// 直接在主进程中执行并通过 SSE 推送输出
func mcpCallToolStreamHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !IsMCPEndpointEnabled() {
		writeMCPEndpointDisabled(w)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	var req mcpCallHTTPReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if req.Arguments == nil {
		req.Arguments = map[string]interface{}{}
	}

	// SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Minute)
	defer cancel()

	// 发送开始事件
	sseEvent(w, "start", "streaming started")
	flusher.Flush()

	// 查找工具配置
	toolCfg, ok := findToolByName(req.Name)
	if !ok {
		sseEvent(w, "error", fmt.Sprintf("tool not found: %s", req.Name))
		flusher.Flush()
		return
	}

	// 直接在主进程中执行工具并流式输出
	output, err := executeToolStreamSSE(ctx, toolCfg, req.Arguments, w, flusher)
	if err != nil {
		sseEvent(w, "error", err.Error())
		flusher.Flush()
		return
	}

	// 发送最终结果
	result := map[string]interface{}{
		"content": []map[string]string{
			{
				"type": "text",
				"text": output,
			},
		},
	}
	b, _ := json.Marshal(result)
	sseEvent(w, "done", string(b))
	flusher.Flush()
}

func sseEvent(w http.ResponseWriter, event string, data string) {
	data = strings.ReplaceAll(data, "\r\n", "\n")
	lines := strings.Split(data, "\n")
	fmt.Fprintf(w, "event: %s\n", event)
	for _, ln := range lines {
		fmt.Fprintf(w, "data: %s\n", ln)
	}
	fmt.Fprint(w, "\n")
}

// executeToolStreamSSE 执行工具并通过 SSE 推送流式输出
func executeToolStreamSSE(ctx context.Context, tool ToolConfig, args map[string]interface{}, w http.ResponseWriter, flusher http.Flusher) (string, error) {
	req := ExecutionRequest{
		ToolID:     tool.ID,
		Command:    tool.Command,
		Params:     args,
		WorkingDir: tool.WorkingDir,
		Encoding:   "utf-8",
		ToolParams: tool.Parameters,
		Terminal:   0,
	}

	cmdArgs := buildCommandArgs(req.Params, req.ToolParams)
	return executeCommandStreamSSEImpl(ctx, req.Command, cmdArgs, req.WorkingDir, w, flusher)
}

// executeCommandStreamSSEImpl 执行命令并通过 SSE 推送流式输出
func executeCommandStreamSSEImpl(ctx context.Context, command string, args []string, workingDir string, w http.ResponseWriter, flusher http.Flusher) (string, error) {
	var cmd *exec.Cmd
	useShell := needsShellExecution(command, args, workingDir)

	if useShell {
		switch runtime.GOOS {
		case "windows":
			full := command + " " + strings.Join(args, " ")
			cmd = exec.CommandContext(ctx, "cmd", "/C", full)
		default:
			full := command + " " + strings.Join(args, " ")
			cmd = exec.CommandContext(ctx, "/bin/sh", "-c", full)
		}
	} else {
		cmd = exec.CommandContext(ctx, command, args...)
	}

	setupCommandEncoding(cmd, "utf-8")

	if workingDir != "" {
		cmd.Dir = workingDir
	} else {
		cmd.Dir, _ = os.Getwd()
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("stdout pipe error: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("stderr pipe error: %v", err)
	}

	var outputBuf strings.Builder

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start command failed: %v", err)
	}

	// 读取管道并通过 SSE 推送
	readPipeSSE := func(rc io.ReadCloser, prefix string) error {
		scanner := bufio.NewScanner(rc)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if prefix != "" {
				line = prefix + line
			}
			processed := processLineEncoding(line, "utf-8")
			outputBuf.WriteString(processed + "\n")
			sseEvent(w, "chunk", processed+"\n")
			flusher.Flush()
		}
		return scanner.Err()
	}

	// 并发读取 stdout 和 stderr
	errCh := make(chan error, 2)
	go func() {
		errCh <- readPipeSSE(stdout, "")
	}()
	go func() {
		errCh <- readPipeSSE(stderr, "[STDERR] ")
	}()

	// 等待两个管道读取完成
	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil {
			// 记录错误但继续等待另一个管道
		}
	}

	if err := cmd.Wait(); err != nil {
		sseEvent(w, "chunk", "[command exit with error]\n")
		flusher.Flush()
		return outputBuf.String(), err
	}

	sseEvent(w, "chunk", "[command completed]\n")
	flusher.Flush()
	return outputBuf.String(), nil
}
