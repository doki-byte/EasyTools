package pcap

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// AnalysisConfig 分析配置
type AnalysisConfig struct {
	FilePath          string `json:"filePath"`
	FlagRegex         string `json:"flagRegex"`
	SqlRegex          string `json:"sqlRegex"`
	CredentialRegex   string `json:"credentialRegex"`
	KeyRegex          string `json:"keyRegex"`
	ExtractFiles      bool   `json:"extractFiles"`
	DetectFlags       bool   `json:"detectFlags"`
	DetectSQL         bool   `json:"detectSQL"`
	DetectCredentials bool   `json:"detectCredentials"`
	DetectKeys        bool   `json:"detectKeys"`
}

// AnalysisStatus 分析状态
type AnalysisStatus struct {
	IsProcessing   bool   `json:"isProcessing"`
	CurrentTask    string `json:"currentTask"`
	Packets        int    `json:"packets"`
	Matches        int    `json:"matches"`
	ExtractedFiles int    `json:"extractedFiles"`
	Progress       string `json:"progress"`
}

// AnalysisResult 分析结果
type AnalysisResult struct {
	Type        string `json:"type"`
	PacketNum   int    `json:"packetNum"`
	Content     string `json:"content"`
	FileName    string `json:"fileName,omitempty"`
	Protocol    string `json:"protocol,omitempty"`
	Source      string `json:"source,omitempty"`
	Destination string `json:"destination,omitempty"`
	Timestamp   string `json:"timestamp,omitempty"`
}

// ExtractionResult 文件提取结果
type ExtractionResult struct {
	FileName    string `json:"fileName"`
	FilePath    string `json:"filePath"`
	FileType    string `json:"fileType"`
	Size        int64  `json:"size"`
	Protocol    string `json:"protocol"`
	ContentType string `json:"contentType,omitempty"`
	MD5         string `json:"md5,omitempty"`
}

// PcapAnalyzer 分析服务
type PcapAnalyzer struct {
	ctx          context.Context
	mu           sync.Mutex
	isProcessing bool
	currentTask  string
	cancelFunc   context.CancelFunc
}

// NewPcapAnalyzer 创建新的分析服务
func NewPcapAnalyzer() *PcapAnalyzer {
	return &PcapAnalyzer{}
}

// Startup 服务启动
func (a *PcapAnalyzer) Startup(ctx context.Context) {
	a.ctx = ctx
	log.Println("分析服务启动完成")
}

