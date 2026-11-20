package helper

import (
	"EasyTools/app/controller/connect/nosql/internal/define"
	"EasyTools/app/controller/system"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/go-redis/redis/v8"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/ssh"
)

// sshConnWrapper 包装SSH连接，忽略deadline相关操作
// SSH通道不支持SetReadDeadline和SetWriteDeadline，需要包装以兼容Redis客户端
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

// sshContextDialer 实现MongoDB驱动的ContextDialer接口，通过SSH隧道连接
type sshContextDialer struct {
	sshClient *ssh.Client
}

// DialContext 实现ContextDialer接口的DialContext方法
func (d *sshContextDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	conn, err := d.sshClient.Dial(network, addr)
	if err != nil {
		return nil, fmt.Errorf("通过SSH隧道连接MongoDB失败 [目标: %s]: %v", addr, err)
	}
	// 包装SSH连接以忽略deadline操作（SSH通道不支持deadline）
	return &sshConnWrapper{Conn: conn}, nil
}

func GetConnection(identity string) (*define.Connection, error) {
	conf := new(define.Config)
	baseDir := system.GetAppBaseDir()
	// 假设baseDir已经定义
	configDir := filepath.Join(baseDir, "config", "nosql")
	ConfigName := filepath.Join(configDir, "nosql-client.conf")

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
	
	// 优先使用 SSH 隧道
	if conn.Type == "ssh" {
		// 创建SSH客户端（复用同一个客户端）
		sshAddr := net.JoinHostPort(conn.SSHAddr, conn.SSHPort)
		sshClient, err := getSSHClient(conn.SSHUsername, conn.SSHPassword, sshAddr)
		if err != nil {
			return nil, fmt.Errorf("SSH连接失败 [%s@%s]: %v", conn.SSHUsername, sshAddr, err)
		}
		
		// 使用自定义拨号器，通过SSH隧道转发连接
		redisOption.Dialer = func(ctx context.Context, network, addr string) (net.Conn, error) {
			// 通过SSH客户端创建连接
			conn, err := sshClient.Dial(network, addr)
			if err != nil {
				return nil, fmt.Errorf("通过SSH隧道连接Redis失败 [目标: %s]: %v", addr, err)
			}
			// 包装SSH连接以忽略deadline操作（SSH通道不支持deadline）
			return &sshConnWrapper{Conn: conn}, nil
		}
		// SSH通道不支持SetReadDeadline和SetWriteDeadline，所以不设置超时
		// 设置为0表示不超时，避免Redis客户端尝试调用deadline相关方法
		redisOption.ReadTimeout = 0
		redisOption.WriteTimeout = 0
	}
	// 简连助手不使用代理，直接连接
	
	rdb := redis.NewClient(redisOption)
	return rdb, err
}

// GetMongoDBClient 获取 MongoDB 客户端对象
//
// connectionIdentity 连接唯一标识
// database 选中的数据库名称
func GetMongoDBClient(connectionIdentity string, database string) (*mongo.Client, *mongo.Database, error) {
	conn, err := GetConnection(connectionIdentity)
	if err != nil {
		return nil, nil, err
	}

	if conn.Port == "" {
		conn.Port = "27017"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 构建 MongoDB 连接URI
	var uri string
	if conn.Username != "" && conn.Password != "" {
		uri = fmt.Sprintf("mongodb://%s:%s@%s:%s/?connectTimeoutMS=10000&serverSelectionTimeoutMS=10000",
			conn.Username, conn.Password, conn.Addr, conn.Port)
	} else {
		uri = fmt.Sprintf("mongodb://%s:%s/?connectTimeoutMS=10000&serverSelectionTimeoutMS=10000",
			conn.Addr, conn.Port)
	}

	// 创建 MongoDB 客户端选项
	clientOptions := options.Client().ApplyURI(uri)

	// 如果是 SSH 连接，需要通过SSH隧道
	if conn.Type == "ssh" {
		// 验证SSH配置完整性
		if conn.SSHAddr == "" {
			return nil, nil, errors.New("SSH隧道模式下，SSH地址不能为空")
		}
		if conn.SSHPort == "" {
			conn.SSHPort = "22"
		}
		if conn.SSHUsername == "" {
			return nil, nil, errors.New("SSH隧道模式下，SSH用户名不能为空")
		}
		if conn.SSHPassword == "" {
			return nil, nil, errors.New("SSH隧道模式下，SSH密码不能为空")
		}

		// 创建SSH客户端（复用同一个客户端）
		sshAddr := net.JoinHostPort(conn.SSHAddr, conn.SSHPort)
		sshClient, err := getSSHClient(conn.SSHUsername, conn.SSHPassword, sshAddr)
		if err != nil {
			return nil, nil, fmt.Errorf("SSH连接失败 [%s@%s]: %v", conn.SSHUsername, sshAddr, err)
		}

		// 设置自定义Dialer，通过SSH隧道连接MongoDB
		clientOptions.SetDialer(&sshContextDialer{sshClient: sshClient})
	}

	// 连接MongoDB（支持直连和SSH隧道两种方式）
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, nil, fmt.Errorf("MongoDB连接失败: %v", err)
	}

	// 测试连接
	err = client.Ping(ctx, nil)
	if err != nil {
		client.Disconnect(ctx)
		return nil, nil, fmt.Errorf("MongoDB连接测试失败: %v", err)
	}

	// 如果没有指定数据库，使用admin作为默认数据库
	if database == "" {
		database = "admin"
	}

	db := client.Database(database)
	return client, db, nil
}
