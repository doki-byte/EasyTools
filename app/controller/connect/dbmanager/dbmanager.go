package dbmanager

import (
	"EasyTools/app/controller/connect"
	"EasyTools/app/controller/connect/dbmanager/app/config"
	middleware2 "EasyTools/app/controller/connect/dbmanager/app/middleware"
	model2 "EasyTools/app/controller/connect/dbmanager/app/model"
	service2 "EasyTools/app/controller/connect/dbmanager/app/service"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"
)

var connectorService = service2.NewConnectorService()

func StartWebDBManager() {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("DBManager服务启动panic", "err", r)
		}
	}()

	// 在启动时主动初始化数据库，确保表已创建
	if err := model2.DbMigrate(config.DefaultConfig.DbType, config.DefaultConfig.DbDsn); err != nil {
		slog.Error("数据库初始化失败", "err", err)
		// 不退出，让中间件处理后续的数据库检查
	}

	gin.SetMode(gin.ReleaseMode)
	var engine = gin.Default()
	port, _ := strconv.Atoi(config.DefaultConfig.Port)
	engine.Use(middleware2.Cors(port), middleware2.DbCheck())

	// 配置管理
	{
		engine.GET("/api/config", service2.ConfigFindAll)
		engine.GET("/api/config/:id", service2.ConfigFindByID)
		engine.POST("/api/config", service2.ConfigCreate)
		engine.PUT("/api/config", service2.ConfigUpdateById)
		engine.DELETE("/api/config/:id", service2.ConfigDeleteById)
	}

	// 连接器操作
	{
		engine.POST("/api/ping", connectorService.Ping)
		engine.POST("/api/databases", connectorService.ShowDatabases)
		engine.POST("/api/tables", connectorService.ShowTables)
		engine.POST("/api/columns", connectorService.Column)
		engine.POST("/api/ddl", connectorService.Ddl)
		engine.POST("/api/query", connectorService.Query)
		engine.POST("/api/execute", connectorService.Execute)
	}

	// SQL 模板管理
	{
		engine.GET("/api/config/sql-templates", connect.GetSQLTemplates)
		engine.POST("/api/config/sql-templates", connect.CreateSQLTemplate)
		engine.PUT("/api/config/sql-templates/:id", connect.UpdateSQLTemplate)
		engine.DELETE("/api/config/sql-templates/:id", connect.DeleteSQLTemplate)
		engine.POST("/api/config/sql-templates/reset", connect.ResetSQLTemplates)
	}

	// SQL 模板分类管理
	{
		engine.GET("/api/config/sql-template-categories", connect.GetSQLTemplateCategories)
		engine.POST("/api/config/sql-template-categories", connect.CreateSQLTemplateCategory)
		engine.PUT("/api/config/sql-template-categories/:id", connect.UpdateSQLTemplateCategory)
		engine.DELETE("/api/config/sql-template-categories/:id", connect.DeleteSQLTemplateCategory)
	}

	// 端口配置接口（用于前端检查服务是否启动）
	{
		engine.GET("/api/port-config", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"code": 0,
				"data": gin.H{
					"port": config.DefaultConfig.Port,
				},
			})
		})
	}

	address := fmt.Sprintf("%s:%s", config.DefaultConfig.Address, config.DefaultConfig.Port)
	slog.Info("DBManager服务启动", "address", address)
	fmt.Printf("DBManager服务正在启动，监听地址: %s\n", address)
	err := engine.Run(address)
	if err != nil {
		slog.Error("DBManager服务启动失败", "msg", err.Error())
		fmt.Printf("DBManager服务启动失败: %v\n", err)
		return
	}
}

// StartWebDBManagerWithPort 使用指定端口启动数据库管理服务
func StartWebDBManagerWithPort(port int) {
	if port <= 0 {
		slog.Error("DBManager端口无效", "port", port)
		return
	}
	// 修改配置中的端口
	config.DefaultConfig.Port = strconv.Itoa(port)
	slog.Info("启动DBManager服务", "port", port)
	StartWebDBManager()
}

