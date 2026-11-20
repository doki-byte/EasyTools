package system

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"EasyTools/app/controller/proxy"
)

// AIConfig AI大模型配置
type AIConfig struct {
	ID          string    `json:"id"`          // 配置ID
	Name        string    `json:"name"`        // 配置名称
	Provider    string    `json:"provider"`    // 提供商：openai, deepseek, qianwen, custom
	APIKey      string    `json:"api_key"`     // API密钥
	APIBase     string    `json:"api_base"`    // API基础URL
	Model       string    `json:"model"`       // 模型名称
	Enabled     bool      `json:"enabled"`     // 是否启用
	IsDefault   bool      `json:"is_default"`  // 是否默认
	Description string    `json:"description"` // 描述
	CreatedAt   time.Time `json:"created_at"`  // 创建时间
	UpdatedAt   time.Time `json:"updated_at"`  // 更新时间
}

// AIConfigManager AI配置管理器
type AIConfigManager struct {
	Base
	configs    []AIConfig
	configsMu  sync.RWMutex
	configPath string
}

var (
	aiConfigManagerOnce sync.Once
	aiConfigManager     *AIConfigManager
)

// GetAIConfigManager 获取AI配置管理器单例
func GetAIConfigManager() *AIConfigManager {
	aiConfigManagerOnce.Do(func() {
		baseDir := GetAppBaseDir()
		aiConfigManager = &AIConfigManager{
			configs:    []AIConfig{},
			configPath: filepath.Join(baseDir, "config", "ai_config.json"),
		}
		aiConfigManager.loadConfigs()
	})
	return aiConfigManager
}

// NewAIConfigManager 创建AI配置管理器
func NewAIConfigManager() *AIConfigManager {
	return GetAIConfigManager()
}

// Startup 启动时设置上下文
func (acm *AIConfigManager) Startup(ctx context.Context) {
	acm.Ctx = ctx
}

// loadConfigs 加载配置
func (acm *AIConfigManager) loadConfigs() {
	acm.configsMu.Lock()
	defer acm.configsMu.Unlock()

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(acm.configPath), 0755); err != nil {
		return
	}

	// 读取配置文件
	data, err := os.ReadFile(acm.configPath)
	if err != nil {
		// 文件不存在，创建默认配置
		acm.configs = acm.getDefaultConfigs()
		acm.saveConfigs()
		return
	}

	// 解析JSON
	if err := json.Unmarshal(data, &acm.configs); err != nil {
		acm.configs = acm.getDefaultConfigs()
		acm.saveConfigs()
		return
	}

	// 移除已弃用的旧版 AI 提供商条目（历史 JSON 中 provider 为 "iflow"）并回写
	if acm.stripDeprecatedAIProvidersLocked() {
		if len(acm.configs) == 0 {
			acm.configs = acm.getDefaultConfigs()
		}
		_ = acm.saveConfigs()
	}

	// 如果没有配置，创建默认配置
	if len(acm.configs) == 0 {
		acm.configs = acm.getDefaultConfigs()
		acm.saveConfigs()
	}
}

func isDeprecatedAIProvider(provider string) bool {
	return strings.EqualFold(strings.TrimSpace(provider), "iflow")
}

// stripDeprecatedAIProvidersLocked 从已加载列表中删除已弃用的提供商配置（须在持有 configsMu 写锁时调用）
func (acm *AIConfigManager) stripDeprecatedAIProvidersLocked() bool {
	var next []AIConfig
	changed := false
	for _, c := range acm.configs {
		if isDeprecatedAIProvider(c.Provider) {
			changed = true
			continue
		}
		next = append(next, c)
	}
	if changed {
		acm.configs = next
	}
	return changed
}

