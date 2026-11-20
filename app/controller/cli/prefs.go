package cli

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"EasyTools/app/controller/system"
)

// cliExecutorPrefs 持久化 CLI 执行器相关偏好（与独立 mcp stdio 子进程无关）
type cliExecutorPrefs struct {
	MCPHTTPEnabled bool `json:"mcp_http_enabled"` // 主进程 FuzzPort 上 /cli/mcp/* 是否响应，默认 true
}

var (
	prefsMu     sync.RWMutex
	prefs       = cliExecutorPrefs{MCPHTTPEnabled: true}
	prefsPath   string
	prefsOnce   sync.Once
)

func cliPrefsFile() string {
	if prefsPath == "" {
		prefsPath = filepath.Join(system.GetAppBaseDir(), "config", "cli_executor_prefs.json")
	}
	return prefsPath
}

func loadCLIPrefs() {
	prefsMu.Lock()
	defer prefsMu.Unlock()
	data, err := os.ReadFile(cliPrefsFile())
	if err != nil {
		return
	}
	var p cliExecutorPrefs
	if json.Unmarshal(data, &p) != nil {
		return
	}
	prefs = p
}

func ensureCLIPrefsLoaded() {
	prefsOnce.Do(loadCLIPrefs)
}

// IsMCPEndpointEnabled 本地 HTTP MCP（/cli/mcp/tools、/call 等）是否开启
func IsMCPEndpointEnabled() bool {
	ensureCLIPrefsLoaded()
	prefsMu.RLock()
	defer prefsMu.RUnlock()
	return prefs.MCPHTTPEnabled
}

// SetMCPEndpointEnabled 设置并保存；关闭后外部通过 FuzzPort 调用 MCP 将收到 403
func SetMCPEndpointEnabled(enabled bool) error {
	ensureCLIPrefsLoaded()
	prefsMu.Lock()
	prefs.MCPHTTPEnabled = enabled
	toSave := prefs
	prefsMu.Unlock()

	dir := filepath.Dir(cliPrefsFile())
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(toSave, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cliPrefsFile(), data, 0644)
}

func writeMCPEndpointDisabled(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusForbidden)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error": "MCP HTTP 接口已在应用中关闭，请在「CLI 工具 → AI 大模型」中开启",
	})
}
