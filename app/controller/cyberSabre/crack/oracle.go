package crack

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// OracleScan Oracle协议爆破（使用并发模式）
func OracleScan(ctx, ctrlCtx context.Context, taskId, host string, usernames, passwords []string, sshConfig *SSHConfig, config *CrackConfig) {
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

	sendCrackLog(ctx, "info", fmt.Sprintf("Oracle爆破开始: 目标=%s, 用户名数量=%d, 密码数量=%d, 总尝试次数=%d, 并发数=%d, 超时=%v", host, len(usernames), len(passwords), totalAttempts, workers, timeout))

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

				flag, err := OracleConn(host, t.User, pass, sshConfig)
				if flag && err == nil {
					credential := fmt.Sprintf("%s/%s", t.User, pass)
					vulnURL := fmt.Sprintf("oracle://%s", host)
					runtime.EventsEmit(ctx, "nucleiResult", map[string]interface{}{
						"task_id":     taskId,
						"template_id": "oracle-weak-password",
						"vuln_name":   "Oracle弱口令",
						"vuln_url":    vulnURL,
						"severity":    "HIGH",
						"protocol":    "Oracle",
						"extract":     credential,
					})

					emitCrackResult(ctx, taskId, "Oracle", vulnURL, credential, t.User, pass)

					saveCrackResultToDB(
						ctx,
						taskId,
						"oracle-weak-password",
						"Oracle弱口令",
						vulnURL,
						"HIGH",
						"Oracle",
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
			sendCrackLog(ctx, "warning", fmt.Sprintf("Oracle爆破超时: 目标=%s, 已尝试=%d次", host, atomic.LoadInt64(&attempted)))
			return
		}
	}

	finalAttempted := atomic.LoadInt64(&attempted)
	if !success {
		sendCrackLog(ctx, "info", fmt.Sprintf("Oracle爆破完成: 目标=%s, 已尝试=%d次, 未发现弱口令", host, finalAttempted))
	}
}

// OracleConn 测试Oracle连接（简化实现，需要godror驱动）
func OracleConn(host, user, pass string, sshConfig *SSHConfig) (bool, error) {
	// 先测试TCP连接
	conn, err := dialWithSSH(sshConfig, "tcp", host, 10*time.Second)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// Oracle TNS协议握手（简化实现）
	// 发送TNS连接包
	tnsPacket := buildTNSPacket(user, pass)
	_, err = conn.Write(tnsPacket)
	if err != nil {
		return false, err
	}

	// 读取响应
	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil {
		return false, err
	}

	// 检查响应（简化处理）
	// 实际Oracle认证需要完整的TNS协议实现
	if n > 0 {
		// 如果能收到响应，说明服务存在
		// 完整认证需要godror驱动或完整的TNS协议实现
		return false, fmt.Errorf("Oracle协议复杂，需要godror驱动或完整的TNS协议实现")
	}

	return false, fmt.Errorf("连接失败")
}

// buildTNSPacket 构建TNS连接包（简化实现）
func buildTNSPacket(user, pass string) []byte {
	// TNS协议格式（简化版本）
	// 实际TNS协议非常复杂
	packet := []byte{0x00, 0x00} // 长度占位
	packet = append(packet, 0x01) // 包类型：CONNECT
	packet = append(packet, 0x00) // 保留
	packet = append(packet, 0x00, 0x00) // 头部校验和

	// 添加用户名和密码（简化处理）
	userBytes := []byte(user)
	passBytes := []byte(pass)
	packet = append(packet, byte(len(userBytes)))
	packet = append(packet, userBytes...)
	packet = append(packet, byte(len(passBytes)))
	packet = append(packet, passBytes...)

	// 更新长度字段
	length := len(packet)
	packet[0] = byte(length >> 8)
	packet[1] = byte(length & 0xFF)

	return packet
}

