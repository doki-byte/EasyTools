package crack

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// RmiScan RMI协议爆破
func RmiScan(ctx, ctrlCtx context.Context, taskId, host string, usernames, passwords []string, sshConfig *SSHConfig, config *CrackConfig) {
	// RMI通常不需要用户名密码，但某些实现可能有认证
	// 先检查未授权访问
	flag, err := RmiUnauth(host, sshConfig)
	if flag && err == nil {
		vulnURL := fmt.Sprintf("rmi://%s", host)
		runtime.EventsEmit(ctx, "nucleiResult", map[string]interface{}{
			"task_id":     taskId,
			"template_id": "rmi-unauthorized",
			"vuln_name":    "RMI未授权访问",
			"vuln_url":     vulnURL,
			"severity":     "HIGH",
			"protocol":     "RMI",
		})
		
		// 发送crackResult事件到密码爆破模块（未授权访问，无用户名密码）
		emitCrackResult(ctx, taskId, "RMI", vulnURL, "", "", "未授权访问")
		
		// 保存结果到数据库
		saveCrackResultToDB(
			ctx,
			taskId,
			"rmi-unauthorized",
			"RMI未授权访问",
			vulnURL,
			"HIGH",
			"RMI",
			"",
			"",
			"",
		)
		return
	}

	// RMI协议本身不直接支持密码认证，但可以通过JMX或其他方式
	// 这里提供一个占位实现
	vulnURL := fmt.Sprintf("rmi://%s", host)
	response := "RMI协议通常不需要密码认证，但可能存在反序列化漏洞"
	runtime.EventsEmit(ctx, "nucleiResult", map[string]interface{}{
		"task_id":     taskId,
		"template_id": "rmi-info",
		"vuln_name":    "RMI服务检测",
		"vuln_url":     vulnURL,
		"severity":     "INFO",
		"protocol":     "RMI",
		"response":     response,
	})
	
	// 发送crackResult事件到密码爆破模块（信息类结果，无用户名密码）
	emitCrackResult(ctx, taskId, "RMI", vulnURL, "", "", "未授权访问")
	
	// 保存结果到数据库
	saveCrackResultToDB(
		ctx,
		taskId,
		"rmi-info",
		"RMI服务检测",
		vulnURL,
		"INFO",
		"RMI",
		"",
		"",
		response,
	)
}

// RmiUnauth 检查RMI未授权访问（支持SSH隧道）
func RmiUnauth(address string, sshConfig *SSHConfig) (bool, error) {
	conn, err := dialWithSSH(sshConfig, "tcp", address, 10*time.Second)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// RMI协议握手（简化实现）
	// RMI使用Java RMI协议，通常以特定的字节序列开始
	// 发送RMI握手请求
	handshake := []byte{0x4a, 0x52, 0x4d, 0x49} // "JRMI"
	_, err = conn.Write(handshake)
	if err != nil {
		return false, err
	}

	// 读取响应
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		return false, err
	}

	// 检查是否是RMI响应
	if n > 0 {
		response := string(buffer[:n])
		if strings.Contains(response, "JRMI") || n > 4 {
			return true, nil
		}
	}

	return false, fmt.Errorf("不是RMI服务")
}

