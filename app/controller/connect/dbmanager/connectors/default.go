package connectors

import (
	"database/sql"
	"fmt"
	"github.com/jmoiron/sqlx"
	"log/slog"
	"strconv"
)

type DefaultConnector struct {
}

func (c DefaultConnector) ShowDatabases(db *sqlx.DB) ([]string, error) {
	var result []string
	sqlStr := "show databases"
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

func (c DefaultConnector) ShowTables(db *sqlx.DB, database string) (*[]Table, error) {
	var tables []Table
	var sqlStr string
	var err error
	var rows *sqlx.Rows
	
	// 如果数据库名为空，返回空结果而不是查询空字符串
	if database == "" {
		return &tables, nil
	}
	
	sqlStr = `SELECT TABLE_NAME, TABLE_ROWS, TABLE_COLLATION, CREATE_TIME, UPDATE_TIME
			FROM INFORMATION_SCHEMA.TABLES
			WHERE TABLE_SCHEMA = ?`

	rows, err = db.Queryx(sqlStr, database)
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
		var tableRows sql.NullString
		var tableCollation sql.NullString
		var createTime sql.NullString
		var updateTime sql.NullString
		var err = rows.Scan(&tableName, &tableRows, &tableCollation, &createTime, &updateTime)
		if err != nil {
			return nil, err
		}

		if tableName.Valid {
			table.Name = tableName.String
		}
		if tableRows.Valid {
			table.Rows = tableRows.String
		}
		if tableCollation.Valid {
			table.Collation = tableCollation.String
		}
		if createTime.Valid {
			table.CreateTime = createTime.String
		}
		if updateTime.Valid {
			table.UpdateTime = updateTime.String
		}

		tables = append(tables, table)
	}
	return &tables, err
}

func (c DefaultConnector) Column(db *sqlx.DB, params ...string) (*[]Column, error) {
	var columns []Column
	schema := params[0]
	table := params[1]
	
	// 如果数据库名或表名为空，返回错误
	if schema == "" || table == "" {
		return nil, fmt.Errorf("database and table name cannot be empty")
	}
	
	sqlStr := `SELECT 
    			COLUMN_NAME as field,
       			COLUMN_TYPE as type,
       			CHARACTER_MAXIMUM_LENGTH AS 'length',
       			IS_NULLABLE AS 'nullable',
       			COLUMN_KEY AS 'key',
       			COLUMN_COMMENT as comment, 
				COLUMN_DEFAULT AS 'default'
			FROM INFORMATION_SCHEMA.COLUMNS
			WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?`

	rows, err := db.Queryx(sqlStr, schema, table)
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
			if nullable.String == "YES" {
				column.Nullable = true
			} else {
				column.Nullable = false
			}
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
	return &columns, err
}

func (c DefaultConnector) Query(db *sqlx.DB, sqlStr string) (*Query, error) {
	rows, err := db.Queryx(sqlStr)
	if err != nil {
		return nil, err
	}
	defer func() {
		if rows != nil {
			_ = rows.Close()
		}
	}()

	columnNames, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	values := make([][]byte, len(columnNames))
	scans := make([]interface{}, len(columnNames))
	columns := make([]Column, len(columnNames))

	for i := range columns {
		scans[i] = &values[i]
		columnType, err := convertColumnType(columnTypes[i].DatabaseTypeName())
		if err != nil {
			return nil, err
		}
		columns[i] = Column{
			Field: columnNames[i],
			Type:  columnType,
		}
	}

	for rows.Next() {
		err = rows.Scan(scans...)
		if err != nil {
			return nil, err
		}
		row := make(map[string]interface{})
		for i, value := range values {
			columnName := columnNames[i] // 使用列名而不是columnType.Name()
			columnType := columnTypes[i]
			typeName := columnType.DatabaseTypeName()
			
			// 检查value是否为nil或空
			if value == nil || len(value) == 0 {
				row[columnName] = nil
				continue
			}

			cColumnType, err := convertColumnType(columnTypes[i].DatabaseTypeName())
			if err != nil {
				return nil, err
			}
			if cColumnType == Text {
				row[columnName] = string(value)
			} else if cColumnType == Number {
				val, err := strconv.ParseFloat(string(value), 64)
				if err != nil {
					return nil, err
				}
				row[columnName] = val
			} else {
				slog.Error("dont support type", "type name", typeName)
				break
			}
		}
		result = append(result, row)
	}

	query := &Query{
		Rows:    result,
		Columns: columns,
		Total:   len(result),
	}

	return query, err
}

func (c DefaultConnector) Count(db *sqlx.DB, sqlStr string) (int, error) {
	rows, err := db.Queryx(sqlStr)
	if err != nil {
		return 0, err
	}
	defer func() {
		if rows != nil {
			_ = rows.Close()
		}
	}()
	for rows.Next() {
		var total int
		err = rows.Scan(&total)
		if err != nil {
			return 0, err
		}
		return total, err
	}
	return 0, err
}

func (c DefaultConnector) Ddl(db *sqlx.DB, table string) (string, error) {
	sqlStr := "SHOW CREATE TABLE " + table
	row := db.QueryRowx(sqlStr)
	var tableName string
	var ddl string
	err := row.Scan(&tableName, &ddl)
	return ddl, err
}

func (c DefaultConnector) Execute(db *sqlx.DB, sqlStr string) (int, error) {
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

func convertColumnType(columnType string) (string, error) {
	switch columnType {
	case "TINYINT", "SMALLINT", "INT", "INTEGER", "BIGINT", "UNSIGNED BIGINT", "UNSIGNED TINYINT":
		return Number, nil
	case "TEXT", "VARCHAR", "DATETIME", "BIT", "DATE", "CHAR", "TIMESTAMP", "JSON":
		return Text, nil
	default:
		return Text, nil
	}
}

