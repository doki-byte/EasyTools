package service

import (
	"EasyTools/app/controller/connect/dbmanager/app/model"
	"EasyTools/app/controller/connect/dbmanager/connectors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
)

type ConnectorService struct {
	connectorFactory connectors.ConnectorFactory
}

func NewConnectorService() ConnectorService {
	return ConnectorService{
		connectorFactory: connectors.ConnectorFactory{},
	}
}

type QueryReq struct {
	Cid      uint   `json:"cid" binding:"required"`
	Sql      string `json:"sql"`
	Database string `json:"database"`
	Table    string `json:"table"`
}

func (s ConnectorService) Ping(c *gin.Context) {
	var config model.DbConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	// 与保存配置保持一致的类型校验
	if err := validateDbConfig(&config); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	connector, err := s.connectorFactory.GetInstance(config.Type)
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	err = connector.Ping(&config)
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "msg": "连接成功"})
}

func (s ConnectorService) ShowDatabases(c *gin.Context) {
	var req QueryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	// 获取配置
	var config model.DbConfig
	config, err := config.FindByID(req.Cid)
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": fmt.Sprintf("获取配置失败: %v", err)})
		return
	}

	connector, err := s.connectorFactory.GetInstance(config.Type)
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	// 对于 SQL 数据库，使用原有逻辑
	db, _, err := s.getConnector(&req)
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	defer db.Close()

	data, err := connector.ShowDatabases(db)
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "msg": "ok", "data": data})
}

func (s ConnectorService) ShowTables(c *gin.Context) {
	var req QueryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	// 获取配置
	var config model.DbConfig
	config, err := config.FindByID(req.Cid)
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": fmt.Sprintf("获取配置失败: %v", err)})
		return
	}

	connector, err := s.connectorFactory.GetInstance(config.Type)
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	// 对于 SQL 数据库，使用原有逻辑
	db, _, err := s.getConnector(&req)
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	defer db.Close()

	data, err := connector.ShowTables(db, req.Database)
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	// 如果返回的数据为空，返回空数组而不是错误
	if data == nil {
		data = &[]connectors.Table{}
	}

	c.JSON(200, gin.H{"code": 0, "msg": "ok", "data": data})
}

func (s ConnectorService) Column(c *gin.Context) {
	var req QueryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	db, connector, err := s.getConnector(&req)
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	defer db.Close()

	data, err := connector.Column(db, req.Database, req.Table)
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	if data == nil {
		c.JSON(200, gin.H{"code": 0, "msg": "ok", "data": []interface{}{}})
		return
	}

	c.JSON(200, gin.H{"code": 0, "msg": "ok", "data": data})
}

func (s ConnectorService) Ddl(c *gin.Context) {
	var req QueryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	db, connector, err := s.getConnector(&req)
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	defer db.Close()

	data, err := connector.Ddl(db, req.Table)
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "msg": "ok", "data": data})
}

