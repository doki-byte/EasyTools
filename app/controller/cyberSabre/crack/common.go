package crack

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/crypto/ssh"
)

// task 爆破任务结构
type task struct {
	User string
	Pass string
}

// sendCrackLog 发送爆破日志（通用函数，供所有协议使用）
func sendCrackLog(ctx context.Context, level, message string) {
	if ctx == nil {
		return
	}
	eventData := map[string]interface{}{
		"level":   level,
		"message": message,
		"time":    time.Now().Format("15:04:05"),
		"source":  "密码爆破",
	}
	runtime.EventsEmit(ctx, "log_event", eventData)
}

// saveCrackResultToDB 保存密码爆破结果到JSON（不再保存到数据库，改为通过事件发送到前端保存到JSON）
// 注意：函数名保留为 saveCrackResultToDB 以保持兼容性，但实际不再保存到数据库
// 结果已经通过 nucleiResult 事件发送到前端，前端会调用 EnsureCrackTaskRecord 保存到JSON
func saveCrackResultToDB(ctx context.Context, taskId, templateId, vulnName, vulnURL, severity, protocol, extract, request, response string) {
	// 结果已经通过 nucleiResult 事件发送到前端，前端会调用 EnsureCrackTaskRecord 保存到JSON
	// 这里不再保存到数据库，只是记录日志
	if ctx != nil {
		sendCrackLog(ctx, "info", fmt.Sprintf("密码爆破结果已发送到前端保存到JSON: %s - %s", protocol, vulnURL))
	}
}

// emitCrackResult 发送密码爆破结果事件到前端（供密码爆破页面保存）
// 解析vulnURL（格式：protocol://host:port）和extract（格式：username/password 或 password）
// 并发送crackResult事件
func emitCrackResult(ctx context.Context, taskId, protocol, vulnURL, extract, username, password string) {
	if ctx == nil {
		return
	}
	
	// 解析host和port
	hostPart := ""
	portPart := ""
	
	// 尝试多种格式解析
	protocolLower := strings.ToLower(protocol)
	protocolPrefix := protocolLower + "://"
	
	if strings.HasPrefix(vulnURL, protocolPrefix) {
		addr := strings.TrimPrefix(vulnURL, protocolPrefix)
		if strings.Contains(addr, ":") {
			parts := strings.Split(addr, ":")
			if len(parts) == 2 {
				hostPart = parts[0]
				portPart = parts[1]
			} else if len(parts) > 2 {
				// 处理IPv6地址或其他包含多个冒号的情况
				hostPart = strings.Join(parts[:len(parts)-1], ":")
				portPart = parts[len(parts)-1]
			}
		} else {
			hostPart = addr
			// 根据协议设置默认端口
			defaultPorts := map[string]string{
				"FTP":          "21",
				"SSH":          "22",
				"Telnet":       "23",
				"LDAP":         "389",
				"SMB":          "445",
				"RDP":          "3389",
				"MySQL":        "3306",
				"MSSQL":        "1433",
				"PostgreSQL":   "5432",
				"Oracle":       "1521",
				"Redis":        "6379",
				"MongoDB":      "27017",
				"VNC":          "5900",
				"Socks5":       "1080",
				"MQTT":         "1883",
				"MemCached":    "11211",
				"ActiveMQ":     "61616",
				"Rsync":        "873",
				"Kafka":        "9092",
				"RMI":          "1099",
				"JDWP":         "5005",
				"ADB":          "5555",
				"Zookeeper":    "2181",
				"RabbitMQ":     "5672",
				"Elasticsearch": "9200",
				"Etcd":         "2379",
			}
			if port, ok := defaultPorts[protocol]; ok {
				portPart = port
			}
		}
	} else {
		// 如果vulnURL格式不正确，尝试从vulnURL中直接提取
		// 或者使用传入的username和password中的信息
		if strings.Contains(vulnURL, "://") {
			// 尝试提取协议后的部分
			parts := strings.SplitN(vulnURL, "://", 2)
			if len(parts) == 2 {
				addr := parts[1]
				if strings.Contains(addr, ":") {
					addrParts := strings.Split(addr, ":")
					if len(addrParts) == 2 {
						hostPart = addrParts[0]
						portPart = addrParts[1]
					}
				} else {
					hostPart = addr
				}
			}
		}
	}
	
	// 如果hostPart仍然为空，记录警告
	if hostPart == "" {
		sendCrackLog(ctx, "warning", fmt.Sprintf("无法从vulnURL解析host: %s, protocol: %s", vulnURL, protocol))
	}
	
	// 解析用户名和密码
	finalUsername := username
	finalPassword := password
	if extract != "" && finalUsername == "" && finalPassword == "" {
		if strings.Contains(extract, "/") {
			parts := strings.SplitN(extract, "/", 2)
			if len(parts) == 2 {
				finalUsername = parts[0]
				finalPassword = parts[1]
			} else {
				finalPassword = extract
			}
		} else {
			finalPassword = extract
		}
	}
	
	// 发送crackResult事件
	crackResultData := map[string]interface{}{
		"task_id":    taskId,
		"protocol":   protocol,
		"host":       hostPart,
		"port":       portPart,
		"username":   finalUsername,
		"password":   finalPassword,
		"status":     "success",
		"message":    "密码爆破成功",
		"credential": extract,
		"vuln_url":   vulnURL,
	}
	
	// 记录发送事件日志
	sendCrackLog(ctx, "info", fmt.Sprintf("发送crackResult事件: protocol=%s, host=%s, port=%s, username=%s", protocol, hostPart, portPart, finalUsername))
	
	runtime.EventsEmit(ctx, "crackResult", crackResultData)
}

