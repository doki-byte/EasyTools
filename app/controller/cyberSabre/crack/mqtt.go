package crack

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func mqttEncodeRemainingLength(length int) []byte {
	if length < 0 {
		return []byte{0}
	}
	encoded := make([]byte, 0, 4)
	for {
		digit := byte(length % 128)
		length /= 128
		if length > 0 {
			digit |= 0x80
		}
		encoded = append(encoded, digit)
		if length == 0 {
			break
		}
	}
	return encoded
}

func mqttWriteUTF8Field(buf []byte, value string) []byte {
	buf = append(buf, byte(len(value)>>8), byte(len(value)&0xFF))
	return append(buf, []byte(value)...)
}

func mqttBuildConnectPacket(clientID, user, pass string, keepAlive uint16) []byte {
	var variableAndPayload []byte
	variableAndPayload = mqttWriteUTF8Field(variableAndPayload, "MQTT")
	variableAndPayload = append(variableAndPayload, byte(4)) // MQTT 3.1.1

	connectFlags := byte(0x02) // Clean Session
	if user != "" {
		connectFlags |= 0x80
	}
	if pass != "" {
		connectFlags |= 0x40
	}
	variableAndPayload = append(variableAndPayload, connectFlags)
	variableAndPayload = append(variableAndPayload, byte(keepAlive>>8), byte(keepAlive&0xFF))

	variableAndPayload = mqttWriteUTF8Field(variableAndPayload, clientID)
	if user != "" {
		variableAndPayload = mqttWriteUTF8Field(variableAndPayload, user)
	}
	if pass != "" {
		variableAndPayload = mqttWriteUTF8Field(variableAndPayload, pass)
	}

	packet := []byte{0x10}
	packet = append(packet, mqttEncodeRemainingLength(len(variableAndPayload))...)
	packet = append(packet, variableAndPayload...)
	return packet
}

func mqttReadPacket(conn net.Conn) (packetType byte, flags byte, payload []byte, err error) {
	header := make([]byte, 1)
	if _, err = io.ReadFull(conn, header); err != nil {
		return 0, 0, nil, err
	}
	packetType = header[0] >> 4
	flags = header[0] & 0x0F

	multiplier := 1
	remainingLength := 0
	for i := 0; i < 4; i++ {
		var b [1]byte
		if _, err = io.ReadFull(conn, b[:]); err != nil {
			return 0, 0, nil, err
		}
		remainingLength += int(b[0]&127) * multiplier
		if (b[0] & 128) == 0 {
			break
		}
		multiplier *= 128
	}

	payload = make([]byte, remainingLength)
	if remainingLength > 0 {
		if _, err = io.ReadFull(conn, payload); err != nil {
			return 0, 0, nil, err
		}
	}
	return packetType, flags, payload, nil
}

func mqttConnect(address, user, pass string, sshConfig *SSHConfig, timeout time.Duration) (net.Conn, byte, error) {
	conn, err := dialWithSSH(sshConfig, "tcp", address, timeout)
	if err != nil {
		return nil, 0, err
	}

	_ = conn.SetDeadline(time.Now().Add(timeout))
	clientID := fmt.Sprintf("easytools-%d", time.Now().UnixNano()%1000000)
	connectPacket := mqttBuildConnectPacket(clientID, user, pass, 60)
	if _, err = conn.Write(connectPacket); err != nil {
		conn.Close()
		return nil, 0, err
	}

	pt, _, payload, err := mqttReadPacket(conn)
	if err != nil {
		conn.Close()
		return nil, 0, fmt.Errorf("读取 CONNACK 失败: %v", err)
	}
	if pt != 2 || len(payload) < 2 {
		conn.Close()
		return nil, 0, fmt.Errorf("MQTT 响应不是 CONNACK")
	}
	returnCode := payload[1]
	if returnCode != 0x00 {
		conn.Close()
		return nil, returnCode, fmt.Errorf("MQTT 认证失败，返回码=%d", returnCode)
	}
	return conn, returnCode, nil
}

