package crack

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	_ "github.com/microsoft/go-mssqldb"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// MssqlScan MSSQL协议爆破（使用并发模式）
func MssqlScan(ctx, ctrlCtx context.Context, taskId, host string, usernames, passwords []string, sshConfig *SSHConfig, config *CrackConfig) {
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

	sendCrackLog(ctx, "info", fmt.Sprintf("MSSQL爆破开始: 目标=%s, 用户名数量=%d, 密码数量=%d, 总尝试次数=%d, 并发数=%d, 超时=%v", host, len(usernames), len(passwords), totalAttempts, workers, timeout))

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

				flag, err := MssqlConn(host, t.User, pass, sshConfig)
				if flag && err == nil {
					credential := fmt.Sprintf("%s/%s", t.User, pass)
					vulnURL := fmt.Sprintf("mssql://%s", host)
					runtime.EventsEmit(ctx, "nucleiResult", map[string]interface{}{
						"task_id":     taskId,
						"template_id": "mssql-weak-password",
						"vuln_name":   "MSSQL弱口令",
						"vuln_url":    vulnURL,
						"severity":    "HIGH",
						"protocol":    "MSSQL",
						"extract":     credential,
					})

					emitCrackResult(ctx, taskId, "MSSQL", vulnURL, credential, t.User, pass)

					saveCrackResultToDB(
						ctx,
						taskId,
						"mssql-weak-password",
						"MSSQL弱口令",
						vulnURL,
						"HIGH",
						"MSSQL",
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
			sendCrackLog(ctx, "warning", fmt.Sprintf("MSSQL爆破超时: 目标=%s, 已尝试=%d次", host, atomic.LoadInt64(&attempted)))
			return
		}
	}

	finalAttempted := atomic.LoadInt64(&attempted)
	if !success {
		sendCrackLog(ctx, "info", fmt.Sprintf("MSSQL爆破完成: 目标=%s, 已尝试=%d次, 未发现弱口令", host, finalAttempted))
	}
}

// MssqlConn 测试MSSQL连接（支持SSH隧道）
func MssqlConn(host, user, pass string, sshConfig *SSHConfig) (bool, error) {
	// MSSQL连接字符串
	connStr := fmt.Sprintf("server=%s;user id=%s;password=%s;database=master;connection timeout=10;encrypt=disable", host, user, pass)
	
	// MSSQL驱动不支持直接设置Dialer，需要通过net.Dialer实现
	// 这里暂时不支持SSH隧道，因为go-mssqldb驱动不支持自定义Dialer
	// 如果需要支持，需要使用其他方法（如本地端口转发）
	if sshConfig != nil && sshConfig.Enabled {
		// TODO: 实现MSSQL的SSH隧道支持（可能需要使用本地端口转发）
		return false, fmt.Errorf("MSSQL SSH隧道支持暂未实现")
	}
	
	db, err := sql.Open("sqlserver", connStr)
	if err != nil {
		return false, err
	}
	defer db.Close()

	db.SetConnMaxLifetime(10 * time.Second)
	db.SetConnMaxIdleTime(10 * time.Second)
	db.SetMaxIdleConns(0)

	err = db.Ping()
	if err != nil {
		return false, err
	}

	return true, nil
}

