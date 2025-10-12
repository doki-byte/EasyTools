package controller

import (
	"EasyTools/app/connect/ftp"
	"EasyTools/app/connect/ssh"
	"fmt"
	"path/filepath"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Server 本地文件服务
type Server struct {
	Base
	address   string
	port      int
	staticDir string
}

// 日志条目结构体
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	FilePath  string `json:"file_path"`
	Message   string `json:"message"`
}

// NewServer 创建 Server 实例
func NewServer() *Server {
	return &Server{
		address: "127.0.0.1", // 默认监听本地地址
	}
}

// start 启动文件服务
func (s *Server) start(port int) *Server {

	baseDir := s.getAppPath()
	s.staticDir = filepath.Join(baseDir, "icon")
	s.port = port

	go func() {
		gin.SetMode(gin.ReleaseMode)
		r := gin.Default()

		// 配置跨域支持
		r.Use(cors.New(cors.Config{
			AllowAllOrigins:  true,
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"*"},
			ExposeHeaders:    []string{"Content-Length", "Authorization", "Content-Type"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}))

		// 添加静态文件服务
		r.Static("/icon", s.staticDir)
		cyberDir := filepath.Join(baseDir, "tools", "CyberChef")
		r.Static("/CyberChef", cyberDir)

		// 启动ssh服务
		go ssh.StartWebSSH() // 52868
		// 启动ftp服务
		go ftp.StartWebFTP() // 52869
		// 启动fuzz服务
		go NewFuzzer() // 52870

		// 启动服务
		err := r.Run(fmt.Sprintf("%s:%d", s.address, s.port))
		if err != nil {
			panic(fmt.Sprintf("Failed to start server: %v", err))
		}
	}()

	return s
}
