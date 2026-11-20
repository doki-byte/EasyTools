package connectors

import (
	"EasyTools/app/controller/connect/dbmanager/app/model"
	"database/sql"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/microsoft/go-mssqldb"
)

type SQLServerConnector struct {
	DefaultConnector
}

func (c SQLServerConnector) Ping(config *model.DbConfig) error {
	db, err := sqlx.Open("sqlserver", c.dsn(config))
	if err != nil {
		return err
	}
	defer db.Close()
	err = db.Ping()
	return err
}

func (c SQLServerConnector) Connect(config *model.DbConfig) (*sqlx.DB, error) {
	return sqlx.Open("sqlserver", c.dsn(config))
}

// 简连助手不使用代理，直接连接

func (c SQLServerConnector) Use(db *sqlx.DB, database string) error {
	// 如果数据库名为空，直接返回，不执行USE语句
	if database == "" {
		return nil
	}
	// SQL Server使用方括号包裹数据库名，避免数据库名为纯数字或包含特殊字符时出现语法错误
	sqlStr := fmt.Sprintf("USE [%s]", database)
	result, err := db.Exec(sqlStr)
	if err != nil {
		slog.Error("Failed to use database", "database", database, "error", err)
		return err
	}
	_ = result
	return nil
}

func (c SQLServerConnector) ShowDatabases(db *sqlx.DB) ([]string, error) {
	var result []string
	sqlStr := "SELECT name FROM sys.databases WHERE database_id > 4"
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

func (c SQLServerConnector) ShowTables(db *sqlx.DB, database string) (*[]Table, error) {
	var tables []Table
	// 先切换到指定数据库
	if database != "" {
		err := c.Use(db, database)
		if err != nil {
			return nil, err
		}
	}

	// SQL Server 使用当前数据库上下文，不需要指定 TABLE_CATALOG
	sqlStr := `SELECT TABLE_NAME 
		FROM INFORMATION_SCHEMA.TABLES 
		WHERE TABLE_TYPE = 'BASE TABLE'`

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
		var table Table
		var tableName string
		err = rows.Scan(&tableName)
		if err != nil {
			return nil, err
		}
		table.Name = tableName
		tables = append(tables, table)
	}
	return &tables, err
}

