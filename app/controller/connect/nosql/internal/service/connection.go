package service

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
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	uuid "github.com/satori/go.uuid"
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

// ConnectionTest 测试连接
func ConnectionTest(conn *define.Connection) error {
	if conn.Addr == "" {
		return errors.New("连接地址不能为空")
	}

	// 如果没有指定数据库类型，默认为 Redis
	if conn.DbType == "" {
		conn.DbType = "redis"
	}

	// 根据数据库类型选择测试方法
	dbType := strings.ToLower(conn.DbType)
	switch dbType {
	case "redis":
		return testRedisConnection(conn)
	case "mongodb", "mongo":
		return testMongoDBConnection(conn)
	default:
		return fmt.Errorf("不支持的数据库类型: %s，支持 Redis 和 MongoDB", conn.DbType)
	}
}

// testRedisConnection 测试 Redis 连接
func testRedisConnection(conn *define.Connection) error {
	if conn.Port == "" {
		conn.Port = "6379"
	}

	// 创建 Redis 客户端配置
	redisOpts := &redis.Options{
		Addr:     fmt.Sprintf("%s:%s", conn.Addr, conn.Port),
		Username: conn.Username, // Redis 6.0+ 支持 ACL，用户名可选
		Password: conn.Password,
		DB:       0, // 默认数据库
	}

	var rdb *redis.Client
	var sshClient *ssh.Client

	// 如果是 SSH 连接，先建立 SSH 隧道
	if conn.Type == "ssh" {
		// 验证SSH配置完整性
		if conn.SSHAddr == "" {
			return errors.New("SSH隧道模式下，SSH地址不能为空")
		}
		if conn.SSHPort == "" {
			conn.SSHPort = "22" // 默认SSH端口
		}
		if conn.SSHUsername == "" {
			return errors.New("SSH隧道模式下，SSH用户名不能为空")
		}
		if conn.SSHPassword == "" {
			return errors.New("SSH隧道模式下，SSH密码不能为空")
		}

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
		sshAddr := fmt.Sprintf("%s:%s", conn.SSHAddr, conn.SSHPort)
		sshClient, err = ssh.Dial("tcp", sshAddr, sshConfig)
		if err != nil {
			// 提供详细的错误信息，包含SSH服务器地址和用户名
			return fmt.Errorf("SSH连接失败 [%s@%s]: %v", conn.SSHUsername, sshAddr, err)
		}
		defer sshClient.Close()

		// 测试通过SSH隧道能否连接到Redis服务器
		redisAddr := fmt.Sprintf("%s:%s", conn.Addr, conn.Port)
		sshConn, err := sshClient.Dial("tcp", redisAddr)
		if err != nil {
			// 提供详细的错误信息，包含Redis服务器地址
			return fmt.Errorf("通过SSH隧道连接Redis失败 [目标: %s]: %v", redisAddr, err)
		}
		defer sshConn.Close()

		// 使用自定义拨号器，通过SSH隧道转发连接
		redisOpts.Dialer = func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := sshClient.Dial(network, addr)
			if err != nil {
				return nil, err
			}
			// 包装SSH连接以忽略deadline操作（SSH通道不支持deadline）
			return &sshConnWrapper{Conn: conn}, nil
		}
		// SSH通道不支持SetReadDeadline和SetWriteDeadline，所以不设置超时
		redisOpts.ReadTimeout = 0
		redisOpts.WriteTimeout = 0
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
		errMsg := err.Error()
		if conn.Type == "ssh" {
			return fmt.Errorf("Redis连接测试失败（SSH隧道模式）: %v", errMsg)
		}
		return fmt.Errorf("Redis连接测试失败: %v", errMsg)
	}

	return nil
}

