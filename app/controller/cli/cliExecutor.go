package cli

import (
	"EasyTools/app/controller/localguard"
	"EasyTools/app/controller/system"
	extractor "EasyTools/app/embed"
	embedconfig "EasyTools/app/embed/config"
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// 工具配置结构体
type ToolConfig struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Command     string          `json:"command"`
	Description string          `json:"description"`
	WorkingDir  string          `json:"workingDir,omitempty"`
	Parameters  []ToolParameter `json:"parameters"`
	OutputFile  string          `json:"outputFile,omitempty"`
	CreatedAt   string          `json:"createdAt"`
	UpdatedAt   string          `json:"updatedAt"`
}

type ToolParameter struct {
	ID           string      `json:"id"`
	Name         string      `json:"name"`
	Label        string      `json:"label"`
	Type         string      `json:"type"` // input, file, number, select, switch, text
	ArgType      string      `json:"argType"`
	Required     bool        `json:"required"`
	DefaultValue interface{} `json:"defaultValue,omitempty"`
	Options      []string    `json:"options,omitempty"`
	Placeholder  string      `json:"placeholder,omitempty"`
	Description  string      `json:"description,omitempty"`
}

// 修改执行请求结构体，添加终端模式和编码字段
type ExecutionRequest struct {
	ToolID        string                 `json:"toolId"`
	Command       string                 `json:"command"`
	Params        map[string]interface{} `json:"params"`
	WorkingDir    string                 `json:"workingDir,omitempty"`
	Encoding      string                 `json:"encoding,omitempty"`
	ToolParams    []ToolParameter        `json:"toolParams,omitempty"`
	Terminal      int                    `json:"terminal,omitempty"`      // 0: 无终端, 1: 有终端
	ExecutionMode string                 `json:"executionMode,omitempty"` // normal, interactive, web
}

// 执行响应
type ExecutionResponse struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Output    string `json:"output,omitempty"`
	Error     string `json:"error,omitempty"`
	Timestamp string `json:"timestamp"`
}

// 执行状态
type ExecutionStatus struct {
	ID        string    `json:"id"`
	Command   string    `json:"command"`
	Status    string    `json:"status"` // running, completed, failed, cancelled
	StartTime time.Time `json:"startTime"`
	EndTime   time.Time `json:"endTime,omitempty"`
	Output    string    `json:"output,omitempty"`
	Error     string    `json:"error,omitempty"`
}

var (
	executionMutex     sync.Mutex
	activeExecutions   = make(map[string]*executionContext)
	executionHistory   []ExecutionStatus
	maxHistorySize     = 1000
	toolsConfigPath    = filepath.Join(system.GetAppBaseDir(), "config", "tools_config.json")
	toolsMutex         sync.RWMutex
	tools              []ToolConfig
	cliServerPort      int // 保存 CLI 服务器端口号
	cliServerPortMutex sync.RWMutex
)

// GetCLIServerPort 获取 CLI 服务器端口号
func GetCLIServerPort() int {
	cliServerPortMutex.RLock()
	defer cliServerPortMutex.RUnlock()
	return cliServerPort
}

// 修改 executionContext 结构体，添加输出通道、终端模式和编码字段
type executionContext struct {
	id         string
	cmd        *exec.Cmd
	cancel     context.CancelFunc
	output     strings.Builder
	startTime  time.Time
	status     string
	outputChan chan string
	terminal   int // 0: 无终端, 1: 有终端
	completed  bool
	encoding   string // 新增：存储编码信息
}

// 初始化工具配置
func init() {
	// MCP 模式下禁用日志输出（避免干扰 stdio）
	if len(os.Args) >= 2 && os.Args[1] == "mcp" {
		log.SetOutput(io.Discard)
	}
	loadToolsConfig()
}