func (c SQLServerConnector) Column(db *sqlx.DB, params ...string) (*[]Column, error) {
	var columns []Column
	database := params[0]
	table := params[1]

	// 如果数据库名或表名为空，返回错误
	if database == "" || table == "" {
		slog.Error("Database or table name is empty", "database", database, "table", table)
		return nil, fmt.Errorf("database and table name cannot be empty")
	}

	// 先切换到指定数据库，确保查询在正确的数据库上下文中执行
	err := c.Use(db, database)
	if err != nil {
		slog.Error("Failed to switch database in Column method", "database", database, "error", err)
		return nil, fmt.Errorf("failed to switch database: %w", err)
	}

	// 验证当前数据库上下文，并获取实际数据库名称用于后续查询
	var currentDB string
	err = db.Get(&currentDB, "SELECT DB_NAME()")
	if err == nil && currentDB != "" {
		// 使用实际获取的数据库名称，确保大小写一致
		database = currentDB
	}

	// 解析表名，提取 schema 和表名
	// 移除可能的方括号和数据库前缀
	schemaName := "dbo" // 默认 schema
	tableName := table

	// 移除可能的方括号
	originalTable := table
	table = strings.Trim(table, "[]")

	// 检查是否包含 schema
	if strings.Contains(table, ".") {
		parts := strings.Split(table, ".")
		// 如果格式是 database.schema.table，取最后两部分
		// 如果格式是 schema.table，取这两部分
		if len(parts) >= 3 {
			// database.schema.table 格式，取 schema 和 table
			schemaName = parts[len(parts)-2]
			tableName = parts[len(parts)-1]
		} else if len(parts) == 2 {
			// schema.table 格式
			schemaName = parts[0]
			tableName = parts[1]
		} else {
			// 只有表名
			tableName = parts[0]
		}
	} else {
		// 只有表名，先尝试从 INFORMATION_SCHEMA 查找实际的 schema
		// 因为表可能不在 dbo schema 中，使用实际数据库名称查询
		// SQL Server 驱动不支持 ? 占位符，需要直接拼接字符串
		escapedDBForSchema := strings.ReplaceAll(database, "'", "''")
		escapedTableForSchema := strings.ReplaceAll(table, "'", "''")
		findSchemaSQL := fmt.Sprintf(`SELECT TOP 1 TABLE_SCHEMA 
			FROM INFORMATION_SCHEMA.TABLES 
			WHERE TABLE_CATALOG = '%s' AND TABLE_NAME = '%s' 
			ORDER BY CASE WHEN TABLE_SCHEMA = 'dbo' THEN 0 ELSE 1 END`, 
			escapedDBForSchema, escapedTableForSchema)
		var foundSchema string
		err = db.Get(&foundSchema, findSchemaSQL)
		if err == nil && foundSchema != "" {
			schemaName = foundSchema
		}
		tableName = table
	}

	// 移除可能残留的方括号
	schemaName = strings.Trim(schemaName, "[]")
	tableName = strings.Trim(tableName, "[]")

	// 使用 INFORMATION_SCHEMA.COLUMNS 视图查询列信息
	// SQL Server 的 go-mssqldb 驱动不支持 ? 占位符，需要直接拼接字符串
	// 转义单引号防止 SQL 注入（对于 INFORMATION_SCHEMA 查询，这些值应该是安全的）
	escapedDB := strings.ReplaceAll(database, "'", "''")
	escapedSchema := strings.ReplaceAll(schemaName, "'", "''")
	escapedTable := strings.ReplaceAll(tableName, "'", "''")
	
	// 构建完整的表名用于 OBJECT_ID 函数
	fullTableName := fmt.Sprintf("[%s].[%s].[%s]", escapedDB, escapedSchema, escapedTable)
	
	sqlStr := fmt.Sprintf(`SELECT 
		c.COLUMN_NAME AS field,
		c.DATA_TYPE AS type,
		CASE 
			WHEN c.DATA_TYPE IN ('nvarchar', 'varchar', 'nchar', 'char') THEN 
				CASE WHEN c.CHARACTER_MAXIMUM_LENGTH = -1 THEN 'MAX' 
					 WHEN c.CHARACTER_MAXIMUM_LENGTH IS NULL THEN NULL
					 ELSE CAST(c.CHARACTER_MAXIMUM_LENGTH AS VARCHAR) END
			WHEN c.DATA_TYPE IN ('decimal', 'numeric') THEN 
				CAST(c.NUMERIC_PRECISION AS VARCHAR) + ',' + CAST(c.NUMERIC_SCALE AS VARCHAR)
			WHEN c.DATA_TYPE IN ('ntext', 'text', 'image', 'xml', 'sql_variant') THEN NULL
			ELSE CASE WHEN c.CHARACTER_MAXIMUM_LENGTH IS NULL THEN NULL 
					  ELSE CAST(c.CHARACTER_MAXIMUM_LENGTH AS VARCHAR) END
		END AS 'length',
		c.IS_NULLABLE AS 'nullable',
		ISNULL(c.COLUMN_DEFAULT, '') AS 'default',
		CASE WHEN ic.OBJECT_ID IS NOT NULL THEN 1 ELSE 0 END AS 'is_identity'
	FROM INFORMATION_SCHEMA.COLUMNS c
	LEFT JOIN sys.identity_columns ic 
		ON ic.object_id = OBJECT_ID('%s')
		AND ic.name = c.COLUMN_NAME
	WHERE c.TABLE_CATALOG = '%s' AND c.TABLE_SCHEMA = '%s' AND c.TABLE_NAME = '%s'
	ORDER BY c.ORDINAL_POSITION`, 
		fullTableName, escapedDB, escapedSchema, escapedTable)

	rows, err := db.Queryx(sqlStr)
	if err != nil {
		slog.Error("查询列信息失败", "database", database, "schema", schemaName, "table", tableName, "originalTable", originalTable, "error", err)
		return nil, fmt.Errorf("failed to query columns for table %s.%s.%s: %w", database, schemaName, tableName, err)
	}
	defer func() {
		if rows != nil {
			_ = rows.Close()
		}
	}()

	hasRows := false
	columnCount := 0
	for rows.Next() {
		hasRows = true
		columnCount++
		var column Column
		var columnName string
		var columnType string
		var length interface{}
		var nullable string
		var columnDefault interface{}
		var isIdentity int
		err = rows.Scan(&columnName, &columnType, &length, &nullable, &columnDefault, &isIdentity)
		if err != nil {
			slog.Error("扫描列信息失败", "error", err)
			return nil, fmt.Errorf("failed to scan column information: %w", err)
		}

		column.Field = columnName
		column.Type = columnType
		if length != nil {
			lengthStr := fmt.Sprintf("%v", length)
			// 处理 NULL 值的情况
			if lengthStr != "<nil>" && lengthStr != "" {
				column.Length = lengthStr
			}
		}
		column.Nullable = (nullable == "YES")
		column.AutoIncrement = (isIdentity == 1)
		if columnDefault != nil {
			defaultStr := fmt.Sprintf("%v", columnDefault)
			if defaultStr != "<nil>" {
				column.Default = defaultStr
			}
		}

		// 检查主键（使用 INFORMATION_SCHEMA），使用实际数据库名称查询
		// SQL Server 驱动不支持 ? 占位符，需要直接拼接字符串
		escapedColumnName := strings.ReplaceAll(columnName, "'", "''")
		pkSql := fmt.Sprintf(`SELECT COUNT(*) 
			FROM INFORMATION_SCHEMA.TABLE_CONSTRAINTS tc
			INNER JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE kcu 
				ON tc.CONSTRAINT_NAME = kcu.CONSTRAINT_NAME 
				AND tc.TABLE_CATALOG = kcu.TABLE_CATALOG
				AND tc.TABLE_SCHEMA = kcu.TABLE_SCHEMA 
				AND tc.TABLE_NAME = kcu.TABLE_NAME
			WHERE tc.CONSTRAINT_TYPE = 'PRIMARY KEY'
				AND tc.TABLE_CATALOG = '%s'
				AND tc.TABLE_SCHEMA = '%s'
				AND tc.TABLE_NAME = '%s'
				AND kcu.COLUMN_NAME = '%s'`, 
			escapedDB, escapedSchema, escapedTable, escapedColumnName)
		var pkCount int
		err = db.Get(&pkCount, pkSql)
		if err != nil {
			slog.Error("查询主键信息失败", "schema", schemaName, "table", tableName, "column", columnName, "error", err)
		} else if pkCount > 0 {
			column.Key = Primary
		}

		columns = append(columns, column)
	}

	// 如果没有查询到任何列，返回错误
	if !hasRows {
		slog.Warn("未查询到任何列", "database", database, "schema", schemaName, "table", tableName, "originalTable", originalTable)
		// 验证表是否存在，使用实际数据库名称查询
		// SQL Server 驱动不支持 ? 占位符，需要直接拼接字符串
		checkTableSQL := fmt.Sprintf(`SELECT COUNT(*) FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_CATALOG = '%s' AND TABLE_SCHEMA = '%s' AND TABLE_NAME = '%s'`, 
			escapedDB, escapedSchema, escapedTable)
		var tableExists int
		err = db.Get(&tableExists, checkTableSQL)
		if err != nil {
			slog.Error("检查表是否存在失败", "database", database, "schema", schemaName, "table", tableName, "error", err)
			return nil, fmt.Errorf("failed to check if table exists: %w", err)
		}
		if tableExists == 0 {
			slog.Error("表不存在", "database", database, "schema", schemaName, "table", tableName)
			return nil, fmt.Errorf("table %s.%s does not exist in database %s", schemaName, tableName, database)
		}
		slog.Warn("表存在但没有列", "database", database, "schema", schemaName, "table", tableName)
		return nil, fmt.Errorf("no columns found for table %s.%s.%s (table exists but has no columns)", database, schemaName, tableName)
	}

	return &columns, err
}

