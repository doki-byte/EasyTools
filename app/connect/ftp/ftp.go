package ftp

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jlaffaye/ftp"
)

func connectFTP(host string, port int, username, password string) (*ftp.ServerConn, error) {
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := ftp.Dial(address, ftp.DialWithTimeout(5*time.Second))
	if err != nil {
		return nil, err
	}
	if err := conn.Login(username, password); err != nil {
		return nil, err
	}
	return conn, nil
}

func StartWebFTP() {
	//gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(cors.Default())

	// 测试FTP连接
	r.POST("/api/ftp/connect", func(c *gin.Context) {
		var req struct {
			Host     string `json:"host"`
			Port     int    `json:"port"`
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		conn, err := connectFTP(req.Host, req.Port, req.Username, req.Password)
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

		conn, err := connectFTP(host, port, username, password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer conn.Quit()

		// 创建管道进行流式传输
		pr, pw := io.Pipe()
		defer pr.Close()

		// 启动goroutine进行流式复制
		go func() {
			defer pw.Close()
			_, err := io.Copy(pw, src)
			if err != nil {
				pw.CloseWithError(err)
			}
		}()

		// 流式上传到FTP
		err = conn.Stor(remotePath, pr)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "文件上传成功"})
	})

	// 下载文件 (流式处理)
	r.POST("/api/ftp/download", func(c *gin.Context) {
		var req struct {
			Host     string `json:"host"`
			Port     int    `json:"port"`
			Username string `json:"username"`
			Password string `json:"password"`
			Path     string `json:"path"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 连接FTP
		conn, err := connectFTP(req.Host, req.Port, req.Username, req.Password)
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

	// 删除文件或目录
	r.POST("/api/ftp/delete", func(c *gin.Context) {
		var req struct {
			Host     string `json:"host"`
			Port     int    `json:"port"`
			Username string `json:"username"`
			Password string `json:"password"`
			Path     string `json:"path"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		conn, err := connectFTP(req.Host, req.Port, req.Username, req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer conn.Quit()

		// 判断是文件还是目录
		parentDir := filepath.Dir(req.Path)
		baseName := filepath.Base(req.Path)
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
			opErr = conn.RemoveDir(req.Path)
		} else {
			opErr = conn.Delete(req.Path)
		}
		if opErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("删除失败: %v", opErr)})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
	})

	// 列表接口（带分页），错误时也返回空数组
	r.POST("/api/ftp/list", func(c *gin.Context) {
		var req struct {
			Host     string `json:"host"`
			Port     int    `json:"port"`
			Username string `json:"username"`
			Password string `json:"password"`
			Path     string `json:"path"`
			Page     int    `json:"page"`
			PageSize int    `json:"pageSize"`
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

		conn, err := connectFTP(req.Host, req.Port, req.Username, req.Password)
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
			Host       string `json:"host"`
			Port       int    `json:"port"`
			Username   string `json:"username"`
			Password   string `json:"password"`
			Path       string `json:"path"`       // 目标父目录
			FolderName string `json:"folderName"` // 新文件夹名
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		conn, err := connectFTP(req.Host, req.Port, req.Username, req.Password)
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

	r.Run("127.0.0.1:52869")
}

// 安全的路径清理
func safePath(path string) string {
	return filepath.Clean("/" + path)[1:]
}
