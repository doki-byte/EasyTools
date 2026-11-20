package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"EasyTools/app/controller/system"
)

// MCP 消息结构（严格按照 MCP 协议）
type mcpRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type mcpResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *mcpError   `json:"error,omitempty"`
}

type mcpError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type mcpTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"` // MCP 协议使用驼峰命名
}

type mcpCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

var useNDJSONFormat bool // 全局变量，记录是否使用 NDJSON 格式

// StartMCPServerStdio 启动 MCP stdio 服务器（严格按照 MCP 协议）
func StartMCPServerStdio() error {
	// 加载工具配置
	if len(tools) == 0 {
		loadToolsConfig()
	}
	log.Printf("MCP server ready with %d tools", len(tools))

	// 检查 stdin 状态
	log.Printf("Stdin check: Fd=%v, nil=%v", os.Stdin.Fd(), os.Stdin == nil)

	reader := bufio.NewReader(os.Stdin)
	log.Printf("Starting to read from stdin...")

	// 主循环：读取并处理消息
	for {
		log.Printf("Waiting for message from stdin...")
		// 读取消息（阻塞等待）
		body, err := readFramedMessage(reader)
		if err != nil {
			if err == io.EOF {
				log.Printf("EOF received, exiting")
				return nil
			}
			log.Printf("Read error: %v", err)
			// 对于读取错误，发送错误响应后继续
			if len(body) > 0 {
				// 尝试解析请求 ID
				var req mcpRequest
				if json.Unmarshal(body, &req) == nil && req.ID != nil {
					sendError(req.ID, -32700, "Read error", map[string]interface{}{
						"error": err.Error(),
					})
				}
			}
			// 继续等待下一个消息
			continue
		}

		log.Printf("Received message, length=%d", len(body))

		// 解析请求
		var req mcpRequest
		if err := json.Unmarshal(body, &req); err != nil {
			log.Printf("Parse error: %v, body: %s", err, string(body))
			// 尝试提取 ID（如果可能）
			var tempReq mcpRequest
			var reqID interface{}
			if json.Unmarshal(body, &tempReq) == nil {
				reqID = tempReq.ID
			}
			sendError(reqID, -32700, "Parse error", map[string]interface{}{
				"originalError": err.Error(),
				"body":          string(body),
			})
			continue
		}

		log.Printf("Received request: method=%s, id=%v", req.Method, req.ID)

		// 处理请求
		handleRequest(&req)

		log.Printf("Handled request: method=%s", req.Method)
	}
}

// readFramedMessage 读取消息（支持两种格式）
// 1. 标准 MCP stdio 格式：Content-Length: <数字>\r\n\r\n<消息体>
// 2. NDJSON 格式（兼容部分 MCP 客户端）：每行一个 JSON 对象
func readFramedMessage(reader *bufio.Reader) ([]byte, error) {
	log.Printf("Reading message from stdin...")

	// 先 peek 第一行，判断消息格式
	peekBytes, err := reader.Peek(1)
	if err != nil {
		if err == io.EOF {
			return nil, io.EOF
		}
		log.Printf("Error peeking first byte: %v", err)
		return nil, err
	}

	// 如果第一个字符是 '{'，说明是直接发送的 JSON（NDJSON 格式）
	if peekBytes[0] == '{' {
		log.Printf("Detected NDJSON format (direct JSON), reading line...")
		useNDJSONFormat = true // 设置全局标志
		// 读取一行 JSON
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF && len(line) > 0 {
				// EOF 但已有数据，返回数据
				return []byte(strings.TrimRight(line, "\r\n")), nil
			}
			log.Printf("Error reading JSON line: %v", err)
			return nil, err
		}
		// 移除行尾的 \r\n 或 \n
		line = strings.TrimRight(line, "\r\n")
		log.Printf("Read JSON line, length=%d", len(line))
		return []byte(line), nil
	}

	// 检测到 Content-Length 格式
	useNDJSONFormat = false

	// 否则按标准 MCP stdio 格式读取（Content-Length 格式）
	log.Printf("Detected Content-Length format, reading header...")
	var length int
	headerLines := []string{}

	// 读取头部，直到遇到空行（\r\n\r\n）
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Error reading header line: %v", err)
			return nil, err
		}
		// 移除 \r\n 或 \n
		line = strings.TrimRight(line, "\r\n")
		headerLines = append(headerLines, line)
		log.Printf("Read header line: %q", line)

		// 空行表示 header 结束（\r\n\r\n 中的第二个 \r\n）
		if line == "" {
			break
		}

		// 解析 Content-Length 头部（不区分大小写）
		if strings.HasPrefix(strings.ToLower(line), "content-length:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				if n, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
					length = n
					log.Printf("Found Content-Length: %d", length)
				} else {
					log.Printf("Failed to parse Content-Length: %v", err)
				}
			}
		}
	}

	if length <= 0 {
		log.Printf("Invalid content length: %d, headers: %v", length, headerLines)
		return nil, errors.New("invalid content length")
	}

	log.Printf("Reading %d bytes of message body...", length)
	buf := make([]byte, length)
	if _, err := io.ReadFull(reader, buf); err != nil {
		log.Printf("Error reading message body: %v", err)
		return nil, err
	}
	log.Printf("Successfully read %d bytes", length)
	return buf, nil
}

