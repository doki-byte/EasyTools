package connectors

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"EasyTools/app/controller/connect/dbmanager/app/model"
	"fmt"
	"net"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

type PostgreSQLConnector struct {
	DefaultConnector
}

func (c PostgreSQLConnector) Ping(config *model.DbConfig) error {
	db, err := c.Connect(config)
	if err != nil {
		return err
	}
	defer db.Close()
	err = db.Ping()
	return err
}

func (c PostgreSQLConnector) Connect(config *model.DbConfig) (*sqlx.DB, error) {
	dsn := c.dsn(config)
	
	// 如果配置了SSH隧道，使用自定义Dialer
	if hasSSHConfig(config) {
		dialer, err := getSSHDialer(config)
		if err != nil {
			return nil, err
		}
		
		// 使用pq.DialOpen支持自定义Dialer
		pgDialer := &pgSSHDialer{dialer: dialer}
		conn, err := pq.DialOpen(pgDialer, dsn)
		if err != nil {
			return nil, fmt.Errorf("通过SSH隧道连接PostgreSQL失败: %v", err)
		}
		
		// 使用sql.OpenDB包装连接
		connector := &postgresSSHConnector{
			conn: conn,
		}
		
		sqlDB := sql.OpenDB(connector)
		return sqlx.NewDb(sqlDB, "postgres"), nil
	}
	
	return sqlx.Open("postgres", dsn)
}

// pgSSHDialer 实现pq.Dialer接口以支持SSH隧道
type pgSSHDialer struct {
	dialer func(network, addr string) (net.Conn, error)
}

func (d *pgSSHDialer) Dial(network, address string) (net.Conn, error) {
	return d.dialer(network, address)
}

func (d *pgSSHDialer) DialTimeout(network, address string, timeout time.Duration) (net.Conn, error) {
	// 使用context实现超时
	conn, err := d.dialer(network, address)
	if err != nil {
		return nil, err
	}
	// 设置连接超时
	conn.SetDeadline(time.Now().Add(timeout))
	return conn, nil
}

// postgresSSHConnector 包装PostgreSQL连接以支持SSH隧道
type postgresSSHConnector struct {
	conn driver.Conn
}

func (c *postgresSSHConnector) Connect(ctx context.Context) (driver.Conn, error) {
	return c.conn, nil
}

func (c *postgresSSHConnector) Driver() driver.Driver {
	return &pq.Driver{}
}

func (c PostgreSQLConnector) Use(db *sqlx.DB, database string) error {
	// PostgreSQL 需要在连接时指定数据库，不能使用 USE 命令
	// 由于连接已经建立，我们无法切换数据库，只能通过重新连接来实现
	// 这里不做任何操作，数据库切换需要在连接时通过 dbname 参数完成
	// 注意：PostgreSQL 的连接总是连接到配置中指定的数据库（默认是 postgres）
	// 要查询其他数据库的表，需要在连接时指定该数据库
	return nil
}

func (c PostgreSQLConnector) ShowDatabases(db *sqlx.DB) ([]string, error) {
	var result []string
	sqlStr := "SELECT datname FROM pg_database WHERE datistemplate = false"
	rows, err := db.Queryx(sqlStr)
	defer func() {
		_ = rows.Close()
		_ = db.Close()
	}()
	for rows.Next() {
		var row string
		err = rows.Scan(&row)
		if err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, err
}

func (c PostgreSQLConnector) ShowTables(db *sqlx.DB, database string) (*[]Table, error) {
	var tables []Table
	// 如果数据库名为空，返回空结果
	if database == "" {
		return &tables, nil
	}
	// 需要先切换到目标数据库
	// 注意：PostgreSQL 的 database 参数实际上是数据库名，表在 public schema 中
	sqlStr := "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_catalog = $1"
	rows, err := db.Queryx(sqlStr, database)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var table Table
		var tableName sql.NullString
		var err = rows.Scan(&tableName)
		if err != nil {
			return nil, err
		}

		if tableName.Valid {
			table.Name = tableName.String
		}

		tables = append(tables, table)
	}
	return &tables, err
}

func (c PostgreSQLConnector) Column(db *sqlx.DB, params ...string) (*[]Column, error) {
	var columns []Column
	database := params[0]
	table := params[1]
	
	// 如果数据库名或表名为空，返回错误
	if database == "" || table == "" {
		return nil, fmt.Errorf("database and table name cannot be empty")
	}
	
	sqlStr := `SELECT 
		column_name as field,
		data_type as type,
		character_maximum_length as length,
		is_nullable as nullable,
		column_default as default
	FROM information_schema.columns
	WHERE table_catalog = $1 AND table_schema = 'public' AND table_name = $2
	ORDER BY ordinal_position`

	rows, err := db.Queryx(sqlStr, database, table)
	if err != nil {
		return nil, err
	}
	defer func() {
		if rows != nil {
			_ = rows.Close()
		}
	}()

	for rows.Next() {
		var column Column
		var columnName string
		var columnType string
		var length interface{}
		var nullable string
		var columnDefault interface{}
		err = rows.Scan(&columnName, &columnType, &length, &nullable, &columnDefault)
		if err != nil {
			return nil, err
		}

		column.Field = columnName
		column.Type = columnType
		if length != nil {
			column.Length = fmt.Sprintf("%v", length)
		}
		column.Nullable = (nullable == "YES")
		if columnDefault != nil {
			column.Default = fmt.Sprintf("%v", columnDefault)
		}

		// 检查主键
		pkSql := `SELECT COUNT(*) 
			FROM information_schema.table_constraints tc
			JOIN information_schema.key_column_usage kcu 
				ON tc.constraint_name = kcu.constraint_name
				AND tc.table_schema = kcu.table_schema
			WHERE tc.table_catalog = $1 
				AND tc.table_schema = 'public'
				AND tc.table_name = $2 
				AND kcu.column_name = $3
				AND tc.constraint_type = 'PRIMARY KEY'`
		var pkCount int
		_ = db.Get(&pkCount, pkSql, database, table, columnName)
		if pkCount > 0 {
			column.Key = Primary
		}

		columns = append(columns, column)
	}
	return &columns, err
}

func (c PostgreSQLConnector) dsn(database *model.DbConfig) string {
	// 默认连接到 postgres 数据库，确保可以正常执行 Ping 和元数据查询
	return fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=postgres sslmode=disable", database.Username, database.Password, database.Host, database.Port)
}

// DsnWithDb 生成包含指定数据库名的 DSN
func (c PostgreSQLConnector) DsnWithDb(database *model.DbConfig, dbName string) string {
	if dbName == "" {
		dbName = "postgres"
	}
	return fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s sslmode=disable", database.Username, database.Password, database.Host, database.Port, dbName)
}

