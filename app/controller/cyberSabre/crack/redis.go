package crack

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// RedisScan Redis协议爆破（使用并发模式）
func RedisScan(ctx, ctrlCtx context.Context, taskId, host string, usernames, passwords []string, sshConfig *SSHConfig, config *CrackConfig) {
	workers := config.Threads
	if workers <= 0 {
		workers = 5
	}

	timeout := time.Duration(config.Timeout) * time.Millisecond
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	// 先检查未授权访问
	flag, err := RedisUnauth(host, sshConfig)
	if flag && err == nil {
		vulnURL := fmt.Sprintf("redis://%s", host)
		runtime.EventsEmit(ctx, "nucleiResult", map[string]interface{}{
			"task_id":     taskId,
			"template_id": "redis-unauthorized",
			"vuln_name":   "Redis未授权访问",
			"vuln_url":    vulnURL,
			"severity":    "HIGH",
			"protocol":    "Redis",
		})

		emitCrackResult(ctx, taskId, "Redis", vulnURL, "", "", "未授权访问")

		saveCrackResultToDB(
			ctx,
			taskId,
			"redis-unauthorized",
			"Redis未授权访问",
			vulnURL,
			"HIGH",
			"Redis",
			"",
			"",
			"",
		)
		return
	}

	// 使用并发模式进行密码爆破
	totalAttempts := len(passwords)
	var attempted int64

	sendCrackLog(ctx, "info", fmt.Sprintf("Redis爆破开始: 目标=%s, 密码数量=%d, 总尝试次数=%d, 并发数=%d, 超时=%v", host, len(passwords), totalAttempts, workers, timeout))

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

				pass := strings.ReplaceAll(t.Pass, "{user}", "redis")
				currentAttempt := atomic.AddInt64(&attempted, 1)

				if currentAttempt%10 == 0 || currentAttempt == 1 {
					sendCrackLog(ctx, "info", fmt.Sprintf("尝试 [%d/%d]: %s", currentAttempt, totalAttempts, pass))
				}

				flag, err := RedisConn(host, pass, sshConfig)
				if flag && err == nil {
					vulnURL := fmt.Sprintf("redis://%s", host)
					runtime.EventsEmit(ctx, "nucleiResult", map[string]interface{}{
						"task_id":     taskId,
						"template_id": "redis-weak-password",
						"vuln_name":   "Redis弱口令",
						"vuln_url":    vulnURL,
						"severity":    "HIGH",
						"protocol":    "Redis",
						"extract":     pass,
					})

					emitCrackResult(ctx, taskId, "Redis", vulnURL, pass, "", pass)

					saveCrackResultToDB(
						ctx,
						taskId,
						"redis-weak-password",
						"Redis弱口令",
						vulnURL,
						"HIGH",
						"Redis",
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

	// 分发任务（Redis只需要密码，不需要用户名）
	go func() {
		defer close(taskChan)
		for _, pass := range passwords {
			if ctrlCtx.Err() != nil {
				return
			}
			select {
			case taskChan <- task{User: "redis", Pass: pass}:
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
			sendCrackLog(ctx, "warning", fmt.Sprintf("Redis爆破超时: 目标=%s, 已尝试=%d次", host, atomic.LoadInt64(&attempted)))
			return
		}
	}

	finalAttempted := atomic.LoadInt64(&attempted)
	if !success {
		sendCrackLog(ctx, "info", fmt.Sprintf("Redis爆破完成: 目标=%s, 已尝试=%d次, 未发现弱口令", host, finalAttempted))
	}
}

// RedisConn 测试Redis连接（带密码，支持SSH隧道）
func RedisConn(address, password string, sshConfig *SSHConfig) (bool, error) {
	conn, err := dialWithSSH(sshConfig, "tcp", address, 10*time.Second)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// 发送AUTH命令
	_, err = fmt.Fprintf(conn, "AUTH %s\r\n", password)
	if err != nil {
		return false, err
	}

	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil {
		return false, err
	}

	if strings.Contains(string(buffer[:n]), "+OK") {
		return true, nil
	}

	return false, fmt.Errorf("认证失败")
}

// RedisUnauth 检查Redis未授权访问（支持SSH隧道）
func RedisUnauth(address string, sshConfig *SSHConfig) (bool, error) {
	conn, err := dialWithSSH(sshConfig, "tcp", address, 10*time.Second)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// 发送INFO命令
	_, err = conn.Write([]byte("INFO\r\n"))
	if err != nil {
		return false, err
	}

	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil {
		return false, err
	}

	if strings.Contains(string(buffer[:n]), "redis_version") {
		return true, nil
	}

	return false, fmt.Errorf("需要认证")
}

