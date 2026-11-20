package service

import (
	gin2 "EasyTools/app/controller/connect/ssh/gin"
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"sync"
)

// SFTP 上传进度，全局按 uploadId 存储
var (
	sftpUploadProgress   = make(map[string]*UploadProgress)
	sftpUploadProgressMu sync.RWMutex
	// SFTP 上传取消标记，key 为 uploadId
	sftpUploadCanceled = make(map[string]bool)
)

// UploadProgress 表示一次上传的进度
type UploadProgress struct {
	Total    int64  `json:"total"`
	Uploaded int64  `json:"uploaded"`
	Done     bool   `json:"done"`
	Error    string `json:"error"`
}

// progressReaderSftp 用于在复制数据时统计已读字节数并写入进度表
type progressReaderSftp struct {
	r        io.Reader
	id       string
	total    int64
	uploaded int64
}

func (p *progressReaderSftp) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	if n > 0 {
		sftpUploadProgressMu.Lock()
		defer sftpUploadProgressMu.Unlock()
		p.uploaded += int64(n)
		if prog, ok := sftpUploadProgress[p.id]; ok {
			prog.Uploaded = p.uploaded
			if prog.Total > 0 && prog.Uploaded > prog.Total {
				prog.Uploaded = prog.Total
			}
		}
		// 如果被标记为取消，则中断当前文件上传
		if canceled, ok := sftpUploadCanceled[p.id]; ok && canceled {
			return 0, io.EOF
		}
	}
	return n, err
}

func getSshConn(sessionId string) (*SshConn, error) {
	cli, ok := OnlineClients.Load(sessionId)
	if !ok {
		slog.Error("加载ssh连接错误")
		return nil, errors.New("加载ssh连接错误")
	}
	conn, ok := cli.(*SshConn)
	if conn != nil && !ok {
		slog.Error("断言ssh连接错误")
		return nil, errors.New("断言ssh连接错误")
	}
	return conn, nil
}

// SshDesktopBridge 提供给 Wails 前端调用的桥接结构（桌面端专用）
// 通过 session_id 与已有的 OnlineClients / sftpClient 复用同一条 SSH/SFTP 连接，
// 避免重新建立连接。
type SshDesktopBridge struct{}

// UploadLocalPaths 根据本地绝对路径列表，将文件/目录上传到当前 SFTP 会话的目标目录。
// sessionID: SSH 会话 ID
// targetDir: 远程目标目录，例如 "/tmp" 或 "/"
// paths: 本地绝对路径列表（文件或目录）
func (b *SshDesktopBridge) UploadLocalPaths(sessionID string, targetDir string, paths []string) (string, error) {
	if sessionID == "" {
		return "", errors.New("session_id 不能为空")
	}
	if len(paths) == 0 {
		return "", errors.New("本地路径列表为空")
	}

	if targetDir == "" {
		targetDir = "/"
	}
	// 规范化远程目录，统一使用 / 结尾
	if !strings.HasPrefix(targetDir, "/") {
		targetDir = "/" + targetDir
	}
	targetDir = path.Clean(targetDir)

	conn, err := getSshConn(sessionID)
	if err != nil {
		return "", err
	}

	var uploaded, failed int

	for _, p := range paths {
		if p == "" {
			failed++
			continue
		}

		info, statErr := os.Stat(p)
		if statErr != nil {
			slog.Error("本地文件状态读取失败", "path", p, "err", statErr)
			failed++
			continue
		}

		if info.IsDir() {
			// 目录：递归上传，保持目录结构
			if err := b.uploadDirRecursive(conn, p, targetDir, &uploaded, &failed); err != nil {
				slog.Error("上传目录失败", "local_dir", p, "err", err)
				// 递归内部已统计失败数量，这里不再重复增加
			}
		} else {
			// 单文件：直接上传到目标目录
			if err := b.uploadSingleFile(conn, p, targetDir, ""); err != nil {
				slog.Error("上传文件失败", "local_file", p, "err", err)
				failed++
			} else {
				uploaded++
			}
		}
	}

	msg := fmt.Sprintf("成功: %d 个，失败: %d 个", uploaded, failed)
	if failed > 0 {
		return msg, errors.New(msg)
	}
	return msg, nil
}

