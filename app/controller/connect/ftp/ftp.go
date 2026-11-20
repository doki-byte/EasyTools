package ftp

import (
	"archive/zip"
	"crypto/md5"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"EasyTools/app/controller/connect/ftp/app/config"
	middleware2 "EasyTools/app/controller/connect/ftp/app/middleware"
	model2 "EasyTools/app/controller/connect/ftp/app/model"
	service2 "EasyTools/app/controller/connect/ftp/app/service"

	"github.com/gin-gonic/gin"
	"github.com/jlaffaye/ftp"
	"golang.org/x/crypto/ssh"
)

var (
	downloadTokens = make(map[string]FTPConfig)
	tokenMutex     sync.RWMutex

	// FTP 上传进度，全局按 uploadId 存储
	ftpUploadProgress   = make(map[string]*UploadProgress)
	ftpUploadProgressMu sync.RWMutex
)

// UploadProgress 表示一次上传的进度
type UploadProgress struct {
	Total    int64  `json:"total"`
	Uploaded int64  `json:"uploaded"`
	Done     bool   `json:"done"`
	Error    string `json:"error"`
}

// progressReader 用于在复制数据时统计已读字节数并写入进度表
type progressReader struct {
	r        io.Reader
	id       string
	total    int64
	uploaded int64
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		ftpUploadProgressMu.Lock()
		defer ftpUploadProgressMu.Unlock()
		p.uploaded += int64(n)
		if prog, ok := ftpUploadProgress[p.id]; ok {
			prog.Uploaded = p.uploaded
			// 防止超过 total
			if prog.Total > 0 && prog.Uploaded > prog.Total {
				prog.Uploaded = prog.Total
			}
		}
	}
	return n, err
}

type FTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	Path     string
	Created  time.Time
	// SSH 隧道配置
	SSHAddr     string
	SSHPort     int
	SSHUsername string
	SSHPassword string
}

// 清理过期的令牌
func cleanupTokens() {
	tokenMutex.Lock()
	defer tokenMutex.Unlock()

	now := time.Now()
	for token, config := range downloadTokens {
		if now.Sub(config.Created) > 10*time.Minute { // 10分钟过期
			delete(downloadTokens, token)
		}
	}
}

// ftpSSHDialer 实现自定义dialer以支持SSH隧道
type ftpSSHDialer struct {
	sshClient *ssh.Client
}

func (d *ftpSSHDialer) Dial(network, addr string) (net.Conn, error) {
	return d.sshClient.Dial(network, addr)
}

