package crack

import (
	"context"
	"fmt"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// JdwpScan JDWP协议爆破
func JdwpScan(ctx, ctrlCtx context.Context, taskId, host string, usernames, passwords []string, sshConfig *SSHConfig, config *CrackConfig) {
	// JDWP（Java Debug Wire Protocol）通常不需要密码
	// 先检查未授权访问
	flag, err := JdwpUnauth(host, sshConfig)
	if flag && err == nil {
		vulnURL := fmt.Sprintf("jdwp://%s", host)
		response := "JDWP未授权访问可能导致代码执行"
		runtime.EventsEmit(ctx, "nucleiResult", map[string]interface{}{
			"task_id":     taskId,
			"template_id": "jdwp-unauthorized",
			"vuln_name":    "JDWP未授权访问",
			"vuln_url":     vulnURL,
			"severity":     "HIGH",
			"protocol":     "JDWP",
			"response":     response,
		})
		
		// 发送crackResult事件到密码爆破模块（未授权访问，无用户名密码）
		emitCrackResult(ctx, taskId, "JDWP", vulnURL, "", "", "未授权访问")
		
		// 保存结果到数据库
		saveCrackResultToDB(
			ctx,
			taskId,
			"jdwp-unauthorized",
			"JDWP未授权访问",
			vulnURL,
			"HIGH",
			"JDWP",
			"",
			"",
			response,
		)
		return
	}
}

// JdwpUnauth 检查JDWP未授权访问（支持SSH隧道）
func JdwpUnauth(address string, sshConfig *SSHConfig) (bool, error) {
	conn, err := dialWithSSH(sshConfig, "tcp", address, 10*time.Second)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// JDWP协议握手
	// JDWP握手包格式：长度(4字节) + "JDWP-Handshake"(14字节)
	handshake := "JDWP-Handshake"
	handshakeLen := len(handshake)
	packet := []byte{
		byte(handshakeLen >> 24),
		byte(handshakeLen >> 16),
		byte(handshakeLen >> 8),
		byte(handshakeLen),
	}
	packet = append(packet, []byte(handshake)...)

	_, err = conn.Write(packet)
	if err != nil {
		return false, err
	}

	// 读取响应
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return false, err
	}

	// 检查响应是否是JDWP握手响应
	if n >= len(handshake) {
		response := string(buffer[4:n]) // 跳过长度字段
		if response == handshake {
			return true, nil
		}
	}

	return false, fmt.Errorf("不是JDWP服务")
}