// uploadDirRecursive 递归上传本地目录下的所有文件，保持相对目录结构
func (b *SshDesktopBridge) uploadDirRecursive(conn *SshConn, localRoot, remoteRoot string, uploaded, failed *int) error {
	localRoot = filepath.Clean(localRoot)

	walkErr := filepath.Walk(localRoot, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			slog.Error("遍历本地目录出错", "path", p, "err", err)
			*failed++
			return nil
		}
		if info.IsDir() {
			// 目录本身由 MkdirAll 在上传文件时顺带创建，不单独处理
			return nil
		}

		rel, relErr := filepath.Rel(localRoot, p)
		if relErr != nil {
			slog.Error("计算相对路径失败", "root", localRoot, "path", p, "err", relErr)
			*failed++
			return nil
		}

		subDir := filepath.Dir(rel)
		if subDir == "." {
			subDir = ""
		}

		if err := b.uploadSingleFile(conn, p, remoteRoot, subDir); err != nil {
			slog.Error("上传目录中的文件失败", "local_file", p, "err", err)
			*failed++
		} else {
			*uploaded++
		}
		return nil
	})

	return walkErr
}

// uploadSingleFile 上传单个本地文件到远程，subDir 为相对 remoteRoot 的子目录，可为空
func (b *SshDesktopBridge) uploadSingleFile(conn *SshConn, localPath, remoteRoot, subDir string) error {
	f, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close()
	}()

	fileName := filepath.Base(localPath)

	remoteDir := remoteRoot
	if subDir != "" && subDir != "." {
		// 将本地子目录转换为远程路径（统一使用 / ）
		remoteDir = path.Join(remoteRoot, filepath.ToSlash(subDir))
	}

	// 确保远程目录存在
	if err := conn.sftpClient.MkdirAll(remoteDir); err != nil {
		slog.Error("远程目录创建失败", "remote_dir", remoteDir, "err", err)
		return err
	}

	remotePath := path.Join(remoteDir, fileName)

	dst, err := conn.sftpClient.Create(remotePath)
	if err != nil {
		slog.Error("远程文件创建失败", "remote_path", remotePath, "err", err)
		return err
	}
	defer func() {
		_ = dst.Close()
	}()

	if _, err := io.Copy(dst, f); err != nil {
		slog.Error("复制文件内容失败", "local", localPath, "remote", remotePath, "err", err)
		return err
	}
	return nil
}

// SftpList GET sftp 获取指定目录下文件信息
func SftpList(c *gin2.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(200, gin2.H{"code": 4, "msg": "读取目录错误"})
			return
		}
	}()
	dirPath := c.PostForm("path")
	conn, err := getSshConn(c.PostForm("session_id"))
	if err != nil {
		slog.Error(err.Error())
		c.JSON(200, gin2.H{"code": 2, "msg": err.Error()})
		return
	}
	if dirPath == "" {
		if wd, err := conn.sftpClient.Getwd(); err == nil && wd != "" {
			dirPath = wd
		} else {
			dirPath = "/"
		}
	}

	files, err := conn.sftpClient.ReadDir(dirPath)
	if err != nil {
		slog.Error("sftp客户端ReadDir错误", "err_msg", err.Error())
		c.JSON(200, gin2.H{"code": 3, "msg": "sftp客户端读取目录错误"})
		return
	}

	fileCount := 0
	dirCount := 0
	var fileList []interface{}
	for _, file := range files {
		fileInfo := map[string]interface{}{}
		fileInfo["path"] = path.Join(dirPath, file.Name())
		fileInfo["name"] = file.Name()
		fileInfo["mode"] = file.Mode().String()
		fileInfo["size"] = file.Size()
		fileInfo["mod_time"] = file.ModTime().Format("2006-01-02 15:04:05")
		if file.IsDir() {
			fileInfo["type"] = "d"
			dirCount += 1
		} else {
			fileInfo["type"] = "f"
			fileCount += 1
		}
		fileList = append(fileList, fileInfo)
	}

	// 内部方法,处理路径信息
	pathHandler := func(dirPath string) (paths []map[string]string) {
		tmp := strings.Split(dirPath, "/")

		var dirs []string
		if strings.HasPrefix(dirPath, "/") {
			dirs = append(dirs, "/")
		}

		for _, item := range tmp {
			name := strings.TrimSpace(item)
			if len(name) > 0 {
				dirs = append(dirs, name)
			}
		}

		for i, item := range dirs {
			fullPath := path.Join(dirs[:i+1]...)
			pathInfo := map[string]string{}
			pathInfo["name"] = item
			pathInfo["dir"] = fullPath
			paths = append(paths, pathInfo)
		}
		return paths
	}

	data := map[string]interface{}{
		"files":       fileList,
		"file_count":  fileCount,
		"dir_count":   dirCount,
		"paths":       pathHandler(dirPath),
		"current_dir": dirPath,
	}

	c.JSON(200, gin2.H{"code": 0, "data": data, "msg": "ok"})
}