// 加载工具配置
func loadToolsConfig() {
	toolsMutex.Lock()
	defer toolsMutex.Unlock()

	// 确保默认工具配置文件存在（第一次运行或文件被删除时）
	if err := ensureDefaultToolsConfig(); err != nil {
		log.Printf("Error ensuring tools config: %v", err)
		tools = createDefaultTools()
		return
	}

	data, err := os.ReadFile(toolsConfigPath)
	if err != nil {
		log.Printf("Error reading tools config: %v", err)
		tools = createDefaultTools()
		return
	}

	if len(data) == 0 {
		log.Printf("Tools config file is empty, using embedded defaults")
		if err := resetToolsConfig(); err != nil {
			log.Printf("Error resetting tools config from embedded defaults: %v", err)
			tools = createDefaultTools()
			return
		}
		data, err = os.ReadFile(toolsConfigPath)
		if err != nil {
			log.Printf("Error rereading tools config after reset: %v", err)
			tools = createDefaultTools()
			return
		}
	}

	if err := json.Unmarshal(data, &tools); err != nil {
		log.Printf("Error parsing tools config: %v", err)

		// 尝试使用内置默认配置恢复
		if err := resetToolsConfig(); err != nil {
			log.Printf("Error resetting tools config from embedded defaults: %v", err)
			tools = createDefaultTools()
			return
		}

		data, err = os.ReadFile(toolsConfigPath)
		if err != nil {
			log.Printf("Error rereading tools config after reset: %v", err)
			tools = createDefaultTools()
			return
		}

		if err := json.Unmarshal(data, &tools); err != nil {
			log.Printf("Error parsing embedded tools config: %v", err)
			tools = createDefaultTools()
		}
	}
}

// ensureDefaultToolsConfig 确保本地存在默认的 tools_config.json
func ensureDefaultToolsConfig() error {
	info, err := os.Stat(toolsConfigPath)
	if err == nil && info.Size() > 0 {
		// 文件已存在且非空
		return nil
	}

	// 优先使用解压器释放整个 config 目录（包含 cms_finger.db、工具配置等所有文件）
	extractor.InitExtractor()
	if err := extractor.GetExtractor().ExtractOne("config"); err != nil {
		log.Printf("解压 config 资源失败，将尝试直接写入默认 tools_config.json: %v", err)
	}

	// 如果解压后文件已经存在且非空，直接返回
	info, err = os.Stat(toolsConfigPath)
	if err == nil && info.Size() > 0 {
		return nil
	}

	// 兜底：单独写入内置默认 tools_config.json
	if err := os.MkdirAll(filepath.Dir(toolsConfigPath), 0755); err != nil {
		return fmt.Errorf("创建工具配置目录失败: %v", err)
	}

	data, err := embedconfig.Assets.ReadFile("tools_config.json")
	if err != nil {
		return fmt.Errorf("读取内置工具配置失败: %v", err)
	}

	if err := os.WriteFile(toolsConfigPath, data, 0644); err != nil {
		return fmt.Errorf("写入工具配置失败: %v", err)
	}

	return nil
}

// resetToolsConfig 使用内置默认配置覆盖本地配置（用于文件损坏时的恢复）
func resetToolsConfig() error {
	// 从内置资源中读取默认配置
	data, err := embedconfig.Assets.ReadFile("tools_config.json")
	if err != nil {
		return fmt.Errorf("读取内置工具配置失败: %v", err)
	}

	if err := os.WriteFile(toolsConfigPath, data, 0644); err != nil {
		return fmt.Errorf("写入工具配置失败: %v", err)
	}

	return nil
}

