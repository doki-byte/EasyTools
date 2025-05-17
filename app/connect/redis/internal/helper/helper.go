package helper

import (
	"EasyTools/app/connect/redis/internal/define"
	"context"
	"encoding/json"
	"errors"
	"github.com/go-redis/redis/v8"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"time"
)

func GetConnection(identity string) (*define.Connection, error) {
	conf := new(define.Config)
	nowPath := GetConfPath()
	data, err := ioutil.ReadFile(nowPath + string(os.PathSeparator) + define.ConfigName)
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

// GetConfPath 获取配置文件路径
func GetConfPath() string {
	current, _ := user.Current()
	return current.HomeDir
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
