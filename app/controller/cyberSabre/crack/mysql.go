package crack

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// MysqlScan MySQL协议爆破（使用并发模式）
func MysqlScan(ctx, ctrlCtx context.Context, taskId, host string, usernames, passwords []string, sshConfig *SSHConfig, config *CrackConfig) {
	totalAttempts := len(usernames) * len(passwords)
	var attempted int64

	workers := config.Threads
	if workers <= 0 {
		workers = 5
	}

	timeout := time.Duration(config.Timeout) * time.Millisecond
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	sendCrackLog(ctx, "info", fmt.Sprintf("MySQL爆破开始: 目标=%s, 用户名数量=%d, 密码数量=%d, 总尝试次数=%d, 并发数=%d, 超时=%v", host, len(usernames), len(passwords), totalAttempts, workers, timeout))

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

				flag, err := MysqlConn(host, t.User, pass, sshConfig, timeout)
				if flag && err == nil {
					credential := fmt.Sprintf("%s/%s", t.User, pass)
					sendCrackLog(ctx, "success", fmt.Sprintf("MySQL爆破成功: %s@%s", credential, host))
					sendCrackLog(ctx, "info", fmt.Sprintf("已发送到漏洞扫描模块: MySQL弱口令 - %s@%s (账号密码: %s)", host, credential, credential))

					vulnURL := fmt.Sprintf("mysql://%s", host)
					runtime.EventsEmit(ctx, "nucleiResult", map[string]interface{}{
						"task_id":     taskId,
						"template_id": "mysql-weak-password",
						"vuln_name":   "MySQL弱口令",
						"vuln_url":    vulnURL,
						"severity":    "HIGH",
						"protocol":    "MySQL",
						"extract":     credential,
					})

					emitCrackResult(ctx, taskId, "MySQL", vulnURL, credential, t.User, pass)

					saveCrackResultToDB(
						ctx,
						taskId,
						"mysql-weak-password",
						"MySQL弱口令",
						vulnURL,
						"HIGH",
						"MySQL",
						credential,
						"",
						"",
					)

					select {
					case successChan <- true:
					default:
					}
					return
				} else if err != nil {
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
			sendCrackLog(ctx, "warning", fmt.Sprintf("MySQL爆破超时: 目标=%s, 已尝试=%d次", host, atomic.LoadInt64(&attempted)))
			return
		}
	}

	finalAttempted := atomic.LoadInt64(&attempted)
	if !success {
		sendCrackLog(ctx, "info", fmt.Sprintf("MySQL爆破完成: 目标=%s, 已尝试=%d次, 未发现弱口令", host, finalAttempted))
	}
}

var (
	mysqlDialMutex   sync.Mutex
	mysqlDialCounter int64
)

// MysqlConn 测试MySQL连接（支持SSH隧道）
func MysqlConn(host, user, pass string, sshConfig *SSHConfig, timeout time.Duration) (bool, error) {
	var lastErr error
	
		// 构建数据源名称，使用传入的timeout
		timeoutStr := fmt.Sprintf("%.0fs", timeout.Seconds())
		buildDSN := func(database string) string {
			if database == "" {
				return fmt.Sprintf("%s:%s@tcp(%s)/?charset=utf8&timeout=%s&readTimeout=%s&writeTimeout=%s", user, pass, host, timeoutStr, timeoutStr, timeoutStr)
			}
			return fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8&timeout=%s&readTimeout=%s&writeTimeout=%s", user, pass, host, database, timeoutStr, timeoutStr, timeoutStr)
		}
	
	// 如果配置了SSH隧道，使用自定义Dialer
	if sshConfig != nil && sshConfig.Enabled {
		dialer, err := getSSHDialer(sshConfig)
		if err != nil {
			return false, fmt.Errorf("创建SSH隧道失败: %v", err)
		}
		
		// MySQL驱动支持通过RegisterDial注册自定义dial函数
		mysqlDialMutex.Lock()
		mysqlDialCounter++
		dialKey := fmt.Sprintf("ssh_tunnel_%d", mysqlDialCounter)
		mysqlDialMutex.Unlock()
		
		// 注册自定义dial函数
		mysql.RegisterDial(dialKey, func(addr string) (net.Conn, error) {
			return dialer("tcp", addr)
		})
		
		// 尝试连接，不指定数据库（使用空数据库名）
		cfg, err := mysql.ParseDSN(buildDSN(""))
		if err != nil {
			return false, fmt.Errorf("解析DSN失败: %v", err)
		}
		cfg.Net = dialKey
		
		connector, err := mysql.NewConnector(cfg)
		if err != nil {
			return false, fmt.Errorf("创建MySQL连接器失败: %v", err)
		}
		
		db := sql.OpenDB(connector)
		defer db.Close()
		
		db.SetConnMaxLifetime(timeout)
		db.SetConnMaxIdleTime(timeout)
		db.SetMaxIdleConns(0)
		
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		
		err = db.PingContext(ctx)
		if err == nil {
			return true, nil
		}
		lastErr = err
		
		// 如果空数据库连接失败，尝试指定数据库
		for _, database := range []string{"mysql", "information_schema", "test"} {
			cfg, err := mysql.ParseDSN(buildDSN(database))
			if err != nil {
				continue
			}
			cfg.Net = dialKey
			
			connector, err := mysql.NewConnector(cfg)
			if err != nil {
				lastErr = err
				continue
			}
			
			db2 := sql.OpenDB(connector)
			defer db2.Close()
			
			db2.SetConnMaxLifetime(timeout)
			db2.SetConnMaxIdleTime(timeout)
			db2.SetMaxIdleConns(0)
			
			ctx2, cancel2 := context.WithTimeout(context.Background(), timeout)
			err = db2.PingContext(ctx2)
			cancel2()
			
			if err == nil {
				return true, nil
			}
			lastErr = err
		}
		
		return false, lastErr
	}
	
	// 直连方式（原有逻辑）
	// 尝试连接，不指定数据库（使用空数据库名）
	dataSourceName := buildDSN("")
	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		return false, fmt.Errorf("打开连接失败: %v", err)
	}
	defer db.Close()

	db.SetConnMaxLifetime(timeout)
	db.SetConnMaxIdleTime(timeout)
	db.SetMaxIdleConns(0)

	// 使用 Ping 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	err = db.PingContext(ctx)
	if err == nil {
		return true, nil
	}
	
	// 如果空数据库连接失败，尝试指定数据库
	for _, database := range []string{"mysql", "information_schema", "test"} {
		dataSourceName := buildDSN(database)
		db2, err := sql.Open("mysql", dataSourceName)
		if err != nil {
			lastErr = err
			continue
		}
		defer db2.Close()

		db2.SetConnMaxLifetime(timeout)
		db2.SetConnMaxIdleTime(timeout)
		db2.SetMaxIdleConns(0)
		
		ctx2, cancel2 := context.WithTimeout(context.Background(), timeout)
		err = db2.PingContext(ctx2)
		cancel2()
		
		if err == nil {
			return true, nil
		}
		lastErr = err
	}
	
	return false, lastErr
}

