package crack

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ExecuteEtcdCommand 执行 etcd HTTP(v2) 精简命令（支持SSH隧道）
//
// 说明：
// - 为了避免引入 etcd gRPC 客户端依赖，这里使用 v2 HTTP API（许多未授权场景仍然可用）
// - 支持命令（参考 MPET-main 语义，做精简）：status, version, list [prefix], get <key> [--recursive], put <key> <value>, del <key>
func ExecuteEtcdCommand(host, port string, sshConfig *SSHConfig, command string) (string, error) {
	cmdline := strings.TrimSpace(command)
	if cmdline == "" {
		return "", fmt.Errorf("命令不能为空")
	}

	parts := strings.Fields(cmdline)
	if len(parts) == 0 {
		return "", fmt.Errorf("无效的命令")
	}

	if port == "" {
		port = "2379"
	}

	// 支持 host 传入 http(s)://host:port
	scheme := "http"
	hostInput := strings.TrimSpace(host)
	lower := strings.ToLower(hostInput)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		if parsed, err := url.Parse(hostInput); err == nil {
			scheme = parsed.Scheme
			hostInput = parsed.Hostname()
			if parsed.Port() != "" {
				port = parsed.Port()
			}
		}
	}

	baseURL := fmt.Sprintf("%s://%s", scheme, net.JoinHostPort(hostInput, port))

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialWithSSHContext(ctx, sshConfig, network, addr, 10*time.Second)
		},
		ResponseHeaderTimeout: 10 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		DisableKeepAlives:     true,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: transport, Timeout: 15 * time.Second}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	verb := strings.ToLower(parts[0])
	switch verb {
	case "version":
		return httpGet(ctx, client, baseURL+"/version")

	case "status":
		// etcd v2 常见：/v2/stats/self 或 /v2/stats/leader
		self, _ := httpGet(ctx, client, baseURL+"/v2/stats/self")
		leader, _ := httpGet(ctx, client, baseURL+"/v2/stats/leader")
		var b strings.Builder
		b.WriteString("=== /v2/stats/self ===\n")
		b.WriteString(self)
		b.WriteString("\n\n=== /v2/stats/leader ===\n")
		b.WriteString(leader)
		return b.String(), nil

	case "list", "ls":
		prefix := "/"
		if len(parts) > 1 {
			prefix = parts[1]
		}
		u := baseURL + "/v2/keys" + ensureLeadingSlash(prefix)
		// 列目录通常用 ?recursive=true&sorted=true（输出可能很大，用户可自行限制前缀）
		u += "?recursive=true&sorted=true"
		return httpGet(ctx, client, u)

	case "get":
		if len(parts) < 2 {
			return "", fmt.Errorf("用法: get <key> [--recursive]")
		}
		key := parts[1]
		u := baseURL + "/v2/keys" + ensureLeadingSlash(key)
		if len(parts) > 2 && parts[2] == "--recursive" {
			u += "?recursive=true"
		}
		return httpGet(ctx, client, u)

	case "put":
		if len(parts) < 3 {
			return "", fmt.Errorf("用法: put <key> <value>")
		}
		key := parts[1]
		value := strings.Join(parts[2:], " ")
		u := baseURL + "/v2/keys" + ensureLeadingSlash(key)
		return httpForm(ctx, client, http.MethodPut, u, "value="+url.QueryEscape(value))

	case "del", "delete":
		if len(parts) < 2 {
			return "", fmt.Errorf("用法: del <key>")
		}
		key := parts[1]
		u := baseURL + "/v2/keys" + ensureLeadingSlash(key)
		return httpForm(ctx, client, http.MethodDelete, u, "")

	default:
		return "", fmt.Errorf("不支持的命令: %s\n支持的命令: status, version, list [prefix], get <key> [--recursive], put <key> <value>, del <key>", verb)
	}
}

func ensureLeadingSlash(s string) string {
	if s == "" {
		return "/"
	}
	if strings.HasPrefix(s, "/") {
		return s
	}
	return "/" + s
}