// saveConfigs 保存配置
func (acm *AIConfigManager) saveConfigs() error {
	data, err := json.MarshalIndent(acm.configs, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(acm.configPath, data, 0644)
}

// getDefaultConfigs 获取默认配置
func (acm *AIConfigManager) getDefaultConfigs() []AIConfig {
	return []AIConfig{
		{
			ID:          "openai-default",
			Name:        "OpenAI",
			Provider:    "openai",
			APIKey:      "",
			APIBase:     "https://api.openai.com/v1",
			Model:       "gpt-3.5-turbo",
			Enabled:     false,
			IsDefault:   false,
			Description: "OpenAI API配置",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		},
	}
}

// GetAllConfigs 获取所有配置
func (acm *AIConfigManager) GetAllConfigs() (string, error) {
	acm.configsMu.RLock()
	defer acm.configsMu.RUnlock()

	data, err := json.Marshal(acm.configs)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// GetConfig 获取指定配置
func (acm *AIConfigManager) GetConfig(configID string) (string, error) {
	acm.configsMu.RLock()
	defer acm.configsMu.RUnlock()

	for _, config := range acm.configs {
		if config.ID == configID {
			data, err := json.Marshal(config)
			if err != nil {
				return "", err
			}
			return string(data), nil
		}
	}

	return "", fmt.Errorf("配置不存在: %s", configID)
}

// GetDefaultConfig 获取默认配置
func (acm *AIConfigManager) GetDefaultConfig() (string, error) {
	acm.configsMu.RLock()
	defer acm.configsMu.RUnlock()

	for _, config := range acm.configs {
		if config.IsDefault && config.Enabled {
			data, err := json.Marshal(config)
			if err != nil {
				return "", err
			}
			return string(data), nil
		}
	}

	return "", fmt.Errorf("未找到默认配置")
}

// AddConfig 添加配置
func (acm *AIConfigManager) AddConfig(configJSON string) error {
	var config AIConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return fmt.Errorf("解析配置失败: %v", err)
	}
	if isDeprecatedAIProvider(config.Provider) {
		return fmt.Errorf("该提供商已移除，请使用 OpenAI、DeepSeek、通义千问或自定义(OpenAI兼容)")
	}

	acm.configsMu.Lock()
	defer acm.configsMu.Unlock()

	// 生成ID
	if config.ID == "" {
		config.ID = fmt.Sprintf("config-%d", time.Now().UnixNano())
	}

	// 设置时间
	if config.CreatedAt.IsZero() {
		config.CreatedAt = time.Now()
	}
	config.UpdatedAt = time.Now()

	// 如果设置为默认，取消其他默认配置
	if config.IsDefault {
		for i := range acm.configs {
			acm.configs[i].IsDefault = false
		}
	}

	acm.configs = append(acm.configs, config)

	if err := acm.saveConfigs(); err != nil {
		return err
	}

	// 发送事件
	if acm.Ctx != nil {
		runtime.EventsEmit(acm.Ctx, "ai-config-updated")
	}

	return nil
}

// UpdateConfig 更新配置
func (acm *AIConfigManager) UpdateConfig(configJSON string) error {
	var config AIConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return fmt.Errorf("解析配置失败: %v", err)
	}
	if isDeprecatedAIProvider(config.Provider) {
		return fmt.Errorf("该提供商已移除，请使用 OpenAI、DeepSeek、通义千问或自定义(OpenAI兼容)")
	}

	acm.configsMu.Lock()
	defer acm.configsMu.Unlock()

	config.UpdatedAt = time.Now()

	// 如果设置为默认，取消其他默认配置
	if config.IsDefault {
		for i := range acm.configs {
			if acm.configs[i].ID != config.ID {
				acm.configs[i].IsDefault = false
			}
		}
	}

	// 更新配置
	found := false
	for i := range acm.configs {
		if acm.configs[i].ID == config.ID {
			acm.configs[i] = config
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("配置不存在: %s", config.ID)
	}

	if err := acm.saveConfigs(); err != nil {
		return err
	}

	// 发送事件
	if acm.Ctx != nil {
		runtime.EventsEmit(acm.Ctx, "ai-config-updated")
	}

	return nil
}

// DeleteConfig 删除配置
func (acm *AIConfigManager) DeleteConfig(configID string) error {
	acm.configsMu.Lock()
	defer acm.configsMu.Unlock()

	var newConfigs []AIConfig
	found := false

	for _, config := range acm.configs {
		if config.ID == configID {
			found = true
			continue
		}
		newConfigs = append(newConfigs, config)
	}

	if !found {
		return fmt.Errorf("配置不存在: %s", configID)
	}

	acm.configs = newConfigs

	if err := acm.saveConfigs(); err != nil {
		return err
	}

	// 发送事件
	if acm.Ctx != nil {
		runtime.EventsEmit(acm.Ctx, "ai-config-updated")
	}

	return nil
}

// SetDefaultConfig 设置默认配置
func (acm *AIConfigManager) SetDefaultConfig(configID string) error {
	acm.configsMu.Lock()
	defer acm.configsMu.Unlock()

	found := false
	for i := range acm.configs {
		if acm.configs[i].ID == configID {
			acm.configs[i].IsDefault = true
			acm.configs[i].UpdatedAt = time.Now()
			found = true
		} else {
			acm.configs[i].IsDefault = false
		}
	}

	if !found {
		return fmt.Errorf("配置不存在: %s", configID)
	}

	if err := acm.saveConfigs(); err != nil {
		return err
	}

	// 发送事件
	if acm.Ctx != nil {
		runtime.EventsEmit(acm.Ctx, "ai-config-updated")
	}

	return nil
}

// TestConfig 测试配置
func (acm *AIConfigManager) TestConfig(configJSON string) (string, error) {
	var config AIConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return "", fmt.Errorf("解析配置失败: %v", err)
	}
	if isDeprecatedAIProvider(config.Provider) {
		return "", fmt.Errorf("该提供商已移除，请改用 OpenAI、DeepSeek、通义千问或自定义(OpenAI兼容)")
	}

	// 验证必填字段
	if config.APIKey == "" {
		return "", fmt.Errorf("API密钥不能为空")
	}

	// 根据提供商调用不同的API进行测试
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var apiURL string
	var reqBody map[string]interface{}
	var modelName string

	switch config.Provider {
	case "openai":
		apiURL = "https://api.openai.com/v1/chat/completions"
		if config.APIBase != "" {
			// 如果用户输入的是base URL，需要拼接完整路径
			if strings.HasSuffix(config.APIBase, "/v1") || strings.HasSuffix(config.APIBase, "/v1/") {
				apiURL = strings.TrimSuffix(config.APIBase, "/") + "/chat/completions"
			} else if !strings.Contains(config.APIBase, "/chat/completions") {
				apiURL = strings.TrimSuffix(config.APIBase, "/") + "/chat/completions"
			} else {
				apiURL = config.APIBase
			}
		}
		modelName = config.Model
		if modelName == "" {
			modelName = "gpt-3.5-turbo"
		}
		reqBody = map[string]interface{}{
			"model": modelName,
			"messages": []map[string]string{
				{
					"role":    "user",
					"content": "test",
				},
			},
			"max_tokens": 5,
		}
	case "deepseek":
		apiURL = "https://api.deepseek.com/v1/chat/completions"
		if config.APIBase != "" {
			// 如果用户输入的是base URL，需要拼接完整路径
			if strings.HasSuffix(config.APIBase, "/v1") || strings.HasSuffix(config.APIBase, "/v1/") {
				apiURL = strings.TrimSuffix(config.APIBase, "/") + "/chat/completions"
			} else if !strings.Contains(config.APIBase, "/chat/completions") {
				apiURL = strings.TrimSuffix(config.APIBase, "/") + "/chat/completions"
			} else {
				apiURL = config.APIBase
			}
		}
		modelName = config.Model
		if modelName == "" {
			modelName = "deepseek-chat"
		}
		reqBody = map[string]interface{}{
			"model": modelName,
			"messages": []map[string]string{
				{
					"role":    "user",
					"content": "test",
				},
			},
			"max_tokens": 5,
			"stream":     false,
		}
	case "qianwen":
		// 阿里云千机百炼API地址
		apiURL = "https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation"
		if config.APIBase != "" {
			apiURL = config.APIBase
		}
		modelName = config.Model
		if modelName == "" {
			modelName = "qwen-turbo"
		}
		// 阿里云千机百炼使用不同的请求格式
		reqBody = map[string]interface{}{
			"model": modelName,
			"input": map[string]interface{}{
				"messages": []map[string]string{
					{
						"role":    "user",
						"content": "test",
					},
				},
			},
			"parameters": map[string]interface{}{
				"max_tokens": 5,
			},
		}
	case "custom":
		if config.APIBase == "" {
			return "", fmt.Errorf("自定义API需要配置API Base URL")
		}
		apiURL = config.APIBase
		// 如果用户输入的是base URL，尝试拼接完整路径
		if strings.HasSuffix(apiURL, "/v1") || strings.HasSuffix(apiURL, "/v1/") {
			apiURL = strings.TrimSuffix(apiURL, "/") + "/chat/completions"
		} else if !strings.Contains(apiURL, "/chat/completions") && !strings.Contains(apiURL, "/generation") {
			apiURL = strings.TrimSuffix(apiURL, "/") + "/chat/completions"
		}
		modelName = config.Model
		if modelName == "" {
			modelName = "gpt-3.5-turbo"
		}
		reqBody = map[string]interface{}{
			"model": modelName,
			"messages": []map[string]string{
				{
					"role":    "user",
					"content": "test",
				},
			},
			"max_tokens": 5,
		}
	default:
		return "", fmt.Errorf("不支持的AI提供商: %s", config.Provider)
	}

	// 构建请求
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("构建请求失败: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.APIKey)

	// 发送请求（使用代理配置）
	client := proxy.GlobalProxyManager.GetHTTPClient()
	client.Timeout = 30 * time.Second
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, _ := io.ReadAll(resp.Body)

	// 检查状态码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	// 验证响应格式
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("API响应格式错误: %v", err)
	}

	// 根据提供商验证不同的响应格式
	if config.Provider == "qianwen" {
		// 阿里云千机百炼的响应格式
		output, ok := result["output"].(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("API响应格式错误: 缺少output字段")
		}
		choices, ok := output["choices"].([]interface{})
		if !ok || len(choices) == 0 {
			return "", fmt.Errorf("API响应格式错误: 缺少choices字段")
		}
	} else {
		// OpenAI兼容格式（OpenAI、DeepSeek、Custom）
		choices, ok := result["choices"].([]interface{})
		if !ok || len(choices) == 0 {
			return "", fmt.Errorf("API响应格式错误: 缺少choices字段")
		}
	}

	return fmt.Sprintf("配置测试成功，模型: %s", modelName), nil
}