// 创建默认工具配置
func createDefaultTools() []ToolConfig {
	return []ToolConfig{
		{
			ID:          "tool-nmap-scan",
			Name:        "NMAP 扫描",
			Command:     "nmap",
			Description: "网络发现和安全审计工具",
			WorkingDir:  "",
			Parameters: []ToolParameter{
				{
					ID:          "param-target",
					Name:        "",
					Label:       "目标地址",
					Type:        "input",
					ArgType:     "positional",
					Required:    true,
					Placeholder: "192.168.1.1/24",
				},
				{
					ID:           "param-scan-type",
					Name:         "",
					Label:        "扫描类型",
					Type:         "select",
					ArgType:      "flag",
					Required:     false,
					Options:      []string{"-sS", "-sT", "-sU", "-sV"},
					DefaultValue: "-sS",
				},
				{
					ID:          "param-ports",
					Name:        "-p",
					Label:       "端口范围",
					Type:        "input",
					ArgType:     "key-value",
					Required:    false,
					Placeholder: "1-1000",
				},
				{
					ID:           "param-timing",
					Name:         "-T",
					Label:        "时序模板",
					Type:         "select",
					ArgType:      "key-value",
					Required:     false,
					Options:      []string{"1", "2", "3", "4", "5"},
					DefaultValue: "4",
				},
				{
					ID:           "param-verbose",
					Name:         "-v",
					Label:        "详细输出",
					Type:         "switch",
					ArgType:      "flag",
					Required:     false,
					DefaultValue: false,
				},
			},
			CreatedAt: time.Now().Format(time.RFC3339),
			UpdatedAt: time.Now().Format(time.RFC3339),
		},
		{
			ID:          "tool-curl-request",
			Name:        "CURL 请求",
			Command:     "curl",
			Description: "数据传输工具",
			WorkingDir:  "",
			Parameters: []ToolParameter{
				{
					ID:          "param-url",
					Name:        "",
					Label:       "目标URL",
					Type:        "input",
					ArgType:     "positional",
					Required:    true,
					Placeholder: "https://example.com",
				},
				{
					ID:           "param-method",
					Name:         "-X",
					Label:        "HTTP方法",
					Type:         "select",
					ArgType:      "key-value",
					Required:     false,
					Options:      []string{"GET", "POST", "PUT", "DELETE"},
					DefaultValue: "GET",
				},
				{
					ID:          "param-headers",
					Name:        "-H",
					Label:       "请求头",
					Type:        "text",
					ArgType:     "key-value",
					Required:    false,
					Placeholder: "Content-Type: application/json",
				},
				{
					ID:          "param-data",
					Name:        "-d",
					Label:       "请求数据",
					Type:        "text",
					ArgType:     "key-value",
					Required:    false,
					Placeholder: `{"key": "value"}`,
				},
				{
					ID:          "param-output",
					Name:        "-o",
					Label:       "输出文件",
					Type:        "input",
					ArgType:     "key-value",
					Required:    false,
					Placeholder: "output.html",
				},
			},
			CreatedAt: time.Now().Format(time.RFC3339),
			UpdatedAt: time.Now().Format(time.RFC3339),
		},
	}
}

// 保存工具配置
func saveToolsConfig() error {
	data, err := json.MarshalIndent(tools, "", "  ")
	if err != nil {
		return err
	}

	// 确保目录存在
	dir := filepath.Dir(toolsConfigPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(toolsConfigPath, data, 0644)
}

// 获取所有工具配置
func getToolsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	toolsMutex.RLock()
	defer toolsMutex.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tools)
}

// 保存工具配置
func saveToolsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var updatedTools []ToolConfig
	if err := json.NewDecoder(r.Body).Decode(&updatedTools); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	toolsMutex.Lock()
	tools = updatedTools
	if err := saveToolsConfig(); err != nil {
		toolsMutex.Unlock()
		http.Error(w, "Failed to save tools config: "+err.Error(), http.StatusInternalServerError)
		return
	}
	toolsMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// 启动终端执行 (不收集输出)