// MqttScan MQTT协议爆破（使用并发模式）
func MqttScan(ctx, ctrlCtx context.Context, taskId, host string, usernames, passwords []string, sshConfig *SSHConfig, config *CrackConfig) {
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

	sendCrackLog(ctx, "info", fmt.Sprintf("MQTT爆破开始: 目标=%s, 用户名数量=%d, 密码数量=%d, 总尝试次数=%d, 并发数=%d, 超时=%v", host, len(usernames), len(passwords), totalAttempts, workers, timeout))

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

				flag, err := MqttConn(host, t.User, pass, sshConfig)
				if flag && err == nil {
					credential := fmt.Sprintf("%s/%s", t.User, pass)
					vulnURL := fmt.Sprintf("mqtt://%s", host)
					runtime.EventsEmit(ctx, "nucleiResult", map[string]interface{}{
						"task_id":     taskId,
						"template_id": "mqtt-weak-password",
						"vuln_name":   "MQTT弱口令",
						"vuln_url":    vulnURL,
						"severity":    "HIGH",
						"protocol":    "MQTT",
						"extract":     credential,
					})

					emitCrackResult(ctx, taskId, "MQTT", vulnURL, credential, t.User, pass)

					saveCrackResultToDB(
						ctx,
						taskId,
						"mqtt-weak-password",
						"MQTT弱口令",
						vulnURL,
						"HIGH",
						"MQTT",
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
			sendCrackLog(ctx, "warning", fmt.Sprintf("MQTT爆破超时: 目标=%s, 已尝试=%d次", host, atomic.LoadInt64(&attempted)))
			return
		}
	}

	finalAttempted := atomic.LoadInt64(&attempted)
	if !success {
		sendCrackLog(ctx, "info", fmt.Sprintf("MQTT爆破完成: 目标=%s, 已尝试=%d次, 未发现弱口令", host, finalAttempted))
	}
}

// MqttConn 测试MQTT连接
func MqttConn(address, user, pass string, sshConfig *SSHConfig) (bool, error) {
	conn, _, err := mqttConnect(address, user, pass, sshConfig, 10*time.Second)
	if err != nil {
		return false, err
	}
	defer conn.Close()
	return true, nil
}

