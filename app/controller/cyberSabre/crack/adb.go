package crack

import (
	"context"
	"fmt"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// AdbScan ADB协议爆破
func AdbScan(ctx, ctrlCtx context.Context, taskId, host string, usernames, passwords []string, sshConfig *SSHConfig, config *CrackConfig) {
	// ADB（Android Debug Bridge）通常不需要密码，但某些实现可能有认证
	// 先检查未授权访问
	flag, err := AdbUnauth(host, sshConfig)
	if flag && err == nil {
		vulnURL := fmt.Sprintf("adb://%s", host)
		response := "ADB未授权访问可能导致设备被控制"
		runtime.EventsEmit(ctx, "nucleiResult", map[string]interface{}{
			"task_id":     taskId,
			"template_id": "adb-unauthorized",
			"vuln_name":    "ADB未授权访问",
			"vuln_url":     vulnURL,
			"severity":     "HIGH",
			"protocol":     "ADB",
			"response":     response,
		})
		
		// 发送crackResult事件到密码爆破模块（未授权访问，无用户名密码）
		emitCrackResult(ctx, taskId, "ADB", vulnURL, "", "", "未授权访问")
		
		// 保存结果到数据库
		saveCrackResultToDB(
			ctx,
			taskId,
			"adb-unauthorized",
			"ADB未授权访问",
			vulnURL,
			"HIGH",
			"ADB",
			"",
			"",
			response,
		)
		return
	}
}

// AdbUnauth 检查ADB未授权访问（支持SSH隧道）
func AdbUnauth(address string, sshConfig *SSHConfig) (bool, error) {
	conn, err := dialWithSSH(sshConfig, "tcp", address, 10*time.Second)
	if err != nil {
		return false, err
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(10 * time.Second))

	// ADB协议握手
	// ADB协议以"CNXN"命令开始
	// 格式：命令(4字节) + 版本(4字节) + 最大数据长度(4字节) + 系统类型(4字节) + 序列号(4字节) + 消息长度(4字节) + 消息内容
	cnxnCommand := []byte{'C', 'N', 'X', 'N'} // CNXN命令
	version := uint32(0x01000000)              // 版本1.0.0.0
	maxData := uint32(256 * 1024)              // 256KB
	systemType := []byte{'h', 'o', 's', 't'}   // host
	sequence := uint32(1)
	message := "host::\x00"
	messageLen := uint32(len(message))

	packet := []byte{}
	packet = append(packet, cnxnCommand...)
	packet = append(packet, byte(version>>24), byte(version>>16), byte(version>>8), byte(version))
	packet = append(packet, byte(maxData>>24), byte(maxData>>16), byte(maxData>>8), byte(maxData))
	packet = append(packet, systemType...)
	packet = append(packet, byte(sequence>>24), byte(sequence>>16), byte(sequence>>8), byte(sequence))
	packet = append(packet, byte(messageLen>>24), byte(messageLen>>16), byte(messageLen>>8), byte(messageLen))
	packet = append(packet, []byte(message)...)

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

	// 检查响应是否是CNXN响应
	if n >= 4 {
		responseCmd := string(buffer[0:4])
		if responseCmd == "CNXN" {
			return true, nil
		}
	}

	return false, fmt.Errorf("不是ADB服务")
}

