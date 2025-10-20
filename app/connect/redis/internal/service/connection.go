package service

import (
	"EasyTools/app/connect/redis/internal/define"
	"EasyTools/app/connect/redis/internal/helper"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/go-redis/redis/v8"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/crypto/ssh"
)

// ConnectionTest 测试连接
func ConnectionTest(conn *define.Connection) error {
	if conn.Addr == "" {
		return errors.New("连接地址不能为空")
	}
	if conn.Port == "" {
		conn.Port = "6379"
	}

	// 创建 Redis 客户端配置
	redisOpts := &redis.Options{
		Addr:     fmt.Sprintf("%s:%s", conn.Addr, conn.Port),
		Password: conn.Password,
		DB:       0, // 默认数据库
	}

	var rdb *redis.Client
	var sshClient *ssh.Client

	// 如果是 SSH 连接，先建立 SSH 隧道
	if conn.Type == "ssh" && conn.SSHAddr != "" {
		sshConfig := &ssh.ClientConfig{
			User: conn.SSHUsername,
			Auth: []ssh.AuthMethod{
				ssh.Password(conn.SSHPassword),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         10 * time.Second,
		}

		// 建立 SSH 连接
		var err error
		sshClient, err = ssh.Dial("tcp", fmt.Sprintf("%s:%s", conn.SSHAddr, conn.SSHPort), sshConfig)
		if err != nil {
			return fmt.Errorf("SSH连接失败: %v", err)
		}
		defer sshClient.Close()

		// 通过 SSH 隧道建立 Redis 连接
		sshConn, err := sshClient.Dial("tcp", fmt.Sprintf("%s:%s", conn.Addr, conn.Port))
		if err != nil {
			return fmt.Errorf("通过SSH连接Redis失败: %v", err)
		}
		defer sshConn.Close()

		// 使用自定义拨号器
		redisOpts.Dialer = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return sshClient.Dial(network, addr)
		}
		// 重置地址为本地，因为实际连接是通过 SSH 隧道
		redisOpts.Addr = "127.0.0.1:6379"
	}

	// 创建 Redis 客户端
	rdb = redis.NewClient(redisOpts)
	defer rdb.Close()

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 执行 PING 命令测试连接
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("Redis连接测试失败: %v", err)
	}

	return nil
}

// ConnectionList 连接列表
func ConnectionList() ([]*define.Connection, error) {
	baseDir := helper.GetAppBaseDir()
	// 假设baseDir已经定义
	configDir := filepath.Join(baseDir, "tools", "redis")
	ConfigName := filepath.Join(configDir, "redis-client.conf")
	data, err := os.ReadFile(ConfigName)
	if errors.Is(err, os.ErrNotExist) {
		return nil, errors.New("暂无连接数据")
	}
	conf := new(define.Config)
	err = json.Unmarshal(data, conf)
	if err != nil {
		return nil, err
	}
	return conf.Connections, nil
}

// ConnectionCreate 创建连接
func ConnectionCreate(conn *define.Connection) error {
	if conn.Addr == "" {
		return errors.New("连接地址不能为空")
	}
	// 参数默认值处理
	if conn.Name == "" {
		conn.Name = conn.Addr
	}
	if conn.Port == "" {
		conn.Port = "6379"
	}
	conn.Identity = uuid.NewV4().String()
	conf := new(define.Config)
	baseDir := helper.GetAppBaseDir()
	configDir := filepath.Join(baseDir, "tools", "redis")
	ConfigName := filepath.Join(configDir, "redis-client.conf")
	data, err := os.ReadFile(ConfigName)
	if errors.Is(err, os.ErrNotExist) {
		// 配置文件的内容初始化
		conf.Connections = []*define.Connection{conn}
		data, _ = json.Marshal(conf)
		// 写入配置内容
		os.MkdirAll(configDir, 755)
		os.WriteFile(ConfigName, data, 0666)
		return nil
	}
	json.Unmarshal(data, conf)
	conf.Connections = append(conf.Connections, conn)
	data, _ = json.Marshal(conf)
	os.WriteFile(ConfigName, data, 0666)
	return nil
}

// ConnectionEdit 编辑连接
func ConnectionEdit(conn *define.Connection) error {
	if conn.Identity == "" {
		return errors.New("连接唯一标识不能为空")
	}
	if conn.Addr == "" {
		return errors.New("连接地址不能为空")
	}
	// 参数默认值处理
	if conn.Name == "" {
		conn.Name = conn.Addr
	}
	if conn.Port == "" {
		conn.Port = "6379"
	}
	conf := new(define.Config)
	baseDir := helper.GetAppBaseDir()
	// 假设baseDir已经定义
	configDir := filepath.Join(baseDir, "tools", "redis")
	ConfigName := filepath.Join(configDir, "redis-client.conf")
	data, err := os.ReadFile(ConfigName)
	if err != nil {
		return err
	}
	json.Unmarshal(data, conf)
	for i, v := range conf.Connections {
		if v.Identity == conn.Identity {
			conf.Connections[i] = conn
		}
	}
	data, _ = json.Marshal(conf)
	os.WriteFile(ConfigName, data, 0666)
	return nil
}

// ConnectionDelete 删除连接
func ConnectionDelete(identity string) error {
	if identity == "" {
		return errors.New("连接唯一标识不能为空")
	}
	conf := new(define.Config)
	baseDir := helper.GetAppBaseDir()
	// 假设baseDir已经定义
	configDir := filepath.Join(baseDir, "tools", "redis")
	ConfigName := filepath.Join(configDir, "redis-client.conf")
	data, err := os.ReadFile(ConfigName)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, conf)
	if err != nil {
		return err
	}
	for i, v := range conf.Connections {
		if v.Identity == identity {
			conf.Connections = append(conf.Connections[:i], conf.Connections[i+1:]...)
			break
		}
	}
	data, _ = json.Marshal(conf)
	os.WriteFile(ConfigName, data, 0666)
	return nil
}