func startTerminalExecution(executionID string, cmd *exec.Cmd) {
	// 设置标准输入输出
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// 启动命令
	if err := cmd.Start(); err != nil {
		log.Printf("Failed to start terminal command: %v", err)
		handleExecutionError(executionID, err)
		return
	}

	// 等待命令完成
	go func() {
		err := cmd.Wait()
		executionMutex.Lock()
		defer executionMutex.Unlock()

		execCtx, exists := activeExecutions[executionID]
		if !exists {
			return
		}

		endTime := time.Now()
		if err != nil {
			execCtx.status = "failed"
			log.Printf("Terminal command failed: %v", err)
		} else {
			execCtx.status = "completed"
		}

		execCtx.completed = true

		// 更新历史记录
		for i := range executionHistory {
			if executionHistory[i].ID == executionID {
				executionHistory[i].Status = execCtx.status
				executionHistory[i].EndTime = endTime
				if err != nil {
					executionHistory[i].Error = err.Error()
				}
				break
			}
		}

		// 清理活跃执行记录
		delete(activeExecutions, executionID)
	}()
}

// 启动非终端执行 (正常收集输出)
func startNonTerminalExecution(executionID string, cmd *exec.Cmd) error {
	// 设置中文环境变量，确保子进程使用正确的编码
	env := os.Environ()
	env = append(env, "LANG=zh_CN.UTF-8")
	env = append(env, "LC_ALL=zh_CN.UTF-8")
	cmd.Env = env

	// 获取命令输出管道
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	// 启动命令
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %v", err)
	}

	// 启动输出收集goroutine
	go collectOutput(executionID, stdout, stderr)

	// 启动等待goroutine
	go waitForCompletion(executionID, cmd)

	return nil
}

// 构建命令参数 - 修改：空值不进行参数拼接
func buildCommandArgs(params map[string]interface{}, toolParams []ToolParameter) []string {
	args := []string{}

	// 首先处理位置参数
	for _, param := range toolParams {
		if param.ArgType == "positional" {
			if value, exists := params[param.ID]; exists && !isEmptyValue(value) {
				args = append(args, fmt.Sprintf("%v", value))
			}
		}
	}

	// 然后处理其他类型的参数
	for _, param := range toolParams {
		if param.ArgType != "positional" {
			value, exists := params[param.ID]
			if !exists || isEmptyValue(value) {
				continue
			}

			switch param.ArgType {
			case "key-value":
				// 键值对参数: -name value
				args = append(args, param.Name, fmt.Sprintf("%v", value))
			case "flag":
				// 开关参数: 只有为 true 时才添加
				if boolValue, ok := value.(bool); ok && boolValue {
					args = append(args, param.Name)
				}
			case "combined":
				// 组合参数: -name=value
				args = append(args, fmt.Sprintf("%s=%v", param.Name, value))
			case "value-only":
				// 只有值，没有参数名
				args = append(args, fmt.Sprintf("%v", value))
			}
		}
	}

	return args
}

// 新增：判断值是否为空的辅助函数
func isEmptyValue(value interface{}) bool {
	if value == nil {
		return true
	}

	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) == ""
	case bool:
		return false // 布尔值永远不为空，false 也是有意义的
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return false // 数字值永远不为空，0 也是有意义的
	case float32, float64:
		return false // 浮点数永远不为空，0.0 也是有意义的
	default:
		// 对于其他类型，转换为字符串判断
		str := fmt.Sprintf("%v", value)
		return strings.TrimSpace(str) == ""
	}
}

