package crack

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/projectdiscovery/go-smb2"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// SmbScan SMB协议爆破（使用并发模式）
func SmbScan(ctx, ctrlCtx context.Context, taskId, host string, usernames, passwords []string, sshConfig *SSHConfig, config *CrackConfig) {
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

	sendCrackLog(ctx, "info", fmt.Sprintf("SMB爆破开始: 目标=%s, 用户名数量=%d, 密码数量=%d, 总尝试次数=%d, 并发数=%d, 超时=%v", host, len(usernames), len(passwords), totalAttempts, workers, timeout))

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

				flag, err := SmbConn(host, t.User, pass, sshConfig)
				if flag && err == nil {
					credential := fmt.Sprintf("%s/%s", t.User, pass)
					vulnURL := fmt.Sprintf("smb://%s", host)
					runtime.EventsEmit(ctx, "nucleiResult", map[string]interface{}{
						"task_id":     taskId,
						"template_id": "smb-weak-password",
						"vuln_name":   "SMB弱口令",
						"vuln_url":    vulnURL,
						"severity":    "HIGH",
						"protocol":    "SMB",
						"extract":     credential,
					})

					emitCrackResult(ctx, taskId, "SMB", vulnURL, credential, t.User, pass)

					saveCrackResultToDB(
						ctx,
						taskId,
						"smb-weak-password",
						"SMB弱口令",
						vulnURL,
						"HIGH",
						"SMB",
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
			sendCrackLog(ctx, "warning", fmt.Sprintf("SMB爆破超时: 目标=%s, 已尝试=%d次", host, atomic.LoadInt64(&attempted)))
			return
		}
	}

	finalAttempted := atomic.LoadInt64(&attempted)
	if !success {
		sendCrackLog(ctx, "info", fmt.Sprintf("SMB爆破完成: 目标=%s, 已尝试=%d次, 未发现弱口令", host, finalAttempted))
	}
}

// SmbConn 测试SMB连接
func SmbConn(address, user, pass string, sshConfig *SSHConfig) (bool, error) {
	// 先建立TCP连接
	conn, err := dialWithSSH(sshConfig, "tcp", address, 10*time.Second)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	// 使用projectdiscovery的go-smb2库
	dialer := &smb2.Dialer{
		Initiator: &smb2.NTLMInitiator{
			User:     user,
			Password: pass,
		},
	}

	session, err := dialer.Dial(conn)
	if err != nil {
		return false, err
	}
	defer session.Logoff()

	// 尝试列出共享
	_, err = session.ListSharenames()
	if err != nil {
		return false, err
	}

	return true, nil
}

