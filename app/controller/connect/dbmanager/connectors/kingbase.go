package connectors

import (
	"EasyTools/app/controller/connect/dbmanager/app/model"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

type KingbaseConnector struct {
	DefaultConnector
}

func (c KingbaseConnector) Ping(config *model.DbConfig) error {
	db, err := c.Connect(config)
	if err != nil {
		return err
	}
	defer db.Close()
	err = db.Ping()
	return err
}

func (c KingbaseConnector) Connect(config *model.DbConfig) (*sqlx.DB, error) {
	dsn := c.dsn(config)

	// 如果配置了SSH隧道，使用自定义Dialer
	if hasSSHConfig(config) {
		dialer, err := getSSHDialer(config)
		if err != nil {
			return nil, err
		}

		// Kingbase使用PostgreSQL协议，使用相同的方法
		pgDialer := &pgSSHDialer{dialer: dialer}
		conn, err := pq.DialOpen(pgDialer, dsn)
		if err != nil {
			return nil, fmt.Errorf("通过SSH隧道连接Kingbase失败: %v", err)
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

func (c KingbaseConnector) Use(db *sqlx.DB, database string) error {
	// 人大金仓兼容 PostgreSQL，不需要 USE 命令
	return nil
}

func (c KingbaseConnector) ShowDatabases(db *sqlx.DB) ([]string, error) {
	var result []string
	sqlStr := "SELECT datname FROM pg_database WHERE datistemplate = false"
	rows, err := db.Queryx(sqlStr)
	if err != nil {
		return nil, err
	}
	defer func() {
		if rows != nil {
			_ = rows.Close()
		}
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

func (c KingbaseConnector) ShowTables(db *sqlx.DB, database string) (*[]Table, error) {
	var tables []Table
	// 如果数据库名为空，返回空结果
	if database == "" {
		return &tables, nil
	}
	// 人大金仓兼容 PostgreSQL，使用 table_catalog 过滤数据库
	sqlStr := "SELECT table_name FROM information_schema.tables WHERE table_schema = 'public' AND table_catalog = $1"
	rows, err := db.Queryx(sqlStr, database)
	if err != nil {
		return nil, err
	}
	defer func() {
		if rows != nil {
			_ = rows.Close()
		}
	}()

	for rows.Next() {
		var table Table
		var tableName sql.NullString
		err = rows.Scan(&tableName)
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

func (c KingbaseConnector) Column(db *sqlx.DB, params ...string) (*[]Column, error) {
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
			WHERE tc.table_catalog = $1 
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

func (c KingbaseConnector) dsn(config *model.DbConfig) string {
	// 人大金仓兼容 PostgreSQL 连接字符串格式
	return fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=postgres sslmode=disable",
		config.Username, config.Password, config.Host, config.Port)
}
