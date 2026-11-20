package crack

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ActiveMQScan ActiveMQ协议爆破（使用并发模式）
func ActiveMQScan(ctx, ctrlCtx context.Context, taskId, host string, usernames, passwords []string, sshConfig *SSHConfig, config *CrackConfig) {
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

	sendCrackLog(ctx, "info", fmt.Sprintf("ActiveMQ爆破开始: 目标=%s, 用户名数量=%d, 密码数量=%d, 总尝试次数=%d, 并发数=%d, 超时=%v", host, len(usernames), len(passwords), totalAttempts, workers, timeout))

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

				flag, err := ActiveMQConn(host, t.User, pass, sshConfig)
				if flag && err == nil {
					credential := fmt.Sprintf("%s/%s", t.User, pass)
					vulnURL := fmt.Sprintf("activemq://%s", host)
					runtime.EventsEmit(ctx, "nucleiResult", map[string]interface{}{
						"task_id":     taskId,
						"template_id": "activemq-weak-password",
						"vuln_name":   "ActiveMQ弱口令",
						"vuln_url":    vulnURL,
						"severity":    "HIGH",
						"protocol":    "ActiveMQ",
						"extract":     credential,
					})

					emitCrackResult(ctx, taskId, "ActiveMQ", vulnURL, credential, t.User, pass)

					saveCrackResultToDB(
						ctx,
						taskId,
						"activemq-weak-password",
						"ActiveMQ弱口令",
						vulnURL,
						"HIGH",
						"ActiveMQ",
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
			sendCrackLog(ctx, "warning", fmt.Sprintf("ActiveMQ爆破超时: 目标=%s, 已尝试=%d次", host, atomic.LoadInt64(&attempted)))
			return
		}
	}

	finalAttempted := atomic.LoadInt64(&attempted)
	if !success {
		sendCrackLog(ctx, "info", fmt.Sprintf("ActiveMQ爆破完成: 目标=%s, 已尝试=%d次, 未发现弱口令", host, finalAttempted))
	}
}

// ActiveMQConn 测试ActiveMQ连接（OpenWire协议）
func ActiveMQConn(address, user, pass string, sshConfig *SSHConfig) (bool, error) {
	conn, err := dialWithSSH(sshConfig, "tcp", address, 10*time.Second)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// OpenWire协议连接
	// 发送连接请求（简化实现）
	connectPacket := buildOpenWireConnectPacket(user, pass)
	_, err = conn.Write(connectPacket)
	if err != nil {
		return false, err
	}

	// 读取响应
	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil {
		return false, err
	}

	// 检查响应中是否包含成功标识
	// OpenWire协议响应通常包含连接ID或成功标志
	if n > 0 {
		// 简化检查：如果收到响应且不是错误消息，认为连接成功
		response := string(buffer[:n])
		if !strings.Contains(strings.ToLower(response), "error") &&
			!strings.Contains(strings.ToLower(response), "denied") &&
			!strings.Contains(strings.ToLower(response), "failed") {
			return true, nil
		}
	}

	return false, fmt.Errorf("认证失败")
}

// buildOpenWireConnectPacket 构建OpenWire连接包（简化实现）
func buildOpenWireConnectPacket(user, pass string) []byte {
	// OpenWire协议格式（简化版本）
	// 实际实现需要完整的OpenWire协议解析
	packet := []byte{0x00, 0x00, 0x00, 0x00} // 长度占位
	packet = append(packet, 0x01)             // 连接命令类型

	// 添加用户名和密码（简化处理）
	userBytes := []byte(user)
	passBytes := []byte(pass)
	packet = append(packet, byte(len(userBytes)))
	packet = append(packet, userBytes...)
	packet = append(packet, byte(len(passBytes)))
	packet = append(packet, passBytes...)

	// 更新长度字段
	length := len(packet) - 4
	packet[0] = byte(length >> 24)
	packet[1] = byte(length >> 16)
	packet[2] = byte(length >> 8)
	packet[3] = byte(length)

	return packet
}