// normalizeSQLForSQLServer 规范化 SQL 语句，将 [database].[schema].[table] 格式转换为 [schema].[table]
// 因为在使用 USE [database] 后，不需要在表名中包含数据库名
func (c SQLServerConnector) normalizeSQLForSQLServer(sqlStr string) string {
	// 使用正则表达式匹配 [database].[schema].[table] 格式
	// 匹配模式：\[([^\]]+)\]\.\[([^\]]+)\]\.\[([^\]]+)\]
	re := regexp.MustCompile(`\[([^\]]+)\]\.\[([^\]]+)\]\.\[([^\]]+)\]`)
	result := re.ReplaceAllStringFunc(sqlStr, func(match string) string {
		// 提取各部分
		parts := re.FindStringSubmatch(match)
		if len(parts) == 4 {
			// 格式：[database].[schema].[table]，转换为 [schema].[table]
			return fmt.Sprintf("[%s].[%s]", parts[2], parts[3])
		}
		return match
	})
	return result
}

func (c SQLServerConnector) Query(db *sqlx.DB, sqlStr string) (*Query, error) {
	// SQL Server 可以直接使用包含数据库名的完整表名，不需要规范化
	// 因为 USE 语句可能在某些情况下不会立即生效，或者连接池可能复用连接
	// 所以直接使用原始 SQL，让 SQL Server 自己解析完整的三部分名称
	querySQL := sqlStr

	// 直接使用原始 SQL，不进行规范化
	rows, err := db.Queryx(querySQL)
	if err != nil {
		slog.Error("SQL Server Query failed", "error", err, "sql", querySQL)
		// 如果原始 SQL 失败，尝试规范化（作为备选方案）
		normalizedSQL := c.normalizeSQLForSQLServer(sqlStr)
		rows, err = db.Queryx(normalizedSQL)
		if err != nil {
			slog.Error("Both original and normalized SQL failed", "error", err)
			return nil, err
		}
		querySQL = normalizedSQL
	}
	defer func() {
		if rows != nil {
			_ = rows.Close()
		}
	}()

	columnNames, err := rows.Columns()
	if err != nil {
		slog.Error("Failed to get columns", "error", err)
		return nil, err
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		slog.Error("Failed to get column types", "error", err)
		return nil, err
	}

	var result []map[string]interface{}
	// 使用 [][]byte 作为扫描目标，这是最可靠的方式，可以处理所有数据类型
	// SQL Server 驱动会将所有数据转换为字节数组
	values := make([][]byte, len(columnNames))
	scans := make([]interface{}, len(columnNames))
	columns := make([]Column, len(columnNames))

	// 初始化扫描目标和列信息
	for i := range columns {
		scans[i] = &values[i]
		columnType, err := convertSQLServerColumnType(columnTypes[i].DatabaseTypeName())
		if err != nil {
			slog.Warn("Failed to convert column type", "type", columnTypes[i].DatabaseTypeName(), "error", err)
			columnType = Text
		}
		columns[i] = Column{
			Field: columnNames[i],
			Type:  columnType,
		}
	}

	// 使用 Scan 方法扫描数据
	rowCount := 0

	for rows.Next() {
		rowCount++

		err = rows.Scan(scans...)
		if err != nil {
			slog.Error("Failed to scan row", "error", err, "row", rowCount)
			return nil, err
		}

		row := make(map[string]interface{})
		for i, value := range values {
			columnName := columnNames[i]
			columnType := columnTypes[i]
			typeName := columnType.DatabaseTypeName()

			// 检查 value 是否为空
			// 对于 []byte，空切片表示 NULL 值（nil 切片的 len() 也是 0）
			if len(value) == 0 {
				row[columnName] = nil
				continue
			}

			// 获取列类型
			cColumnType, err := convertSQLServerColumnType(typeName)
			if err != nil {
				slog.Warn("Failed to convert column type", "type", typeName, "error", err)
				cColumnType = Text
			}

			// 根据类型转换值
			valStr := string(value)
			if cColumnType == Number {
				// 尝试解析为数字
				val, err := strconv.ParseFloat(valStr, 64)
				if err != nil {
					// 如果解析失败，尝试作为整数解析
					intVal, intErr := strconv.ParseInt(valStr, 10, 64)
					if intErr != nil {
						slog.Warn("Failed to parse number", "value", valStr, "type", typeName, "error", err)
						// 解析失败，作为字符串返回
						row[columnName] = valStr
					} else {
						row[columnName] = float64(intVal)
					}
				} else {
					row[columnName] = val
				}
			} else {
				// 文本类型，直接转换为字符串
				row[columnName] = valStr
			}
		}

		result = append(result, row)
	}

	// 检查迭代过程中是否有错误
	if err = rows.Err(); err != nil {
		slog.Error("Error during rows iteration", "error", err)
		return nil, err
	}

	// 确保列信息被正确返回（即使没有数据行）
	if len(columns) == 0 && len(columnNames) > 0 {
		// 如果没有列信息，从 rows.Columns() 重新构建
		columns = make([]Column, 0, len(columnNames))
		for i, colName := range columnNames {
			columnType := Text
			if i < len(columnTypes) {
				ct, err := convertSQLServerColumnType(columnTypes[i].DatabaseTypeName())
				if err == nil {
					columnType = ct
				}
			}
			columns = append(columns, Column{
				Field: colName,
				Type:  columnType,
			})
		}
	}

	// 确保 columns 不为 nil（即使为空数组，也要确保是有效的切片）
	if columns == nil {
		columns = []Column{}
	}

	query := &Query{
		Rows:    result,
		Columns: columns,
		Total:   len(result),
	}

	return query, nil
}

