package ssh

import (
	"EasyTools/app/controller/connect/ssh/app/config"
	middleware2 "EasyTools/app/controller/connect/ssh/app/middleware"
	model2 "EasyTools/app/controller/connect/ssh/app/model"
	service2 "EasyTools/app/controller/connect/ssh/app/service"
	gin2 "EasyTools/app/controller/connect/ssh/gin"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
)

func StartWebSSH() {
	// 在启动时主动初始化数据库，确保表已创建
	if err := model2.DbMigrate(config.DefaultConfig.DbType, config.DefaultConfig.DbDsn); err != nil {
		slog.Error("数据库初始化失败", "err", err)
		// 不退出，让中间件处理后续的数据库检查
	}

	gin2.SetMode(gin2.ReleaseMode)
	var engine = gin2.Default()
	engine.Use(middleware2.Cors(), middleware2.DbCheck())

	engine.NoRoute(func(c *gin2.Context) {
		c.Redirect(http.StatusMovedPermanently, "/app")
	})

	// 不需要认证的路由
	engine.Group("")

	// 需要认证的路由
	var auth = engine.Group("", middleware2.JWTAuth())

	{ // SSH 连接配置
		auth.GET("/api/conn_conf", service2.ConfFindAll)
		auth.GET("/api/conn_conf/:id", service2.ConfFindByID)
		auth.POST("/api/conn_conf", service2.ConfCreate)
		auth.PUT("/api/conn_conf", service2.ConfUpdateById)
		auth.DELETE("/api/conn_conf/:id", service2.ConfDeleteById)
	}

	{ // 命令收藏
		auth.GET("/api/cmd_note", service2.CmdNoteFindAll)
		auth.GET("/api/cmd_note/:id", service2.CmdNoteFindByID)
		auth.POST("/api/cmd_note", service2.CmdNoteCreate)
		auth.PUT("/api/cmd_note", service2.CmdNoteUpdateById)
		auth.DELETE("/api/cmd_note/:id", service2.CmdNoteDeleteById)
	}

	{ // SSH链接
		auth.GET("/api/conn_manage/online_client", service2.GetOnlineClient)
		auth.PUT("/api/conn_manage/refresh_conn_time", service2.RefreshConnTime)
		auth.POST("/api/sftp/create_dir", service2.SftpCreateDir)
		auth.POST("/api/sftp/list", service2.SftpList)
		auth.GET("/api/sftp/download", service2.SftpDownLoad)
		auth.GET("/api/sftp/download_dir", service2.SftpDownLoadDir)
		auth.PUT("/api/sftp/upload", service2.SftpUpload)
		auth.GET("/api/sftp/upload/progress", service2.SftpUploadProgress)
		auth.GET("/api/sftp/upload/cancel", service2.SftpUploadCancel)
		auth.DELETE("/api/sftp/delete", service2.SftpDelete)
		auth.GET("/api/ssh/conn", service2.NewSshConn)
		auth.PATCH("/api/ssh/conn", service2.ResizeWindow)
		auth.POST("/api/ssh/exec", service2.ExecCommand)
		auth.POST("/api/ssh/disconnect", service2.Disconnect)
		auth.POST("/api/ssh/create_session", service2.CreateSessionId)
	}

	address := fmt.Sprintf("%s:%s", config.DefaultConfig.Address, config.DefaultConfig.Port)
	_, certErr := os.Open(config.DefaultConfig.CertFile)
	_, keyErr := os.Open(config.DefaultConfig.KeyFile)

	// 如果证书和私钥文件存在,就使用https协议,否则使用http协议
	if certErr == nil && keyErr == nil {
		slog.Debug("https_server_start")
		err := engine.RunTLS(address, config.DefaultConfig.CertFile, config.DefaultConfig.KeyFile)
		if err != nil {
			slog.Error("RunServeTLSError:", "msg", err.Error())
			os.Exit(1)
			return
		}
	} else {
		slog.Debug("http_server_start")
		err := engine.Run(address)
		if err != nil {
			slog.Error("RunServeError:", "msg", err.Error())
			os.Exit(1)
			return
		}
	}
}

// StartWebSSHWithPort 使用指定端口启动SSH服务
func StartWebSSHWithPort(port int) {
	// 修改配置中的端口
	config.DefaultConfig.Port = strconv.Itoa(port)
	StartWebSSH()
}
