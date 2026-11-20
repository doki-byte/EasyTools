package crack

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/crypto/ssh"
)

// SshScan SSH协议爆破（SSH协议本身不需要SSH隧道）
func SshScan(ctx, ctrlCtx context.Context, taskId, host string, usernames, passwords []string, sshConfig *SSHConfig, config *CrackConfig) {
	totalAttempts := len(usernames) * len(passwords)
	var attempted int64
	
	workers := config.Threads
	if workers <= 0 {
		workers = 5
	}
	
	sendCrackLog(ctx, "info", fmt.Sprintf("SSH爆破开始: 目标=%s, 用户名数量=%d, 密码数量=%d, 总尝试次数=%d, 并发数=%d", host, len(usernames), len(passwords), totalAttempts, workers))
	
	taskChan := make(chan task, 100)
	doneChan := make(chan struct{})
	successChan := make(chan bool, 1) // 成功信号

	// 启动worker
	for i := 0; i < workers; i++ {
		go func() {
			for t := range taskChan {
				// 检查是否需要停止
				if ctrlCtx.Err() != nil {
					continue
				}

				pass := strings.ReplaceAll(t.Pass, "{user}", t.User)
				currentAttempt := atomic.AddInt64(&attempted, 1)
				
				// 记录每次尝试（每10次记录一次，避免日志过多）
				if currentAttempt%10 == 0 || currentAttempt == 1 {
					sendCrackLog(ctx, "info", fmt.Sprintf("尝试 [%d/%d]: %s/%s", currentAttempt, totalAttempts, t.User, pass))
				}
				
				timeout := time.Duration(config.Timeout) * time.Millisecond
				if timeout <= 0 {
					timeout = 10 * time.Second
				}
				flag, err := SshConn(host, t.User, pass, timeout)
				if flag && err == nil {
					// 爆破成功，发送结果
					credential := fmt.Sprintf("%s/%s", t.User, pass)
					sendCrackLog(ctx, "success", fmt.Sprintf("SSH爆破成功: %s@%s", credential, host))
					
					result, err := ExecuteSshCommand(host, t.User, pass, "whoami")
					if err != nil {
						result = err.Error()
					}

				sendCrackLog(ctx, "info", fmt.Sprintf("已发送到漏洞扫描模块: SSH弱口令 - %s@%s (账号密码: %s)", host, credential, credential))
				
				// 解析host和port
				hostPart := host
				portPart := "22"
				if strings.Contains(host, ":") {
					parts := strings.Split(host, ":")
					if len(parts) == 2 {
						hostPart = parts[0]
						portPart = parts[1]
					}
				}
				
				// 构建结果数据（发送到漏洞扫描模块）
				vulnURL := fmt.Sprintf("ssh://%s", host)
				resultData := map[string]interface{}{
					"task_id":     taskId,
					"template_id": "ssh-weak-password",
					"vuln_name":   "SSH弱口令",
					"vuln_url":    vulnURL,
					"severity":    "HIGH",
					"protocol":    "SSH",
					"extract":     credential,
					"request":     "[Command] whoami",
					"response":    result,
				}
				
				// 发送事件到前端（漏洞扫描模块）
				runtime.EventsEmit(ctx, "nucleiResult", resultData)
				
				// 同时发送到密码爆破模块（直接保存到数据库）
				crackResultData := map[string]interface{}{
					"task_id":    taskId,
					"protocol":   "SSH",
					"host":       hostPart,
					"port":       portPart,
					"username":   t.User,
					"password":   pass,
					"status":     "success",
					"message":    "密码爆破成功",
					"credential": credential,
					"vuln_url":   vulnURL,
					"command_result": result,
				}
				runtime.EventsEmit(ctx, "crackResult", crackResultData)
				
				// 保存结果到数据库
				saveCrackResultToDB(
					ctx,
					taskId,
					"ssh-weak-password",
					"SSH弱口令",
					vulnURL,
					"HIGH",
					"SSH",
					credential,
					"[Command] whoami",
					result,
				)

					// 发送成功信号
					select {
					case successChan <- true:
					default:
					}
					return
				} else if err != nil {
					// 记录错误（每50次记录一次，避免日志过多）
					if currentAttempt%50 == 0 {
						errMsg := err.Error()
						if len(errMsg) > 100 {
							errMsg = errMsg[:100] + "..."
						}
						sendCrackLog(ctx, "info", fmt.Sprintf("尝试失败 [%d/%d]: %s/%s - %s", currentAttempt, totalAttempts, t.User, pass, errMsg))
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
			for _, pass := range passwords {
				if ctrlCtx.Err() != nil {
					return
				}
				select {
				case taskChan <- task{User: user, Pass: pass}:
				case <-ctrlCtx.Done():
					return
				case <-successChan:
					// 如果已经成功，停止分发任务
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
			// 成功，如果配置了仅破解一个账户，立即停止
			if config.OnlyOneAccount {
				success = true
				// 关闭任务通道，停止分发新任务
				// 注意：这里不能直接close(taskChan)，因为它在另一个goroutine中
				// 等待所有worker完成
				for doneCount < workers {
					<-doneChan
					doneCount++
				}
				return
			}
			// 否则继续等待所有worker完成
			success = true
			for doneCount < workers {
				<-doneChan
				doneCount++
			}
		case <-doneChan:
			doneCount++
		case <-time.After(30 * time.Minute):
			// 超时
			sendCrackLog(ctx, "warning", fmt.Sprintf("SSH爆破超时: 目标=%s, 已尝试=%d次", host, atomic.LoadInt64(&attempted)))
			return
		}
	}
	
	finalAttempted := atomic.LoadInt64(&attempted)
	if !success {
		sendCrackLog(ctx, "info", fmt.Sprintf("SSH爆破完成: 目标=%s, 已尝试=%d次, 未发现弱口令", host, finalAttempted))
	}
}

// SshConn 测试SSH连接
func SshConn(host, user, pass string, timeout time.Duration) (bool, error) {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(pass)},
		Timeout:         timeout,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", host, config)
	if err != nil {
		return false, err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return false, err
	}
	defer session.Close()

	return true, nil
}

// ExecuteSshCommand 执行SSH命令
func ExecuteSshCommand(host, username, password, command string) (string, error) {
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		Timeout:         5 * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", host, config)
	if err != nil {
		return "", fmt.Errorf("连接失败: %v", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建会话失败: %v", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)
	if err != nil {
		return "", fmt.Errorf("执行命令失败: %v", err)
	}

	return string(output), nil
}
