package crack

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ExecuteRabbitMQCommand 精简版 RabbitMQ Management API 命令（支持SSH隧道）
//
// 说明：
// - 只做管理 API 的 GET 查询，不做 AMQP 协议操作（避免额外依赖）
// - 依赖记录中已有 username/password（爆破结果）作为 BasicAuth
// - 支持命令：overview, connections, channels, exchanges, queues, nodes
func ExecuteRabbitMQCommand(host, port, username, password string, sshConfig *SSHConfig, command string) (string, error) {
	cmd := strings.TrimSpace(strings.ToLower(command))
	if cmd == "" {
		return "", fmt.Errorf("命令为空")
	}

	// 管理端口：15672 常见（15671 TLS），若爆破结果里 port 是 AMQP(5672)，这里默认切到 15672
	managementPort := port
	if managementPort == "" || managementPort == "5672" || managementPort == "5671" {
		managementPort = "15672"
	}

	pathMap := map[string]string{
		"overview":    "/api/overview",
		"connections": "/api/connections",
		"channels":    "/api/channels",
		"exchanges":   "/api/exchanges",
		"queues":      "/api/queues",
		"nodes":       "/api/nodes",
	}
	apiPath, ok := pathMap[cmd]
	if !ok {
		return "", fmt.Errorf("不支持的命令: %s\n支持的命令: overview, connections, channels, exchanges, queues, nodes", command)
	}

	targetURL := fmt.Sprintf("http://%s%s", net.JoinHostPort(host, managementPort), apiPath)

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialWithSSHContext(ctx, sshConfig, network, addr, 10*time.Second)
		},
		ResponseHeaderTimeout: 10 * time.Second,
	}
	client := &http.Client{Transport: transport, Timeout: 15 * time.Second}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建请求失败: %v", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "EasyTools-RabbitMQ-Client/1.0")
	if username != "" || password != "" {
		req.SetBasicAuth(username, password)
	}

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

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("请求失败 (HTTP %d)\n\n%s", resp.StatusCode, body)
	}

	// 尝试把 JSON 缩进一下，便于展示
	var pretty any
	if json.Unmarshal(bodyBytes, &pretty) == nil {
		if formatted, e := json.MarshalIndent(pretty, "", "  "); e == nil {
			body = string(formatted)
		}
	}

	return fmt.Sprintf("HTTP %d (%s)\n\n%s", resp.StatusCode, targetURL, body), nil
}

// RabbitMQScan RabbitMQ协议爆破（使用并发模式）
func RabbitMQScan(ctx, ctrlCtx context.Context, taskId, host string, usernames, passwords []string, sshConfig *SSHConfig, config *CrackConfig) {
	totalAttempts := len(usernames) * len(passwords)
	var attempted int64

	workers := config.Threads
	if workers <= 0 {
		workers = 5
	}

	timeout := time.Duration(config.Timeout) * time.Millisecond
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	sendCrackLog(ctx, "info", fmt.Sprintf("RabbitMQ爆破开始: 目标=%s, 用户名数量=%d, 密码数量=%d, 总尝试次数=%d, 并发数=%d, 超时=%v", host, len(usernames), len(passwords), totalAttempts, workers, timeout))

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

				flag, err := RabbitMQConn(host, t.User, t.Pass, sshConfig)
				if flag && err == nil {
					vulnURL := fmt.Sprintf("rabbitmq://%s", host)
					credential := fmt.Sprintf("%s/%s", t.User, t.Pass)
					runtime.EventsEmit(ctx, "nucleiResult", map[string]interface{}{
						"task_id":     taskId,
						"template_id": "rabbitmq-weak-password",
						"vuln_name":   "RabbitMQ弱口令",
						"vuln_url":    vulnURL,
						"severity":    "HIGH",
						"protocol":    "RabbitMQ",
						"extract":     credential,
					})

					emitCrackResult(ctx, taskId, "RabbitMQ", vulnURL, credential, t.User, t.Pass)

					saveCrackResultToDB(
						ctx,
						taskId,
						"rabbitmq-weak-password",
						"RabbitMQ弱口令",
						vulnURL,
						"HIGH",
						"RabbitMQ",
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
			sendCrackLog(ctx, "warning", fmt.Sprintf("RabbitMQ爆破超时: 目标=%s, 已尝试=%d次", host, atomic.LoadInt64(&attempted)))
			return
		}
	}

	finalAttempted := atomic.LoadInt64(&attempted)
	if !success {
		sendCrackLog(ctx, "info", fmt.Sprintf("RabbitMQ爆破完成: 目标=%s, 已尝试=%d次, 未发现弱口令", host, finalAttempted))
	}
}

// RabbitMQConn 测试RabbitMQ连接（AMQP协议，支持SSH隧道）
func RabbitMQConn(address, username, password string, sshConfig *SSHConfig) (bool, error) {
	// RabbitMQ使用AMQP协议，这里通过HTTP Management API进行认证测试
	// 如果端口是5672（AMQP），尝试访问15672（Management API）
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		host = address
		port = "5672"
	}

	// 尝试Management API端口
	managementPort := port
	if port == "5672" || port == "5671" || port == "" {
		managementPort = "15672"
	}

	targetURL := fmt.Sprintf("http://%s/api/whoami", net.JoinHostPort(host, managementPort))

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialWithSSHContext(ctx, sshConfig, network, addr, 10*time.Second)
		},
		ResponseHeaderTimeout: 10 * time.Second,
	}
	client := &http.Client{Transport: transport, Timeout: 10 * time.Second}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Accept", "application/json")
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