// testMongoDBConnection 测试 MongoDB 连接
func testMongoDBConnection(conn *define.Connection) error {
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
			return errors.New("SSH隧道模式下，SSH地址不能为空")
		}
		if conn.SSHPort == "" {
			conn.SSHPort = "22"
		}
		if conn.SSHUsername == "" {
			return errors.New("SSH隧道模式下，SSH用户名不能为空")
		}
		if conn.SSHPassword == "" {
			return errors.New("SSH隧道模式下，SSH密码不能为空")
		}

		sshConfig := &ssh.ClientConfig{
			User: conn.SSHUsername,
			Auth: []ssh.AuthMethod{
				ssh.Password(conn.SSHPassword),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         10 * time.Second,
		}

		// 建立 SSH 连接
		sshAddr := fmt.Sprintf("%s:%s", conn.SSHAddr, conn.SSHPort)
		sshClient, err := ssh.Dial("tcp", sshAddr, sshConfig)
		if err != nil {
			return fmt.Errorf("SSH连接失败 [%s@%s]: %v", conn.SSHUsername, sshAddr, err)
		}
		defer sshClient.Close()

		// 设置自定义Dialer，通过SSH隧道连接MongoDB
		clientOptions.SetDialer(&sshContextDialer{sshClient: sshClient})
	}

	// 连接MongoDB（支持直连和SSH隧道两种方式）
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return fmt.Errorf("MongoDB连接失败: %v", err)
	}
	defer client.Disconnect(ctx)

	// 测试连接
	err = client.Ping(ctx, nil)
	if err != nil {
		return fmt.Errorf("MongoDB连接测试失败: %v", err)
	}

	return nil
}

// ConnectionList 连接列表
func ConnectionList() ([]*define.Connection, error) {
	baseDir := system.GetAppBaseDir()
	// 假设baseDir已经定义
	configDir := filepath.Join(baseDir, "config", "nosql")
	ConfigName := filepath.Join(configDir, "nosql-client.conf")
	data, err := os.ReadFile(ConfigName)
	if errors.Is(err, os.ErrNotExist) {
		// 没有历史配置时返回空列表，避免前端误报错误
		return []*define.Connection{}, nil
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

	// 如果没有指定数据库类型，默认为 Redis
	if conn.DbType == "" {
		conn.DbType = "redis"
	}

	// 根据数据库类型设置默认端口
	if conn.Port == "" {
		dbType := strings.ToLower(conn.DbType)
		if dbType == "mongodb" || dbType == "mongo" {
			conn.Port = "27017"
		} else {
			conn.Port = "6379" // Redis默认端口
		}
	}

	conn.Identity = uuid.NewV4().String()
	conf := new(define.Config)
	baseDir := system.GetAppBaseDir()
	configDir := filepath.Join(baseDir, "config", "nosql")
	ConfigName := filepath.Join(configDir, "nosql-client.conf")
	data, err := os.ReadFile(ConfigName)
	if errors.Is(err, os.ErrNotExist) {
		// 配置文件的内容初始化
		conf.Connections = []*define.Connection{conn}
		data, _ = json.Marshal(conf)
		// 写入配置内容
		os.MkdirAll(configDir, 0755)
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

	// 如果没有指定数据库类型，默认为 Redis
	if conn.DbType == "" {
		conn.DbType = "redis"
	}

	// 根据数据库类型设置默认端口
	if conn.Port == "" {
		dbType := strings.ToLower(conn.DbType)
		if dbType == "mongodb" || dbType == "mongo" {
			conn.Port = "27017"
		} else {
			conn.Port = "6379" // Redis默认端口
		}
	}

	conf := new(define.Config)
	baseDir := system.GetAppBaseDir()
	// 假设baseDir已经定义
	configDir := filepath.Join(baseDir, "config", "nosql")
	ConfigName := filepath.Join(configDir, "nosql-client.conf")
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
	baseDir := system.GetAppBaseDir()
	// 假设baseDir已经定义
	configDir := filepath.Join(baseDir, "config", "nosql")
	ConfigName := filepath.Join(configDir, "nosql-client.conf")
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