// resolveOpenAIStyleChatURL 将 api_base 解析为 chat/completions 完整地址（OpenAI 兼容）
func resolveOpenAIStyleChatURL(apiBase, defaultV1Base string) (string, error) {
	base := strings.TrimSpace(apiBase)
	if base == "" {
		base = defaultV1Base
	}
	if base == "" {
		return "", fmt.Errorf("未配置 API Base URL")
	}
	if strings.HasSuffix(base, "/v1") || strings.HasSuffix(base, "/v1/") {
		return strings.TrimSuffix(base, "/") + "/chat/completions", nil
	}
	if strings.Contains(base, "/chat/completions") {
		return base, nil
	}
	return strings.TrimSuffix(base, "/") + "/chat/completions", nil
}

// ChatCompletion 使用系统管理中的配置发起对话补全（OpenAI 兼容协议或阿里云千问）
func ChatCompletion(ctx context.Context, config AIConfig, messages []map[string]string, maxTokens int, temperature float64) (string, error) {
	if !config.Enabled {
		return "", fmt.Errorf("AI配置未启用")
	}
	if config.APIKey == "" {
		return "", fmt.Errorf("API密钥未配置")
	}
	if len(messages) == 0 {
		return "", fmt.Errorf("消息不能为空")
	}
	if maxTokens <= 0 {
		maxTokens = 2048
	}
	if temperature < 0 {
		temperature = 0.7
	}

	var apiURL string
	var reqBody map[string]interface{}
	modelName := strings.TrimSpace(config.Model)

	switch config.Provider {
	case "openai":
		u, err := resolveOpenAIStyleChatURL(config.APIBase, "https://api.openai.com/v1")
		if err != nil {
			return "", err
		}
		apiURL = u
		if modelName == "" {
			modelName = "gpt-3.5-turbo"
		}
		reqBody = map[string]interface{}{
			"model":       modelName,
			"messages":    messages,
			"max_tokens":  maxTokens,
			"temperature": temperature,
		}
	case "deepseek":
		u, err := resolveOpenAIStyleChatURL(config.APIBase, "https://api.deepseek.com/v1")
		if err != nil {
			return "", err
		}
		apiURL = u
		if modelName == "" {
			modelName = "deepseek-chat"
		}
		reqBody = map[string]interface{}{
			"model":       modelName,
			"messages":    messages,
			"max_tokens":  maxTokens,
			"temperature": temperature,
			"stream":      false,
		}
	case "custom":
		if strings.TrimSpace(config.APIBase) == "" {
			return "", fmt.Errorf("自定义API需要配置API Base URL")
		}
		u, err := resolveOpenAIStyleChatURL(config.APIBase, "")
		if err != nil {
			return "", err
		}
		apiURL = u
		if modelName == "" {
			modelName = "gpt-3.5-turbo"
		}
		reqBody = map[string]interface{}{
			"model":       modelName,
			"messages":    messages,
			"max_tokens":  maxTokens,
			"temperature": temperature,
		}
	case "qianwen":
		apiURL = "https://dashscope.aliyuncs.com/api/v1/services/aigc/text-generation/generation"
		if strings.TrimSpace(config.APIBase) != "" {
			apiURL = config.APIBase
		}
		if modelName == "" {
			modelName = "qwen-turbo"
		}
		reqBody = map[string]interface{}{
			"model": modelName,
			"input": map[string]interface{}{
				"messages": messages,
			},
			"parameters": map[string]interface{}{
				"max_tokens":  maxTokens,
				"temperature": temperature,
			},
		}
	default:
		return "", fmt.Errorf("不支持的AI提供商: %s", config.Provider)
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("构建请求失败: %v", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.APIKey)

	client := proxy.GlobalProxyManager.GetHTTPClient()
	client.Timeout = 120 * time.Second
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("API响应格式错误: %v", err)
	}

	if config.Provider == "qianwen" {
		output, ok := result["output"].(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("API响应格式错误: 缺少output字段")
		}
		choices, ok := output["choices"].([]interface{})
		if !ok || len(choices) == 0 {
			return "", fmt.Errorf("API响应格式错误: 缺少choices字段")
		}
		choice, ok := choices[0].(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("API响应格式错误")
		}
		message, ok := choice["message"].(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("API响应格式错误")
		}
		content, ok := message["content"].(string)
		if !ok {
			return "", fmt.Errorf("API响应格式错误: 内容非文本")
		}
		return content, nil
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("API响应格式错误: 缺少choices字段")
	}
	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("API响应格式错误")
	}
	message, ok := choice["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("API响应格式错误")
	}
	content, ok := message["content"].(string)
	if !ok {
		return "", fmt.Errorf("API响应格式错误: 内容非文本")
	}
	return content, nil
}