// AnalyzePcap 分析PCAP文件
func (a *PcapAnalyzer) AnalyzePcap(config AnalysisConfig) string {
	// 加载正则配置
	regexConfig, err := a.loadConfig()
	if err != nil {
		a.sendLogEvent("warning", fmt.Sprintf("加载正则配置失败，使用默认配置: %v", err))
		regexConfig = defaultRegexConfig
	}

	// 使用配置文件中的正则表达式
	config.FlagRegex = regexConfig.FlagRegex
	config.SqlRegex = regexConfig.SqlRegex
	config.CredentialRegex = regexConfig.CredentialRegex
	config.KeyRegex = regexConfig.KeyRegex

	a.mu.Lock()
	if a.isProcessing {
		a.mu.Unlock()
		return "❌ 已有任务正在进行中，请等待当前任务完成"
	}
	a.isProcessing = true
	a.currentTask = "analysis"

	var ctx context.Context
	var cancel context.CancelFunc
	if a.ctx != nil {
		ctx, cancel = context.WithCancel(a.ctx)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	a.cancelFunc = cancel
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.isProcessing = false
		a.currentTask = ""
		a.cancelFunc = nil
		a.mu.Unlock()
	}()

	a.sendLogEvent("info", "=== 开始分析PCAP文件 ===")
	a.sendLogEvent("info", fmt.Sprintf("📁 文件路径: %s", config.FilePath))
	a.sendLogEvent("info", fmt.Sprintf("⚙️ 配置 - 提取文件: %t, 检测Flag: %t, 检测SQL: %t, 检测凭据: %t, 检测密钥: %t",
		config.ExtractFiles, config.DetectFlags, config.DetectSQL, config.DetectCredentials, config.DetectKeys))

	// 验证输入
	if config.FilePath == "" {
		a.sendLogEvent("error", "文件路径为空")
		return a.errorResult("文件路径为空")
	}

	if _, err := os.Stat(config.FilePath); os.IsNotExist(err) {
		a.sendLogEvent("error", fmt.Sprintf("文件不存在: %s", config.FilePath))
		return a.errorResult(fmt.Sprintf("文件不存在: %s", config.FilePath))
	}

	var results []AnalysisResult
	var extractedFiles []ExtractionResult
	var fileContentResults []AnalysisResult

	// 执行数据包分析
	analysisResults, err := a.performAnalysis(ctx, config)
	if err != nil {
		a.sendLogEvent("error", fmt.Sprintf("分析失败: %v", err))
		return a.errorResult(fmt.Sprintf("分析失败: %v", err))
	}
	results = analysisResults

	// 如果需要提取文件
	if config.ExtractFiles {
		a.sendLogEvent("info", "开始提取文件并进行内容分析...")
		files, fileMatches, err := a.extractFilesFromPcap(ctx, config.FilePath, config)
		if err != nil {
			a.sendLogEvent("warning", fmt.Sprintf("文件提取过程中出现错误: %v", err))
		} else {
			extractedFiles = files
			fileContentResults = fileMatches
			a.sendLogEvent("success", fmt.Sprintf("✅ 成功提取 %d 个文件，文件内容匹配 %d 个结果", len(extractedFiles), len(fileContentResults)))
		}
	}

	// 合并数据包分析结果和文件内容分析结果
	allResults := append(results, fileContentResults...)

	// 准备返回结果
	response := map[string]interface{}{
		"analysisResults": allResults, // 包含数据包匹配和文件内容匹配
		"extractedFiles":  extractedFiles,
		"summary": map[string]interface{}{
			"totalPackets":       len(results),
			"fileContentMatches": len(fileContentResults),
			"totalMatches":       len(allResults),
			"extractedFiles":     len(extractedFiles),
			"analysisTime":       time.Now().Format("2006-01-02 15:04:05"),
		},
	}

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		a.sendLogEvent("error", fmt.Sprintf("JSON序列化失败: %v", err))
		return a.errorResult(fmt.Sprintf("JSON序列化失败: %v", err))
	}

	finalMsg := fmt.Sprintf("✅ 分析完成，数据包匹配 %d 个结果，文件内容匹配 %d 个结果，提取 %d 个文件",
		len(results), len(fileContentResults), len(extractedFiles))
	a.sendLogEvent("success", finalMsg)
	return string(jsonResponse)
}

func (a *PcapAnalyzer) analyzeFileContent(content []byte, fileName, protocol string, config AnalysisConfig, flagRe, sqlRe, credentialRe, keyRe *regexp.Regexp) []AnalysisResult {
	var results []AnalysisResult

	// 将内容转换为字符串进行处理
	contentStr := string(content)

	// 检测Flag
	if config.DetectFlags && flagRe != nil {
		matches := flagRe.FindAllString(contentStr, -1)
		for _, match := range matches {
			if len(match) > 0 && a.isValidFlag(match) {
				results = append(results, AnalysisResult{
					Type:     "file_flag",
					Content:  match,
					FileName: fileName,
					Protocol: protocol,
				})
			}
		}
	}

	// 检测SQL语句
	if config.DetectSQL && sqlRe != nil {
		matches := sqlRe.FindAllString(contentStr, -1)
		for _, match := range matches {
			if len(match) > 0 {
				results = append(results, AnalysisResult{
					Type:     "file_sql",
					Content:  match,
					FileName: fileName,
					Protocol: protocol,
				})
			}
		}
	}

	// 检测登录凭据
	if config.DetectCredentials && credentialRe != nil {
		matches := credentialRe.FindAllStringSubmatch(contentStr, -1)
		for _, match := range matches {
			for i := 1; i < len(match); i++ {
				if len(match) > i && match[i] != "" && a.isValidCredential(match[i]) {
					results = append(results, AnalysisResult{
						Type:     "file_credential",
						Content:  match[i],
						FileName: fileName,
						Protocol: protocol,
					})
					break
				}
			}
		}
	}

	// 检测密钥
	if config.DetectKeys && keyRe != nil {
		matches := keyRe.FindAllStringSubmatch(contentStr, -1)
		for _, match := range matches {
			for i := 1; i < len(match); i++ {
				if len(match) > i && match[i] != "" && a.isValidKey(match[i]) {
					results = append(results, AnalysisResult{
						Type:     "file_key",
						Content:  match[i],
						FileName: fileName,
						Protocol: protocol,
					})
					break
				}
			}
		}
	}

	return results
}

