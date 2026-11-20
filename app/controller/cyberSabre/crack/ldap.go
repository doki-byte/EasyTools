package crack

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-ldap/ldap/v3"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// LdapScan LDAP协议爆破（使用并发模式）
func LdapScan(ctx, ctrlCtx context.Context, taskId, host string, usernames, passwords []string, sshConfig *SSHConfig, config *CrackConfig) {
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

	sendCrackLog(ctx, "info", fmt.Sprintf("LDAP爆破开始: 目标=%s, 用户名数量=%d, 密码数量=%d, 总尝试次数=%d, 并发数=%d, 超时=%v", host, len(usernames), len(passwords), totalAttempts, workers, timeout))

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

				flag, err := LdapConn(host, t.User, pass, sshConfig)
				if flag && err == nil {
					credential := fmt.Sprintf("%s/%s", t.User, pass)
					vulnURL := fmt.Sprintf("ldap://%s", host)
					runtime.EventsEmit(ctx, "nucleiResult", map[string]interface{}{
						"task_id":     taskId,
						"template_id": "ldap-weak-password",
						"vuln_name":   "LDAP弱口令",
						"vuln_url":    vulnURL,
						"severity":    "HIGH",
						"protocol":    "LDAP",
						"extract":     credential,
					})

					emitCrackResult(ctx, taskId, "LDAP", vulnURL, credential, t.User, pass)

					saveCrackResultToDB(
						ctx,
						taskId,
						"ldap-weak-password",
						"LDAP弱口令",
						vulnURL,
						"HIGH",
						"LDAP",
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
			sendCrackLog(ctx, "warning", fmt.Sprintf("LDAP爆破超时: 目标=%s, 已尝试=%d次", host, atomic.LoadInt64(&attempted)))
			return
		}
	}

	finalAttempted := atomic.LoadInt64(&attempted)
	if !success {
		sendCrackLog(ctx, "info", fmt.Sprintf("LDAP爆破完成: 目标=%s, 已尝试=%d次, 未发现弱口令", host, finalAttempted))
	}
}

// LdapConn 测试LDAP连接（支持SSH隧道）
func LdapConn(address, user, pass string, sshConfig *SSHConfig) (bool, error) {
	var conn *ldap.Conn
	var err error
	
	// 如果配置了SSH隧道，使用自定义连接
	if sshConfig != nil && sshConfig.Enabled {
		// 通过SSH隧道建立TCP连接
		netConn, err := dialWithSSH(sshConfig, "tcp", address, 10*time.Second)
		if err != nil {
			return false, err
		}
		
		// 使用ldap.NewConn创建LDAP连接
		conn = ldap.NewConn(netConn, false)
		conn.Start()
	} else {
		// 直连方式
		conn, err = ldap.DialURL(fmt.Sprintf("ldap://%s", address))
		if err != nil {
			return false, err
		}
	}
	defer conn.Close()

	conn.SetTimeout(10 * time.Second)

	// 尝试不同的DN格式
	dnFormats := []string{
		fmt.Sprintf("cn=%s,dc=example,dc=com", user),
		fmt.Sprintf("uid=%s,dc=example,dc=com", user),
		fmt.Sprintf("cn=%s,ou=users,dc=example,dc=com", user),
		fmt.Sprintf("%s", user), // 直接使用用户名
	}

	for _, dn := range dnFormats {
		err = conn.Bind(dn, pass)
		if err == nil {
			return true, nil
		}
		// 如果是无效凭证错误，继续尝试下一个格式
		if ldap.IsErrorWithCode(err, ldap.LDAPResultInvalidCredentials) {
			continue
		}
		// 其他错误直接返回
		return false, err
	}

	return false, fmt.Errorf("认证失败")
}

