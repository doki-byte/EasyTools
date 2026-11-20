package connectors

import (
	"EasyTools/app/controller/connect/dbmanager/app/model"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	// 达梦数据库驱动需要单独安装: go get gitee.com/chunanyong/dm
	// 如果未安装，请注释掉下面的导入，或使用 ODBC 方式连接
	_ "gitee.com/chunanyong/dm"
)

type DMConnector struct {
	DefaultConnector
}

func (c DMConnector) Ping(config *model.DbConfig) error {
	db, err := sqlx.Open("dm", c.dsn(config))
	if err != nil {
		return err
	}
	defer db.Close()
	err = db.Ping()
	return err
}

func (c DMConnector) Connect(config *model.DbConfig) (*sqlx.DB, error) {
	return sqlx.Open("dm", c.dsn(config))
}

// 简连助手不使用代理，直接连接

func (c DMConnector) ShowDatabases(db *sqlx.DB) ([]string, error) {
	// 达梦数据库使用模式（schema）概念，类似 Oracle
	var result []string
	sqlStr := "SELECT DISTINCT OWNER FROM DBA_OBJECTS WHERE OBJECT_TYPE = 'TABLE'"
	rows, err := db.Queryx(sqlStr)
	if err != nil {
		// 如果权限不足，尝试查询当前用户的模式
		sqlStr = "SELECT USER FROM DUAL"
		rows, err = db.Queryx(sqlStr)
		if err != nil {
			return nil, err
		}
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

func (c DMConnector) ShowTables(db *sqlx.DB, database string) (*[]Table, error) {
	var tables []Table
	var sqlStr string
	var rows *sqlx.Rows
	var err error
	
	// 达梦数据库：如果database为空，查询当前用户的表
	if database == "" {
		sqlStr = "SELECT TABLE_NAME FROM USER_TABLES"
		rows, err = db.Queryx(sqlStr)
	} else {
		// 使用参数化查询防止SQL注入
		sqlStr = "SELECT TABLE_NAME FROM DBA_TABLES WHERE OWNER = ?"
		rows, err = db.Queryx(sqlStr, database)
	}
	
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

func (c DMConnector) Column(db *sqlx.DB, params ...string) (*[]Column, error) {
	var columns []Column
	database := params[0]
	table := params[1]
	
	// 如果表名为空，返回错误
	if table == "" {
		return nil, fmt.Errorf("table name cannot be empty")
	}
	
	var sqlStr string
	var rows *sqlx.Rows
	var err error
	
	// 达梦数据库：如果database为空，查询当前用户的表
	if database == "" {
		sqlStr = `SELECT 
			COLUMN_NAME as field,
			DATA_TYPE as type,
			DATA_LENGTH as length,
			NULLABLE as nullable,
			DATA_DEFAULT as default
		FROM USER_TAB_COLUMNS
		WHERE TABLE_NAME = ?`
		rows, err = db.Queryx(sqlStr, table)
	} else {
		// 使用参数化查询防止SQL注入
		sqlStr = `SELECT 
			COLUMN_NAME as field,
			DATA_TYPE as type,
			DATA_LENGTH as length,
			NULLABLE as nullable,
			DATA_DEFAULT as default
		FROM DBA_TAB_COLUMNS
		WHERE OWNER = ? AND TABLE_NAME = ?`
		rows, err = db.Queryx(sqlStr, database, table)
	}
	
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
		column.Nullable = (nullable == "Y")
		if columnDefault != nil {
			column.Default = fmt.Sprintf("%v", columnDefault)
		}
		
		// 检查主键，使用参数化查询防止SQL注入
		var pkSql string
		var pkCount int
		if database == "" {
			pkSql = `SELECT COUNT(*) 
				FROM USER_CONS_COLUMNS 
				WHERE TABLE_NAME = ? AND COLUMN_NAME = ? 
				AND CONSTRAINT_NAME IN (
					SELECT CONSTRAINT_NAME 
					FROM USER_CONSTRAINTS 
					WHERE CONSTRAINT_TYPE = 'P' AND TABLE_NAME = ?
				)`
			_ = db.Get(&pkCount, pkSql, table, columnName, table)
		} else {
			pkSql = `SELECT COUNT(*) 
				FROM DBA_CONS_COLUMNS 
				WHERE OWNER = ? AND TABLE_NAME = ? AND COLUMN_NAME = ? 
				AND CONSTRAINT_NAME IN (
					SELECT CONSTRAINT_NAME 
					FROM DBA_CONSTRAINTS 
					WHERE OWNER = ? AND CONSTRAINT_TYPE = 'P' AND TABLE_NAME = ?
				)`
			_ = db.Get(&pkCount, pkSql, database, table, columnName, database, table)
		}
		if pkCount > 0 {
			column.Key = Primary
		}
		
		columns = append(columns, column)
	}
	return &columns, err
}

func (c DMConnector) dsn(config *model.DbConfig) string {
	// 达梦数据库连接字符串格式: dm://用户名:密码@主机:端口
	return fmt.Sprintf("dm://%s:%s@%s:%s", config.Username, config.Password, config.Host, config.Port)
}

