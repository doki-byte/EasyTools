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
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type MariaDBConnector struct {
	DefaultConnector
}

var (
	mariadbDialMutex sync.Mutex
	mariadbDialCounter int64
)

func (c MariaDBConnector) Ping(config *model.DbConfig) error {
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

func (c MariaDBConnector) Connect(config *model.DbConfig) (*sqlx.DB, error) {
	dsn := c.dsn(config)
	
	// 如果配置了SSH隧道，使用自定义Dialer
	if hasSSHConfig(config) {
		dialer, err := getSSHDialer(config)
		if err != nil {
			return nil, err
		}
		
		// MariaDB使用MySQL协议，使用相同的方法
		mariadbDialMutex.Lock()
		mariadbDialCounter++
		dialKey := fmt.Sprintf("ssh_tunnel_mariadb_%d", mariadbDialCounter)
		mariadbDialMutex.Unlock()
		
		// 注册自定义dial函数
		mysql.RegisterDial(dialKey, func(addr string) (net.Conn, error) {
			return dialer("tcp", addr)
		})
		
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
			return nil, fmt.Errorf("创建MariaDB连接器失败: %v", err)
		}
		
		sqlDB := sql.OpenDB(connector)
		return sqlx.NewDb(sqlDB, "mysql"), nil
	}
	
	return sqlx.Open("mysql", dsn)
}

func (c MariaDBConnector) dsn(config *model.DbConfig) string {
	return buildMySQLProtocolDSN(config)
}

