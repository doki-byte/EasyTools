package crack

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jlaffaye/ftp"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// FtpScan FTP协议爆破（使用并发模式）
func FtpScan(ctx, ctrlCtx context.Context, taskId, host string, usernames, passwords []string, sshConfig *SSHConfig, config *CrackConfig) {
	totalAttempts := len(usernames) * len(passwords)
	var attempted int64

	workers := config.Threads
	if workers <= 0 {
		workers = 5
	}

	sendCrackLog(ctx, "info", fmt.Sprintf("FTP爆破开始: 目标=%s, 用户名数量=%d, 密码数量=%d, 总尝试次数=%d, 并发数=%d, 超时=%dms", host, len(usernames), len(passwords), totalAttempts, workers, config.Timeout))

	// 先检查匿名登录
	sendCrackLog(ctx, "info", fmt.Sprintf("检查FTP匿名登录: %s", host))
	timeout := time.Duration(config.Timeout) * time.Millisecond
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	flag, directories, err := FtpConn(host, "anonymous", "", sshConfig, timeout)
	if flag && err == nil {
		sendCrackLog(ctx, "success", fmt.Sprintf("FTP匿名登录成功: %s", host))

		vulnURL := fmt.Sprintf("ftp://%s", host)
		response := strings.Join(directories, "\n")

		runtime.EventsEmit(ctx, "nucleiResult", map[string]interface{}{
			"task_id":     taskId,
			"template_id": "ftp-unauthorized",
			"vuln_name":   "FTP匿名登录",
			"vuln_url":    vulnURL,
			"severity":    "HIGH",
			"protocol":    "FTP",
			"response":    response,
		})

		// 发送crackResult事件到密码爆破模块（匿名登录）
		emitCrackResult(ctx, taskId, "FTP", vulnURL, "anonymous/", "anonymous", "")

		// 保存结果到数据库
		saveCrackResultToDB(
			ctx,
			taskId,
			"ftp-unauthorized",
			"FTP匿名登录",
			vulnURL,
			"HIGH",
			"FTP",
			"anonymous/",
			"",
			response,
		)
		return
	} else if err != nil {
		sendCrackLog(ctx, "info", fmt.Sprintf("FTP匿名登录失败: %v", err))
	}

	// 使用并发模式进行密码爆破
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

				flag, directories, err := FtpConn(host, t.User, pass, sshConfig, timeout)
				if flag && err == nil {
					// 爆破成功，发送结果
					credential := fmt.Sprintf("%s/%s", t.User, pass)
					sendCrackLog(ctx, "success", fmt.Sprintf("FTP爆破成功: %s@%s", credential, host))
					sendCrackLog(ctx, "info", fmt.Sprintf("已发送到漏洞扫描模块: FTP弱口令 - %s@%s (账号密码: %s)", host, credential, credential))

					vulnURL := fmt.Sprintf("ftp://%s", host)
					response := strings.Join(directories, "\n")

					runtime.EventsEmit(ctx, "nucleiResult", map[string]interface{}{
						"task_id":     taskId,
						"template_id": "ftp-weak-password",
						"vuln_name":   "FTP弱口令",
						"vuln_url":    vulnURL,
						"severity":    "HIGH",
						"protocol":    "FTP",
						"extract":     credential,
						"response":    response,
					})

					// 发送crackResult事件到密码爆破模块
					emitCrackResult(ctx, taskId, "FTP", vulnURL, credential, t.User, pass)

					// 保存结果到数据库
					saveCrackResultToDB(
						ctx,
						taskId,
						"ftp-weak-password",
						"FTP弱口令",
						vulnURL,
						"HIGH",
						"FTP",
						credential,
						"",
						response,
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
			sendCrackLog(ctx, "warning", fmt.Sprintf("FTP爆破超时: 目标=%s, 已尝试=%d次", host, atomic.LoadInt64(&attempted)))
			return
		}
	}

	finalAttempted := atomic.LoadInt64(&attempted)
	if !success {
		sendCrackLog(ctx, "info", fmt.Sprintf("FTP爆破完成: 目标=%s, 已尝试=%d次, 未发现弱口令", host, finalAttempted))
	}
}

// FtpConn 测试FTP连接（支持SSH隧道）
func FtpConn(address, user, pass string, sshConfig *SSHConfig, timeout time.Duration) (bool, []string, error) {
	var conn *ftp.ServerConn
	var err error

	if timeout <= 0 {
		timeout = 12 * time.Second
	}

	// 如果配置了SSH隧道，使用本地端口转发
	if sshConfig != nil && sshConfig.Enabled {
		sshClient, err := getSSHClient(sshConfig)
		if err != nil {
			return false, nil, fmt.Errorf("SSH连接失败: %v", err)
		}
		defer sshClient.Close()

		// 创建本地监听端口进行转发
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			return false, nil, fmt.Errorf("创建本地监听端口失败: %v", err)
		}
		defer listener.Close()

		localAddr := listener.Addr().String()

		// 启动goroutine进行端口转发
		go func() {
			for {
				localConn, err := listener.Accept()
				if err != nil {
					return
				}

				// 通过SSH隧道连接到远程FTP服务器
				remoteConn, err := sshClient.Dial("tcp", address)
				if err != nil {
					localConn.Close()
					continue
				}

				// 进行双向数据转发
				go func() {
					defer localConn.Close()
					defer remoteConn.Close()
					io.Copy(localConn, remoteConn)
				}()
				go func() {
					defer localConn.Close()
					defer remoteConn.Close()
					io.Copy(remoteConn, localConn)
				}()
			}
		}()

		// 连接到本地端口（通过SSH隧道转发）
		conn, err = ftp.Dial(localAddr, ftp.DialWithTimeout(timeout))
		if err != nil {
			return false, nil, fmt.Errorf("通过SSH隧道连接FTP失败: %v", err)
		}
	} else {
		// 直连方式
		conn, err = ftp.Dial(address, ftp.DialWithTimeout(timeout))
		if err != nil {
			return false, nil, err
		}
	}

	defer func() {
		if conn != nil {
			conn.Quit()
		}
	}()

	err = conn.Login(user, pass)
	if err != nil {
		return false, nil, err
	}

	// 获取目录信息
	dirs, err := conn.List("")
	directories := []string{}
	if err == nil && len(dirs) > 0 {
		maxDirs := 6
		if len(dirs) < maxDirs {
			maxDirs = len(dirs)
		}
		for i := 0; i < maxDirs; i++ {
			name := dirs[i].Name
			if len(name) > 50 {
				name = name[:50]
			}
			directories = append(directories, name)
		}
	}

	return true, directories, nil
}