func connectFTP(host string, port int, username, password string, sshAddr string, sshPort int, sshUsername, sshPassword string) (*ftp.ServerConn, error) {
	address := fmt.Sprintf("%s:%d", host, port)

	// 如果配置了SSH隧道，通过SSH隧道连接
	if sshAddr != "" && sshPort > 0 && sshUsername != "" {
		sshClient, err := getSSHClient(sshUsername, sshPassword, sshAddr, sshPort)
		if err != nil {
			return nil, fmt.Errorf("SSH连接失败: %v", err)
		}
		// 注意：这里不能defer close，因为连接需要保持打开

		// 通过SSH隧道连接到FTP服务器
		// 由于jlaffaye/ftp库不支持自定义dialer，我们需要使用端口转发
		// 创建一个本地监听端口进行转发
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			sshClient.Close()
			return nil, fmt.Errorf("创建本地监听端口失败: %v", err)
		}

		localAddr := listener.Addr().String()

		// 启动goroutine进行端口转发
		go func() {
			defer listener.Close()
			for {
				localConn, err := listener.Accept()
				if err != nil {
					return
				}

				// 通过SSH隧道连接到远程FTP服务器
				remoteConn, err := sshClient.Dial("tcp", address)
				if err != nil {
					localConn.Close()
					continue
				}

				// 进行双向数据转发
				go func() {
					defer localConn.Close()
					defer remoteConn.Close()
					io.Copy(localConn, remoteConn)
				}()
				go func() {
					defer localConn.Close()
					defer remoteConn.Close()
					io.Copy(remoteConn, localConn)
				}()
			}
		}()

		// 连接到本地端口（通过SSH隧道转发）
		conn, err := ftp.Dial(localAddr, ftp.DialWithTimeout(5*time.Second))
		if err != nil {
			listener.Close()
			sshClient.Close()
			return nil, fmt.Errorf("通过SSH隧道连接FTP失败: %v", err)
		}

		// 保存SSH客户端引用，以便在连接关闭时清理
		// 注意：这里我们需要确保SSH连接在FTP连接关闭时也关闭
		// 但由于FTP连接的生命周期管理，我们暂时不在这里关闭SSH连接
		// 在实际使用中，可能需要更复杂的连接管理

		if err := conn.Login(username, password); err != nil {
			conn.Quit()
			listener.Close()
			sshClient.Close()
			return nil, err
		}

		// 注意：这里listener和sshClient需要在适当的时候关闭
		// 但由于FTP连接的生命周期，我们暂时保持它们打开
		// 在实际生产环境中，需要实现更完善的资源管理

		return conn, nil
	}

	// 直接连接
	conn, err := ftp.Dial(address, ftp.DialWithTimeout(5*time.Second))
	if err != nil {
		return nil, err
	}
	if err := conn.Login(username, password); err != nil {
		return nil, err
	}
	return conn, nil
}

// getSSHClient 创建SSH客户端连接
func getSSHClient(username, password, addr string, port int) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}

	sshAddr := fmt.Sprintf("%s:%d", addr, port)
	client, err := ssh.Dial("tcp", sshAddr, config)
	if err != nil {
		return nil, err
	}

	return client, nil
}

// StartWebFTPWithPort 使用指定端口启动FTP服务
func StartWebFTPWithPort(port int) {
	startFTP(port)
}

func StartWebFTP() {
	startFTP(52869)
}