func (c SQLServerConnector) Count(db *sqlx.DB, sqlStr string) (int, error) {
	// 规范化 SQL，移除表名中的数据库前缀
	normalizedSQL := c.normalizeSQLForSQLServer(sqlStr)

	rows, err := db.Queryx(normalizedSQL)
	if err != nil {
		return 0, err
	}
	defer func() {
		if rows != nil {
			_ = rows.Close()
		}
	}()

	var total int
	if rows.Next() {
		err = rows.Scan(&total)
		if err != nil {
			return 0, err
		}
		return total, err
	}
	return 0, err
}

// convertSQLServerColumnType 转换 SQL Server 列类型
func convertSQLServerColumnType(columnType string) (string, error) {
	switch columnType {
	case "TINYINT", "SMALLINT", "INT", "BIGINT", "BIT", "DECIMAL", "NUMERIC", "FLOAT", "REAL", "MONEY", "SMALLMONEY":
		return Number, nil
	case "NVARCHAR", "VARCHAR", "NCHAR", "CHAR", "NTEXT", "TEXT", "DATE", "TIME", "DATETIME", "DATETIME2", "DATETIMEOFFSET", "SMALLDATETIME", "TIMESTAMP", "UNIQUEIDENTIFIER", "XML", "JSON", "SQL_VARIANT":
		return Text, nil
	default:
		return Text, nil
	}
}

