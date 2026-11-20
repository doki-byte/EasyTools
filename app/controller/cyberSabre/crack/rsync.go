package crack

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// RsyncScan Rsync协议爆破（使用并发模式）
func RsyncScan(ctx, ctrlCtx context.Context, taskId, host string, usernames, passwords []string, sshConfig *SSHConfig, config *CrackConfig) {
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

	sendCrackLog(ctx, "info", fmt.Sprintf("Rsync爆破开始: 目标=%s, 用户名数量=%d, 密码数量=%d, 总尝试次数=%d, 并发数=%d, 超时=%v", host, len(usernames), len(passwords), totalAttempts, workers, timeout))

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

				flag, err := RsyncConn(host, t.User, pass, sshConfig)
				if flag && err == nil {
					credential := fmt.Sprintf("%s/%s", t.User, pass)
					vulnURL := fmt.Sprintf("rsync://%s", host)
					runtime.EventsEmit(ctx, "nucleiResult", map[string]interface{}{
						"task_id":     taskId,
						"template_id": "rsync-weak-password",
						"vuln_name":   "Rsync弱口令",
						"vuln_url":    vulnURL,
						"severity":    "HIGH",
						"protocol":    "Rsync",
						"extract":     credential,
					})

					emitCrackResult(ctx, taskId, "Rsync", vulnURL, credential, t.User, pass)

					saveCrackResultToDB(
						ctx,
						taskId,
						"rsync-weak-password",
						"Rsync弱口令",
						vulnURL,
						"HIGH",
						"Rsync",
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
			sendCrackLog(ctx, "warning", fmt.Sprintf("Rsync爆破超时: 目标=%s, 已尝试=%d次", host, atomic.LoadInt64(&attempted)))
			return
		}
	}

	finalAttempted := atomic.LoadInt64(&attempted)
	if !success {
		sendCrackLog(ctx, "info", fmt.Sprintf("Rsync爆破完成: 目标=%s, 已尝试=%d次, 未发现弱口令", host, finalAttempted))
	}
}

// RsyncConn 测试Rsync连接
func RsyncConn(address, user, pass string, sshConfig *SSHConfig) (bool, error) {
	conn, err := dialWithSSH(sshConfig, "tcp", address, 10*time.Second)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// Rsync协议握手
	// 发送协议版本
	protocolVersion := "@RSYNCD: 31.0\n"
	_, err = conn.Write([]byte(protocolVersion))
	if err != nil {
		return false, err
	}

	// 读取服务器响应
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return false, err
	}

	response := string(buffer[:n])
	if !strings.Contains(response, "@RSYNCD:") {
		return false, fmt.Errorf("不是Rsync服务")
	}

	// 发送模块列表请求
	moduleList := "#\n"
	_, err = conn.Write([]byte(moduleList))
	if err != nil {
		return false, err
	}

	// 读取模块列表
	buffer = make([]byte, 4096)
	n, err = conn.Read(buffer)
	if err != nil {
		return false, err
	}

	// 尝试访问一个模块（通常使用用户名作为模块名）
	moduleName := user
	moduleRequest := fmt.Sprintf("%s\n", moduleName)
	_, err = conn.Write([]byte(moduleRequest))
	if err != nil {
		return false, err
	}

	// 读取模块响应
	buffer = make([]byte, 1024)
	n, err = conn.Read(buffer)
	if err != nil {
		return false, err
	}

	moduleResponse := string(buffer[:n])
	// 如果返回模块信息或需要密码，说明模块存在
	if strings.Contains(moduleResponse, "@RSYNCD: OK") ||
		strings.Contains(moduleResponse, "auth") ||
		strings.Contains(moduleResponse, "password") {
		// 发送密码
		_, err = conn.Write([]byte(pass + "\n"))
		if err != nil {
			return false, err
		}

		// 读取最终响应
		buffer = make([]byte, 1024)
		n, err = conn.Read(buffer)
		if err == nil {
			finalResponse := string(buffer[:n])
			if strings.Contains(finalResponse, "@RSYNCD: OK") ||
				!strings.Contains(strings.ToLower(finalResponse), "auth failed") &&
					!strings.Contains(strings.ToLower(finalResponse), "password") {
				return true, nil
			}
		}
	}

	return false, fmt.Errorf("认证失败")
}

