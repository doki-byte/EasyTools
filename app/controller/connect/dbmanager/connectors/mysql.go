package connectors

import (
	"EasyTools/app/controller/connect/dbmanager/app/model"
	"context"
	"database/sql"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type MysqlConnector struct {
	DefaultConnector
}

var (
	mysqlDialMutex sync.Mutex
	mysqlDialCounter int64
)

func (c MysqlConnector) Ping(config *model.DbConfig) error {
	db, err := c.Connect(config)
	if err != nil {
		return err
	}
	defer db.Close()
	pingTimeout := 15 * time.Second
	if config != nil && config.Timeout > 0 {
		pingTimeout = time.Duration(config.Timeout) * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
	defer cancel()
	return db.PingContext(ctx)
}

func (c MysqlConnector) Connect(config *model.DbConfig) (*sqlx.DB, error) {
	dsn := c.dsn(config)
	
	// 如果配置了SSH隧道，使用自定义Dialer
	if hasSSHConfig(config) {
		dialer, err := getSSHDialer(config)
		if err != nil {
			return nil, err
		}
		
		// MySQL驱动支持通过RegisterDial注册自定义dial函数
		// 为了避免冲突，我们使用一个唯一的dial名称
		mysqlDialMutex.Lock()
		mysqlDialCounter++
		dialKey := fmt.Sprintf("ssh_tunnel_%d", mysqlDialCounter)
		mysqlDialMutex.Unlock()
		
		// 注册自定义dial函数
		mysql.RegisterDial(dialKey, func(addr string) (net.Conn, error) {
			return dialer("tcp", addr)
		})
		
		// 修改DSN使用自定义dial
		// 解析DSN
		cfg, err := mysql.ParseDSN(dsn)
		if err != nil {
			return nil, fmt.Errorf("解析DSN失败: %v", err)
		}
		
		// 设置自定义网络类型
		cfg.Net = dialKey
		
		// 使用新配置创建连接
		connector, err := mysql.NewConnector(cfg)
		if err != nil {
			return nil, fmt.Errorf("创建MySQL连接器失败: %v", err)
		}
		
		sqlDB := sql.OpenDB(connector)
		return sqlx.NewDb(sqlDB, "mysql"), nil
	}
	
	return sqlx.Open("mysql", dsn)
}

func (c DefaultConnector) Use(db *sqlx.DB, database string) error {
	// 如果数据库名为空，直接返回，不执行USE语句
	if database == "" {
		return nil
	}
	// 使用反引号包裹数据库名，避免数据库名为纯数字或包含特殊字符时出现语法错误
	sqlStr := fmt.Sprintf("USE `%s`", database)
	_, err := db.Exec(sqlStr)
	return err
}

func (c MysqlConnector) dsn(database *model.DbConfig) string {
	return buildMySQLProtocolDSN(database)
}

// buildMySQLProtocolDSN 使用 go-sql-driver 的 Config 生成 DSN，正确转义密码/用户名中的 @、:、/ 等字符。
// 不显式设置 TLS（与 Navicat「默认」、Apache Doris FE 9030 常见部署一致）。
// 注意：官方 github.com/go-sql-driver/mysql v1.8+ 握手会附带连接属性，部分旧版 Doris 会异常断开；
// 此类连接请使用数据库类型「Doris (旧版兼容)」（doris_legacy），其使用内嵌的 v1.7.1 协议栈。
func buildMySQLProtocolDSN(cfg *model.DbConfig) string {
	if cfg == nil {
		return ""
	}
	c := mysql.NewConfig()
	c.User = cfg.Username
	c.Passwd = cfg.Password
	c.Net = "tcp"
	c.Addr = net.JoinHostPort(cfg.Host, cfg.Port)
	c.DBName = ""
	if cfg.Timeout > 0 {
		c.Timeout = time.Duration(cfg.Timeout) * time.Second
	} else {
		c.Timeout = 10 * time.Second
	}
	if cfg.ReadTimeout > 0 {
		c.ReadTimeout = time.Duration(cfg.ReadTimeout) * time.Second
	}
	if cfg.WriteTimeout > 0 {
		c.WriteTimeout = time.Duration(cfg.WriteTimeout) * time.Second
	}
	if cfg.Charset != "" {
		if c.Params == nil {
			c.Params = make(map[string]string)
		}
		c.Params["charset"] = cfg.Charset
	}
	return c.FormatDSN()
}