func (c SQLServerConnector) Ddl(db *sqlx.DB, table string) (string, error) {
	// SQL Server 使用系统函数获取 DDL
	sqlStr := `SELECT 
		'CREATE TABLE ' + QUOTENAME(SCHEMA_NAME(schema_id)) + '.' + QUOTENAME(name) + ' (' + CHAR(13) +
		STUFF((
			SELECT CHAR(13) + '    ' + QUOTENAME(c.name) + ' ' + 
				CASE 
					WHEN t.name IN ('nvarchar', 'varchar', 'nchar', 'char') THEN 
						t.name + '(' + CASE WHEN c.max_length = -1 THEN 'MAX' ELSE CAST(c.max_length AS VARCHAR) END + ')'
					WHEN t.name IN ('decimal', 'numeric') THEN 
						t.name + '(' + CAST(c.precision AS VARCHAR) + ',' + CAST(c.scale AS VARCHAR) + ')'
					ELSE t.name
				END +
				CASE WHEN c.is_nullable = 0 THEN ' NOT NULL' ELSE '' END +
				CASE WHEN i.is_primary_key = 1 THEN ' PRIMARY KEY' ELSE '' END
			FROM sys.columns c
			INNER JOIN sys.types t ON c.user_type_id = t.user_type_id
			LEFT JOIN sys.index_columns ic ON c.object_id = ic.object_id AND c.column_id = ic.column_id
			LEFT JOIN sys.indexes i ON ic.object_id = i.object_id AND ic.index_id = i.index_id
			WHERE c.object_id = OBJECT_ID(?)
			ORDER BY c.column_id
			FOR XML PATH('')
		), 1, 4, '') + CHAR(13) + ')' AS DDL
	FROM sys.tables
	WHERE object_id = OBJECT_ID(?)`

	row := db.QueryRowx(sqlStr, table, table)
	var ddl sql.NullString
	err := row.Scan(&ddl)
	if err != nil {
		return "", err
	}
	if ddl.Valid {
		return ddl.String, nil
	}
	return "", fmt.Errorf("无法获取表 %s 的 DDL", table)
}

func (c SQLServerConnector) Execute(db *sqlx.DB, sqlStr string) (int, error) {
	// 规范化 SQL，移除表名中的数据库前缀
	normalizedSQL := c.normalizeSQLForSQLServer(sqlStr)

	result, err := db.Exec(normalizedSQL)
	if err != nil {
		return 0, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(rows), err
}

func (c SQLServerConnector) dsn(config *model.DbConfig) string {
	// SQL Server 连接字符串格式: server=host,port;user id=username;password=password;database=master;connection timeout=30;encrypt=disable
	// 注意：端口使用逗号分隔，不是冒号
	return fmt.Sprintf("server=%s,%s;user id=%s;password=%s;database=master;connection timeout=30;encrypt=disable",
		config.Host, config.Port, config.Username, config.Password)
}
