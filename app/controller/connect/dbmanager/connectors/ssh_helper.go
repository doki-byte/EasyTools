package connectors

import (
	"EasyTools/app/controller/connect/dbmanager/app/model"
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

// getSSHClient 创建SSH客户端连接
func getSSHClient(config *model.DbConfig) (*ssh.Client, error) {
	if config.SSHAddr == "" || config.SSHPort == "" || config.SSHUsername == "" {
		return nil, fmt.Errorf("SSH配置不完整")
	}

	sshConfig := &ssh.ClientConfig{
		User: config.SSHUsername,
		Auth: []ssh.AuthMethod{
			ssh.Password(config.SSHPassword),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}

	addr := fmt.Sprintf("%s:%s", config.SSHAddr, config.SSHPort)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("SSH连接失败: %v", err)
	}

	return client, nil
}

// getSSHDialer 获取通过SSH隧道连接的Dialer函数
func getSSHDialer(config *model.DbConfig) (func(network, addr string) (net.Conn, error), error) {
	sshClient, err := getSSHClient(config)
	if err != nil {
		return nil, err
	}

	// 返回一个Dialer函数，该函数通过SSH隧道连接到目标地址
	return func(network, addr string) (net.Conn, error) {
		// 通过SSH隧道连接到目标数据库地址
		conn, err := sshClient.Dial(network, addr)
		if err != nil {
			return nil, fmt.Errorf("通过SSH隧道连接失败: %v", err)
		}
		return conn, nil
	}, nil
}

// hasSSHConfig 检查是否应按 SSH 隧道连接。
// 必须与 connection_type 一致：仅当显式选择 ssh 且 SSH 三要素齐全时才走隧道；
// 否则即使用户曾保存过 SSH 字段（历史数据或改直连后未清库），也应直连，避免与 Navicat 等行为不一致。
func hasSSHConfig(config *model.DbConfig) bool {
	if config.ConnectionType != "ssh" {
		return false
	}
	return config.SSHAddr != "" && config.SSHPort != "" && config.SSHUsername != ""
}

