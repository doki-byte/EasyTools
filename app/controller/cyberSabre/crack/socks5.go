package crack

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// Socks5Scan SOCKS5协议爆破（使用并发模式）
func Socks5Scan(ctx, ctrlCtx context.Context, taskId, host string, usernames, passwords []string, sshConfig *SSHConfig, config *CrackConfig) {
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

	sendCrackLog(ctx, "info", fmt.Sprintf("SOCKS5爆破开始: 目标=%s, 用户名数量=%d, 密码数量=%d, 总尝试次数=%d, 并发数=%d, 超时=%v", host, len(usernames), len(passwords), totalAttempts, workers, timeout))

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

				flag, err := Socks5Conn(host, t.User, pass, sshConfig)
				if flag && err == nil {
					credential := fmt.Sprintf("%s/%s", t.User, pass)
					vulnURL := fmt.Sprintf("socks5://%s", host)
					runtime.EventsEmit(ctx, "nucleiResult", map[string]interface{}{
						"task_id":     taskId,
						"template_id": "socks5-weak-password",
						"vuln_name":   "SOCKS5弱口令",
						"vuln_url":    vulnURL,
						"severity":    "HIGH",
						"protocol":    "SOCKS5",
						"extract":     credential,
					})

					emitCrackResult(ctx, taskId, "Socks5", vulnURL, credential, t.User, pass)

					saveCrackResultToDB(
						ctx,
						taskId,
						"socks5-weak-password",
						"SOCKS5弱口令",
						vulnURL,
						"HIGH",
						"SOCKS5",
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
			sendCrackLog(ctx, "warning", fmt.Sprintf("SOCKS5爆破超时: 目标=%s, 已尝试=%d次", host, atomic.LoadInt64(&attempted)))
			return
		}
	}

	finalAttempted := atomic.LoadInt64(&attempted)
	if !success {
		sendCrackLog(ctx, "info", fmt.Sprintf("SOCKS5爆破完成: 目标=%s, 已尝试=%d次, 未发现弱口令", host, finalAttempted))
	}
}

// Socks5Conn 测试SOCKS5连接
func Socks5Conn(address, user, pass string, sshConfig *SSHConfig) (bool, error) {
	conn, err := dialWithSSH(sshConfig, "tcp", address, 10*time.Second)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// SOCKS5握手：发送认证方法
	// 0x05 0x01 0x02 表示：版本5，1种方法，用户名密码认证(0x02)
	authMethods := []byte{0x05, 0x01, 0x02}
	_, err = conn.Write(authMethods)
	if err != nil {
		return false, err
	}

	// 读取服务器响应
	buffer := make([]byte, 2)
	n, err := conn.Read(buffer)
	if err != nil || n < 2 {
		return false, fmt.Errorf("读取认证方法响应失败")
	}

	// 服务器应该返回 0x05 0x02 (版本5，选择用户名密码认证)
	if buffer[0] != 0x05 || buffer[1] != 0x02 {
		return false, fmt.Errorf("服务器不支持用户名密码认证")
	}

	// 发送用户名密码
	userLen := byte(len(user))
	passLen := byte(len(pass))
	authPacket := []byte{0x01} // 版本1
	authPacket = append(authPacket, userLen)
	authPacket = append(authPacket, []byte(user)...)
	authPacket = append(authPacket, passLen)
	authPacket = append(authPacket, []byte(pass)...)

	_, err = conn.Write(authPacket)
	if err != nil {
		return false, err
	}

	// 读取认证结果
	buffer = make([]byte, 2)
	n, err = conn.Read(buffer)
	if err != nil || n < 2 {
		return false, fmt.Errorf("读取认证结果失败")
	}

	// 0x01 0x00 表示认证成功
	if buffer[0] == 0x01 && buffer[1] == 0x00 {
		return true, nil
	}

	return false, fmt.Errorf("认证失败")
}

