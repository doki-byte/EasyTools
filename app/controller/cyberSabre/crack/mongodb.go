package crack

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/crypto/ssh"
)

// MongodbScan MongoDB协议爆破（使用并发模式）
func MongodbScan(ctx, ctrlCtx context.Context, taskId, host string, usernames, passwords []string, sshConfig *SSHConfig, config *CrackConfig) {
	workers := config.Threads
	if workers <= 0 {
		workers = 5
	}

	timeout := time.Duration(config.Timeout) * time.Millisecond
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	// 先检查未授权访问
	flag, err := MongodbUnauth(host, sshConfig)
	if flag && err == nil {
		vulnURL := fmt.Sprintf("mongodb://%s", host)
		runtime.EventsEmit(ctx, "nucleiResult", map[string]interface{}{
			"task_id":     taskId,
			"template_id": "mongodb-unauthorized",
			"vuln_name":   "MongoDB未授权访问",
			"vuln_url":    vulnURL,
			"severity":    "HIGH",
			"protocol":    "MongoDB",
		})

		emitCrackResult(ctx, taskId, "MongoDB", vulnURL, "", "", "未授权访问")

		saveCrackResultToDB(
			ctx,
			taskId,
			"mongodb-unauthorized",
			"MongoDB未授权访问",
			vulnURL,
			"HIGH",
			"MongoDB",
			"",
			"",
			"",
		)
		return
	}

	// 使用并发模式进行密码爆破
	totalAttempts := len(usernames) * len(passwords)
	var attempted int64

	sendCrackLog(ctx, "info", fmt.Sprintf("MongoDB爆破开始: 目标=%s, 用户名数量=%d, 密码数量=%d, 总尝试次数=%d, 并发数=%d, 超时=%v", host, len(usernames), len(passwords), totalAttempts, workers, timeout))

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

				flag, err := MongodbConn(host, t.User, pass, sshConfig)
				if flag && err == nil {
					credential := fmt.Sprintf("%s/%s", t.User, pass)
					vulnURL := fmt.Sprintf("mongodb://%s", host)
					runtime.EventsEmit(ctx, "nucleiResult", map[string]interface{}{
						"task_id":     taskId,
						"template_id": "mongodb-weak-password",
						"vuln_name":   "MongoDB弱口令",
						"vuln_url":    vulnURL,
						"severity":    "HIGH",
						"protocol":    "MongoDB",
						"extract":     credential,
					})

					emitCrackResult(ctx, taskId, "MongoDB", vulnURL, credential, t.User, pass)

					saveCrackResultToDB(
						ctx,
						taskId,
						"mongodb-weak-password",
						"MongoDB弱口令",
						vulnURL,
						"HIGH",
						"MongoDB",
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
			sendCrackLog(ctx, "warning", fmt.Sprintf("MongoDB爆破超时: 目标=%s, 已尝试=%d次", host, atomic.LoadInt64(&attempted)))
			return
		}
	}

	finalAttempted := atomic.LoadInt64(&attempted)
	if !success {
		sendCrackLog(ctx, "info", fmt.Sprintf("MongoDB爆破完成: 目标=%s, 已尝试=%d次, 未发现弱口令", host, finalAttempted))
	}
}

// MongodbConn 测试MongoDB连接（带认证，支持SSH隧道）
func MongodbConn(address, user, pass string, sshConfig *SSHConfig) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client()
	
	// 如果配置了SSH隧道，设置自定义Dialer
	if sshConfig != nil && sshConfig.Enabled {
		dialer, err := getSSHDialer(sshConfig)
		if err != nil {
			return false, fmt.Errorf("创建SSH隧道失败: %v", err)
		}
		
		sshDialer := &sshContextDialer{sshClient: nil}
		sshDialer.dialer = dialer
		clientOptions.SetDialer(sshDialer)
	}

	// 尝试不同的数据库名
	for _, dbName := range []string{"admin", "test"} {
		uri := fmt.Sprintf("mongodb://%s:%s@%s/%s?authSource=%s&connectTimeoutMS=10000",
			user, pass, address, dbName, dbName)
		client, err := mongo.Connect(ctx, clientOptions.ApplyURI(uri))
		if err != nil {
			continue
		}

		err = client.Ping(ctx, nil)
		if err == nil {
			client.Disconnect(ctx)
			return true, nil
		}
		client.Disconnect(ctx)
	}
	return false, fmt.Errorf("连接失败")
}

// MongodbUnauth 检查MongoDB未授权访问（支持SSH隧道）
func MongodbUnauth(address string, sshConfig *SSHConfig) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client()
	
	// 如果配置了SSH隧道，设置自定义Dialer
	if sshConfig != nil && sshConfig.Enabled {
		dialer, err := getSSHDialer(sshConfig)
		if err != nil {
			return false, fmt.Errorf("创建SSH隧道失败: %v", err)
		}
		
		sshDialer := &sshContextDialer{sshClient: nil}
		sshDialer.dialer = dialer
		clientOptions.SetDialer(sshDialer)
	}

	uri := fmt.Sprintf("mongodb://%s/?connectTimeoutMS=10000", address)
	client, err := mongo.Connect(ctx, clientOptions.ApplyURI(uri))
	if err != nil {
		return false, err
	}
	defer client.Disconnect(ctx)

	err = client.Ping(ctx, nil)
	if err == nil {
		return true, nil
	}
	return false, fmt.Errorf("需要认证")
}

// sshContextDialer 实现MongoDB驱动的ContextDialer接口，通过SSH隧道连接
type sshContextDialer struct {
	sshClient *ssh.Client
	dialer    func(network, addr string) (net.Conn, error)
}

// DialContext 实现ContextDialer接口的DialContext方法
func (d *sshContextDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	if d.dialer != nil {
		conn, err := d.dialer(network, addr)
		if err != nil {
			return nil, fmt.Errorf("通过SSH隧道连接MongoDB失败 [目标: %s]: %v", addr, err)
		}
		return &sshConnWrapper{Conn: conn}, nil
	}
	return nil, fmt.Errorf("SSH dialer未初始化")
}
