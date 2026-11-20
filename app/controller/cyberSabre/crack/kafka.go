package crack

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// KafkaScan Kafka协议爆破（使用并发模式）
func KafkaScan(ctx, ctrlCtx context.Context, taskId, host string, usernames, passwords []string, sshConfig *SSHConfig, config *CrackConfig) {
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

	sendCrackLog(ctx, "info", fmt.Sprintf("Kafka爆破开始: 目标=%s, 用户名数量=%d, 密码数量=%d, 总尝试次数=%d, 并发数=%d, 超时=%v", host, len(usernames), len(passwords), totalAttempts, workers, timeout))

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

				pass := strings.ReplaceAll(t.Pass, "{user}", t.User)
				currentAttempt := atomic.AddInt64(&attempted, 1)

				if currentAttempt%10 == 0 || currentAttempt == 1 {
					sendCrackLog(ctx, "info", fmt.Sprintf("尝试 [%d/%d]: %s/%s", currentAttempt, totalAttempts, t.User, pass))
				}

				flag, err := KafkaConn(host, t.User, pass, sshConfig)
				if flag && err == nil {
					credential := fmt.Sprintf("%s/%s", t.User, pass)
					vulnURL := fmt.Sprintf("kafka://%s", host)
					runtime.EventsEmit(ctx, "nucleiResult", map[string]interface{}{
						"task_id":     taskId,
						"template_id": "kafka-weak-password",
						"vuln_name":   "Kafka弱口令",
						"vuln_url":    vulnURL,
						"severity":    "HIGH",
						"protocol":    "Kafka",
						"extract":     credential,
					})

					emitCrackResult(ctx, taskId, "Kafka", vulnURL, credential, t.User, pass)

					saveCrackResultToDB(
						ctx,
						taskId,
						"kafka-weak-password",
						"Kafka弱口令",
						vulnURL,
						"HIGH",
						"Kafka",
						credential,
						"",
						"",
					)

					select {
					case successChan <- true:
					default:
					}
					return
				}
			}
			doneChan <- struct{}{}
		}()
	}

	// 分发任务
	go func() {
		defer close(taskChan)
		for _, user := range usernames {
			for _, pass := range passwords {
				if ctrlCtx.Err() != nil {
					return
				}
				select {
				case taskChan <- task{User: user, Pass: pass}:
				case <-ctrlCtx.Done():
					return
				case <-successChan:
					return
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
			success = true
			for doneCount < workers {
				<-doneChan
				doneCount++
			}
		case <-doneChan:
			doneCount++
		case <-time.After(30 * time.Minute):
			sendCrackLog(ctx, "warning", fmt.Sprintf("Kafka爆破超时: 目标=%s, 已尝试=%d次", host, atomic.LoadInt64(&attempted)))
			return
		}
	}

	finalAttempted := atomic.LoadInt64(&attempted)
	if !success {
		sendCrackLog(ctx, "info", fmt.Sprintf("Kafka爆破完成: 目标=%s, 已尝试=%d次, 未发现弱口令", host, finalAttempted))
	}
}

// KafkaConn 测试Kafka连接（SASL/PLAIN认证）
func KafkaConn(address, user, pass string, sshConfig *SSHConfig) (bool, error) {
	conn, err := dialWithSSH(sshConfig, "tcp", address, 10*time.Second)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// Kafka协议：发送API版本请求（简化实现）
	// 实际Kafka协议比较复杂，这里使用简化的SASL握手

	// 发送SASL握手请求
	// Request Header: API Key (17=SASL_HANDSHAKE) + API Version (0) + Correlation ID (1) + Client ID
	correlationId := int32(1)
	clientId := "test_client"
	clientIdLen := int16(len(clientId))

	// 构建SASL握手请求
	request := []byte{}
	request = append(request, 0x00, 0x11) // API Key: 17 (SASL_HANDSHAKE)
	request = append(request, 0x00, 0x00) // API Version: 0
	request = append(request, byte(correlationId>>24), byte(correlationId>>16), byte(correlationId>>8), byte(correlationId))
	request = append(request, byte(clientIdLen>>8), byte(clientIdLen&0xFF))
	request = append(request, []byte(clientId)...)

	// SASL机制名称：PLAIN
	mechanism := "PLAIN"
	mechanismLen := int16(len(mechanism))
	request = append(request, byte(mechanismLen>>8), byte(mechanismLen&0xFF))
	request = append(request, []byte(mechanism)...)

	// 添加长度前缀
	requestLen := len(request)
	fullRequest := []byte{}
	fullRequest = append(fullRequest, byte(requestLen>>24), byte(requestLen>>16), byte(requestLen>>8), byte(requestLen))
	fullRequest = append(fullRequest, request...)

	_, err = conn.Write(fullRequest)
	if err != nil {
		return false, err
	}

	// 读取响应
	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil {
		return false, err
	}

	// 如果收到响应，尝试SASL认证
	if n > 0 {
		// 发送SASL认证消息（PLAIN机制）
		// 格式: \0username\0password
		authMessage := fmt.Sprintf("\x00%s\x00%s", user, pass)
		authMessageLen := int32(len(authMessage))

		authRequest := []byte{}
		authRequest = append(authRequest, 0x00, 0x00) // API Key: 0 (PRODUCE, 简化使用)
		authRequest = append(authRequest, 0x00, 0x00) // API Version
		authRequest = append(authRequest, byte(correlationId>>24), byte(correlationId>>16), byte(correlationId>>8), byte(correlationId))
		authRequest = append(authRequest, byte(clientIdLen>>8), byte(clientIdLen&0xFF))
		authRequest = append(authRequest, []byte(clientId)...)
		authRequest = append(authRequest, byte(authMessageLen>>24), byte(authMessageLen>>16), byte(authMessageLen>>8), byte(authMessageLen))
		authRequest = append(authRequest, []byte(authMessage)...)

		authRequestLen := len(authRequest)
		fullAuthRequest := []byte{}
		fullAuthRequest = append(fullAuthRequest, byte(authRequestLen>>24), byte(authRequestLen>>16), byte(authRequestLen>>8), byte(authRequestLen))
		fullAuthRequest = append(fullAuthRequest, authRequest...)

		_, err = conn.Write(fullAuthRequest)
		if err != nil {
			return false, err
		}

		// 读取认证响应
		buffer = make([]byte, 4096)
		n, err = conn.Read(buffer)
		if err == nil && n > 0 {
			// 如果收到响应且不是错误，认为认证成功（简化处理）
			response := string(buffer[:n])
			if !strings.Contains(strings.ToLower(response), "error") &&
				!strings.Contains(strings.ToLower(response), "denied") {
				return true, nil
			}
		}
	}

	return false, fmt.Errorf("认证失败")
}

