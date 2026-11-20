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

type GBaseConnector struct {
	DefaultConnector
}

var (
	gbaseDialMutex sync.Mutex
	gbaseDialCounter int64
)

func (c GBaseConnector) Ping(config *model.DbConfig) error {
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

func (c GBaseConnector) Connect(config *model.DbConfig) (*sqlx.DB, error) {
	dsn := c.dsn(config)
	
	// 如果配置了SSH隧道，使用自定义Dialer
	if hasSSHConfig(config) {
		dialer, err := getSSHDialer(config)
		if err != nil {
			return nil, err
		}
		
		// GBase使用MySQL协议，使用相同的方法
		gbaseDialMutex.Lock()
		gbaseDialCounter++
		dialKey := fmt.Sprintf("ssh_tunnel_gbase_%d", gbaseDialCounter)
		gbaseDialMutex.Unlock()
		
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
			return nil, fmt.Errorf("创建GBase连接器失败: %v", err)
		}
		
		sqlDB := sql.OpenDB(connector)
		return sqlx.NewDb(sqlDB, "mysql"), nil
	}
	
	return sqlx.Open("mysql", dsn)
}

func (c GBaseConnector) dsn(config *model.DbConfig) string {
	return buildMySQLProtocolDSN(config)
}

