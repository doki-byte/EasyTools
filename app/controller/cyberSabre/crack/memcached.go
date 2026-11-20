package crack

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// MemcachedScan Memcached协议爆破（使用并发模式）
func MemcachedScan(ctx, ctrlCtx context.Context, taskId, host string, usernames, passwords []string, sshConfig *SSHConfig, config *CrackConfig) {
	workers := config.Threads
	if workers <= 0 {
		workers = 5
	}

	timeout := time.Duration(config.Timeout) * time.Millisecond
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	// 先检查未授权访问
	flag, err := MemcachedUnauth(host, sshConfig)
	if flag && err == nil {
		vulnURL := fmt.Sprintf("memcached://%s", host)
		runtime.EventsEmit(ctx, "nucleiResult", map[string]interface{}{
			"task_id":     taskId,
			"template_id": "memcached-unauthorized",
			"vuln_name":   "Memcached未授权访问",
			"vuln_url":    vulnURL,
			"severity":    "HIGH",
			"protocol":    "Memcached",
		})

		emitCrackResult(ctx, taskId, "Memcached", vulnURL, "", "", "未授权访问")

		saveCrackResultToDB(
			ctx,
			taskId,
			"memcached-unauthorized",
			"Memcached未授权访问",
			vulnURL,
			"HIGH",
			"Memcached",
			"",
			"",
			"",
		)
		return
	}

	// 使用并发模式进行密码爆破（Memcached通常只需要密码，SASL认证）
	totalAttempts := len(passwords)
	var attempted int64

	sendCrackLog(ctx, "info", fmt.Sprintf("Memcached爆破开始: 目标=%s, 密码数量=%d, 总尝试次数=%d, 并发数=%d, 超时=%v", host, len(passwords), totalAttempts, workers, timeout))

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

				pass := strings.ReplaceAll(t.Pass, "{user}", "")
				currentAttempt := atomic.AddInt64(&attempted, 1)

				if currentAttempt%10 == 0 || currentAttempt == 1 {
					sendCrackLog(ctx, "info", fmt.Sprintf("尝试 [%d/%d]: %s", currentAttempt, totalAttempts, pass))
				}

				flag, err := MemcachedConn(host, pass, sshConfig)
				if flag && err == nil {
					vulnURL := fmt.Sprintf("memcached://%s", host)
					runtime.EventsEmit(ctx, "nucleiResult", map[string]interface{}{
						"task_id":     taskId,
						"template_id": "memcached-weak-password",
						"vuln_name":   "Memcached弱口令",
						"vuln_url":    vulnURL,
						"severity":    "HIGH",
						"protocol":    "Memcached",
						"extract":     pass,
					})

					emitCrackResult(ctx, taskId, "Memcached", vulnURL, pass, "", pass)

					saveCrackResultToDB(
						ctx,
						taskId,
						"memcached-weak-password",
						"Memcached弱口令",
						vulnURL,
						"HIGH",
						"Memcached",
						pass,
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

	// 分发任务（Memcached只需要密码）
	go func() {
		defer close(taskChan)
		for _, pass := range passwords {
			if ctrlCtx.Err() != nil {
				return
			}
			select {
			case taskChan <- task{User: "", Pass: pass}:
			case <-ctrlCtx.Done():
				return
			case <-successChan:
				return
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
			sendCrackLog(ctx, "warning", fmt.Sprintf("Memcached爆破超时: 目标=%s, 已尝试=%d次", host, atomic.LoadInt64(&attempted)))
			return
		}
	}

	finalAttempted := atomic.LoadInt64(&attempted)
	if !success {
		sendCrackLog(ctx, "info", fmt.Sprintf("Memcached爆破完成: 目标=%s, 已尝试=%d次, 未发现弱口令", host, finalAttempted))
	}
}

// MemcachedConn 测试Memcached连接（SASL认证）
func MemcachedConn(address, password string, sshConfig *SSHConfig) (bool, error) {
	conn, err := dialWithSSH(sshConfig, "tcp", address, 10*time.Second)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// SASL认证：发送认证请求
	authCmd := "set test_key 0 0 4\r\ntest\r\n"
	_, err = conn.Write([]byte(authCmd))
	if err != nil {
		return false, err
	}

	// 读取响应
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return false, err
	}

	// 如果返回STORED，说明可以写入（未授权或认证成功）
	response := string(buffer[:n])
	if strings.Contains(response, "STORED") || strings.Contains(response, "NOT_STORED") {
		// 尝试SASL认证（如果支持）
		saslCmd := "set sasl_auth 0 0 " + fmt.Sprintf("%d", len(password)) + "\r\n" + password + "\r\n"
		_, err = conn.Write([]byte(saslCmd))
		if err != nil {
			return false, err
		}

		buffer = make([]byte, 1024)
		n, err = conn.Read(buffer)
		if err == nil {
			response = string(buffer[:n])
			if strings.Contains(response, "STORED") || strings.Contains(response, "OK") {
				return true, nil
			}
		}
	}

	return false, fmt.Errorf("认证失败")
}

// MemcachedUnauth 检查Memcached未授权访问（支持SSH隧道）
func MemcachedUnauth(address string, sshConfig *SSHConfig) (bool, error) {
	conn, err := dialWithSSH(sshConfig, "tcp", address, 10*time.Second)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// 尝试执行stats命令
	_, err = conn.Write([]byte("stats\r\n"))
	if err != nil {
		return false, err
	}

	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil {
		return false, err
	}

	response := string(buffer[:n])
	if strings.Contains(response, "STAT") {
		return true, nil
	}

	return false, fmt.Errorf("需要认证")
}

