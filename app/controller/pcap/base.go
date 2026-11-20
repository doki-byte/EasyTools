package pcap

import (
	"EasyTools/app/controller/system"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// RegexConfig 正则表达式配置
type RegexConfig struct {
	FlagRegex       string            `json:"flagRegex"`
	SqlRegex        string            `json:"sqlRegex"`
	CredentialRegex string            `json:"credentialRegex"`
	KeyRegex        string            `json:"keyRegex"`
	CustomRegex     map[string]string `json:"customRegex"`
}

// 默认正则配置
var defaultRegexConfig = RegexConfig{
	FlagRegex:       `flag\{[^}]+\}|FLAG\{[^}]+\}|ctf\{[^}]+\}|CTF\{[^}]+\}|key\{[^}]+\}|flag\[[^\]]+\]`,
	SqlRegex:        `(SELECT|INSERT|UPDATE|DELETE).*?FROM|UNION.*?SELECT|DROP.*TABLE|INSERT.*INTO|UPDATE.*SET`,
	CredentialRegex: `password\s*[=:]\s*['"]?([^'"\s]{3,50})|passwd\s*[=:]\s*['"]?([^'"\s]{3,50})|pwd\s*[=:]\s*['"]?([^'"\s]{3,50})|username\s*[=:]\s*['"]?([^'"\s]{3,50})|user\s*[=:]\s*['"]?([^'"\s]{3,50})|login\s*[=:]\s*['"]?([^'"\s]{3,50})|auth\s*[=:]\s*['"]?([^'"\s]{3,50})`,
	KeyRegex:        `api[_-]?key\s*[=:]\s*['"]?([a-zA-Z0-9_-]{10,50})['"]?|secret[_-]?key\s*[=:]\s*['"]?([a-zA-Z0-9_-]{10,50})['"]?|access[_-]?token\s*[=:]\s*['"]?([a-zA-Z0-9_-]{10,50})['"]?|bearer\s+([a-zA-Z0-9_-]{20,100})|token\s*[=:]\s*['"]?([a-zA-Z0-9_-]{10,50})['"]?`,
	CustomRegex:     make(map[string]string),
}

func PathExist(path string) string {
	_, err := os.Stat(path)
	if err != nil && os.IsNotExist(err) {
		_ = os.MkdirAll(path, os.ModePerm)
	}
	return path
}

// 正则配置管理方法
func (a *PcapAnalyzer) GetRegexConfig() string {
	a.mu.Lock()
	defer a.mu.Unlock()

	config, err := a.loadConfig()
	if err != nil {
		// 如果加载失败，返回默认配置
		config = defaultRegexConfig
	}

	jsonData, err := json.Marshal(config)
	if err != nil {
		return `{"error": "序列化配置失败"}`
	}

	return string(jsonData)
}

func (a *PcapAnalyzer) UpdateRegexConfig(configJson string) string {
	a.mu.Lock()
	defer a.mu.Unlock()

	var config RegexConfig
	if err := json.Unmarshal([]byte(configJson), &config); err != nil {
		return fmt.Sprintf("❌ 配置解析失败: %v", err)
	}

	// 验证正则表达式
	if err := a.validateRegex(config); err != nil {
		return fmt.Sprintf("❌ 正则表达式验证失败: %v", err)
	}

	// 保存配置
	if err := a.saveConfig(config); err != nil {
		return fmt.Sprintf("❌ 配置保存失败: %v", err)
	}

	a.sendLogEvent("success", "正则表达式配置已更新")
	return "✅ 正则表达式配置已保存"
}

// ResetRegexConfig 重置为默认配置
func (a *PcapAnalyzer) ResetRegexConfig() string {
	a.mu.Lock()
	defer a.mu.Unlock()

	if err := a.saveConfig(defaultRegexConfig); err != nil {
		return fmt.Sprintf("❌ 重置配置失败: %v", err)
	}

	a.sendLogEvent("success", "正则表达式配置已重置为默认值")
	return "✅ 正则表达式配置已重置为默认值"
}

// 加载配置文件
func (a *PcapAnalyzer) loadConfig() (RegexConfig, error) {
	var config RegexConfig

	configFilePath := filepath.Join(system.GetAppBaseDir(), "config", "pcap_regex_config.json")

	data, err := os.ReadFile(configFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不存在，创建默认配置
			if err := a.saveConfig(defaultRegexConfig); err != nil {
				return defaultRegexConfig, err
			}
			return defaultRegexConfig, nil
		}
		return defaultRegexConfig, err
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return defaultRegexConfig, err
	}

	return config, nil
}

