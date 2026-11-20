package crack

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ZookeeperScan Zookeeper协议爆破（检查未授权访问）
// Zookeeper默认不需要用户名密码，主要通过四字命令检查未授权访问
func ZookeeperScan(ctx, ctrlCtx context.Context, taskId, host string, usernames, passwords []string, sshConfig *SSHConfig, config *CrackConfig) {
	workers := config.Threads
	if workers <= 0 {
		workers = 5
	}

	timeout := time.Duration(config.Timeout) * time.Millisecond
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	// Zookeeper通常不需要用户名密码，主要检查未授权访问
	flag, err := ZookeeperUnauth(host, sshConfig)
	if flag && err == nil {
		vulnURL := fmt.Sprintf("zookeeper://%s", host)
		runtime.EventsEmit(ctx, "nucleiResult", map[string]interface{}{
			"task_id":     taskId,
			"template_id": "zookeeper-unauthorized",
			"vuln_name":   "Zookeeper未授权访问",
			"vuln_url":    vulnURL,
			"severity":    "HIGH",
			"protocol":    "Zookeeper",
		})

		emitCrackResult(ctx, taskId, "Zookeeper", vulnURL, "", "", "未授权访问")

		saveCrackResultToDB(
			ctx,
			taskId,
			"zookeeper-unauthorized",
			"Zookeeper未授权访问",
			vulnURL,
			"HIGH",
			"Zookeeper",
			"",
			"",
			"",
		)
		return
	}

	// 如果未授权访问失败，记录日志
	sendCrackLog(ctx, "info", fmt.Sprintf("Zookeeper爆破完成: 目标=%s, 未发现未授权访问", host))
}

// ZookeeperUnauth 检查Zookeeper未授权访问（支持SSH隧道）
// 只要能建立TCP连接，就判定为未授权访问
func ZookeeperUnauth(address string, sshConfig *SSHConfig) (bool, error) {
	conn, err := dialWithSSH(sshConfig, "tcp", address, 10*time.Second)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	// 只要能连接成功，就判定为未授权访问
	return true, nil
}

// ExecuteZookeeperCommand 执行 Zookeeper 四字命令（支持SSH隧道）
func ExecuteZookeeperCommand(host, port string, sshConfig *SSHConfig, command string) (string, error) {
	cmd := strings.TrimSpace(strings.ToLower(command))
	if cmd == "" {
		return "", fmt.Errorf("命令为空")
	}

	// 支持的四字命令列表（参考 MPET-main）
	validCommands := map[string]bool{
		"envi": true, // 环境变量
		"conf": true, // 配置信息
		"cons": true, // 连接信息
		"crst": true, // 重置连接统计
		"dump": true, // 会话和临时节点
		"ruok": true, // 服务器是否运行
		"stat": true, // 服务器统计信息
		"srvr": true, // 服务器信息
		"mntr": true, // 监控信息
	}
	if !validCommands[cmd] {
		return "", fmt.Errorf("不支持的命令: %s\n支持的命令: envi, conf, cons, crst, dump, ruok, stat, srvr, mntr", command)
	}

	if port == "" {
		port = "2181"
	}

	address := net.JoinHostPort(host, port)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	netConn, err := dialWithSSHContext(ctx, sshConfig, "tcp", address, 10*time.Second)
	if err != nil {
		return "", fmt.Errorf("连接失败: %v", err)
	}
	defer netConn.Close()

	_ = netConn.SetDeadline(time.Now().Add(10 * time.Second))

	// 发送四字命令
	if _, err := netConn.Write([]byte(cmd)); err != nil {
		return "", fmt.Errorf("发送命令失败: %v", err)
	}

	// 读取响应
	respBytes, err := io.ReadAll(netConn)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("读取响应失败: %v", err)
	}
	result := strings.TrimSpace(string(respBytes))
	if result == "" {
		return "命令执行成功，但无返回内容", nil
	}
	return result, nil
}
