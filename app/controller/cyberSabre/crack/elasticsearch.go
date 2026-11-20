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
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ExecuteElasticsearchCommand 执行 Elasticsearch API 命令（支持SSH隧道）
//
// command 支持（参考 MPET-main）：health, stats, nodes, indices, shards, allocation, count, settings, version, tasks, pending
func ExecuteElasticsearchCommand(host, port string, sshConfig *SSHConfig, command string) (string, error) {
	validCommands := map[string]string{
		"health":     "/_cluster/health",
		"stats":      "/_cluster/stats",
		"nodes":      "/_nodes",
		"indices":    "/_cat/indices?v",
		"shards":     "/_cat/shards?v",
		"allocation": "/_cat/allocation?v",
		"count":      "/_count",
		"settings":   "/_cluster/settings",
		"version":    "/",
		"tasks":      "/_tasks",
		"pending":    "/_cluster/pending_tasks",
	}

	cmd := strings.TrimSpace(strings.ToLower(command))
	apiPath, ok := validCommands[cmd]
	if !ok {
		return "", fmt.Errorf("不支持的命令: %s\n支持的命令: health, stats, nodes, indices, shards, allocation, count, settings, version, tasks, pending", command)
	}

	if port == "" {
		port = "9200"
	}

	// 允许 host 传入 http(s)://host:port 形式
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

	targetURL := fmt.Sprintf("%s://%s%s", scheme, net.JoinHostPort(hostInput, port), apiPath)

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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Accept", "application/json, text/plain;q=0.9, */*;q=0.8")
	req.Header.Set("User-Agent", "EasyTools-Elasticsearch-Client/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}
	body := strings.TrimSpace(string(bodyBytes))
	if body == "" {
		body = "(响应为空)"
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return fmt.Sprintf("HTTP %d\n\n%s", resp.StatusCode, body), nil
	}
	return "", fmt.Errorf("请求失败 (HTTP %d)\n\n%s", resp.StatusCode, body)
}

// ElasticsearchScan Elasticsearch协议爆破（检查未授权访问）
// Elasticsearch默认不需要用户名密码，主要通过HTTP API检查未授权访问
func ElasticsearchScan(ctx, ctrlCtx context.Context, taskId, host string, usernames, passwords []string, sshConfig *SSHConfig, config *CrackConfig) {
	workers := config.Threads
	if workers <= 0 {
		workers = 5
	}

	timeout := time.Duration(config.Timeout) * time.Millisecond
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	// Elasticsearch通常不需要用户名密码，主要检查未授权访问
	flag, err := ElasticsearchUnauth(host, sshConfig)
	if flag && err == nil {
		vulnURL := fmt.Sprintf("elasticsearch://%s", host)
		runtime.EventsEmit(ctx, "nucleiResult", map[string]interface{}{
			"task_id":     taskId,
			"template_id": "elasticsearch-unauthorized",
			"vuln_name":   "Elasticsearch未授权访问",
			"vuln_url":    vulnURL,
			"severity":    "HIGH",
			"protocol":    "Elasticsearch",
		})

		emitCrackResult(ctx, taskId, "Elasticsearch", vulnURL, "", "", "未授权访问")

		saveCrackResultToDB(
			ctx,
			taskId,
			"elasticsearch-unauthorized",
			"Elasticsearch未授权访问",
			vulnURL,
			"HIGH",
			"Elasticsearch",
			"",
			"",
			"",
		)
		return
	}

	// 如果未授权访问失败，尝试用户名密码认证（如果Elasticsearch启用了安全认证）
	if len(usernames) > 0 && len(passwords) > 0 {
		totalAttempts := len(usernames) * len(passwords)
		var attempted int64

		sendCrackLog(ctx, "info", fmt.Sprintf("Elasticsearch爆破开始: 目标=%s, 用户名数量=%d, 密码数量=%d, 总尝试次数=%d, 并发数=%d, 超时=%v", host, len(usernames), len(passwords), totalAttempts, workers, timeout))

		taskChan := make(chan task, 100)
		doneChan := make(chan struct{})
		successChan := make(chan bool, 1)

		// 启动worker
		for i := 0; i < workers; i++ {
			go func() {
				for t := range taskChan {
					if ctrlCtx.Err() != nil {
						continue
					}

					currentAttempt := atomic.AddInt64(&attempted, 1)

					if currentAttempt%10 == 0 || currentAttempt == 1 {
						sendCrackLog(ctx, "info", fmt.Sprintf("尝试 [%d/%d]: %s/%s", currentAttempt, totalAttempts, t.User, t.Pass))
					}

					flag, err := ElasticsearchConn(host, t.User, t.Pass, sshConfig)
					if flag && err == nil {
						vulnURL := fmt.Sprintf("elasticsearch://%s", host)
						credential := fmt.Sprintf("%s/%s", t.User, t.Pass)
						runtime.EventsEmit(ctx, "nucleiResult", map[string]interface{}{
							"task_id":     taskId,
							"template_id": "elasticsearch-weak-password",
							"vuln_name":   "Elasticsearch弱口令",
							"vuln_url":    vulnURL,
							"severity":    "HIGH",
							"protocol":    "Elasticsearch",
							"extract":     credential,
						})

						emitCrackResult(ctx, taskId, "Elasticsearch", vulnURL, credential, t.User, t.Pass)

						saveCrackResultToDB(
							ctx,
							taskId,
							"elasticsearch-weak-password",
							"Elasticsearch弱口令",
							vulnURL,
							"HIGH",
							"Elasticsearch",
							credential,
							"",
							"",
						)

						select {
						case successChan <- true:
						default:
						}
						if config.OnlyOneAccount {
							return
						}
					}
				}
				doneChan <- struct{}{}
			}()
		}

		// 分发任务
		go func() {
			defer close(taskChan)
			for _, user := range usernames {
				if ctrlCtx.Err() != nil {
					return
				}
				for _, pass := range passwords {
					if ctrlCtx.Err() != nil {
						return
					}
					select {
					case taskChan <- task{User: user, Pass: pass}:
					case <-ctrlCtx.Done():
						return
					case <-successChan:
						if config.OnlyOneAccount {
							return
						}
					}
				}
			}
		}()

		// 等待所有worker完成或成功
		doneCount := 0
		success := false

		for doneCount < workers && !success {
			select {
			case <-successChan:
				if config.OnlyOneAccount {
					success = true
					for doneCount < workers {
						<-doneChan
						doneCount++
					}
					return
				}
			case <-doneChan:
				doneCount++
			case <-time.After(30 * time.Minute):
				sendCrackLog(ctx, "warning", fmt.Sprintf("Elasticsearch爆破超时: 目标=%s, 已尝试=%d次", host, atomic.LoadInt64(&attempted)))
				return
			}
		}

		finalAttempted := atomic.LoadInt64(&attempted)
		if !success {
			sendCrackLog(ctx, "info", fmt.Sprintf("Elasticsearch爆破完成: 目标=%s, 已尝试=%d次, 未发现弱口令", host, finalAttempted))
		}
	} else {
		sendCrackLog(ctx, "info", fmt.Sprintf("Elasticsearch爆破完成: 目标=%s, 未发现未授权访问", host))
	}
}

// ElasticsearchUnauth 检查Elasticsearch未授权访问（支持SSH隧道）
func ElasticsearchUnauth(address string, sshConfig *SSHConfig) (bool, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		host = address
		port = "9200"
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

	targetURL := fmt.Sprintf("%s://%s/", scheme, net.JoinHostPort(hostInput, port))

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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("User-Agent", "EasyTools-Elasticsearch-Client/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	// HTTP 200且响应包含Elasticsearch版本信息，说明未授权
	if resp.StatusCode == 200 {
		body := string(bodyBytes)
		if strings.Contains(body, "version") || strings.Contains(body, "cluster_name") || strings.Contains(body, "name") {
			return true, nil
		}
	}

	return false, fmt.Errorf("需要认证或连接失败")
}

// ElasticsearchConn 测试Elasticsearch连接（带用户名密码，支持SSH隧道）
func ElasticsearchConn(address, username, password string, sshConfig *SSHConfig) (bool, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		host = address
		port = "9200"
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

	targetURL := fmt.Sprintf("%s://%s/", scheme, net.JoinHostPort(hostInput, port))

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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("User-Agent", "EasyTools-Elasticsearch-Client/1.0")
	if username != "" || password != "" {
		req.SetBasicAuth(username, password)
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// HTTP 200表示认证成功
	if resp.StatusCode == 200 {
		return true, nil
	}

	return false, fmt.Errorf("认证失败 (HTTP %d)", resp.StatusCode)
}