// analyzePacketFields 分析数据包字段
func (a *PcapAnalyzer) analyzePacketFields(fields []string, packetNum int, config AnalysisConfig) []AnalysisResult {
	var results []AnalysisResult

	// 合并所有字段内容，特别关注payload字段
	var payloadFields []string
	for i, field := range fields {
		if field != "" {
			// 特别关注payload字段（通常在第4、5个位置）
			if i >= 4 && i <= 5 && len(field) > 0 {
				payloadFields = append(payloadFields, field)
			}
		}
	}

	// 尝试从payload中解码hex数据
	var decodedContent strings.Builder
	for _, payload := range payloadFields {
		// 尝试hex解码
		if decoded, err := hex.DecodeString(payload); err == nil && len(decoded) > 0 {
			decodedContent.Write(decoded)
			decodedContent.WriteString(" ")
		} else {
			// 如果不是hex，直接使用原始内容
			decodedContent.WriteString(payload)
			decodedContent.WriteString(" ")
		}
	}

	content := decodedContent.String()
	if content == "" {
		// 如果没有payload，使用所有字段
		content = strings.Join(fields, " ")
	}

	// 预编译正则表达式以提高性能
	var flagRe, sqlRe, credentialRe, keyRe *regexp.Regexp
	var err error

	// 编译所有启用的正则表达式
	if config.DetectFlags && config.FlagRegex != "" {
		flagRe, err = regexp.Compile(config.FlagRegex)
		if err != nil {
			log.Printf("⚠️ Flag正则表达式编译失败: %v", err)
		}
	}

	if config.DetectSQL && config.SqlRegex != "" {
		sqlRe, err = regexp.Compile(config.SqlRegex)
		if err != nil {
			log.Printf("⚠️ SQL正则表达式编译失败: %v", err)
		}
	}

	if config.DetectCredentials && config.CredentialRegex != "" {
		credentialRe, err = regexp.Compile(config.CredentialRegex)
		if err != nil {
			log.Printf("⚠️ 凭据正则表达式编译失败: %v", err)
		}
	}

	if config.DetectKeys && config.KeyRegex != "" {
		keyRe, err = regexp.Compile(config.KeyRegex)
		if err != nil {
			log.Printf("⚠️ 密钥正则表达式编译失败: %v", err)
		}
	}

	// 检测Flag - 使用配置文件中的正则
	if config.DetectFlags && flagRe != nil {
		matches := flagRe.FindAllString(content, -1)
		for _, match := range matches {
			if len(match) > 0 && a.isValidFlag(match) {
				results = append(results, AnalysisResult{
					Type:      "flag",
					PacketNum: packetNum,
					Content:   match,
				})
			}
		}
	}

	// 检测SQL语句 - 使用配置文件中的正则
	if config.DetectSQL && sqlRe != nil {
		matches := sqlRe.FindAllString(content, -1)
		for _, match := range matches {
			if len(match) > 0 {
				results = append(results, AnalysisResult{
					Type:      "sql",
					PacketNum: packetNum,
					Content:   match,
				})
			}
		}
	}

	// 检测登录凭据 - 使用配置文件中的正则
	if config.DetectCredentials && credentialRe != nil {
		matches := credentialRe.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			// 取第一个捕获组
			for i := 1; i < len(match); i++ {
				if len(match) > i && match[i] != "" && a.isValidCredential(match[i]) {
					results = append(results, AnalysisResult{
						Type:      "credential",
						PacketNum: packetNum,
						Content:   match[i],
					})
					break // 只取第一个有效的捕获组
				}
			}
		}
	}

	// 检测密钥 - 使用配置文件中的正则
	if config.DetectKeys && keyRe != nil {
		matches := keyRe.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			// 取第一个捕获组
			for i := 1; i < len(match); i++ {
				if len(match) > i && match[i] != "" && a.isValidKey(match[i]) {
					results = append(results, AnalysisResult{
						Type:      "key",
						PacketNum: packetNum,
						Content:   match[i],
					})
					break // 只取第一个有效的捕获组
				}
			}
		}
	}

	return results
}

