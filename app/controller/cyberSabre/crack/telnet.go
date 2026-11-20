package crack

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// TelnetScan Telnet协议爆破（使用并发模式）
func TelnetScan(ctx, ctrlCtx context.Context, taskId, host string, usernames, passwords []string, sshConfig *SSHConfig, config *CrackConfig) {
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

	sendCrackLog(ctx, "info", fmt.Sprintf("Telnet爆破开始: 目标=%s, 用户名数量=%d, 密码数量=%d, 总尝试次数=%d, 并发数=%d, 超时=%v", host, len(usernames), len(passwords), totalAttempts, workers, timeout))

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

				flag, err := TelnetConn(host, t.User, pass, sshConfig)
				if flag && err == nil {
					credential := fmt.Sprintf("%s/%s", t.User, pass)
					vulnURL := fmt.Sprintf("telnet://%s", host)
					runtime.EventsEmit(ctx, "nucleiResult", map[string]interface{}{
						"task_id":     taskId,
						"template_id": "telnet-weak-password",
						"vuln_name":   "Telnet弱口令",
						"vuln_url":    vulnURL,
						"severity":    "HIGH",
						"protocol":    "Telnet",
						"extract":     credential,
					})

					emitCrackResult(ctx, taskId, "Telnet", vulnURL, credential, t.User, pass)

					saveCrackResultToDB(
						ctx,
						taskId,
						"telnet-weak-password",
						"Telnet弱口令",
						vulnURL,
						"HIGH",
						"Telnet",
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
			sendCrackLog(ctx, "warning", fmt.Sprintf("Telnet爆破超时: 目标=%s, 已尝试=%d次", host, atomic.LoadInt64(&attempted)))
			return
		}
	}

	finalAttempted := atomic.LoadInt64(&attempted)
	if !success {
		sendCrackLog(ctx, "info", fmt.Sprintf("Telnet爆破完成: 目标=%s, 已尝试=%d次, 未发现弱口令", host, finalAttempted))
	}
}

// TelnetConn 测试Telnet连接
func TelnetConn(address, user, pass string, sshConfig *SSHConfig) (bool, error) {
	conn, err := dialWithSSH(sshConfig, "tcp", address, 10*time.Second)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	if err := telnetLoginSession(conn, user, pass); err != nil {
		return false, err
	}
	return true, nil
}

// telnetLoginSession 与 TelnetConn 相同的用户名/密码握手，成功后连接处于 shell 就绪状态
func telnetLoginSession(conn net.Conn, user, pass string) error {
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil && n == 0 {
		return err
	}

	_, err = conn.Write([]byte(user + "\r\n"))
	if err != nil {
		return err
	}

	time.Sleep(500 * time.Millisecond)
	buffer = make([]byte, 4096)
	n, err = conn.Read(buffer)
	if err != nil && n == 0 {
		return err
	}

	_, err = conn.Write([]byte(pass + "\r\n"))
	if err != nil {
		return err
	}

	time.Sleep(1 * time.Second)
	buffer = make([]byte, 4096)
	n, err = conn.Read(buffer)
	if err != nil && n == 0 {
		return err
	}

	response := strings.ToLower(string(buffer[:n]))
	if strings.Contains(response, "$") || strings.Contains(response, "#") ||
		strings.Contains(response, ">") || strings.Contains(response, "welcome") ||
		strings.Contains(response, "last login") || !strings.Contains(response, "login incorrect") &&
		!strings.Contains(response, "password incorrect") && !strings.Contains(response, "access denied") {
		return nil
	}

	return fmt.Errorf("认证失败")
}

// ExecuteTelnetCommand 在 Telnet 登录会话中执行一条命令（握手逻辑与爆破检测一致，支持 SSH 隧道）
func ExecuteTelnetCommand(host, port, username, password string, sshConfig *SSHConfig, command string) (string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return "", fmt.Errorf("命令为空")
	}
	if host == "" {
		return "", fmt.Errorf("主机地址为空")
	}
	if port == "" {
		port = "23"
	}
	address := net.JoinHostPort(host, port)

	conn, err := dialWithSSH(sshConfig, "tcp", address, 15*time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	if err := telnetLoginSession(conn, username, password); err != nil {
		return "", err
	}

	_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if _, err := conn.Write([]byte(command + "\r\n")); err != nil {
		return "", fmt.Errorf("发送命令失败: %w", err)
	}

	var out strings.Builder
	buf := make([]byte, 4096)
	overall := time.Now().Add(12 * time.Second)
	for time.Now().Before(overall) {
		_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, rerr := conn.Read(buf)
		if n > 0 {
			out.Write(buf[:n])
		}
		if rerr != nil {
			if ne, ok := rerr.(net.Error); ok && ne.Timeout() {
				if out.Len() > 0 {
					break
				}
				continue
			}
			if out.Len() > 0 {
				break
			}
			return "", fmt.Errorf("读取输出失败: %w", rerr)
		}
		if n == 0 {
			break
		}
	}
	return strings.TrimSpace(out.String()), nil
}