func httpGet(ctx context.Context, client *http.Client, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("User-Agent", "EasyTools-Etcd-Client/1.0")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	body := strings.TrimSpace(string(bodyBytes))
	if body == "" {
		body = "(响应为空)"
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return fmt.Sprintf("HTTP %d\n\n%s", resp.StatusCode, body), nil
	}
	return "", fmt.Errorf("请求失败 (HTTP %d)\n\n%s", resp.StatusCode, body)
}

func httpForm(ctx context.Context, client *http.Client, method, url string, form string) (string, error) {
	var body io.Reader
	if form != "" {
		body = strings.NewReader(form)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("User-Agent", "EasyTools-Etcd-Client/1.0")
	if form != "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := strings.TrimSpace(string(bodyBytes))
	if bodyStr == "" {
		bodyStr = "(响应为空)"
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return fmt.Sprintf("HTTP %d\n\n%s", resp.StatusCode, bodyStr), nil
	}
	return "", fmt.Errorf("请求失败 (HTTP %d)\n\n%s", resp.StatusCode, bodyStr)
}

// EtcdScan Etcd协议爆破（检查未授权访问）
// Etcd默认不需要用户名密码，主要通过HTTP API检查未授权访问
func EtcdScan(ctx, ctrlCtx context.Context, taskId, host string, usernames, passwords []string, sshConfig *SSHConfig, config *CrackConfig) {
	workers := config.Threads
	if workers <= 0 {
		workers = 5
	}

	timeout := time.Duration(config.Timeout) * time.Millisecond
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	// Etcd通常不需要用户名密码，主要检查未授权访问
	flag, err := EtcdUnauth(host, sshConfig)
	if flag && err == nil {
		vulnURL := fmt.Sprintf("etcd://%s", host)
		runtime.EventsEmit(ctx, "nucleiResult", map[string]interface{}{
			"task_id":     taskId,
			"template_id": "etcd-unauthorized",
			"vuln_name":   "Etcd未授权访问",
			"vuln_url":    vulnURL,
			"severity":    "HIGH",
			"protocol":    "Etcd",
		})

		emitCrackResult(ctx, taskId, "Etcd", vulnURL, "", "", "未授权访问")

		saveCrackResultToDB(
			ctx,
			taskId,
			"etcd-unauthorized",
			"Etcd未授权访问",
			vulnURL,
			"HIGH",
			"Etcd",
			"",
			"",
			"",
		)
		return
	}

	// 如果未授权访问失败，记录日志
	sendCrackLog(ctx, "info", fmt.Sprintf("Etcd爆破完成: 目标=%s, 未发现未授权访问", host))
}

// EtcdUnauth 检查Etcd未授权访问（支持SSH隧道）
func EtcdUnauth(address string, sshConfig *SSHConfig) (bool, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		host = address
		port = "2379"
	}

	scheme := "http"
	hostInput := strings.TrimSpace(host)
	lower := strings.ToLower(hostInput)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		if parsed, err := url.Parse(hostInput); err == nil {
			scheme = parsed.Scheme
			hostInput = parsed.Hostname()
			if parsed.Port() != "" {
				port = parsed.Port()
			}
		}
	}

	baseURL := fmt.Sprintf("%s://%s", scheme, net.JoinHostPort(hostInput, port))

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialWithSSHContext(ctx, sshConfig, network, addr, 10*time.Second)
		},
		ResponseHeaderTimeout: 10 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		DisableKeepAlives:     true,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 尝试访问/version端点
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/version", nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("User-Agent", "EasyTools-Etcd-Client/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	// HTTP 200且响应包含etcd版本信息，说明未授权
	if resp.StatusCode == 200 {
		body := string(bodyBytes)
		if strings.Contains(body, "etcd") || strings.Contains(body, "version") {
			return true, nil
		}
	}

	return false, fmt.Errorf("需要认证或连接失败")
}