func (s ConnectorService) Query(c *gin.Context) {
	var req QueryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	db, connector, err := s.getConnector(&req)
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	defer db.Close()

	// 只有当数据库名不为空时才调用Use方法
	// 某些SQL语句（如CREATE DATABASE、USE DATABASE等）不需要切换数据库
	if req.Database != "" {
		err = connector.Use(db, req.Database)
		if err != nil {
			c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
			return
		}
	}

	startTime := time.Now()
	var result *connectors.Query

	var sqlStr = req.Sql
	slog.Info("Query", "sql", sqlStr)

	// 前端目前统一调用 /api/query（即使是 ALTER/CREATE/INSERT/UPDATE 等非查询语句）
	// 为了避免返回 columns 为空导致前端告警，这里自动识别语句类型：
	// - 查询类：SELECT/WITH/PRAGMA/EXPLAIN/SHOW/DESCRIBE 等，走 connector.Query
	// - 执行类：其它语句，走 connector.Execute，并返回一个可展示的结果表（rows_affected）
	stmts := splitSQLStatements(sqlStr)
	lastStmt := ""
	for i := len(stmts) - 1; i >= 0; i-- {
		if strings.TrimSpace(stmts[i]) != "" {
			lastStmt = stmts[i]
			break
		}
	}

	if lastStmt == "" {
		c.JSON(200, gin.H{"code": 1, "msg": "SQL语句不能为空"})
		return
	}

	if isQueryLikeSQL(lastStmt) {
		result, err = connector.Query(db, sqlStr)
		if err != nil {
			c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
			return
		}
	} else {
		// 多语句脚本：逐条执行，返回每条语句的“Navicat 风格”结果行
		rows := make([]map[string]any, 0, len(stmts))
		columns := []connectors.Column{
			{Field: "statement", Type: connectors.Text},
			{Field: "rows_affected", Type: connectors.Number},
			{Field: "message", Type: connectors.Text},
			{Field: "elapsed_ms", Type: connectors.Number},
		}

		for _, raw := range stmts {
			stmt := strings.TrimSpace(raw)
			if stmt == "" {
				continue
			}

			stmtStart := time.Now()
			affected, execErr := connector.Execute(db, stmt)
			stmtElapsed := float64(time.Since(stmtStart).Milliseconds())

			if execErr != nil {
				c.JSON(200, gin.H{"code": 1, "msg": execErr.Error()})
				return
			}

			rows = append(rows, map[string]any{
				"statement":     stmt,
				"rows_affected": affected,
				"message":       "OK",
				"elapsed_ms":    stmtElapsed,
			})
		}

		result = &connectors.Query{
			Columns: columns,
			Rows:    rows,
			Total:   len(rows),
		}
	}

	elapsedTime := time.Since(startTime)
	cost := elapsedTime.Seconds()
	str := fmt.Sprintf("%.3f", cost)
	cost, err = strconv.ParseFloat(str, 64)
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}
	result.ElapsedTime = cost

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		if req.Table != "" {
			columns, err := connector.Column(db, req.Database, req.Table)
			if err != nil {
				slog.Error("Failed to get column info from Column method", "error", err, "table", req.Table)
				// 如果获取详细列信息失败，保留 Query 返回的基本列信息
				// 这样至少可以显示列名
			} else if columns != nil && len(*columns) > 0 {
				// 只有成功获取到非空列信息时才覆盖
				// Column 方法返回的列信息更详细（包含 Length, Nullable, Default 等）
				result.Columns = *columns
			}
		}
		// 如果 req.Table 为空，保留 Query 返回的列信息
	}()
	wg.Wait()

	// 确保 columns 不为 nil（即使为空数组，也要确保是有效的切片）
	if result.Columns == nil {
		result.Columns = []connectors.Column{}
	}

	c.JSON(200, gin.H{"code": 0, "msg": "ok", "data": result})
}

// splitSQLStatements 尝试把 SQL 脚本按分号拆分为多条语句。
// 规则：支持单引号/双引号/反引号中的分号不拆分；支持 -- 行注释 和 /* */ 块注释的基础处理。
func splitSQLStatements(sqlText string) []string {
	var stmts []string
	var buf strings.Builder

	inSingle := false
	inDouble := false
	inBacktick := false
	inLineComment := false
	inBlockComment := false

	runes := []rune(sqlText)
	for i := 0; i < len(runes); i++ {
		ch := runes[i]
		next := rune(0)
		if i+1 < len(runes) {
			next = runes[i+1]
		}

		// 行注释结束
		if inLineComment {
			buf.WriteRune(ch)
			if ch == '\n' {
				inLineComment = false
			}
			continue
		}
		// 块注释结束
		if inBlockComment {
			buf.WriteRune(ch)
			if ch == '*' && next == '/' {
				buf.WriteRune(next)
				i++
				inBlockComment = false
			}
			continue
		}

		// 进入注释（仅在非字符串内）
		if !inSingle && !inDouble && !inBacktick {
			if ch == '-' && next == '-' {
				buf.WriteRune(ch)
				buf.WriteRune(next)
				i++
				inLineComment = true
				continue
			}
			if ch == '/' && next == '*' {
				buf.WriteRune(ch)
				buf.WriteRune(next)
				i++
				inBlockComment = true
				continue
			}
		}

		// 字符串/标识符引号切换（简单处理转义：'' 或 ""）
		if ch == '\'' && !inDouble && !inBacktick {
			// 处理 '' 作为转义
			if inSingle && next == '\'' {
				buf.WriteRune(ch)
				buf.WriteRune(next)
				i++
				continue
			}
			inSingle = !inSingle
			buf.WriteRune(ch)
			continue
		}
		if ch == '"' && !inSingle && !inBacktick {
			// 处理 "" 作为转义
			if inDouble && next == '"' {
				buf.WriteRune(ch)
				buf.WriteRune(next)
				i++
				continue
			}
			inDouble = !inDouble
			buf.WriteRune(ch)
			continue
		}
		if ch == '`' && !inSingle && !inDouble {
			inBacktick = !inBacktick
			buf.WriteRune(ch)
			continue
		}

		// 分号拆分（仅在非字符串/标识符中）
		if ch == ';' && !inSingle && !inDouble && !inBacktick {
			stmt := strings.TrimSpace(buf.String())
			if stmt != "" {
				stmts = append(stmts, stmt)
			}
			buf.Reset()
			continue
		}

		buf.WriteRune(ch)
	}

	last := strings.TrimSpace(buf.String())
	if last != "" {
		stmts = append(stmts, last)
	}
	return stmts
}

