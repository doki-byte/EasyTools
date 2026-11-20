package connectors

import (
	"database/sql"
	"EasyTools/app/controller/connect/dbmanager/app/model"

	"github.com/jmoiron/sqlx"
	go_ora "github.com/sijms/go-ora/v2"
	"log/slog"
	"strconv"
)

// OracleConnector 提供 Oracle 数据库的基础能力支持
// 说明：目前使用 go-ora 驱动，通过主机/端口/用户名/密码进行连接。
// 数据库服务名目前写死为 DefaultOracleServiceName，如需适配实际环境，请修改该常量。
const DefaultOracleServiceName = "XE"

type OracleConnector struct {
}

// Ping 测试 Oracle 连接是否可用
func (c OracleConnector) Ping(config *model.DbConfig) error {
	dsn := c.dsn(config)
	db, err := sqlx.Open("oracle", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	return db.Ping()
}

// Connect 建立到 Oracle 的连接，供后续查询/元数据操作使用
func (c OracleConnector) Connect(config *model.DbConfig) (*sqlx.DB, error) {
	dsn := c.dsn(config)
	return sqlx.Open("oracle", dsn)
}

// 简连助手不使用代理，直接连接

func (c OracleConnector) ShowDatabases(db *sqlx.DB) ([]string, error) {
	// Oracle 一般通过 schema 管理对象，不像 MySQL 那样有多库，这里直接返回空列表
	return nil, nil
}

func (c OracleConnector) Use(db *sqlx.DB, database string) error {
	// Oracle 不支持 USE 语法，连接时已经决定了服务，直接返回即可
	return nil
}

func (c OracleConnector) ShowTables(db *sqlx.DB, database string) (*[]Table, error) {
	var tables []Table
	sqlStr := "SELECT table_name FROM user_tables"

	rows, err := db.Queryx(sqlStr)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = rows.Close()
		_ = db.Close()
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

func (c OracleConnector) Column(db *sqlx.DB, params ...string) (*[]Column, error) {
	// 暂未实现列信息的查询，返回空即可，前端会优雅降级
	return nil, nil
}

func (c OracleConnector) Query(db *sqlx.DB, sqlStr string) (*Query, error) {
	rows, err := db.Queryx(sqlStr)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
		_ = db.Close()
	}()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	columnTypes, err := rows.ColumnTypes()

	var result []map[string]interface{}
	values := make([][]byte, len(columns))
	scans := make([]interface{}, len(columns))

	for i := range columns {
		scans[i] = &values[i]
	}

	for rows.Next() {
		err = rows.Scan(scans...)
		if err != nil {
			return nil, err
		}
		row := make(map[string]interface{})
		for i, value := range values {
			columnType := columnTypes[i]
			typeName := columnType.DatabaseTypeName()

			switch typeName {
			case "TINYINT", "SMALLINT", "INT", "INTEGER", "BIGINT", "UNSIGNED BIGINT", "UNSIGNED TINYINT":
				val, err := strconv.Atoi(string(value))
				if err != nil {
					return nil, err
				}
				row[columnType.Name()] = val
			case "NCHAR", "TimeStampDTY", "CHAR":
				row[columnType.Name()] = string(value)
			default:
				slog.Error("dont support type", "type name", typeName)
			}
		}
		result = append(result, row)
	}

	query := &Query{
		Rows:    result,
		Columns: nil,
		Total:   len(result),
	}

	return query, err
}

func (c OracleConnector) Count(db *sqlx.DB, sqlStr string) (int, error) {
	rows, err := db.Queryx(sqlStr)
	if err != nil {
		return 0, err
	}
	defer func() {
		_ = rows.Close()
		_ = db.Close()
	}()
	for rows.Next() {
		var total int
		err = rows.Scan(&total)
		if err != nil {
			return total, nil
		}
	}
	return 0, err
}

func (c OracleConnector) Ddl(db *sqlx.DB, table string) (string, error) {
	sqlStr := "SELECT DBMS_METADATA.GET_DDL('TABLE', 'SYS_USER') AS TABLE_DDL FROM DUAL"
	row := db.QueryRowx(sqlStr, table)
	var result string
	err := row.Scan(&result)
	return result, err
}

func (c OracleConnector) Execute(db *sqlx.DB, sqlStr string) (int, error) {
	result, err := db.Exec(sqlStr)
	if err != nil {
		return 0, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(rows), err
}

// dsn 构造 Oracle 连接串
// 目前使用 Easy Connect 形式：host:port/serviceName，serviceName 由 DefaultOracleServiceName 控制
func (c OracleConnector) dsn(database *model.DbConfig) string {
	port, _ := strconv.Atoi(database.Port)
	connStr := go_ora.BuildUrl(database.Host, port, DefaultOracleServiceName, database.Username, database.Password, nil)
	return connStr
}