// SftpDownLoad POST sftp 下载文件
func SftpDownLoad(c *gin2.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(200, gin2.H{"code": 1, "msg": "下载错误"})
			return
		}
	}()
	fullPath, err := url.QueryUnescape(c.Query("path"))
	if err != nil {
		slog.Error("获取文件路径参数错误", "err_msg", err.Error())
		c.JSON(200, gin2.H{"code": 2, "msg": "获取文件路径参数错误"})
		return
	}
	conn, err := getSshConn(c.Query("session_id"))
	if err != nil {
		slog.Error(err.Error())
		c.JSON(200, gin2.H{"code": 3, "msg": err.Error()})
		return
	}

	file, err := conn.sftpClient.Open(fullPath)
	defer func() {
		_ = file.Close()
	}()
	if err != nil {
		slog.Error("sftpClient.Openc错误", "err_msg", err.Error())
		c.JSON(200, gin2.H{"code": 4, "msg": "sftp打开文件错误"})
		return
	}

	stat, err := file.Stat()
	if err != nil {
		slog.Error("file.Stat()错误", "err_msg", err.Error())
		c.JSON(200, gin2.H{"code": 4, "msg": "读取文件信息错误"})
		return
	}

	c.Writer.WriteHeader(http.StatusOK)
	c.Header("Content-Disposition", "attachment; filename="+stat.Name())
	c.Header("Content-Type", "application/octet-stream")
	//c.Header("Content-Type", "application/x-download")
	c.Header("Content-Length", fmt.Sprintf("%d", stat.Size()))
	_, err = file.WriteTo(c.Writer)
	if err != nil {
		slog.Error("file.WriteTo错误", "err_msg", err.Error())
		c.JSON(200, gin2.H{"code": 5, "msg": "下载文件错误"})
		return
	}
	c.Writer.Flush()
}