func startFTP(port int) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("FTP服务启动panic", "err", r)
		}
	}()

	// 在启动时主动初始化数据库，确保表已创建
	config2 := config.DefaultConfig
	if err := model2.DbMigrate(config2.DbType, config2.DbDsn); err != nil {
		slog.Error("数据库初始化失败", "err", err)
		// 不退出，让中间件处理后续的数据库检查
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(middleware2.Cors(port), middleware2.DbCheck())

	// 配置管理
	{
		r.GET("/api/config", service2.ConfigFindAll)
		r.GET("/api/config/:id", service2.ConfigFindByID)
		r.POST("/api/config", service2.ConfigCreate)
		r.PUT("/api/config", service2.ConfigUpdateById)
		r.DELETE("/api/config/:id", service2.ConfigDeleteById)
	}

	// 测试FTP连接
	r.POST("/api/ftp/connect", func(c *gin.Context) {
		var req struct {
			Host        string `json:"host"`
			Port        int    `json:"port"`
			Username    string `json:"username"`
			Password    string `json:"password"`
			SSHAddr     string `json:"ssh_addr"`
			SSHPort     int    `json:"ssh_port"`
			SSHUsername string `json:"ssh_username"`
			SSHPassword string `json:"ssh_password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		conn, err := connectFTP(req.Host, req.Port, req.Username, req.Password, req.SSHAddr, req.SSHPort, req.SSHUsername, req.SSHPassword)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer conn.Quit()
		c.JSON(http.StatusOK, gin.H{"message": "Connected successfully"})
	})

	// 上传文件 (流式处理)
	r.POST("/api/ftp/upload", func(c *gin.Context) {
		// 获取表单数据
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 打开文件流
		src, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer src.Close()

		// 连接FTP
		host := c.PostForm("host")
		port, _ := strconv.Atoi(c.PostForm("port"))
		username := c.PostForm("username")
		password := c.PostForm("password")
		remotePath := c.PostForm("remotePath")
		sshAddr := c.PostForm("ssh_addr")
		sshPort, _ := strconv.Atoi(c.PostForm("ssh_port"))
		sshUsername := c.PostForm("ssh_username")
		sshPassword := c.PostForm("ssh_password")

		// 上传进度 ID，由前端传入；如果为空则不记录进度
		uploadID := c.PostForm("uploadId")
		var pr io.Reader = src
		if uploadID != "" {
			// 初始化进度
			ftpUploadProgressMu.Lock()
			ftpUploadProgress[uploadID] = &UploadProgress{
				Total:    file.Size,
				Uploaded: 0,
				Done:     false,
				Error:    "",
			}
			ftpUploadProgressMu.Unlock()
			pr = &progressReader{
				r:     src,
				id:    uploadID,
				total: file.Size,
			}
		}

		conn, err := connectFTP(host, port, username, password, sshAddr, sshPort, sshUsername, sshPassword)
		if err != nil {
			if uploadID != "" {
				ftpUploadProgressMu.Lock()
				if prog, ok := ftpUploadProgress[uploadID]; ok {
					prog.Error = err.Error()
					prog.Done = true
				}
				ftpUploadProgressMu.Unlock()
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer conn.Quit()

		// 创建管道进行流式传输
		pipeReader, pw := io.Pipe()
		defer pipeReader.Close()

		// 启动 goroutine 负责把上传内容写入管道
		go func() {
			defer pw.Close()
			_, err := io.Copy(pw, pr)
			if err != nil {
				_ = pw.CloseWithError(err)
			}
		}()

		// 流式上传到 FTP
		err = conn.Stor(remotePath, pipeReader)
		if err != nil {
			if uploadID != "" {
				ftpUploadProgressMu.Lock()
				if prog, ok := ftpUploadProgress[uploadID]; ok {
					prog.Error = err.Error()
					prog.Done = true
				}
				ftpUploadProgressMu.Unlock()
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if uploadID != "" {
			ftpUploadProgressMu.Lock()
			if prog, ok := ftpUploadProgress[uploadID]; ok {
				prog.Uploaded = prog.Total
				prog.Done = true
			}
			ftpUploadProgressMu.Unlock()
		}

		c.JSON(http.StatusOK, gin.H{"message": "文件上传成功"})
	})

	// 下载文件 (流式处理)
	r.POST("/api/ftp/download", func(c *gin.Context) {
		var req struct {
			Host        string `json:"host"`
			Port        int    `json:"port"`
			Username    string `json:"username"`
			Password    string `json:"password"`
			Path        string `json:"path"`
			SSHAddr     string `json:"ssh_addr"`
			SSHPort     int    `json:"ssh_port"`
			SSHUsername string `json:"ssh_username"`
			SSHPassword string `json:"ssh_password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 连接FTP
		conn, err := connectFTP(req.Host, req.Port, req.Username, req.Password, req.SSHAddr, req.SSHPort, req.SSHUsername, req.SSHPassword)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer conn.Quit()

		// 获取文件大小
		size, err := conn.FileSize(req.Path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取文件大小失败: " + err.Error()})
			return
		}

		// 流式下载
		resp, err := conn.Retr(req.Path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer resp.Close()

		// 设置响应头
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(req.Path)))
		c.Header("Content-Type", "application/octet-stream")
		c.Header("Content-Length", fmt.Sprintf("%d", size))
		c.Status(http.StatusOK)

		// 使用CopyN限制传输速率 (可选)
		buf := make([]byte, 32*1024) // 32KB缓冲区
		_, err = io.CopyBuffer(c.Writer, io.LimitReader(resp, size), buf)
		if err != nil && err != io.EOF {
			fmt.Println("下载中断:", err)
		}
	})

	// 查询 FTP 上传进度
	r.GET("/api/ftp/upload/progress", func(c *gin.Context) {
		id := c.Query("id")
		if id == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing id"})
			return
		}

		ftpUploadProgressMu.RLock()
		prog, ok := ftpUploadProgress[id]
		ftpUploadProgressMu.RUnlock()
		if !ok {
			c.JSON(http.StatusOK, gin.H{
				"exists": false,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"exists":  true,
			"total":   prog.Total,
			"uploaded": prog.Uploaded,
			"done":    prog.Done,
			"error":   prog.Error,
		})

		// 如果已经完成或失败，可以选择性清理
		if prog.Done {
			ftpUploadProgressMu.Lock()
			delete(ftpUploadProgress, id)
			ftpUploadProgressMu.Unlock()
		}
	})

	// 生成下载令牌
	r.POST("/api/ftp/generate-download-url", func(c *gin.Context) {
		var req struct {
			Host        string `json:"host"`
			Port        int    `json:"port"`
			Username    string `json:"username"`
			Password    string `json:"password"`
			Path        string `json:"path"`
			SSHAddr     string `json:"ssh_addr"`
			SSHPort     int    `json:"ssh_port"`
			SSHUsername string `json:"ssh_username"`
			SSHPassword string `json:"ssh_password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 验证FTP连接
		conn, err := connectFTP(req.Host, req.Port, req.Username, req.Password, req.SSHAddr, req.SSHPort, req.SSHUsername, req.SSHPassword)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer conn.Quit()

		// 验证文件存在
		_, err = conn.FileSize(req.Path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "文件不存在: " + err.Error()})
			return
		}

		// 生成唯一令牌
		token := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s%d%s%s%s%d",
			req.Host, req.Port, req.Username, req.Password, req.Path, time.Now().UnixNano()))))

		// 存储令牌信息
		tokenMutex.Lock()
		downloadTokens[token] = FTPConfig{
			Host:        req.Host,
			Port:        req.Port,
			Username:    req.Username,
			Password:    req.Password,
			Path:        req.Path,
			Created:     time.Now(),
			SSHAddr:     req.SSHAddr,
			SSHPort:     req.SSHPort,
			SSHUsername: req.SSHUsername,
			SSHPassword: req.SSHPassword,
		}
		tokenMutex.Unlock()

		// 定期清理令牌
		go cleanupTokens()
		downloadURL := fmt.Sprintf("http://127.0.0.1:%d/api/ftp/direct-download?token=%s", port, token)

		c.JSON(http.StatusOK, gin.H{"downloadUrl": downloadURL})
	})

	// 直接下载接口
	r.GET("/api/ftp/direct-download", func(c *gin.Context) {
		token := c.Query("token")
		if token == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "缺少下载令牌"})
			return
		}

		// 获取令牌对应的配置
		tokenMutex.RLock()
		config, exists := downloadTokens[token]
		tokenMutex.RUnlock()

		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "下载令牌无效或已过期"})
			return
		}

		// 连接FTP
		conn, err := connectFTP(config.Host, config.Port, config.Username, config.Password, config.SSHAddr, config.SSHPort, config.SSHUsername, config.SSHPassword)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer conn.Quit()

		// 获取文件信息
		size, err := conn.FileSize(config.Path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "获取文件大小失败: " + err.Error()})
			return
		}

		// 开始下载
		resp, err := conn.Retr(config.Path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer resp.Close()

		// 设置响应头
		filename := filepath.Base(config.Path)
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
		c.Header("Content-Type", "application/octet-stream")
		c.Header("Content-Length", fmt.Sprintf("%d", size))
		c.Status(http.StatusOK)

		// 流式传输
		buf := make([]byte, 32*1024)
		_, err = io.CopyBuffer(c.Writer, io.LimitReader(resp, size), buf)
		if err != nil && err != io.EOF {
			fmt.Println("下载中断:", err)
		}

		// 下载完成后删除令牌（可选）
		tokenMutex.Lock()
		delete(downloadTokens, token)
		tokenMutex.Unlock()
	})

	// 删除文件或目录（支持递归删除文件夹）
	r.POST("/api/ftp/delete", func(c *gin.Context) {
		var req struct {
			Host        string `json:"host"`
			Port        int    `json:"port"`
			Username    string `json:"username"`
			Password    string `json:"password"`
			Path        string `json:"path"`
			SSHAddr     string `json:"ssh_addr"`
			SSHPort     int    `json:"ssh_port"`
			SSHUsername string `json:"ssh_username"`
			SSHPassword string `json:"ssh_password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		conn, err := connectFTP(req.Host, req.Port, req.Username, req.Password, req.SSHAddr, req.SSHPort, req.SSHUsername, req.SSHPassword)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer conn.Quit()

		// 规范化路径
		normalizedPath := strings.TrimPrefix(req.Path, "/")
		if normalizedPath == "" {
			normalizedPath = "/"
		} else {
			normalizedPath = "/" + normalizedPath
		}

		// 判断是文件还是目录
		parentDir := path.Dir(normalizedPath)
		if parentDir == "." {
			parentDir = "/"
		}
		baseName := path.Base(normalizedPath)
		entries, listErr := conn.List(parentDir)
		if listErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "无法获取父目录列表: " + listErr.Error()})
			return
		}
		isDir := false
		for _, e := range entries {
			if e.Name == baseName && e.Type == ftp.EntryTypeFolder {
				isDir = true
				break
			}
		}

		var opErr error
		if isDir {
			// 递归删除目录
			opErr = deleteFtpDirRecursive(conn, normalizedPath)
		} else {
			opErr = conn.Delete(normalizedPath)
		}
		if opErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("删除失败: %v", opErr)})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
	})

	// 下载文件夹（打包为 zip）
	r.GET("/api/ftp/download_dir", func(c *gin.Context) {
		// 从查询参数获取配置
		host := c.Query("host")
		portStr := c.Query("port")
		username := c.Query("username")
		password := c.Query("password")
		dirPathRaw := c.Query("path")
		sshAddr := c.Query("ssh_addr")
		sshPortStr := c.Query("ssh_port")
		sshUsername := c.Query("ssh_username")
		sshPassword := c.Query("ssh_password")

		if dirPathRaw == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "目录路径不能为空"})
			return
		}

		dirPath, err := url.QueryUnescape(dirPathRaw)
		if err != nil {
			slog.Error("获取目录路径参数错误", "err", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "获取目录路径参数错误"})
			return
		}

		port, err := strconv.Atoi(portStr)
		if err != nil {
			port = 21
		}

		sshPort := 0
		if sshPortStr != "" {
			sshPort, _ = strconv.Atoi(sshPortStr)
		}

		// 连接 FTP
		conn, err := connectFTP(host, port, username, password, sshAddr, sshPort, sshUsername, sshPassword)
		if err != nil {
			slog.Error("连接 FTP 失败", "err", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "连接 FTP 失败: " + err.Error()})
			return
		}
		defer conn.Quit()

		// 规范化路径
		normalizedPath := strings.TrimPrefix(dirPath, "/")
		if normalizedPath == "" {
			normalizedPath = "/"
		} else {
			normalizedPath = "/" + normalizedPath
		}

		// 验证目录是否存在（仅检查是否能成功列出，不使用返回值）
		_, err = conn.List(normalizedPath)
		if err != nil {
			slog.Error("读取目录失败", "path", normalizedPath, "err", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "读取目录失败: " + err.Error()})
			return
		}

		// 生成 zip 文件名
		baseName := path.Base(normalizedPath)
		if baseName == "" || baseName == "/" || baseName == "." {
			baseName = "folder"
		}
		zipName := fmt.Sprintf("%s_%d.zip", baseName, time.Now().Unix())

		// 设置响应头
		c.Writer.WriteHeader(http.StatusOK)
		c.Header("Content-Disposition", "attachment; filename="+url.PathEscape(zipName))
		c.Header("Content-Type", "application/zip")

		// 创建 zip writer
		zipWriter := zip.NewWriter(c.Writer)
		defer func() {
			if err := zipWriter.Close(); err != nil {
				slog.Error("关闭 zipWriter 失败", "err", err)
			}
		}()

		// 递归遍历目录并写入 zip
		err = walkFtpDirAndZip(conn, normalizedPath, "", zipWriter)
		if err != nil {
			slog.Error("打包目录失败", "path", normalizedPath, "err", err)
			return
		}

		// 刷新缓冲
		if flusher, ok := c.Writer.(http.Flusher); ok {
			flusher.Flush()
		}
	})

	// 列表接口（带分页），错误时也返回空数组
	r.POST("/api/ftp/list", func(c *gin.Context) {
		var req struct {
			Host        string `json:"host"`
			Port        int    `json:"port"`
			Username    string `json:"username"`
			Password    string `json:"password"`
			Path        string `json:"path"`
			Page        int    `json:"page"`
			PageSize    int    `json:"pageSize"`
			SSHAddr     string `json:"ssh_addr"`
			SSHPort     int    `json:"ssh_port"`
			SSHUsername string `json:"ssh_username"`
			SSHPassword string `json:"ssh_password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// 默认分页
		if req.Page <= 0 {
			req.Page = 1
		}
		if req.PageSize <= 0 {
			req.PageSize = 100
		} else if req.PageSize > 500 {
			req.PageSize = 500
		}

		conn, err := connectFTP(req.Host, req.Port, req.Username, req.Password, req.SSHAddr, req.SSHPort, req.SSHUsername, req.SSHPassword)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer conn.Quit()

		entries, err := conn.List(req.Path)
		if err != nil {
			// 返回空列表和分页
			c.JSON(http.StatusOK, gin.H{
				"files": []map[string]interface{}{},
				"pagination": map[string]interface{}{
					"total":     0,
					"page":      req.Page,
					"pageSize":  req.PageSize,
					"pageCount": 0,
				},
			})
			return
		}

		total := len(entries)
		start := (req.Page - 1) * req.PageSize
		if start > total {
			start = total
		}
		end := start + req.PageSize
		if end > total {
			end = total
		}
		sliced := entries[start:end]

		files := make([]map[string]interface{}, 0, len(sliced))
		for _, e := range sliced {
			files = append(files, map[string]interface{}{
				"name": e.Name,
				"type": e.Type.String(),
				"size": e.Size,
				"time": e.Time.Format("2006-01-02 15:04:05"),
			})
		}

		c.JSON(http.StatusOK, gin.H{
			"files": files,
			"pagination": map[string]interface{}{
				"total":     total,
				"page":      req.Page,
				"pageSize":  req.PageSize,
				"pageCount": (total + req.PageSize - 1) / req.PageSize,
			},
		})
	})

	// 创建目录
	r.POST("/api/ftp/mkdir", func(c *gin.Context) {
		var req struct {
			Host        string `json:"host"`
			Port        int    `json:"port"`
			Username    string `json:"username"`
			Password    string `json:"password"`
			Path        string `json:"path"`       // 目标父目录
			FolderName  string `json:"folderName"` // 新文件夹名
			SSHAddr     string `json:"ssh_addr"`
			SSHPort     int    `json:"ssh_port"`
			SSHUsername string `json:"ssh_username"`
			SSHPassword string `json:"ssh_password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		conn, err := connectFTP(req.Host, req.Port, req.Username, req.Password, req.SSHAddr, req.SSHPort, req.SSHUsername, req.SSHPassword)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer conn.Quit()

		fullPath := filepath.Join(req.Path, req.FolderName)
		if err := conn.MakeDir(fullPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "目录创建成功"})
	})

	address := fmt.Sprintf("%s:%d", config.DefaultConfig.Address, port)
	slog.Info("FTP服务启动", "address", address)
	fmt.Printf("FTP服务正在启动，监听地址: %s\n", address)
	err := r.Run(address)
	if err != nil {
		slog.Error("FTP服务启动失败", "msg", err.Error())
		fmt.Printf("FTP服务启动失败: %v\n", err)
		return
	}
}