// 验证Flag格式
func (a *PcapAnalyzer) isValidFlag(flag string) bool {
	if len(flag) < 5 || len(flag) > 100 {
		return false
	}
	return true
}

// 验证凭据格式
func (a *PcapAnalyzer) isValidCredential(credential string) bool {
	if len(credential) < 3 || len(credential) > 50 {
		return false
	}
	// 简单的有效性检查
	if strings.Contains(credential, " ") || strings.Contains(credential, "\n") {
		return false
	}
	return true
}

// 验证密钥格式
func (a *PcapAnalyzer) isValidKey(key string) bool {
	if len(key) < 10 || len(key) > 100 {
		return false
	}
	return true
}

// ExtractFiles 从PCAP文件中提取文件（包含内容分析）
func (a *PcapAnalyzer) ExtractFiles(filePath string, enableContentAnalysis bool) string {
	a.mu.Lock()
	if a.isProcessing {
		a.mu.Unlock()
		return "❌ 已有任务正在进行中，请等待当前任务完成"
	}
	a.isProcessing = true
	a.currentTask = "extraction"

	// 使用后台上下文
	var ctx context.Context
	var cancel context.CancelFunc
	if a.ctx != nil {
		ctx, cancel = context.WithCancel(a.ctx)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	a.cancelFunc = cancel
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		a.isProcessing = false
		a.currentTask = ""
		a.cancelFunc = nil
		a.mu.Unlock()
	}()

	log.Printf("=== 开始从PCAP文件中提取文件 ===\n")
	log.Printf("📁 源文件: %s\n", filePath)
	log.Printf("🔍 内容分析: %t\n", enableContentAnalysis)

	if filePath == "" {
		return a.errorResult("文件路径为空")
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return a.errorResult(fmt.Sprintf("文件不存在: %s", filePath))
	}

	// 加载正则配置
	regexConfig, err := a.loadConfig()
	if err != nil {
		log.Printf("⚠️ 加载正则配置失败，使用默认配置: %v", err)
		regexConfig = defaultRegexConfig
	}

	config := AnalysisConfig{
		FilePath:          filePath,
		ExtractFiles:      true,
		DetectFlags:       enableContentAnalysis,
		DetectSQL:         enableContentAnalysis,
		DetectCredentials: enableContentAnalysis,
		DetectKeys:        enableContentAnalysis,
		FlagRegex:         regexConfig.FlagRegex,
		SqlRegex:          regexConfig.SqlRegex,
		CredentialRegex:   regexConfig.CredentialRegex,
		KeyRegex:          regexConfig.KeyRegex,
	}

	// 调用函数，接收所有返回值
	extractedFiles, contentResults, err := a.extractFilesFromPcap(ctx, filePath, config)
	if err != nil {
		return a.errorResult(fmt.Sprintf("文件提取失败: %v", err))
	}

	// 准备返回结果
	response := map[string]interface{}{
		"extractedFiles": extractedFiles,
		"contentResults": contentResults,
		"summary": map[string]interface{}{
			"extractedFiles": len(extractedFiles),
			"contentMatches": len(contentResults),
			"extractionTime": time.Now().Format("2006-01-02 15:04:05"),
		},
	}

	jsonResults, err := json.Marshal(response)
	if err != nil {
		return a.errorResult(fmt.Sprintf("JSON序列化失败: %v", err))
	}

	logMsg := fmt.Sprintf("✅ 文件提取完成，共提取 %d 个文件", len(extractedFiles))
	if enableContentAnalysis {
		logMsg += fmt.Sprintf("，内容匹配 %d 个结果", len(contentResults))
	}
	log.Printf(logMsg)

	return string(jsonResults)
}