// 保存配置文件
func (a *PcapAnalyzer) saveConfig(config RegexConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	configFilePath := filepath.Join(system.GetAppBaseDir(), "config", "pcap_regex_config.json")
	return os.WriteFile(configFilePath, data, 0644)
}

// 验证正则表达式
func (a *PcapAnalyzer) validateRegex(config RegexConfig) error {
	regexes := map[string]string{
		"flagRegex":       config.FlagRegex,
		"sqlRegex":        config.SqlRegex,
		"credentialRegex": config.CredentialRegex,
		"keyRegex":        config.KeyRegex,
	}

	for name, pattern := range regexes {
		if pattern != "" {
			if _, err := regexp.Compile(pattern); err != nil {
				return fmt.Errorf("%s 无效: %v", name, err)
			}
		}
	}

	// 验证自定义正则
	for name, pattern := range config.CustomRegex {
		if pattern != "" {
			if _, err := regexp.Compile(pattern); err != nil {
				return fmt.Errorf("自定义正则 %s 无效: %v", name, err)
			}
		}
	}

	return nil
}

// OpenFile 打开文件对话框
func (a *PcapAnalyzer) OpenFile(dialogOptions map[string]interface{}) string {
	// 默认选项
	options := runtime.OpenDialogOptions{
		Title: "选择PCAP文件",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "PCAP Files (*.pcap;*.pcapng)",
				Pattern:     "*.pcap;*.pcapng",
			},
			{
				DisplayName: "All Files (*.*)",
				Pattern:     "*.*",
			},
		},
	}

	// 应用自定义选项
	if title, ok := dialogOptions["Title"].(string); ok {
		options.Title = title
	}
	if filters, ok := dialogOptions["Filters"].([]interface{}); ok {
		fileFilters := make([]runtime.FileFilter, 0, len(filters))
		for _, filter := range filters {
			if f, ok := filter.(map[string]interface{}); ok {
				displayName, _ := f["DisplayName"].(string)
				pattern, _ := f["Pattern"].(string)
				if displayName != "" && pattern != "" {
					fileFilters = append(fileFilters, runtime.FileFilter{
						DisplayName: displayName,
						Pattern:     pattern,
					})
				}
			}
		}
		if len(fileFilters) > 0 {
			options.Filters = fileFilters
		}
	}

	// 打开文件对话框
	filePath, err := runtime.OpenFileDialog(a.ctx, options)
	if err != nil {
		log.Printf("打开文件对话框错误: %v", err)
		return ""
	}

	if filePath != "" {
		absPath, err := filepath.Abs(filePath)
		if err == nil {
			filePath = absPath
		}
		log.Printf("用户选择文件: %s", filePath)
	}

	return filePath
}

// SaveFile 保存文件对话框
func (a *PcapAnalyzer) SaveFile(dialogOptions map[string]interface{}) string {
	options := runtime.SaveDialogOptions{
		Title: "保存文件",
		Filters: []runtime.FileFilter{
			{
				DisplayName: "JSON Files (*.json)",
				Pattern:     "*.json",
			},
			{
				DisplayName: "Text Files (*.txt)",
				Pattern:     "*.txt",
			},
			{
				DisplayName: "All Files (*.*)",
				Pattern:     "*.*",
			},
		},
		DefaultFilename: "pcap-analysis.json",
	}

	if title, ok := dialogOptions["Title"].(string); ok {
		options.Title = title
	}
	if defaultFilename, ok := dialogOptions["DefaultFilename"].(string); ok {
		options.DefaultFilename = defaultFilename
	}

	filePath, err := runtime.SaveFileDialog(a.ctx, options)
	if err != nil {
		log.Printf("保存文件对话框错误: %v", err)
		return ""
	}

	return filePath
}

// 添加日志事件发送方法
func (a *PcapAnalyzer) sendLogEvent(level, message string) {
	if a.ctx != nil {
		eventData := map[string]interface{}{
			"level":   level,
			"message": message,
			"time":    time.Now().Format("15:04:05"),
			"source":  "CTF/PCAP分析",
		}
		runtime.EventsEmit(a.ctx, "log_event", eventData)
		//log.Printf("📝 [前端日志] %s: %s", level, message)
	}
}