// sendMessage 发送消息（根据客户端格式自动选择）
func sendMessage(data []byte) error {
	if useNDJSONFormat {
		// 使用 NDJSON 格式：直接输出 JSON + \n
		// 必须确保换行符被正确发送
		output := append(data, '\n')
		if _, err := os.Stdout.Write(output); err != nil {
			return err
		}
		// 强制刷新，确保立即发送
		if err := os.Stdout.Sync(); err != nil {
			return err
		}
		return nil
	}

	// 使用标准 Content-Length 格式
	// 格式：Content-Length: <数字>\r\n\r\n<消息体>
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	if _, err := os.Stdout.WriteString(header); err != nil {
		return err
	}
	if _, err := os.Stdout.Write(data); err != nil {
		return err
	}
	return os.Stdout.Sync()
}

// sendResponse 发送响应
func sendResponse(id interface{}, result interface{}) {
	resp := mcpResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	data, err := json.Marshal(resp)
	if err != nil {
		// 在日志中记录，但不输出到 stdout（避免干扰协议）
		log.Printf("Marshal error: %v", err)
		return
	}
	log.Printf("Sending response: id=%v, length=%d", id, len(data))
	if err := sendMessage(data); err != nil {
		log.Printf("Send error: %v", err)
	} else {
		log.Printf("Response sent successfully")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// sendError 发送错误响应（完整的 JSON-RPC 错误格式）
func sendError(id interface{}, code int, msg string, errorData ...interface{}) {
	error := &mcpError{
		Code:    code,
		Message: msg,
	}
	if len(errorData) > 0 && errorData[0] != nil {
		error.Data = errorData[0]
	}
	resp := mcpResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   error,
	}
	data, err := json.Marshal(resp)
	if err != nil {
		// 在日志中记录，但不输出到 stdout（避免干扰协议）
		log.Printf("Marshal error: %v", err)
		return
	}
	if err := sendMessage(data); err != nil {
		log.Printf("Send error: %v", err)
	}
}

// handleRequest 处理请求（严格按照 MCP 协议）
func handleRequest(req *mcpRequest) {
	switch req.Method {
	case "initialize":
		handleInitialize(req)
	case "initialized":
		// 通知消息，无需响应
		log.Printf("Received initialized notification")
	case "notifications/initialized":
		// 部分客户端使用 notifications/initialized 格式
		log.Printf("Received notifications/initialized notification")
	case "ping":
		handlePing(req)
	case "tools/list":
		handleToolsList(req)
	case "tools/call":
		handleToolsCall(req)
	case "shutdown":
		handleShutdown(req)
	case "exit":
		handleExit(req)
	default:
		// 未知方法
		if req.ID != nil {
			sendError(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
		}
	}
}

// handleInitialize 处理 initialize 请求（严格按照 MCP 协议格式）
func handleInitialize(req *mcpRequest) {
	log.Printf("Handling initialize request, id=%v", req.ID)

	// 解析客户端请求的协议版本
	var initParams map[string]interface{}
	clientProtocolVersion := "2024-11-05" // 默认版本
	if req.Params != nil {
		if err := json.Unmarshal(req.Params, &initParams); err == nil {
			if pv, ok := initParams["protocolVersion"].(string); ok && pv != "" {
				clientProtocolVersion = pv
				log.Printf("Client requested protocol version: %s", clientProtocolVersion)
			}
		}
	}

	// 返回客户端请求的协议版本（如果支持）或默认版本
	// 支持 2024-11-05 和 2025-11-25
	result := map[string]interface{}{
		"protocolVersion": clientProtocolVersion,
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{
				"list": true,
				"call": true,
			},
		},
		"serverInfo": map[string]interface{}{
			"name":    "easytools",
			"version": "1.0.0",
		},
	}
	sendResponse(req.ID, result)
	log.Printf("Sent initialize response with protocol version: %s", clientProtocolVersion)
}

// handlePing 处理 ping 请求（心跳机制）
func handlePing(req *mcpRequest) {
	// ping 方法用于保持连接，返回空结果
	sendResponse(req.ID, nil)
}

// handleShutdown 处理 shutdown 请求
func handleShutdown(req *mcpRequest) {
	// 发送响应确认
	sendResponse(req.ID, nil)
	log.Printf("Received shutdown request, exiting...")
	// 注意：shutdown 后应该等待 exit 通知再退出
	// 这里先记录，实际退出由 handleExit 处理
}

// handleExit 处理 exit 请求
func handleExit(req *mcpRequest) {
	// exit 是通知消息，无需响应
	log.Printf("Received exit notification, terminating...")
	// 优雅退出
	os.Exit(0)
}

// handleToolsList 处理 tools/list 请求
func handleToolsList(req *mcpRequest) {
	mcpTools := []mcpTool{}

	// 添加所有 CLI 工具
	toolsMutex.RLock()
	for _, tool := range tools {
		// 优先使用 Name，如果 Name 为空或无效，则使用 ID
		toolName := tool.Name
		if strings.TrimSpace(toolName) == "" {
			toolName = tool.ID
		}

		// 构建增强的描述，包含命令、工作路径等信息
		enhancedDescription := buildEnhancedDescription(tool)

		mcpTools = append(mcpTools, mcpTool{
			Name:        normalizeToolName(toolName),
			Description: enhancedDescription,
			InputSchema: buildToolSchema(tool),
		})
	}
	toolsMutex.RUnlock()

	result := map[string]interface{}{
		"tools": mcpTools,
	}
	sendResponse(req.ID, result)
}

// handleToolsCall 处理 tools/call 请求
func handleToolsCall(req *mcpRequest) {
	var params mcpCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		sendError(req.ID, -32602, "Invalid params", map[string]interface{}{
			"originalError": err.Error(),
		})
		return
	}

	// 查找工具
	toolCfg, ok := findToolByName(params.Name)
	if !ok {
		sendError(req.ID, -32001, "Tool not found", map[string]interface{}{
			"toolName": params.Name,
		})
		return
	}

	// 执行工具（带超时控制）
	output, err := executeTool(toolCfg, params.Arguments)
	if err != nil {
		sendError(req.ID, -32002, "Tool execution failed", map[string]interface{}{
			"toolName": params.Name,
			"error":    err.Error(),
		})
		return
	}

	// 返回结果（严格按照 MCP 协议格式）
	result := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": output,
			},
		},
	}
	sendResponse(req.ID, result)
}