// 安全的路径清理
func safePath(path string) string {
	return filepath.Clean("/" + path)[1:]
}

// deleteFtpDirRecursive 递归删除 FTP 目录及其所有内容
func deleteFtpDirRecursive(conn *ftp.ServerConn, dirPath string) error {
	// 规范化路径
	normalizedPath := strings.TrimPrefix(dirPath, "/")
	if normalizedPath == "" {
		normalizedPath = "/"
	} else {
		normalizedPath = "/" + normalizedPath
	}

	// 列出目录内容
	entries, err := conn.List(normalizedPath)
	if err != nil {
		return fmt.Errorf("列出目录失败: %w", err)
	}

	// 先删除目录内的所有文件和子目录
	for _, entry := range entries {
		if entry.Name == "." || entry.Name == ".." {
			continue
		}

		childPath := path.Join(normalizedPath, entry.Name)
		if entry.Type == ftp.EntryTypeFolder {
			// 递归删除子目录
			if err := deleteFtpDirRecursive(conn, childPath); err != nil {
				return err
			}
		} else {
			// 删除文件
			if err := conn.Delete(childPath); err != nil {
				return fmt.Errorf("删除文件失败 %s: %w", childPath, err)
			}
		}
	}

	// 删除空目录
	if err := conn.RemoveDir(normalizedPath); err != nil {
		return fmt.Errorf("删除目录失败: %w", err)
	}

	return nil
}