// ExecuteMqttCommand 执行 MQTT 协议命令（支持SSH隧道）
// 支持命令：
// - connack
// - ping
// - publish <topic> [message]
// - subscribe <topic> [seconds]
func ExecuteMqttCommand(host, port, username, password string, sshConfig *SSHConfig, command string) (string, error) {
	cmd := strings.TrimSpace(command)
	if cmd == "" {
		return "", fmt.Errorf("命令为空")
	}
	if port == "" {
		port = "1883"
	}

	address := net.JoinHostPort(host, port)
	conn, _, err := mqttConnect(address, username, password, sshConfig, 12*time.Second)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	parts := strings.Fields(cmd)
	mainCmd := strings.ToLower(parts[0])

	switch mainCmd {
	case "connack":
		return fmt.Sprintf("MQTT 连接成功\n目标=%s\n用户=%s\n返回码=0", address, username), nil

	case "ping":
		if _, err := conn.Write([]byte{0xC0, 0x00}); err != nil {
			return "", fmt.Errorf("发送 PINGREQ 失败: %v", err)
		}
		_ = conn.SetReadDeadline(time.Now().Add(8 * time.Second))
		pt, _, _, err := mqttReadPacket(conn)
		if err != nil {
			return "", fmt.Errorf("读取 PINGRESP 失败: %v", err)
		}
		if pt != 13 {
			return "", fmt.Errorf("收到非 PINGRESP 报文，类型=%d", pt)
		}
		return fmt.Sprintf("PING 成功\n目标=%s", address), nil

	case "publish":
		if len(parts) < 2 {
			return "", fmt.Errorf("publish 命令格式错误: publish <topic> [message]")
		}
		topic := parts[1]
		message := ""
		if len(parts) >= 3 {
			message = strings.Join(parts[2:], " ")
		}

		payload := make([]byte, 0, 2+len(topic)+len(message))
		payload = append(payload, byte(len(topic)>>8), byte(len(topic)&0xFF))
		payload = append(payload, []byte(topic)...)
		payload = append(payload, []byte(message)...)
		packet := []byte{0x30} // PUBLISH QoS0
		packet = append(packet, mqttEncodeRemainingLength(len(payload))...)
		packet = append(packet, payload...)
		if _, err := conn.Write(packet); err != nil {
			return "", fmt.Errorf("发送 PUBLISH 失败: %v", err)
		}
		return fmt.Sprintf("PUBLISH 发送成功\n目标=%s\ntopic=%s\nmessage=%s", address, topic, message), nil

	case "subscribe":
		if len(parts) < 2 {
			return "", fmt.Errorf("subscribe 命令格式错误: subscribe <topic> [seconds]")
		}
		topic := parts[1]
		waitSeconds := 3
		if len(parts) >= 3 {
			if n, e := strconv.Atoi(parts[2]); e == nil {
				if n < 1 {
					n = 1
				}
				if n > 30 {
					n = 30
				}
				waitSeconds = n
			}
		}

		packetID := uint16(1)
		var variableAndPayload []byte
		idBuf := make([]byte, 2)
		binary.BigEndian.PutUint16(idBuf, packetID)
		variableAndPayload = append(variableAndPayload, idBuf...)
		variableAndPayload = mqttWriteUTF8Field(variableAndPayload, topic)
		variableAndPayload = append(variableAndPayload, 0x00) // QoS 0

		subPacket := []byte{0x82}
		subPacket = append(subPacket, mqttEncodeRemainingLength(len(variableAndPayload))...)
		subPacket = append(subPacket, variableAndPayload...)

		if _, err := conn.Write(subPacket); err != nil {
			return "", fmt.Errorf("发送 SUBSCRIBE 失败: %v", err)
		}

		_ = conn.SetReadDeadline(time.Now().Add(8 * time.Second))
		pt, _, payload, err := mqttReadPacket(conn)
		if err != nil {
			return "", fmt.Errorf("读取 SUBACK 失败: %v", err)
		}
		if pt != 9 || len(payload) < 3 {
			return "", fmt.Errorf("收到非 SUBACK 报文，类型=%d", pt)
		}
		if payload[2] == 0x80 {
			return "", fmt.Errorf("订阅被拒绝")
		}

		start := time.Now()
		var messages []string
		for time.Since(start) < time.Duration(waitSeconds)*time.Second {
			_ = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			pType, _, pPayload, e := mqttReadPacket(conn)
			if e != nil {
				if ne, ok := e.(net.Error); ok && ne.Timeout() {
					continue
				}
				break
			}
			if pType != 3 || len(pPayload) < 2 {
				continue
			}
			topicLen := int(binary.BigEndian.Uint16(pPayload[:2]))
			if len(pPayload) < 2+topicLen {
				continue
			}
			msgTopic := string(pPayload[2 : 2+topicLen])
			msg := string(pPayload[2+topicLen:])
			messages = append(messages, fmt.Sprintf("[%s] %s", msgTopic, msg))
		}

		if len(messages) == 0 {
			return fmt.Sprintf("订阅成功\n目标=%s\ntopic=%s\n等待=%ds\n未收到消息", address, topic, waitSeconds), nil
		}
		return fmt.Sprintf("订阅成功\n目标=%s\ntopic=%s\n收到消息:\n%s", address, topic, strings.Join(messages, "\n")), nil
	}

	return "", fmt.Errorf("不支持的命令: %s\n支持的命令: connack, ping, publish <topic> [message], subscribe <topic> [seconds]", command)
}