// 修改 collectOutput 函数，修复编码引用
func collectOutput(executionID string, stdout, stderr io.ReadCloser) {
	executionMutex.Lock()
	execCtx, exists := activeExecutions[executionID]
	executionMutex.Unlock()

	if !exists {
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// 修复：使用执行上下文的编码来处理输出
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()

			// 修复：处理编码转换，使用执行上下文的编码
			processedLine := processLineEncoding(line, execCtx.encoding)

			// 添加换行符
			outputLine := processedLine + "\n"

			execCtx.output.WriteString(outputLine)

			select {
			case execCtx.outputChan <- outputLine:
			default:
			}
			broadcastExecutionOutput(executionID, outputLine)
		}

		if err := scanner.Err(); err != nil {
			log.Printf("Error reading stdout: %v", err)
		}
	}()

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			line := "[STDERR] " + scanner.Text()

			// 修复：处理编码转换，使用执行上下文的编码
			processedLine := processLineEncoding(line, execCtx.encoding)

			outputLine := processedLine + "\n"

			execCtx.output.WriteString(outputLine)

			select {
			case execCtx.outputChan <- outputLine:
			default:
			}
			broadcastExecutionOutput(executionID, outputLine)
		}

		if err := scanner.Err(); err != nil {
			log.Printf("Error reading stderr: %v", err)
		}
	}()

	go func() {
		wg.Wait()
		select {
		case execCtx.outputChan <- "[执行完成]\n":
		default:
		}
	}()
}

// 真正的GBK到UTF-8转换
func GBKToUTF8(gbkStr string) (string, error) {
	reader := transform.NewReader(bytes.NewReader([]byte(gbkStr)), simplifiedchinese.GBK.NewDecoder())
	d, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(d), nil
}

// UTF-8到GBK转换
func UTF8ToGBK(utf8Str string) (string, error) {
	reader := transform.NewReader(bytes.NewReader([]byte(utf8Str)), simplifiedchinese.GBK.NewEncoder())
	d, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(d), nil
}

// 改进编码处理函数，添加自动检测和修复
func processLineEncoding(line, encoding string) string {
	// 如果已经是有效的UTF-8，直接返回
	if utf8.ValidString(line) {
		return line
	}

	// 根据指定的编码进行处理
	switch strings.ToLower(encoding) {
	case "gbk", "gb2312":
		// 尝试GBK到UTF-8转换
		if decoded, err := GBKToUTF8(line); err == nil && utf8.ValidString(decoded) {
			return decoded
		}
		// 如果转换失败，尝试修复常见的中文乱码
		return fixCommonGarbledChars(line)
	default:
		// 对于其他编码，尝试转换为有效的UTF-8
		return strings.ToValidUTF8(line, "�")
	}
}

// 新增：修复常见的中文乱码字符
func fixCommonGarbledChars(line string) string {
	// 常见的GBK乱码到正确中文的映射
	replacements := map[string]string{
		"�÷�": "用法",
		"�1":  "参数1",
		"�2":  "参数2",
		"�3":  "参数3",
		"�4":  "参数4",
		"�5":  "参数5",
		"�6":  "参数6",
		"�7":  "参数7",
		"�8":  "参数8",
		"�9":  "参数9",
		"�0":  "参数0",
		// 添加更多常见的乱码映射
	}

	result := line
	for garbled, correct := range replacements {
		result = strings.ReplaceAll(result, garbled, correct)
	}

	return result
}