// SftpDownLoadDir GET sftp 打包下载整个目录（zip）
func SftpDownLoadDir(c *gin2.Context) {
	defer func() {
		if err := recover(); err != nil {
			slog.Error("SftpDownLoadDir panic", "err", err)
			c.JSON(200, gin2.H{"code": 1, "msg": "下载目录错误"})
		}
	}()

	dirPathRaw := c.Query("path")
	if dirPathRaw == "" {
		c.JSON(200, gin2.H{"code": 2, "msg": "目录路径不能为空"})
		return
	}

	dirPath, err := url.QueryUnescape(dirPathRaw)
	if err != nil {
		slog.Error("获取目录路径参数错误", "err_msg", err.Error())
		c.JSON(200, gin2.H{"code": 3, "msg": "获取目录路径参数错误"})
		return
	}

	conn, err := getSshConn(c.Query("session_id"))
	if err != nil {
		slog.Error("获取 SSH 连接失败", "err", err)
		c.JSON(200, gin2.H{"code": 4, "msg": err.Error()})
		return
	}

	// 读取根目录信息，确保存在且为目录
	rootStat, err := conn.sftpClient.Stat(dirPath)
	if err != nil {
		slog.Error("读取目录信息失败", "path", dirPath, "err", err)
		c.JSON(200, gin2.H{"code": 5, "msg": "读取目录信息失败"})
		return
	}
	if !rootStat.IsDir() {
		c.JSON(200, gin2.H{"code": 6, "msg": "目标路径不是目录"})
		return
	}

	// 归一化目录名，生成 zip 名称
	baseName := path.Base(dirPath)
	if baseName == "" || baseName == "/" || baseName == "." {
		baseName = "folder"
	}
	zipName := fmt.Sprintf("%s_%d.zip", baseName, time.Now().Unix())

	// 设置响应头，按流式方式写出 zip 内容
	c.Writer.WriteHeader(http.StatusOK)
	c.Header("Content-Disposition", "attachment; filename="+url.PathEscape(zipName))
	c.Header("Content-Type", "application/zip")
	// 不预先设置 Content-Length，方便流式输出

	zipWriter := zip.NewWriter(c.Writer)
	defer func() {
		if err := zipWriter.Close(); err != nil {
			slog.Error("关闭 zipWriter 失败", "err", err)
		}
	}()

	// 递归遍历目录并写入 zip
	err = walkSftpDirAndZip(conn, dirPath, "", zipWriter)
	if err != nil {
		slog.Error("打包目录失败", "path", dirPath, "err", err)
		// 尽量在响应尾部给出简单 JSON，不再写复杂结构，避免破坏已写入的 zip
		return
	}

	// 尽可能刷新缓冲
	if flusher, ok := c.Writer.(http.Flusher); ok {
		flusher.Flush()
	}
}

// walkSftpDirAndZip 递归遍历远程 SFTP 目录，并写入到 zipWriter 中
// basePath: 远程根目录绝对路径
// relPath:  当前遍历的相对路径前缀（用于 zip 内部路径）
func walkSftpDirAndZip(conn *SshConn, basePath, relPath string, zw *zip.Writer) error {
	entries, err := conn.sftpClient.ReadDir(basePath)
	if err != nil {
		return fmt.Errorf("读取目录失败: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if name == "." || name == ".." {
			continue
		}

		childRemotePath := path.Join(basePath, name)
		childRelPath := name
		if relPath != "" {
			childRelPath = path.Join(relPath, name)
		}

		if entry.IsDir() {
			// 目录：在 zip 中创建一个目录条目（以 / 结尾），然后递归
			header := &zip.FileHeader{
				Name:   childRelPath + "/",
				Method: zip.Deflate,
			}
			header.SetMode(0o755 | os.ModeDir)
			if _, err := zw.CreateHeader(header); err != nil {
				return fmt.Errorf("创建目录 zip 条目失败: %w", err)
			}

			if err := walkSftpDirAndZip(conn, childRemotePath, childRelPath, zw); err != nil {
				return err
			}
		} else {
			// 文件：从 SFTP 读取并写入 zip
			file, err := conn.sftpClient.Open(childRemotePath)
			if err != nil {
				slog.Error("打开远程文件失败", "path", childRemotePath, "err", err)
				continue
			}

			header, err := zip.FileInfoHeader(entry)
			if err != nil {
				_ = file.Close()
				slog.Error("创建文件 zip 头失败", "path", childRemotePath, "err", err)
				continue
			}
			header.Name = childRelPath
			header.Method = zip.Deflate

			w, err := zw.CreateHeader(header)
			if err != nil {
				_ = file.Close()
				slog.Error("创建文件 zip 条目失败", "path", childRemotePath, "err", err)
				continue
			}

			if _, err := io.Copy(w, file); err != nil {
				slog.Error("写入 zip 文件内容失败", "path", childRemotePath, "err", err)
				_ = file.Close()
				continue
			}
			_ = file.Close()
		}
	}
	return nil
}