// 添加匹配结果事件发送方法
func (a *PcapAnalyzer) sendMatchEvent(result AnalysisResult) {
	if a.ctx != nil {
		// 添加时间戳
		result.Timestamp = time.Now().Format("15:04:05")
		runtime.EventsEmit(a.ctx, "match_event", result)
		//log.Printf("🎯 [匹配事件] 发送匹配结果: %s - %s", result.Type, result.Content)
	}
}

func (a *PcapAnalyzer) sendFileEvent(file ExtractionResult) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "file_extracted", file)
		//log.Printf("📁 [文件事件] 发送文件: %s", file.FileName)
	}
}

func (a *PcapAnalyzer) sendStatusEvent(packets, matches int) {
	if a.ctx != nil {
		status := map[string]interface{}{
			"packets":  packets,
			"matches":  matches,
			"progress": fmt.Sprintf("已处理 %d 个数据包", packets),
		}
		runtime.EventsEmit(a.ctx, "analysis_status", status)
	}
}

// 截断字符串用于日志显示
func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length] + "..."
}

// 获取结果类型标签
func (a *PcapAnalyzer) getResultTypeLabel(typeStr string) string {
	labels := map[string]string{
		"flag":       "Flag",
		"sql":        "SQL语句",
		"credential": "登录凭据",
		"key":        "密钥",
	}
	if label, exists := labels[typeStr]; exists {
		return label
	}
	return typeStr
}

// 辅助函数

func (a *PcapAnalyzer) errorResult(message string) string {
	result := map[string]interface{}{
		"error": message,
	}
	jsonData, _ := json.Marshal(result)
	return string(jsonData)
}

func findTshark() (string, error) {
	// 检查常见安装路径
	possiblePaths := []string{
		"tshark",
		"/usr/bin/tshark",
		"/usr/local/bin/tshark",
		"/opt/homebrew/bin/tshark",
		"C:\\Program Files\\Wireshark\\tshark.exe",
		"C:\\Program Files (x86)\\Wireshark\\tshark.exe",
	}

	for _, path := range possiblePaths {
		if _, err := exec.LookPath(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("tshark not found in PATH or common locations")
}

func isValidCredential(cred string) bool {
	// 过滤掉明显不是凭据的匹配
	if len(cred) < 3 || len(cred) > 50 {
		return false
	}

	// 检查是否包含空格
	if strings.Contains(cred, " ") {
		return false
	}

	// 检查是否全是数字
	if isAllDigits(cred) {
		return false
	}

	return true
}

func isValidKey(key string) bool {
	if len(key) < 10 || len(key) > 100 {
		return false
	}

	// 检查是否包含空格
	if strings.Contains(key, " ") {
		return false
	}

	// 检查是否全是相同字符
	if isRepeatedChar(key) {
		return false
	}

	return true
}

func isAllDigits(s string) bool {
	for _, c := range s {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}

func isRepeatedChar(s string) bool {
	if len(s) == 0 {
		return false
	}
	first := s[0]
	for i := 1; i < len(s); i++ {
		if s[i] != first {
			return false
		}
	}
	return true
}

func (a *PcapAnalyzer) detectFileType(filePath string) string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return a.detectFileTypeByExtension(filePath)
	}
	return a.detectFileTypeByContent(content)
}

func (a *PcapAnalyzer) detectFileTypeByExtension(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".png":
		return "PNG图像"
	case ".jpg", ".jpeg", ".jfif":
		return "JPEG图像"
	case ".gif":
		return "GIF图像"
	case ".bmp":
		return "BMP图像"
	case ".pdf":
		return "PDF文档"
	case ".zip", ".rar", ".7z", ".tar", ".gz":
		return "压缩包"
	case ".doc", ".docx":
		return "Word文档"
	case ".xls", ".xlsx":
		return "Excel文档"
	case ".ppt", ".pptx":
		return "PowerPoint文档"
	case ".txt", ".log":
		return "文本文件"
	case ".exe", ".dll":
		return "可执行文件"
	case ".html", ".htm":
		return "网页文件"
	case ".xml", ".json":
		return "数据文件"
	case ".mp3", ".wav":
		return "音频文件"
	case ".mp4", ".avi":
		return "视频文件"
	default:
		return "未知文件"
	}
}