func isQueryLikeSQL(sqlText string) bool {
	s := strings.TrimSpace(sqlText)
	if s == "" {
		return false
	}
	up := strings.ToUpper(s)
	// 常见“返回结果集”的语句前缀（跨数据库 & SQLite PRAGMA）
	prefixes := []string{
		"SELECT", "WITH", "PRAGMA", "EXPLAIN", "SHOW", "DESCRIBE", "DESC",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(up, p) {
			return true
		}
	}
	return false
}

func (s ConnectorService) Execute(c *gin.Context) {
	var req QueryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "参数错误: " + err.Error()})
		return
	}

	if req.Sql == "" {
		c.JSON(200, gin.H{"code": 1, "msg": "SQL语句不能为空"})
		return
	}

	// 基本的SQL注入检测（检测危险关键字组合）
	sqlUpper := strings.ToUpper(strings.TrimSpace(req.Sql))
	// 检测DROP DATABASE、DROP TABLE等危险操作（可以根据需要调整）
	dangerousPatterns := []string{
		"DROP DATABASE",
		"TRUNCATE DATABASE",
		"DELETE FROM", // 这个可能太严格，但可以保留
	}

	// 对于DELETE和DROP操作，记录日志但不阻止（因为这是管理工具）
	for _, pattern := range dangerousPatterns {
		if strings.Contains(sqlUpper, pattern) {
			slog.Warn("执行危险SQL操作", "sql", req.Sql, "pattern", pattern)
		}
	}

	slog.Info("execute", "sql", req.Sql, "database", req.Database)
	db, connector, err := s.getConnector(&req)
	if err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "连接失败: " + err.Error()})
		return
	}
	defer db.Close()

	// Execute方法直接执行SQL，不需要Use数据库（因为SQL中可能包含USE语句）
	// 但如果指定了数据库且SQL中没有USE语句，可以尝试切换数据库
	if req.Database != "" && !strings.Contains(strings.ToUpper(req.Sql), "USE ") {
		// 某些数据库类型需要在执行前切换数据库
		err = connector.Use(db, req.Database)
		if err != nil {
			// 如果切换失败，记录警告但不阻止执行（某些SQL可能不需要数据库上下文）
			slog.Warn("切换数据库失败，继续执行SQL", "database", req.Database, "error", err)
		}
	}

	rows, err := connector.Execute(db, req.Sql)
	if err != nil {
		// 提供更详细的错误信息
		errorMsg := err.Error()
		// 尝试提取更友好的错误信息
		if strings.Contains(errorMsg, "foreign key constraint") {
			errorMsg = "删除失败: 存在外键约束，请先删除关联数据"
		} else if strings.Contains(errorMsg, "duplicate entry") {
			errorMsg = "保存失败: 数据重复，请检查唯一约束"
		} else if strings.Contains(errorMsg, "cannot be null") {
			errorMsg = "保存失败: 必填字段不能为空"
		}
		c.JSON(200, gin.H{"code": 1, "msg": errorMsg})
		return
	}

	// 返回执行结果
	// connector.Execute返回int类型，表示受影响的行数
	result := gin.H{
		"code":          0,
		"msg":           "执行成功",
		"data":          rows,
		"rows_affected": rows, // 直接添加影响的行数
	}

	c.JSON(200, result)
}

func (s ConnectorService) getConnector(req *QueryReq) (*sqlx.DB, connectors.Connector, error) {
	var config model.DbConfig
	config, err := config.FindByID(req.Cid)
	if err != nil {
		return nil, nil, fmt.Errorf("获取配置失败: %v", err)
	}

	connector, err := s.connectorFactory.GetInstance(config.Type)
	if err != nil {
		return nil, nil, err
	}

	// 对于 SQL 数据库，需要 SQL 连接进行操作

	var db *sqlx.DB
	// 对于 PostgreSQL，如果指定了数据库，需要连接到该数据库
	if config.Type == connectors.PostgreSQL && req.Database != "" {
		// PostgreSQL 需要在连接时指定数据库，创建一个包含指定数据库的 DSN
		dsn := fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s sslmode=disable",
			config.Username, config.Password, config.Host, config.Port, req.Database)
		db, err = sqlx.Open("postgres", dsn)
		if err != nil {
			return nil, nil, err
		}
	} else {
		db, err = connector.Connect(&config)
		if err != nil {
			return nil, nil, err
		}
	}

	return db, connector, nil
}
