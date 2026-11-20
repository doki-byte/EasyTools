package controller

import (
	"EasyTools/app/bypass/shell/bypassAv"
	"EasyTools/app/bypass/skyshadow"
	"EasyTools/app/controller/assistive"
	"EasyTools/app/controller/cli"
	"EasyTools/app/controller/connect"
	"EasyTools/app/controller/connect/dbmanager"
	"EasyTools/app/controller/connect/ftp"
	"EasyTools/app/controller/connect/ssh"
	"EasyTools/app/controller/infoCollection"
	"EasyTools/app/controller/localguard"
	"EasyTools/app/controller/system"
	"fmt"
	"net"
	"path/filepath"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Server 本地文件服务
type Server struct {
	system.Base
	address     string
	port        int
	staticDir   string
	PortConfig  *system.PortConfig // 添加端口配置
}

// NewServer 创建 Server 实例
func NewServer() *Server {
	return &Server{
		address: "127.0.0.1", // 默认监听本地地址
	}
}

func (s *Server) start(port int) *Server {
	baseDir := s.GetAppPath()
	s.staticDir = filepath.Join(baseDir, "icon")

	// 初始化进程级本地服务访问 Token（供 Wails 前端注入请求头）
	localguard.InitToken()

	// 如果传入端口为0，则使用动态分配
	if port == 0 {
		// 使用更大的端口范围避免冲突
		mainPort, err := system.PortManager.GetAvailablePortInRange(52000, 52999)
		if err != nil {
			// 如果分配失败，回退到默认端口
			mainPort = 52867
		}

		sshPort, _ := system.PortManager.GetAvailablePortInRange(53000, 53999)
		ftpPort, _ := system.PortManager.GetAvailablePortInRange(54000, 54999)
		fuzzPort, _ := system.PortManager.GetAvailablePortInRange(55000, 55999)
		dbManagerPort, _ := system.PortManager.GetAvailablePortInRange(56000, 56999)

		s.port = mainPort
		s.PortConfig = &system.PortConfig{
			MainPort:      mainPort,
			SSHPort:       sshPort,
			FTPPort:       ftpPort,
			FuzzPort:      fuzzPort,
			DBManagerPort: dbManagerPort,
		}
	} else {
		// 使用固定端口（向后兼容）
		s.port = port
		// 尝试动态分配，如果失败则使用固定偏移
		sshPort, _ := system.PortManager.GetAvailablePortInRange(53000, 53999)
		ftpPort, _ := system.PortManager.GetAvailablePortInRange(54000, 54999)
		fuzzPort, _ := system.PortManager.GetAvailablePortInRange(55000, 55999)
		dbManagerPort, _ := system.PortManager.GetAvailablePortInRange(56000, 56999)

		// 如果动态分配失败，使用固定偏移
		if sshPort == 0 {
			sshPort = port + 1
		}
		if ftpPort == 0 {
			ftpPort = port + 2
		}
		if fuzzPort == 0 {
			fuzzPort = port + 3
		}
		if dbManagerPort == 0 {
			dbManagerPort = port + 4
		}

		s.PortConfig = &system.PortConfig{
			MainPort:      port,
			SSHPort:       sshPort,
			FTPPort:       ftpPort,
			FuzzPort:      fuzzPort,
			DBManagerPort: dbManagerPort,
		}

		// 检查固定端口是否可用，如果不可用则使用动态分配
		if !s.isPortAvailable(port) {
			dynamicPort, err := system.PortManager.GetAvailablePortInRange(52000, 52999)
			if err == nil {
				s.port = dynamicPort
				s.PortConfig.MainPort = dynamicPort
			}
		}
	}

	// 原有的启动代码...
	go func() {
		gin.SetMode(gin.ReleaseMode)
		r := gin.Default()

		// 本地访问防护：校验 Host 与 Origin，阻断恶意网页 CSRF / DNS rebinding / 跨域调用
		r.Use(localguard.GinGuard(s.port))

		// 配置跨域支持（仅放行本地回环来源与 Wails 协议，不再允许任意来源；
		// Wails 协议页面 Origin 序列化为 "null" 时要求携带进程级 Token）
		r.Use(cors.New(cors.Config{
			AllowOrigins: []string{"http://127.0.0.1:" + fmt.Sprint(s.port), "http://localhost:" + fmt.Sprint(s.port), "http://[::1]:" + fmt.Sprint(s.port)},
			AllowOriginWithContextFunc: func(c *gin.Context, origin string) bool {
				if localguard.IsAllowedOrigin(origin) {
					return true
				}
				// null 等非白名单 Origin：必须携带有效 Token（防恶意网页 sandbox iframe CSRF）
				return localguard.ValidateRequestToken(c.Request)
			},
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Content-Type", "Authorization", "X-Requested-With", "X-Auth-Token"},
			ExposeHeaders:    []string{"Content-Length", "Authorization", "Content-Type"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}))

		// 添加静态文件服务
		r.Static("/icon", s.staticDir)
		cyberDir := filepath.Join(baseDir, "tools", "CyberChef")
		r.Static("/CyberChef", cyberDir)

		// 添加端口信息接口
		r.GET("/api/port-config", func(c *gin.Context) {
			c.JSON(200, s.PortConfig)
		})

		// 配置管理路由
		{
			// NoSQL命令管理
			r.GET("/api/config/nosql-commands", connect.GetNoSQLCommands)
			r.POST("/api/config/nosql-commands", connect.CreateNoSQLCommand)
			r.PUT("/api/config/nosql-commands/:id", connect.UpdateNoSQLCommand)
			r.DELETE("/api/config/nosql-commands/:id", connect.DeleteNoSQLCommand)
			r.POST("/api/config/nosql-commands/reset", connect.ResetNoSQLCommands)

			// SQL模板管理
			r.GET("/api/config/sql-templates", connect.GetSQLTemplates)
			r.POST("/api/config/sql-templates", connect.CreateSQLTemplate)
			r.PUT("/api/config/sql-templates/:id", connect.UpdateSQLTemplate)
			r.DELETE("/api/config/sql-templates/:id", connect.DeleteSQLTemplate)
			r.POST("/api/config/sql-templates/reset", connect.ResetSQLTemplates)

			// SQL模板分类管理
			r.GET("/api/config/sql-template-categories", connect.GetSQLTemplateCategories)
			r.POST("/api/config/sql-template-categories", connect.CreateSQLTemplateCategory)
			r.PUT("/api/config/sql-template-categories/:id", connect.UpdateSQLTemplateCategory)
			r.DELETE("/api/config/sql-template-categories/:id", connect.DeleteSQLTemplateCategory)
		}

		// 信息收集模块路由
		{
			spaceMapping := infoCollection.NewSpaceMapping()
			r.GET("/api/info-collection/config", spaceMapping.GetConfig)
			r.POST("/api/info-collection/config", spaceMapping.SaveConfig)
			r.POST("/api/info-collection/space-mapping/query", spaceMapping.Query)
			r.POST("/api/info-collection/space-mapping/cancel", spaceMapping.CancelQuery)
			r.POST("/api/info-collection/space-mapping/verify-key", spaceMapping.VerifyKey)
		}

		// 辅助工具模块路由
		{
			assistive := assistive.NewAssistive()
			r.GET("/api/assistive/baidu-map/proxy", assistive.ProxyBaiduMap)
			r.GET("/api/assistive/google-map/proxy", assistive.ProxyGoogleMap)
		}

		// Bypass C2 免杀模块路由（Mac/Linux/Windows 均可使用）
		r.GET("/modelist", bypassAv.ModeLists)
		r.GET("/platformlist", bypassAv.PlatformList)
		r.GET("/avlist", bypassAv.AvList)
		r.GET("/encodewaylist", bypassAv.EncodeWayList)
		r.GET("/bypass/iconlist", bypassAv.IconList)
		r.GET("/bypass/icon/:name", bypassAv.IconFile)
		r.GET("/initCheck", bypassAv.InitCheckHandler)
		r.POST("/upload", bypassAv.UploadHandler)
		r.POST("/make", bypassAv.RunMakeFile)
		r.POST("/changefilesize", bypassAv.ChangeFileSize)
		r.GET("/opendir", bypassAv.OpenDir)
		r.POST("/binder", bypassAv.RunBinder)
		r.GET("/passqvm", bypassAv.PassQvm)
		r.POST("/passqvm", bypassAv.PassQvm)
		r.GET("/removeupx", bypassAv.RemoveUpx)
		r.GET("/addSignthief", bypassAv.AddSignthief)
		r.POST("/dllproxy", bypassAv.RunDllProxyGenerate)

		// SkyShadow DLL 劫持 Payload 生成模块路由
		r.POST("/skyshadow/scan", skyshadow.RunScanHandler)
		r.POST("/skyshadow/cancel", skyshadow.CancelScanHandler)
		r.GET("/skyshadow/logs", skyshadow.LogsHandler)
		r.GET("/skyshadow/payloads", skyshadow.PayloadsHandler)
		r.POST("/skyshadow/generateTestDll", skyshadow.GenerateTestDllHandler)
		r.POST("/skyshadow/generateTestDlls", skyshadow.GenerateTestDllsHandler)

		// 启动其他服务，使用动态端口
		go ssh.StartWebSSHWithPort(s.PortConfig.SSHPort)
		go ftp.StartWebFTPWithPort(s.PortConfig.FTPPort)
		go cli.StartCLIExecutor(s.PortConfig.FuzzPort)
		if s.PortConfig.DBManagerPort > 0 {
			fmt.Printf("准备启动DBManager服务，端口: %d\n", s.PortConfig.DBManagerPort)
			go func() {
				// 添加小延迟确保端口已释放
				time.Sleep(100 * time.Millisecond)
				dbmanager.StartWebDBManagerWithPort(s.PortConfig.DBManagerPort)
			}()
		} else {
			fmt.Printf("警告: DBManager端口未分配（端口值: %d），跳过启动\n", s.PortConfig.DBManagerPort)
		}

		// 输出端口配置信息用于调试
		fmt.Printf("端口配置 - MainPort: %d, SSHPort: %d, FTPPort: %d, FuzzPort: %d, DBManagerPort: %d\n",
			s.PortConfig.MainPort, s.PortConfig.SSHPort, s.PortConfig.FTPPort, s.PortConfig.FuzzPort, s.PortConfig.DBManagerPort)

		// 启动服务
		err := r.Run(fmt.Sprintf("%s:%d", s.address, s.port))
		if err != nil {
			panic(fmt.Sprintf("Failed to start server: %v", err))
		}
	}()

	return s
}

// 检查端口是否可用
func (s *Server) isPortAvailable(port int) bool {
	timeout := time.Second
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", port)), timeout)
	if err != nil {
		return true
	}
	if conn != nil {
		conn.Close()
		return false
	}
	return true
}

// GetPortConfig 提供给前端调用的方法
func (s *Server) GetPortConfig() *system.PortConfig {
	return s.PortConfig
}

// GetLocalServiceToken 返回本进程的本地服务访问令牌（Wails 绑定，供前端注入请求头）。
// 本地防护在 Origin 不在白名单（如 Wails 协议下为 "null"）时要求携带该令牌。
func (s *Server) GetLocalServiceToken() string {
	return localguard.GetToken()
}

// GetCLIMCPEnabled 主进程 FuzzPort 上 /cli/mcp/* 是否允许外部调用（默认 true）
func (s *Server) GetCLIMCPEnabled() bool {
	return cli.IsMCPEndpointEnabled()
}

// SetCLIMCPEnabled 开启或关闭上述 MCP HTTP 接口；stdio 子进程 `EasyTools mcp --stdio` 不受影响
func (s *Server) SetCLIMCPEnabled(enabled bool) error {
	return cli.SetMCPEndpointEnabled(enabled)
}