// walkFtpDirAndZip 递归遍历 FTP 目录，并写入到 zipWriter 中
// basePath: 远程根目录绝对路径
// relPath:  当前遍历的相对路径前缀（用于 zip 内部路径）
func walkFtpDirAndZip(conn *ftp.ServerConn, basePath, relPath string, zw *zip.Writer) error {
	// 列出目录内容
	entries, err := conn.List(basePath)
	if err != nil {
		return fmt.Errorf("读取目录失败: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name
		if name == "." || name == ".." {
			continue
		}

		childRemotePath := path.Join(basePath, name)
		childRelPath := name
		if relPath != "" {
			childRelPath = path.Join(relPath, name)
		}

		if entry.Type == ftp.EntryTypeFolder {
			// 目录：在 zip 中创建一个目录条目（以 / 结尾），然后递归
			header := &zip.FileHeader{
				Name:   childRelPath + "/",
				Method: zip.Deflate,
			}
			header.SetMode(0o755 | os.ModeDir)
			if _, err := zw.CreateHeader(header); err != nil {
				return fmt.Errorf("创建目录 zip 条目失败: %w", err)
			}

			if err := walkFtpDirAndZip(conn, childRemotePath, childRelPath, zw); err != nil {
				return err
			}
		} else {
			// 文件：从 FTP 读取并写入 zip
			resp, err := conn.Retr(childRemotePath)
			if err != nil {
				slog.Error("打开远程文件失败", "path", childRemotePath, "err", err)
				continue
			}

			header := &zip.FileHeader{
				Name:   childRelPath,
				Method: zip.Deflate,
			}
			header.SetMode(0o644)
			header.Modified = entry.Time

			w, err := zw.CreateHeader(header)
			if err != nil {
				_ = resp.Close()
				slog.Error("创建文件 zip 条目失败", "path", childRemotePath, "err", err)
				continue
			}

			if _, err := io.Copy(w, resp); err != nil {
				slog.Error("写入 zip 文件内容失败", "path", childRemotePath, "err", err)
				_ = resp.Close()
				continue
			}
			_ = resp.Close()
		}
	}
	return nil
}