// SSHConfig SSH隧道配置
type SSHConfig struct {
	Enabled  bool   `json:"enabled"`
	Address  string `json:"address"`  // SSH服务器地址
	Port     string `json:"port"`     // SSH端口
	Username string `json:"username"` // SSH用户名
	Password string `json:"password"` // SSH密码
}

// sshConnWrapper 包装SSH连接，忽略deadline相关操作
// SSH通道不支持SetReadDeadline和SetWriteDeadline，需要包装以兼容某些客户端
type sshConnWrapper struct {
	net.Conn
}

// SetDeadline 忽略deadline设置（SSH通道不支持）
func (w *sshConnWrapper) SetDeadline(t time.Time) error {
	return nil
}

// SetReadDeadline 忽略read deadline设置（SSH通道不支持）
func (w *sshConnWrapper) SetReadDeadline(t time.Time) error {
	return nil
}

// SetWriteDeadline 忽略write deadline设置（SSH通道不支持）
func (w *sshConnWrapper) SetWriteDeadline(t time.Time) error {
	return nil
}

// getSSHClient 创建SSH客户端连接
func getSSHClient(config *SSHConfig) (*ssh.Client, error) {
	if !config.Enabled || config.Address == "" || config.Username == "" {
		return nil, fmt.Errorf("SSH配置不完整")
	}

	port := config.Port
	if port == "" {
		port = "22"
	}

	sshConfig := &ssh.ClientConfig{
		User: config.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(config.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}

	addr := fmt.Sprintf("%s:%s", config.Address, port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("SSH连接失败: %v", err)
	}

	return client, nil
}

// getSSHDialer 获取通过SSH隧道连接的Dialer函数
func getSSHDialer(config *SSHConfig) (func(network, addr string) (net.Conn, error), error) {
	sshClient, err := getSSHClient(config)
	if err != nil {
		return nil, err
	}

	// 返回一个Dialer函数，该函数通过SSH隧道连接到目标地址
	return func(network, addr string) (net.Conn, error) {
		conn, err := sshClient.Dial(network, addr)
		if err != nil {
			return nil, fmt.Errorf("通过SSH隧道连接失败: %v", err)
		}
		// 包装SSH连接以忽略deadline操作
		return &sshConnWrapper{Conn: conn}, nil
	}, nil
}

// dialWithSSH 通过SSH隧道或直连方式建立连接
func dialWithSSH(config *SSHConfig, network, addr string, timeout time.Duration) (net.Conn, error) {
	if config != nil && config.Enabled {
		dialer, err := getSSHDialer(config)
		if err != nil {
			return nil, err
		}
		return dialer(network, addr)
	}
	return net.DialTimeout(network, addr, timeout)
}

// dialWithSSHContext 通过SSH隧道或直连方式建立连接（带上下文）
func dialWithSSHContext(ctx context.Context, config *SSHConfig, network, addr string, timeout time.Duration) (net.Conn, error) {
	if config != nil && config.Enabled {
		dialer, err := getSSHDialer(config)
		if err != nil {
			return nil, err
		}
		// 使用 context 超时控制
		done := make(chan net.Conn, 1)
		errChan := make(chan error, 1)
		go func() {
			conn, err := dialer(network, addr)
			if err != nil {
				errChan <- err
				return
			}
			done <- conn
		}()
		select {
		case conn := <-done:
			return conn, nil
		case err := <-errChan:
			return nil, err
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(timeout):
			return nil, fmt.Errorf("连接超时")
		}
	}
	dialer := &net.Dialer{Timeout: timeout}
	return dialer.DialContext(ctx, network, addr)
}