package crack

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// PostgresScan PostgreSQL协议爆破（使用并发模式）
func PostgresScan(ctx, ctrlCtx context.Context, taskId, host string, usernames, passwords []string, sshConfig *SSHConfig, config *CrackConfig) {
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

	sendCrackLog(ctx, "info", fmt.Sprintf("PostgreSQL爆破开始: 目标=%s, 用户名数量=%d, 密码数量=%d, 总尝试次数=%d, 并发数=%d, 超时=%v", host, len(usernames), len(passwords), totalAttempts, workers, timeout))

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

				flag, err := PostgresConn(host, t.User, pass, sshConfig)
				if flag && err == nil {
					credential := fmt.Sprintf("%s/%s", t.User, pass)
					vulnURL := fmt.Sprintf("postgresql://%s", host)
					runtime.EventsEmit(ctx, "nucleiResult", map[string]interface{}{
						"task_id":     taskId,
						"template_id": "postgresql-weak-password",
						"vuln_name":   "PostgreSQL弱口令",
						"vuln_url":    vulnURL,
						"severity":    "HIGH",
						"protocol":    "PostgreSQL",
						"extract":     credential,
					})

					emitCrackResult(ctx, taskId, "PostgreSQL", vulnURL, credential, t.User, pass)

					saveCrackResultToDB(
						ctx,
						taskId,
						"postgresql-weak-password",
						"PostgreSQL弱口令",
						vulnURL,
						"HIGH",
						"PostgreSQL",
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
			sendCrackLog(ctx, "warning", fmt.Sprintf("PostgreSQL爆破超时: 目标=%s, 已尝试=%d次", host, atomic.LoadInt64(&attempted)))
			return
		}
	}

	finalAttempted := atomic.LoadInt64(&attempted)
	if !success {
		sendCrackLog(ctx, "info", fmt.Sprintf("PostgreSQL爆破完成: 目标=%s, 已尝试=%d次, 未发现弱口令", host, finalAttempted))
	}
}

// pgSSHDialer 实现pq.Dialer接口以支持SSH隧道
type pgSSHDialer struct {
	dialer func(network, addr string) (net.Conn, error)
}

func (d *pgSSHDialer) Dial(network, address string) (net.Conn, error) {
	return d.dialer(network, address)
}

func (d *pgSSHDialer) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	conn, err := d.dialer(network, address)
	if err != nil {
		return nil, err
	}
	conn.SetDeadline(time.Now().Add(timeout))
	return conn, nil
}

// postgresSSHConnector 包装PostgreSQL连接以支持SSH隧道
type postgresSSHConnector struct {
	conn driver.Conn
}

func (c *postgresSSHConnector) Connect(ctx context.Context) (driver.Conn, error) {
	return c.conn, nil
}

func (c *postgresSSHConnector) Driver() driver.Driver {
	return &postgresSSHDriver{}
}

type postgresSSHDriver struct{}

func (d *postgresSSHDriver) Open(name string) (driver.Conn, error) {
	return nil, fmt.Errorf("not implemented")
}

// PostgresConn 测试PostgreSQL连接（支持SSH隧道）
func PostgresConn(host, user, pass string, sshConfig *SSHConfig) (bool, error) {
	hostParts := strings.Split(host, ":")
	hostname := hostParts[0]
	port := "5432"
	if len(hostParts) > 1 {
		port = hostParts[1]
	}
	
	// 尝试连接不同的数据库
	for _, database := range []string{"postgres", "template1"} {
		connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable connect_timeout=10",
			hostname, port, user, pass, database)
		
		var db *sql.DB
		var err error
		
		// 如果配置了SSH隧道，使用自定义Dialer
		if sshConfig != nil && sshConfig.Enabled {
			dialer, err := getSSHDialer(sshConfig)
			if err != nil {
				continue
			}
			
			pgDialer := &pgSSHDialer{dialer: dialer}
			conn, err := pq.DialOpen(pgDialer, connStr)
			if err != nil {
				continue
			}
			
			connector := &postgresSSHConnector{conn: conn}
			db = sql.OpenDB(connector)
		} else {
			db, err = sql.Open("postgres", connStr)
			if err != nil {
				continue
			}
		}
		
		defer db.Close()

		db.SetConnMaxLifetime(10 * time.Second)
		db.SetConnMaxIdleTime(10 * time.Second)
		db.SetMaxIdleConns(0)

		err = db.Ping()
		if err == nil {
			return true, nil
		}
	}
	return false, fmt.Errorf("连接失败")
}