func (a *PcapAnalyzer) detectFileTypeByContent(content []byte) string {
	if len(content) < 4 {
		return "未知文件"
	}

	// 检查文件魔数
	magic := string(content[:8])

	switch {
	case strings.HasPrefix(magic, "\x89PNG\r\n\x1a\n"):
		return "PNG图像"
	case strings.HasPrefix(magic, "\xff\xd8\xff"):
		return "JPEG图像"
	case strings.HasPrefix(magic, "GIF8"):
		return "GIF图像"
	case strings.HasPrefix(magic, "BM"):
		return "BMP图像"
	case strings.HasPrefix(magic, "%PDF"):
		return "PDF文档"
	case strings.HasPrefix(magic, "PK\x03\x04"):
		return "ZIP压缩包"
	case strings.HasPrefix(magic, "Rar!\x1a\x07"):
		return "RAR压缩包"
	case strings.HasPrefix(magic, "\x7fELF"):
		return "可执行文件"
	case strings.HasPrefix(magic, "MZ"):
		return "Windows可执行文件"
	case strings.HasPrefix(string(content[:5]), "<html") || strings.HasPrefix(string(content[:9]), "<!DOCTYPE"):
		return "HTML文档"
	case strings.HasPrefix(string(content[:5]), "<?xml"):
		return "XML文档"
	case strings.HasPrefix(string(content[:1]), "{") || strings.HasPrefix(string(content[:1]), "["):
		// 尝试解析JSON
		var js json.RawMessage
		if json.Unmarshal(content, &js) == nil {
			return "JSON文档"
		}
	}

	// 检查文本文件
	if a.isTextContent(content) {
		return "文本文件"
	}

	return "二进制文件"
}

func (a *PcapAnalyzer) detectContentType(filePath string) string {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "unknown"
	}

	// 简单的Content-Type检测
	fileType := a.detectFileTypeByContent(content)
	switch fileType {
	case "PNG图像":
		return "image/png"
	case "JPEG图像":
		return "image/jpeg"
	case "GIF图像":
		return "image/gif"
	case "PDF文档":
		return "application/pdf"
	case "HTML文档":
		return "text/html"
	case "文本文件":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}

func (a *PcapAnalyzer) isTextContent(content []byte) bool {
	if len(content) == 0 {
		return false
	}

	// 检查前1KB的内容是否主要是可打印字符
	checkSize := len(content)
	if checkSize > 1024 {
		checkSize = 1024
	}

	printableCount := 0
	controlCount := 0

	for i := 0; i < checkSize; i++ {
		b := content[i]
		if b >= 32 && b <= 126 || b == 9 || b == 10 || b == 13 {
			printableCount++
		} else if b < 32 || b == 127 {
			controlCount++
		}
	}

	// 如果控制字符太多，可能不是文本
	if controlCount > checkSize/10 {
		return false
	}

	return float32(printableCount)/float32(checkSize) > 0.8
}

func (a *PcapAnalyzer) getTextPreview(content []byte) string {
	if !a.isTextContent(content) {
		// 显示十六进制预览
		previewSize := 100
		if len(content) < previewSize {
			previewSize = len(content)
		}
		return "⚠️ 二进制文件，十六进制预览:\n" + hex.Dump(content[:previewSize])
	}

	previewSize := 2000
	if len(content) < previewSize {
		previewSize = len(content)
	}

	text := string(content[:previewSize])

	// 清理文本，移除控制字符（保留换行和制表符）
	text = strings.Map(func(r rune) rune {
		if r == 10 || r == 13 || r == 9 || (r >= 32 && r != 127) {
			return r
		}
		return -1
	}, text)

	if len(text) > 500 {
		text = text[:500] + "\n... (内容已截断)"
	}

	return text
}

func formatFileSize(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}
	const unit = 1024
	sizes := []string{"B", "KB", "MB", "GB"}
	i := 0
	for bytes >= unit && i < len(sizes)-1 {
		bytes /= unit
		i++
	}
	return strconv.FormatInt(bytes, 10) + " " + sizes[i]
}

func calculateMD5(data []byte) string {
	if len(data) == 0 {
		return "N/A"
	}
	hash := md5.Sum(data)
	return fmt.Sprintf("%x", hash)
}
