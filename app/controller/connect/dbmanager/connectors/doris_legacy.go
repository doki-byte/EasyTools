package connectors

import (
	"EasyTools/app/controller/connect/dbmanager/app/model"
	mysqllegacy "EasyTools/app/controller/connect/dbmanager/mysqllegacydriver"
	"context"
	"database/sql"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
)

// DorisLegacyConnector 用于旧版 Apache Doris（MySQL 协议，如 9030）等无法兼容官方驱动 v1.8+ 默认连接属性的实例。
// 新版 Doris 请直接使用类型「MySQL」。
type DorisLegacyConnector struct {
	DefaultConnector
}

var (
	dorisLegacyDialMutex   sync.Mutex
	dorisLegacyDialCounter int64
)

const dorisLegacyDriverName = "mysql_doris_legacy"

// dorisEscapeSQLString 将值嵌入字面量 SQL（Doris FE 对客户端预处理语句支持极窄，带 ? 会报 Only support prepare SelectStmt...）
func dorisEscapeSQLString(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

func (c DorisLegacyConnector) Ping(config *model.DbConfig) error {
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

func (c DorisLegacyConnector) Connect(config *model.DbConfig) (*sqlx.DB, error) {
	dsn := buildMySQLProtocolDSN(config)

	if hasSSHConfig(config) {
		dialer, err := getSSHDialer(config)
		if err != nil {
			return nil, err
		}

		dorisLegacyDialMutex.Lock()
		dorisLegacyDialCounter++
		dialKey := fmt.Sprintf("ssh_tunnel_doris_legacy_%d", dorisLegacyDialCounter)
		dorisLegacyDialMutex.Unlock()

		mysqllegacy.RegisterDial(dialKey, func(addr string) (net.Conn, error) {
			return dialer("tcp", addr)
		})

		cfg, err := mysqllegacy.ParseDSN(dsn)
		if err != nil {
			return nil, fmt.Errorf("解析DSN失败: %v", err)
		}
		cfg.Net = dialKey

		connector, err := mysqllegacy.NewConnector(cfg)
		if err != nil {
			return nil, fmt.Errorf("创建Doris(旧版)连接器失败: %v", err)
		}

		sqlDB := sql.OpenDB(connector)
		return sqlx.NewDb(sqlDB, dorisLegacyDriverName), nil
	}

	return sqlx.Open(dorisLegacyDriverName, dsn)
}

// ShowTables Doris FE 对 INFORMATION_SCHEMA.TABLES 多列/类型与 MySQL 不完全一致，
// 仅用 TABLE_NAME 查询；且不能使用预处理参数（?），否则 FE 报错 Only support prepare SelectStmt point query。
func (c DorisLegacyConnector) ShowTables(db *sqlx.DB, database string) (*[]Table, error) {
	if database == "" {
		empty := []Table{}
		return &empty, nil
	}
	sqlStr := fmt.Sprintf(
		`SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = '%s' ORDER BY TABLE_NAME`,
		dorisEscapeSQLString(database),
	)
	rows, err := db.Queryx(sqlStr)
	if err != nil {
		return nil, err
	}
	defer func() {
		if rows != nil {
			_ = rows.Close()
		}
	}()

	var tables []Table
	for rows.Next() {
		var name sql.NullString
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		if name.Valid && name.String != "" {
			tables = append(tables, Table{Name: name.String})
		}
	}
	return &tables, rows.Err()
}

// Column 同上：避免 Queryx(..., schema, table) 触发的 PREPARE。
func (c DorisLegacyConnector) Column(db *sqlx.DB, params ...string) (*[]Column, error) {
	schema := params[0]
	table := params[1]
	if schema == "" || table == "" {
		return nil, fmt.Errorf("database and table name cannot be empty")
	}

	sqlStr := fmt.Sprintf(`SELECT 
    			COLUMN_NAME as field,
       			COLUMN_TYPE as type,
       			CHARACTER_MAXIMUM_LENGTH AS 'length',
       			IS_NULLABLE AS 'nullable',
       			COLUMN_KEY AS 'key',
       			COLUMN_COMMENT as comment, 
				COLUMN_DEFAULT AS 'default'
			FROM INFORMATION_SCHEMA.COLUMNS
			WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME = '%s'`,
		dorisEscapeSQLString(schema), dorisEscapeSQLString(table))

	rows, err := db.Queryx(sqlStr)
	if err != nil {
		return nil, err
	}
	defer func() {
		if rows != nil {
			_ = rows.Close()
		}
	}()

	var columns []Column
	for rows.Next() {
		var column Column
		var columnName sql.NullString
		var columnType sql.NullString
		var columnComment sql.NullString
		var length sql.NullString
		var nullable sql.NullString
		var columnKey sql.NullString
		var columnDefault sql.NullString
		err = rows.Scan(&columnName, &columnType, &length, &nullable, &columnKey, &columnComment, &columnDefault)
		if err != nil {
			return nil, err
		}

		if columnName.Valid {
			column.Field = columnName.String
		}
		if columnType.Valid {
			column.Type = columnType.String
		}
		if columnComment.Valid {
			column.Comment = columnComment.String
		}
		if length.Valid {
			column.Length = length.String
		}
		if nullable.Valid {
			column.Nullable = nullable.String == "YES"
		}
		if columnKey.Valid {
			if columnKey.String == "PRI" {
				column.Key = Primary
			} else {
				column.Key = columnKey.String
			}
		}
		if columnDefault.Valid {
			column.Default = columnDefault.String
		}

		columns = append(columns, column)
	}
	return &columns, rows.Err()
}
