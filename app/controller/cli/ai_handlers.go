package cli

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"EasyTools/app/controller/system"
)

func maskAPIKeyHint(key string) string {
	key = strings.TrimSpace(key)
	if key == "" {
		return ""
	}
	if len(key) <= 4 {
		return "****"
	}
	return "****" + key[len(key)-4:]
}

// GET /cli/ai/default — 返回当前「默认且已启用」的 AI 配置摘要（不含完整密钥）
func defaultAIConfigHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	acm := system.GetAIConfigManager()
	raw, err := acm.GetDefaultConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":      false,
			"message": err.Error(),
		})
		return
	}

	var cfg system.AIConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		http.Error(w, "解析配置失败", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":           true,
		"id":           cfg.ID,
		"name":         cfg.Name,
		"provider":     cfg.Provider,
		"model":        cfg.Model,
		"api_base":     cfg.APIBase,
		"api_key_hint": maskAPIKeyHint(cfg.APIKey),
		"has_api_key":  cfg.APIKey != "",
	})
}

// POST /cli/ai/test — 使用默认配置做一次连通性测试（与系统管理中「测试」一致）
func defaultAITestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	acm := system.GetAIConfigManager()
	raw, err := acm.GetDefaultConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": err.Error()})
		return
	}

	msg, err := acm.TestConfig(raw)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "message": msg})
}

type aiChatRequest struct {
	Messages    []map[string]string `json:"messages"`
	Prompt      string              `json:"prompt"`
	MaxTokens   int                 `json:"max_tokens"`
	Temperature float64             `json:"temperature"`
}

// POST /cli/ai/chat — 使用默认配置调用 OpenAI 兼容对话接口
func defaultAIChatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req aiChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	messages := req.Messages
	if len(messages) == 0 && strings.TrimSpace(req.Prompt) != "" {
		messages = []map[string]string{{"role": "user", "content": strings.TrimSpace(req.Prompt)}}
	}
	if len(messages) == 0 {
		http.Error(w, "messages 或 prompt 必填其一", http.StatusBadRequest)
		return
	}

	acm := system.GetAIConfigManager()
	raw, err := acm.GetDefaultConfig()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": err.Error()})
		return
	}

	var cfg system.AIConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		http.Error(w, "解析配置失败", http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	temp := req.Temperature
	content, err := system.ChatCompletion(ctx, cfg, messages, req.MaxTokens, temp)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "message": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":      true,
		"content": content,
		"model":   cfg.Model,
	})
}