// executeTool 执行工具（带超时控制）
func executeTool(tool ToolConfig, args map[string]interface{}) (string, error) {
	// 将参数名从 label（规范化后）映射回参数 ID
	mappedArgs := map[string]interface{}{}
	for paramName, paramValue := range args {
		// 查找对应的参数
		found := false
		for _, p := range tool.Parameters {
			normalizedLabel := normalizeToolName(p.Label)
			if paramName == normalizedLabel || paramName == p.ID {
				mappedArgs[p.ID] = paramValue
				found = true
				break
			}
		}
		// 如果没找到匹配的参数，直接使用原参数名（向后兼容）
		if !found {
			mappedArgs[paramName] = paramValue
		}
	}

	req := ExecutionRequest{
		ToolID:     tool.ID,
		Command:    tool.Command,
		Params:     mappedArgs,
		WorkingDir: tool.WorkingDir,
		Encoding:   "utf-8",
		ToolParams: tool.Parameters,
		Terminal:   0,
	}

	cmdArgs := buildCommandArgs(req.Params, req.ToolParams)
	// MCP 工具执行超时：默认 5 分钟，可通过环境变量配置
	timeout := 5 * time.Minute
	if timeoutStr := os.Getenv("MCP_TOOL_TIMEOUT"); timeoutStr != "" {
		if parsed, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = parsed
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var cmd *exec.Cmd
	useShell := needsShellExecution(req.Command, cmdArgs, req.WorkingDir)

	if useShell {
		switch runtime.GOOS {
		case "windows":
			cmd = exec.CommandContext(ctx, "cmd", "/C", req.Command+" "+strings.Join(cmdArgs, " "))
		default:
			cmd = exec.CommandContext(ctx, "/bin/sh", "-c", req.Command+" "+strings.Join(cmdArgs, " "))
		}
	} else {
		cmd = exec.CommandContext(ctx, req.Command, cmdArgs...)
	}

	setupCommandEncoding(cmd, req.Encoding)

	if req.WorkingDir != "" {
		cmd.Dir = req.WorkingDir
	} else {
		cmd.Dir, _ = os.Getwd()
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	cmd.Stdin = bytes.NewReader(nil)

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("start failed: %v", err)
	}

	if err := cmd.Wait(); err != nil {
		out := strings.TrimSpace(stdoutBuf.String())
		errOut := strings.TrimSpace(stderrBuf.String())
		if errOut != "" {
			return out, fmt.Errorf("%v: %s", err, errOut)
		}
		return out, err
	}

	output := stdoutBuf.String()
	if output == "" {
		output = stderrBuf.String()
	}
	return output, nil
}

// buildEnhancedDescription 构建增强的工具描述，包含命令、工作路径等信息
func buildEnhancedDescription(t ToolConfig) string {
	var parts []string

	// 添加原始描述
	if strings.TrimSpace(t.Description) != "" {
		parts = append(parts, t.Description)
	}

	// 添加执行信息部分
	var execInfo []string
	if t.Command != "" {
		execInfo = append(execInfo, fmt.Sprintf("Command: %s", t.Command))
	}
	if t.WorkingDir != "" {
		execInfo = append(execInfo, fmt.Sprintf("Working directory: %s", t.WorkingDir))
	}
	if len(execInfo) > 0 {
		parts = append(parts, "")
		parts = append(parts, "Execution details:")
		parts = append(parts, execInfo...)
	}

	// 添加参数说明
	if len(t.Parameters) > 0 {
		parts = append(parts, "")
		parts = append(parts, "Parameters:")
		for _, p := range t.Parameters {
			paramName := normalizeToolName(p.Label)
			if paramName == "" || strings.Trim(paramName, "_") == "" {
				paramName = p.ID
			}

			paramDesc := fmt.Sprintf("  - %s", paramName)
			if p.Label != "" && paramName != p.ID {
				paramDesc += fmt.Sprintf(" (label: %s)", p.Label)
			}
			if p.Name != "" {
				paramDesc += fmt.Sprintf(" (flag: %s)", p.Name)
			}
			if p.Required && p.DefaultValue == nil {
				paramDesc += " [required]"
			} else if p.DefaultValue != nil {
				paramDesc += fmt.Sprintf(" [optional, default: %v]", p.DefaultValue)
			} else {
				paramDesc += " [optional]"
			}
			if p.Description != "" {
				paramDesc += fmt.Sprintf(" - %s", p.Description)
			}
			parts = append(parts, paramDesc)
		}
	}

	if len(parts) == 0 {
		return "MCP tool"
	}

	return strings.Join(parts, "\n")
}

// buildToolSchema 构建工具 schema
// 使用 label（规范化后）作为属性名，使参数更友好
func buildToolSchema(t ToolConfig) map[string]interface{} {
	properties := map[string]interface{}{}
	required := []string{}

	for _, p := range t.Parameters {
		schema := map[string]interface{}{
			"title":       p.Label,
			"description": p.Description,
		}

		switch p.Type {
		case "number":
			schema["type"] = "number"
		case "switch":
			schema["type"] = "boolean"
		default:
			schema["type"] = "string"
		}

		if len(p.Options) > 0 {
			schema["enum"] = p.Options
		}
		if p.DefaultValue != nil {
			schema["default"] = p.DefaultValue
		}

		// 优先使用 label（规范化后）作为属性名，如果 label 为空或不够友好则使用 ID
		propName := normalizeToolName(p.Label)
		if propName == "" || propName == "unnamed_tool" {
			propName = p.ID
		}
		// 如果规范化后的名称全是下划线（说明原 label 全是中文或特殊字符），使用 ID
		if strings.Trim(propName, "_") == "" {
			propName = p.ID
		}

		// 如果参数有默认值，在描述中说明，并提示不需要传递
		if p.DefaultValue != nil {
			defaultDesc := fmt.Sprintf("Has default value: %v. This parameter is optional and should not be provided unless you need to override the default.", p.DefaultValue)
			if desc, ok := schema["description"].(string); ok && desc != "" {
				schema["description"] = fmt.Sprintf("%s. %s", desc, defaultDesc)
			} else {
				schema["description"] = defaultDesc
			}
		}

		properties[propName] = schema
		// 只有没有默认值且 required 的参数才加入 required 列表
		// 有默认值的参数即使 required=true，也可以使用默认值，所以不强制要求传递
		if p.Required && p.DefaultValue == nil {
			required = append(required, propName)
		}
	}

	return map[string]interface{}{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
}

// findToolByName 查找工具
// 支持通过规范化后的名称、原始名称或 ID 查找
func findToolByName(name string) (ToolConfig, bool) {
	toolsMutex.RLock()
	defer toolsMutex.RUnlock()

	// 规范化输入名称
	normalizedName := normalizeToolName(name)

	for _, t := range tools {
		// 优先使用 Name 字段进行匹配
		toolName := t.Name
		if strings.TrimSpace(toolName) == "" {
			toolName = t.ID
		}

		// 匹配规范化后的名称、原始名称或 ID
		if normalizeToolName(toolName) == normalizedName ||
			normalizeToolName(t.ID) == normalizedName ||
			normalizeToolName(t.Name) == normalizedName ||
			t.ID == name ||
			t.Name == name {
			return t, true
		}
	}
	return ToolConfig{}, false
}

// normalizeToolName 规范化工具名称（符合 MCP 规范）
// MCP 工具名称规范：
// - 只能包含小写字母、数字、连字符(-)和下划线(_)
// - 不能以连字符或下划线开头或结尾
// - 不能包含空格或其他特殊字符（包括中文字符）
func normalizeToolName(name string) string {
	// 去除首尾空格
	name = strings.TrimSpace(name)

	// 转换为小写
	name = strings.ToLower(name)

	// 替换不允许的字符为下划线（包括 ASCII 特殊字符）
	replacer := strings.NewReplacer(
		" ", "_",
		"/", "_",
		"\\", "_",
		".", "_",
		":", "_",
		"@", "_",
		"#", "_",
		"$", "_",
		"%", "_",
		"^", "_",
		"&", "_",
		"*", "_",
		"(", "_",
		")", "_",
		"+", "_",
		"=", "_",
		"[", "_",
		"]", "_",
		"{", "_",
		"}", "_",
		"|", "_",
		";", "_",
		"'", "_",
		"\"", "_",
		",", "_",
		"<", "_",
		">", "_",
		"?", "_",
		"!", "_",
		"~", "_",
		"`", "_",
	)
	name = replacer.Replace(name)

	// 处理非 ASCII 字符（包括中文字符），替换为下划线
	var result strings.Builder
	for _, r := range name {
		// 只保留小写字母、数字、连字符和下划线
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			result.WriteRune(r)
		} else {
			// 非 ASCII 字符或其他不允许的字符替换为下划线
			result.WriteRune('_')
		}
	}
	name = result.String()

	// 移除连续的下划线
	for strings.Contains(name, "__") {
		name = strings.ReplaceAll(name, "__", "_")
	}

	// 移除首尾的下划线和连字符
	name = strings.Trim(name, "_-")

	// 如果结果为空，使用默认名称
	if name == "" {
		name = "unnamed_tool"
	}

	// 确保不超过合理长度（MCP 建议不超过 64 字符）
	if len(name) > 64 {
		name = name[:64]
		name = strings.Trim(name, "_-")
	}

	return name
}

// popStreamFlag 从参数中提取并删除 __stream 标志（供 mcp_client.go 使用）
func popStreamFlag(args map[string]interface{}) bool {
	if args == nil {
		return false
	}
	raw, ok := args["__stream"]
	if !ok {
		return false
	}
	delete(args, "__stream")
	if b, ok := raw.(bool); ok {
		return b
	}
	return false
}

// ---------------------------
// HTTP 模式（保留用于前端调用）
// ---------------------------

var (
	mcpHTTPServer     *http.Server
	mcpHTTPServerPort int
	mcpHTTPServerMu   sync.Mutex
)

func StartMCPServerHTTP(port int) (int, error) {
	mcpHTTPServerMu.Lock()
	defer mcpHTTPServerMu.Unlock()

	if mcpHTTPServer != nil {
		mcpHTTPServer.Close()
		mcpHTTPServer = nil
	}

	if port == 0 {
		var err error
		port, err = system.PortManager.GetAvailablePortInRange(57000, 57999)
		if err != nil {
			return 0, fmt.Errorf("无法获取可用端口: %v", err)
		}
	}

	mcpHTTPServerPort = port
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", handleMCPRequest)

	mcpHTTPServer = &http.Server{
		Addr:    fmt.Sprintf("127.0.0.1:%d", port),
		Handler: mux,
	}

	go func() {
		if err := mcpHTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("MCP HTTP 服务器错误: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)
	return port, nil
}

func StopMCPServerHTTP() error {
	mcpHTTPServerMu.Lock()
	defer mcpHTTPServerMu.Unlock()

	if mcpHTTPServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := mcpHTTPServer.Shutdown(ctx)
		mcpHTTPServer = nil
		mcpHTTPServerPort = 0
		return err
	}
	return nil
}

func GetMCPServerPort() int {
	mcpHTTPServerMu.Lock()
	defer mcpHTTPServerMu.Unlock()
	return mcpHTTPServerPort
}

func handleMCPRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	var req mcpRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeHTTPMCPError(w, nil, -32700, "Parse error", map[string]interface{}{
			"originalError": err.Error(),
			"body":          string(body),
		})
		return
	}

	switch req.Method {
	case "initialize":
		writeHTTPMCPResponse(w, req.ID, map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]bool{"list": true, "call": true},
			},
			"serverInfo": map[string]interface{}{
				"name":    "easytools",
				"version": "1.0.0",
			},
		})
	case "tools/list":
		mcpTools := []mcpTool{}
		toolsMutex.RLock()
		for _, t := range tools {
			// 优先使用 Name，如果 Name 为空或无效，则使用 ID
			toolName := t.Name
			if strings.TrimSpace(toolName) == "" {
				toolName = t.ID
			}
			// 构建增强的描述，包含命令、工作路径等信息
			enhancedDescription := buildEnhancedDescription(t)

			mcpTools = append(mcpTools, mcpTool{
				Name:        normalizeToolName(toolName),
				Description: enhancedDescription,
				InputSchema: buildToolSchema(t),
			})
		}
		toolsMutex.RUnlock()
		writeHTTPMCPResponse(w, req.ID, map[string]interface{}{"tools": mcpTools})
	case "tools/call":
		var params mcpCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			writeHTTPMCPError(w, req.ID, -32602, "Invalid params", map[string]interface{}{
				"originalError": err.Error(),
			})
			return
		}
		toolCfg, ok := findToolByName(params.Name)
		if !ok {
			writeHTTPMCPError(w, req.ID, -32001, "Tool not found", map[string]interface{}{
				"toolName": params.Name,
			})
			return
		}
		output, err := executeTool(toolCfg, params.Arguments)
		if err != nil {
			writeHTTPMCPError(w, req.ID, -32002, "Tool execution failed", map[string]interface{}{
				"toolName": params.Name,
				"error":    err.Error(),
			})
			return
		}
		writeHTTPMCPResponse(w, req.ID, map[string]interface{}{
			"content": []map[string]interface{}{
				{"type": "text", "text": output},
			},
		})
	default:
		writeHTTPMCPError(w, req.ID, -32601, "Method not found", map[string]interface{}{
			"method": req.Method,
		})
	}
}

func writeHTTPMCPResponse(w http.ResponseWriter, id interface{}, result interface{}) {
	resp := mcpResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func writeHTTPMCPError(w http.ResponseWriter, id interface{}, code int, msg string, errorData ...interface{}) {
	error := &mcpError{
		Code:    code,
		Message: msg,
	}
	if len(errorData) > 0 && errorData[0] != nil {
		error.Data = errorData[0]
	}
	resp := mcpResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   error,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
