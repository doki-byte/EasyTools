package helper

import (
	"EasyTools/app/controller/connect/redis/internal/define"
	"context"
	"encoding/json"
	"errors"
	"github.com/go-redis/redis/v8"
	"golang.org/x/crypto/ssh"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// 获取应用基础目录
func GetAppBaseDir() string {
	// 如果是 macOS，使用应用支持目录
	if runtime.GOOS == "darwin" {
		appName := "EasyTools"
		homeDir, err := os.UserHomeDir()
		if err != nil {
			panic("获取用户主目录失败: " + err.Error())
		}
		return filepath.Join(homeDir, "Library", "Application Support", appName)
	}

	// 其他系统使用当前目录下的 EasyToolsFiles
	currentPath, err := os.Getwd()
	if err != nil {
		panic("获取当前路径失败: " + err.Error())
	}
	return filepath.Join(currentPath, "EasyToolsFiles")
}

func GetConnection(identity string) (*define.Connection, error) {
	conf := new(define.Config)
	baseDir := GetAppBaseDir()
	// 假设baseDir已经定义
	configDir := filepath.Join(baseDir, "tools", "redis")
	ConfigName := filepath.Join(configDir, "redis-client.conf")

	data, err := os.ReadFile(ConfigName)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, conf)
	if err != nil {
		return nil, err
	}
	for _, v := range conf.Connections {
		if v.Identity == identity {
			return v, nil
		}
	}
	return nil, errors.New("连接数据不存在")
}

func getSSHClient(username, password, addr string) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}
	return ssh.Dial("tcp", addr, config)
}

func getRedisConn(username, password, addr, redisAddr string) (net.Conn, error) {
	client, err := getSSHClient(username, password, addr)
	if err != nil {
		return nil, err
	}
	return client.Dial("tcp", redisAddr)
}

// GetRedisClient 获取 Redis 客户端对象
//
// connectionIdentity 连接唯一标识
// db 选中的数据库
func GetRedisClient(connectionIdentity string, db int) (*redis.Client, error) {
	conn, err := GetConnection(connectionIdentity)
	if err != nil {
		return nil, err
	}
	redisOption := &redis.Options{
		Addr:         net.JoinHostPort(conn.Addr, conn.Port),
		Username:     conn.Username,
		Password:     conn.Password,
		DB:           db,
		ReadTimeout:  -1,
		WriteTimeout: -1,
	}
	if conn.Type == "ssh" {
		redisOption.Dialer = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return getRedisConn(conn.SSHUsername, conn.SSHPassword, conn.SSHAddr+":"+conn.SSHPort, addr)
		}
	}
	rdb := redis.NewClient(redisOption)
	return rdb, err
}
