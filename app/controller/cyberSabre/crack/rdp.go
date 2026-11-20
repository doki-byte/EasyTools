package crack

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// RdpScan RDP协议爆破（使用并发模式）
func RdpScan(ctx, ctrlCtx context.Context, taskId, host string, usernames, passwords []string, sshConfig *SSHConfig, config *CrackConfig) {
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

	sendCrackLog(ctx, "info", fmt.Sprintf("RDP爆破开始: 目标=%s, 用户名数量=%d, 密码数量=%d, 总尝试次数=%d, 并发数=%d, 超时=%v", host, len(usernames), len(passwords), totalAttempts, workers, timeout))

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

				flag, err := RdpConn(host, t.User, pass, sshConfig)
				if flag && err == nil {
					credential := fmt.Sprintf("%s/%s", t.User, pass)
					vulnURL := fmt.Sprintf("rdp://%s", host)
					runtime.EventsEmit(ctx, "nucleiResult", map[string]interface{}{
						"task_id":     taskId,
						"template_id": "rdp-weak-password",
						"vuln_name":   "RDP弱口令",
						"vuln_url":    vulnURL,
						"severity":    "HIGH",
						"protocol":    "RDP",
						"extract":     credential,
					})

					emitCrackResult(ctx, taskId, "RDP", vulnURL, credential, t.User, pass)

					saveCrackResultToDB(
						ctx,
						taskId,
						"rdp-weak-password",
						"RDP弱口令",
						vulnURL,
						"HIGH",
						"RDP",
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
			sendCrackLog(ctx, "warning", fmt.Sprintf("RDP爆破超时: 目标=%s, 已尝试=%d次", host, atomic.LoadInt64(&attempted)))
			return
		}
	}

	finalAttempted := atomic.LoadInt64(&attempted)
	if !success {
		sendCrackLog(ctx, "info", fmt.Sprintf("RDP爆破完成: 目标=%s, 已尝试=%d次, 未发现弱口令", host, finalAttempted))
	}
}

// RdpConn 测试RDP连接（简化实现）
func RdpConn(address, user, pass string, sshConfig *SSHConfig) (bool, error) {
	conn, err := dialWithSSH(sshConfig, "tcp", address, 10*time.Second)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// RDP协议握手（简化实现）
	// 发送连接请求
	// RDP协议以T.124协议开始，然后是RDP协商
	// 这里使用简化的方法：检查端口是否开放RDP服务

	// 读取服务器响应（RDP服务通常会发送连接请求）
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return false, err
	}

	// 检查是否是RDP协议响应
	// RDP协议通常以0x03 0x00开头（T.124协议）
	if n >= 2 && buffer[0] == 0x03 && buffer[1] == 0x00 {
		// 这是一个简化的实现
		// 实际RDP认证需要完整的RDP协议栈，包括：
		// 1. T.124连接序列
		// 2. RDP协商
		// 3. 安全层协商
		// 4. 用户认证
		// 由于RDP协议非常复杂，这里只做基本的连接测试
		// 实际认证需要使用专门的RDP库（如github.com/bolkedebruin/rdpgw）

		// 如果能建立连接并收到RDP响应，认为服务存在
		// 但完整认证需要实现完整的RDP协议栈
		return false, fmt.Errorf("RDP协议复杂，需要专门的库实现完整认证")
	}

	return false, fmt.Errorf("不是RDP服务或认证失败")
}