// detectContentTypeByContent 根据文件内容检测Content-Type
func (a *PcapAnalyzer) detectContentTypeByContent(content []byte) string {
	if len(content) == 0 {
		return "application/octet-stream"
	}

	// 检查文件魔数
	if len(content) >= 8 {
		magic := string(content[:8])

		switch {
		case strings.HasPrefix(magic, "\x89PNG\r\n\x1a\n"):
			return "image/png"
		case strings.HasPrefix(magic, "\xff\xd8\xff"):
			return "image/jpeg"
		case strings.HasPrefix(magic, "GIF8"):
			return "image/gif"
		case strings.HasPrefix(magic, "BM"):
			return "image/bmp"
		case strings.HasPrefix(magic, "%PDF"):
			return "application/pdf"
		case strings.HasPrefix(magic, "PK\x03\x04"):
			return "application/zip"
		case strings.HasPrefix(magic, "Rar!\x1a\x07"):
			return "application/x-rar-compressed"
		case strings.HasPrefix(magic, "\x7fELF"):
			return "application/x-executable"
		case strings.HasPrefix(magic, "MZ"):
			return "application/x-msdownload"
		}
	}

	// 检查文本内容
	if len(content) >= 5 {
		prefix := string(content[:5])
		switch {
		case strings.HasPrefix(prefix, "<html") || strings.HasPrefix(string(content[:9]), "<!DOCTYPE"):
			return "text/html"
		case strings.HasPrefix(prefix, "<?xml"):
			return "application/xml"
		case strings.HasPrefix(prefix, "{\"") || strings.HasPrefix(prefix, "[{"):
			// 尝试解析JSON
			var js json.RawMessage
			if json.Unmarshal(content, &js) == nil {
				return "application/json"
			}
		}
	}

	// 检查是否为文本文件
	if a.isTextContent(content) {
		return "text/plain"
	}

	return "application/octet-stream"
}

// PreviewFile 预览文件内容
func (a *PcapAnalyzer) PreviewFile(filePath string) string {
	log.Printf("🔍 预览文件: %s", filePath)

	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Sprintf("❌ 无法读取文件: %v", err)
	}

	// 文件大小限制：10MB
	if fileInfo.Size() > 10*1024*1024 {
		return "⚠️ 文件过大(>10MB)，建议直接下载查看"
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Sprintf("❌ 读取文件失败: %v", err)
	}

	fileType := a.detectFileTypeByContent(content)
	preview := a.getTextPreview(content)

	return fmt.Sprintf("📊 文件信息:\n名称: %s\n类型: %s\n大小: %s\n路径: %s\nMD5: %s\n\n📄 文本预览:\n%s",
		filepath.Base(filePath), fileType, formatFileSize(fileInfo.Size()),
		filePath, calculateMD5(content), preview)
}

// CancelAnalysis 取消当前分析任务
func (a *PcapAnalyzer) CancelAnalysis() string {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.isProcessing && a.cancelFunc != nil {
		a.cancelFunc()
		a.isProcessing = false
		a.cancelFunc = nil
		log.Println("分析任务已取消")
		return "✅ 分析任务已取消"
	}

	return "⚠️ 没有正在运行的分析任务"
}

// GetAnalysisStatus 获取分析状态
func (a *PcapAnalyzer) GetAnalysisStatus() AnalysisStatus {
	a.mu.Lock()
	defer a.mu.Unlock()

	return AnalysisStatus{
		IsProcessing:   a.isProcessing,
		CurrentTask:    a.currentTask,
		Packets:        0, // 这个需要从分析过程中获取
		Matches:        0, // 这个需要从分析过程中获取
		ExtractedFiles: 0,
		Progress:       "",
	}
}