// SftpUpload PUT sftp 上传文件
func SftpUpload(c *gin2.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(200, gin2.H{"code": 4, "msg": "上传错误"})
			return
		}
	}()

	dstPath := c.PostForm("path")
	//获取上传的文件组
	form, err := c.MultipartForm()
	if err != nil {
		slog.Error("获取form数据错误", "err_msg", err.Error())
		c.JSON(200, gin2.H{"code": 1, "msg": "获取form数据错误"})
		return
	}
	files := form.File["files"]
	// files := c.Request.MultipartForm.File["file"]

	conn, err := getSshConn(c.PostForm("session_id"))
	if err != nil {
		slog.Error(err.Error())
		c.JSON(200, gin2.H{"code": 2, "msg": err.Error()})
		return
	}

	// 单次批量上传使用同一个 uploadId（前端生成），统计总大小与进度
	uploadID := c.PostForm("uploadId")
	var totalSize int64
	for _, f := range files {
		totalSize += f.Size
	}
	if uploadID != "" {
		sftpUploadProgressMu.Lock()
		// 多文件上传时，同一个 uploadId 会被多次调用，这里做累加而不是覆盖
		if prog, ok := sftpUploadProgress[uploadID]; ok {
			// 已有进度，则累加本次请求中所有文件的大小到总大小
			prog.Total += totalSize
			// 标记为未完成，避免上一次请求提前设置 Done=true
			prog.Done = false
		} else {
			// 首次出现该 uploadId，初始化进度
			sftpUploadProgress[uploadID] = &UploadProgress{
				Total:    totalSize,
				Uploaded: 0,
				Done:     false,
				Error:    "",
			}
		}
		sftpUploadProgressMu.Unlock()
	}

	var ret []string
	var wasCanceled bool
	for _, file := range files {
		// 检查是否已取消
		if uploadID != "" {
			sftpUploadProgressMu.RLock()
			canceled, ok := sftpUploadCanceled[uploadID]
			sftpUploadProgressMu.RUnlock()
			if ok && canceled {
				wasCanceled = true
				break
			}
		}

		srcFile, err := file.Open()
		if err != nil {
			continue
		}
		var reader io.Reader = srcFile
		if uploadID != "" {
			reader = &progressReaderSftp{
				r:     srcFile,
				id:    uploadID,
				total: totalSize,
			}
		}

		fileName := file.Filename
		dstFile, err := conn.sftpClient.Create(path.Join(dstPath, fileName))
		if err != nil {
			_ = srcFile.Close()
			continue
		}
		_, err = io.Copy(dstFile, reader)
		if err != nil {
			_ = srcFile.Close()
			_ = dstFile.Close()
			// 检查是否因为取消而中断
			if uploadID != "" {
				sftpUploadProgressMu.RLock()
				canceled, ok := sftpUploadCanceled[uploadID]
				sftpUploadProgressMu.RUnlock()
				if ok && canceled {
					wasCanceled = true
					break
				}
			}
			continue
		}
		_ = srcFile.Close()
		_ = dstFile.Close()
		ret = append(ret, fileName)
	}

	if uploadID != "" {
		sftpUploadProgressMu.Lock()
		if prog, ok := sftpUploadProgress[uploadID]; ok {
			if wasCanceled {
				// 如果被取消，设置错误信息并标记为完成
				prog.Error = "上传已取消"
				prog.Done = true
			} else {
				// 正常完成
				prog.Uploaded = prog.Total
				prog.Done = true
			}
		}
		// 上传结束后清理取消标记
		delete(sftpUploadCanceled, uploadID)
		sftpUploadProgressMu.Unlock()
	}

	msg := strconv.Itoa(len(ret)) + " 个文件上传成功"
	c.JSON(200, gin2.H{"code": 0, "msg": msg, "data": ret})
}

