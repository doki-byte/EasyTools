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
	gin.SetMode(gin.ReleaseMode)
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

	// 删除文件
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

		if err := conn.Delete(req.Path); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "File deleted successfully"})
	})

	// 上传文件
	r.POST("/api/ftp/upload", func(c *gin.Context) {
		file, err := c.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

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

		f, err := file.Open()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer f.Close()

		if err := conn.Stor(remotePath, f); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "File uploaded successfully"})
	})

	// 下载文件
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

		conn, err := connectFTP(req.Host, req.Port, req.Username, req.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer conn.Quit()

		resp, err := conn.Retr(req.Path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer resp.Close()

		data, err := io.ReadAll(resp)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filepath.Base(req.Path)))
		c.Data(http.StatusOK, "application/octet-stream", data)
	})
	// 获取文件列表
	r.POST("/api/ftp/list", func(c *gin.Context) {
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

		entries, err := conn.List(req.Path)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		var files []map[string]interface{}
		for _, entry := range entries {
			files = append(files, map[string]interface{}{
				"name": entry.Name,
				"type": entry.Type.String(),
				"size": entry.Size,
				"time": entry.Time.Format("2006-01-02 15:04:05"),
			})
		}

		c.JSON(http.StatusOK, gin.H{"files": files})
	})

	r.Run("127.0.0.1:10089")
}