// 修改实时输出处理器，使用执行上下文的编码
func getRealtimeOutputHandler(w http.ResponseWriter, r *http.Request) {
	executionID := r.URL.Query().Get("id")
	if executionID == "" {
		http.Error(w, "Execution ID is required", http.StatusBadRequest)
		return
	}

	executionMutex.Lock()
	execCtx, exists := activeExecutions[executionID]
	executionMutex.Unlock()

	if !exists {
		http.Error(w, "Execution not found", http.StatusNotFound)
		return
	}

	if execCtx.terminal == 1 {
		http.Error(w, "Terminal mode does not support realtime output", http.StatusBadRequest)
		return
	}

	// 设置SSE头部
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// 使用执行上下文中的编码
	encoding := execCtx.encoding
	if encoding == "" {
		encoding = "utf-8"
	}

	for {
		select {
		case output, ok := <-execCtx.outputChan:
			if !ok {
				fmt.Fprintf(w, "data: %s\n\n", "[执行完成]")
				flusher.Flush()
				return
			}

			// 处理输出编码
			processedOutput := processOutputEncoding(output, encoding)

			if strings.TrimSpace(processedOutput) == "[执行完成]" {
				fmt.Fprintf(w, "data: %s\n\n", processedOutput)
				flusher.Flush()
				return
			}

			lines := strings.Split(processedOutput, "\n")
			for _, line := range lines {
				if line == "" {
					continue
				}
				encodedLine := processOutputEncoding(line, encoding)
				fmt.Fprintf(w, "data: %s\n\n", encodedLine)
				flusher.Flush()
			}

		case <-r.Context().Done():
			return

		case <-time.After(30 * time.Second):
			fmt.Fprintf(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

// 改进输出编码处理
func processOutputEncoding(text, encoding string) string {
	// 首先确保文本是有效的UTF-8
	if !utf8.ValidString(text) {
		// 尝试根据编码修复
		switch strings.ToLower(encoding) {
		case "gbk", "gb2312":
			if decoded, err := GBKToUTF8(text); err == nil {
				return decoded
			}
			return fixCommonGarbledChars(text)
		default:
			return strings.ToValidUTF8(text, "�")
		}
	}
	return text
}

// 等待命令完成
func waitForCompletion(executionID string, cmd *exec.Cmd) {
	err := cmd.Wait()

	executionMutex.Lock()
	defer executionMutex.Unlock()

	execCtx, exists := activeExecutions[executionID]
	if !exists {
		return
	}

	// 标记为已完成
	execCtx.completed = true

	// 更新执行状态
	endTime := time.Now()
	if err != nil {
		execCtx.status = "failed"
		execCtx.output.WriteString(fmt.Sprintf("\n[ERROR] Command failed: %v\n", err))
	} else {
		execCtx.status = "completed"
	}

	// 关闭输出通道（这会触发实时输出的完成信号）
	close(execCtx.outputChan)

	// 更新历史记录
	for i := range executionHistory {
		if executionHistory[i].ID == executionID {
			executionHistory[i].Status = execCtx.status
			executionHistory[i].EndTime = endTime
			executionHistory[i].Output = execCtx.output.String()
			if err != nil {
				executionHistory[i].Error = err.Error()
			}
			break
		}
	}

	// 清理活跃执行记录
	delete(activeExecutions, executionID)
}

// 停止执行处理器
func stopExecutionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	executionID := r.URL.Query().Get("id")
	if executionID == "" {
		http.Error(w, "Execution ID is required", http.StatusBadRequest)
		return
	}

	executionMutex.Lock()
	execCtx, exists := activeExecutions[executionID]
	executionMutex.Unlock()

	if !exists {
		// 检查是否是web终端会话
		if err := stopWebTerminalSession(executionID); err == nil {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "stopped"})
			return
		}
		http.Error(w, "Execution not found", http.StatusNotFound)
		return
	}

	// 取消执行
	execCtx.cancel()
	execCtx.status = "cancelled"
	execCtx.completed = true

	// 关闭输出通道
	if execCtx.outputChan != nil {
		close(execCtx.outputChan)
	}

	// 更新历史记录
	for i := range executionHistory {
		if executionHistory[i].ID == executionID {
			executionHistory[i].Status = "cancelled"
			executionHistory[i].EndTime = time.Now()
			break
		}
	}

	delete(activeExecutions, executionID)

	response := ExecutionResponse{
		ID:        executionID,
		Status:    "cancelled",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// 获取执行状态处理器
func getExecutionStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	executionID := r.URL.Query().Get("id")
	if executionID == "" {
		// 返回所有执行历史
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(executionHistory)
		return
	}

	executionMutex.Lock()
	defer executionMutex.Unlock()

	// 查找执行记录
	var status *ExecutionStatus
	for i := range executionHistory {
		if executionHistory[i].ID == executionID {
			status = &executionHistory[i]
			break
		}
	}

	if status == nil {
		http.Error(w, "Execution not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// 获取执行输出处理器
func getExecutionOutputHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	executionID := r.URL.Query().Get("id")
	if executionID == "" {
		http.Error(w, "Execution ID is required", http.StatusBadRequest)
		return
	}

	executionMutex.Lock()
	defer executionMutex.Unlock()

	// 查找执行记录
	for i := range executionHistory {
		if executionHistory[i].ID == executionID {
			response := ExecutionResponse{
				ID:        executionID,
				Status:    executionHistory[i].Status,
				Output:    executionHistory[i].Output,
				Error:     executionHistory[i].Error,
				Timestamp: time.Now().Format(time.RFC3339),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	http.Error(w, "Execution not found", http.StatusNotFound)
}

// 工具函数
func generateExecutionID() string {
	return fmt.Sprintf("exec-%d-%s", time.Now().Unix(), randomString(8))
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func handleExecutionError(executionID string, err error) {
	executionMutex.Lock()
	defer executionMutex.Unlock()

	// 更新历史记录
	for i := range executionHistory {
		if executionHistory[i].ID == executionID {
			executionHistory[i].Status = "failed"
			executionHistory[i].EndTime = time.Now()
			executionHistory[i].Error = err.Error()
			break
		}
	}

	delete(activeExecutions, executionID)
}

func broadcastExecutionOutput(executionID string, line string) {
	// 这里可以实现WebSocket或Server-Sent Events进行实时输出推送
	// 当前版本先记录到执行上下文中
}

// 创建全局 CORS 处理器
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS 仅放行本地回环来源（127.0.0.1/localhost）与 Wails 协议；
		// Wails 协议页面 Origin 为 "null"，须携带进程级 Token 才放行
		origin := r.Header.Get("Origin")
		if origin != "" {
			allowed := localguard.IsAllowedOrigin(origin) || localguard.ValidateRequestToken(r)
			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			} else {
				w.Header().Set("Access-Control-Allow-Origin", "null")
			}
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		// X-Auth-Token：前端 main.js 统一注入的本地服务 CSRF Token，预检必须放行该头
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-Auth-Token")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// 如果是预检请求，直接返回
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// 调用下一个处理器
		next.ServeHTTP(w, r)
	})
}

func StartCLIExecutor(port int) {
	// 保存端口号
	cliServerPortMutex.Lock()
	cliServerPort = port
	cliServerPortMutex.Unlock()

	// 创建新的多路复用器
	mux := http.NewServeMux()

	// 注册路由
	mux.HandleFunc("/cli/tools", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getToolsHandler(w, r)
		case http.MethodPost:
			saveToolsHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/cli/execute", genericExecuteHandler)
	mux.HandleFunc("/cli/stop", stopExecutionHandler)
	mux.HandleFunc("/cli/status", getExecutionStatusHandler)
	mux.HandleFunc("/cli/output", getExecutionOutputHandler)
	mux.HandleFunc("/cli/realtime-output", getRealtimeOutputHandler)
	mux.HandleFunc("/cli/ai/default", defaultAIConfigHandler)
	mux.HandleFunc("/cli/ai/test", defaultAITestHandler)
	mux.HandleFunc("/cli/ai/chat", defaultAIChatHandler)
	mux.HandleFunc("/cli/mcp/tools", mcpListToolsHandler)
	mux.HandleFunc("/cli/mcp/call", mcpCallToolHandler)
	mux.HandleFunc("/cli/mcp/call/stream", mcpCallToolStreamHandler)
	mux.HandleFunc("/cli/web-terminal", handleWebTerminal)

	// 安全防护：Host/Origin 校验（阻断恶意网页 CSRF / DNS rebinding）+ CORS 中间件
	handler := localguard.HTTPGuard(port)(mux)
	handler = corsMiddleware(handler)

	log.Printf("Starting CLI executor server on port %d", port)
	log.Printf("Tools config file: %s", toolsConfigPath)
	log.Fatal(http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", port), handler))
}