// SftpUploadProgress 查询 SFTP 上传进度
func SftpUploadProgress(c *gin2.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(200, gin2.H{"code": 1, "msg": "缺少 id"})
		return
	}

	sftpUploadProgressMu.RLock()
	prog, ok := sftpUploadProgress[id]
	sftpUploadProgressMu.RUnlock()
	if !ok {
		c.JSON(200, gin2.H{
			"code":   0,
			"exists": false,
		})
		return
	}

	c.JSON(200, gin2.H{
		"code":    0,
		"exists":  true,
		"total":   prog.Total,
		"uploaded": prog.Uploaded,
		"done":    prog.Done,
		"error":   prog.Error,
	})

	if prog.Done {
		sftpUploadProgressMu.Lock()
		delete(sftpUploadProgress, id)
		delete(sftpUploadCanceled, id)
		sftpUploadProgressMu.Unlock()
	}
}

// SftpUploadCancel 取消 SFTP 上传（基于 uploadId）
func SftpUploadCancel(c *gin2.Context) {
	id := c.Query("id")
	if id == "" {
		id = c.PostForm("id")
	}
	if id == "" {
		c.JSON(200, gin2.H{"code": 1, "msg": "缺少 id"})
		return
	}

	sftpUploadProgressMu.Lock()
	sftpUploadCanceled[id] = true
	sftpUploadProgressMu.Unlock()

	c.JSON(200, gin2.H{"code": 0, "msg": "已请求取消上传"})
}

// SftpDelete DELETE sftp 删除文件或目录
func SftpDelete(c *gin2.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(200, gin2.H{"code": 4, "msg": "删除错误"})
			return
		}
	}()

	type Body struct {
		SessionId string `form:"session_id" binding:"required,min=1,max=128" json:"session_id"`
		Path      string `form:"path" binding:"required,min=1,max=1024" json:"path"`
	}

	var body Body
	if err := c.ShouldBind(&body); err != nil {
		slog.Error("绑定数据错误", "err_msg", err.Error())
		c.JSON(200, gin2.H{"code": 1, "msg": "输入数据不合法"})
		return
	}
	conn, err := getSshConn(body.SessionId)
	if err != nil {
		slog.Error(err.Error())
		c.JSON(200, gin2.H{"code": 1, "msg": err.Error()})
		return
	}

	err = conn.sftpClient.RemoveAll(body.Path)
	if err != nil {
		slog.Error("sftpClient.Remove错误", "err_msg", err.Error())
		c.JSON(200, gin2.H{"code": 2, "msg": "删除文件错误"})
		return
	}
	c.JSON(200, gin2.H{"code": 0, "msg": "删除成功"})
}

// SftpCreateDir sftp 创建目录
func SftpCreateDir(c *gin2.Context) {
	defer func() {
		if err := recover(); err != nil {
			c.JSON(200, gin2.H{"code": 4, "msg": "创建目录错误"})
			return
		}
	}()

	type Body struct {
		SessionId string `form:"session_id" binding:"required,min=1,max=128" json:"session_id"`
		Path      string `form:"path" binding:"required,min=1,max=1024" json:"path"`
	}

	var body Body
	if err := c.ShouldBind(&body); err != nil {
		slog.Error("绑定数据错误", "err_msg", err.Error())
		c.JSON(200, gin2.H{"code": 1, "msg": "输入数据不合法"})
		return
	}
	conn, err := getSshConn(body.SessionId)
	if err != nil {
		slog.Error(err.Error())
		c.JSON(200, gin2.H{"code": 1, "msg": err.Error()})
		return
	}

	err = conn.sftpClient.MkdirAll(body.Path)
	if err != nil {
		slog.Error("sftpClient.MkdirAll错误", "err_msg", err.Error())
		c.JSON(200, gin2.H{"code": 2, "msg": "创建目录错误"})
		return
	}
	c.JSON(200, gin2.H{"code": 0, "msg": "创建目录成功"})
}
