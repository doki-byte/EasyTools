package controller

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Note struct {
	Base
	watcher   *fsnotify.Watcher
	server    *http.Server
	notesDir  string
	serverMux *http.ServeMux
	serverMu  sync.Mutex
	isRunning bool
	httpPort  int
}

// FileInfo 文件信息结构体
type FileInfo struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	IsDir     bool      `json:"isDir"`
	Size      int64     `json:"size"`
	Modified  time.Time `json:"modified"`
	Extension string    `json:"extension"`
}

// FileChangeEvent 文件变化事件
type FileChangeEvent struct {
	Type string `json:"type"` // "create", "delete", "rename", "modify"
	Path string `json:"path"`
	Name string `json:"name"`
}

// NewNote 创建新的 Note 控制器
func NewNote() *Note {
	// 创建文件监听器
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Printf("创建文件监听器失败: %v\n", err)
		return &Note{}
	}

	note := &Note{
		watcher:   watcher,
		serverMux: http.NewServeMux(),
	}

	// 启动监听协程
	go note.watchFiles()
	go note.startFileServer(filepath.Join(GetAppBaseDir(), "notes"))

	return note
}

// 启动文件服务器
func (n *Note) startFileServer(notesDir string) error {
	n.serverMu.Lock()
	defer n.serverMu.Unlock()

	n.notesDir = notesDir

	// 如果服务器已经在运行，先关闭它
	if n.server != nil {
		n.server.Close()
		n.server = nil
		n.isRunning = false
	}

	// 使用随机可用端口
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("启动文件服务器失败: %v", err)
	}

	n.httpPort = listener.Addr().(*net.TCPAddr).Port

	// 重新创建 ServeMux
	n.serverMux = http.NewServeMux()

	// 设置静态文件服务
	n.serverMux.Handle("/", http.FileServer(http.Dir(notesDir)))

	n.server = &http.Server{
		Handler: n.serverMux,
	}

	n.isRunning = true

	go func() {
		if err := n.server.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Printf("文件服务器错误: %v\n", err)
			n.isRunning = false
		}
	}()

	fmt.Printf("文件服务器启动在: http://127.0.0.1:%d\n", n.httpPort)

	return nil
}

// 获取文件服务器的基础URL
func (n *Note) getFileServerURL() string {
	if !n.isRunning {
		return ""
	}
	return fmt.Sprintf("http://127.0.0.1:%d", n.httpPort)
}

// ReadFile 读取 Markdown 文件，将相对路径转换为HTTP URL用于预览
func (n *Note) ReadFile(filePath string) (string, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	content := string(data)

	// 如果没有运行文件服务器，直接返回内容
	if !n.isRunning {
		return content, nil
	}

	// 正则匹配 Markdown 图片语法
	imgPattern := regexp.MustCompile(`!\[(.*?)\]\((.*?)\)`)
	dir := filepath.Dir(filePath)
	baseURL := n.getFileServerURL()

	// 处理图片路径：将相对路径转换为HTTP URL
	processed := imgPattern.ReplaceAllStringFunc(content, func(m string) string {
		matches := imgPattern.FindStringSubmatch(m)
		if len(matches) < 3 {
			return m
		}

		altText := matches[1]
		imgPath := matches[2]

		// 跳过网络图片和base64图片
		if strings.HasPrefix(imgPath, "http://") ||
			strings.HasPrefix(imgPath, "https://") ||
			strings.HasPrefix(imgPath, "data:") {
			return m
		}

		// 处理 Windows 路径分隔符问题
		imgPath = strings.ReplaceAll(imgPath, "\\", "/")

		// 构建完整图片路径
		var imgFullPath string
		if filepath.IsAbs(imgPath) {
			imgFullPath = imgPath
		} else {
			imgFullPath = filepath.Join(dir, imgPath)
		}

		// 检查图片是否存在
		if _, err := os.Stat(imgFullPath); err != nil {
			// 图片不存在，保持原样
			return m
		}

		// 计算相对于notesDir的相对路径
		relPath, err := filepath.Rel(n.notesDir, imgFullPath)
		if err != nil {
			return m
		}

		// 使用正斜杠
		relPath = filepath.ToSlash(relPath)

		// 构建HTTP URL用于预览
		httpURL := baseURL + "/" + relPath
		return "![" + altText + "](" + httpURL + ")"
	})

	return processed, nil
}

// SaveFile 保存文件内容，将HTTP URL转换回相对路径
func (n *Note) SaveFile(filePath, content string) error {
	// 将HTTP URL转换回相对路径
	processedContent := n.convertHTTPToRelative(filePath, content)

	// 处理base64图片并保存到本地
	finalContent, err := n.processBase64Images(filePath, processedContent)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filePath, []byte(finalContent), 0644)
}

// convertHTTPToRelative 将HTTP URL转换回相对路径
func (n *Note) convertHTTPToRelative(filePath, content string) string {
	baseURL := n.getFileServerURL()
	if baseURL == "" {
		return content // 如果没有文件服务器，直接返回
	}

	// 正则匹配图片语法
	imgPattern := regexp.MustCompile(`!\[(.*?)\]\((.*?)\)`)
	dir := filepath.Dir(filePath)

	// 替换HTTP URL为相对路径
	processed := imgPattern.ReplaceAllStringFunc(content, func(m string) string {
		matches := imgPattern.FindStringSubmatch(m)
		if len(matches) < 3 {
			return m
		}

		altText := matches[1]
		imgURL := matches[2]

		// 只处理本地文件服务器的URL
		if !strings.HasPrefix(imgURL, baseURL+"/") {
			return m
		}

		// 提取相对路径（去掉baseURL部分）
		relPath := strings.TrimPrefix(imgURL, baseURL+"/")

		// 构建相对于当前文件的路径
		imgFullPath := filepath.Join(n.notesDir, relPath)
		relativeToFile, err := filepath.Rel(dir, imgFullPath)
		if err != nil {
			return m
		}

		// 使用正斜杠
		relativeToFile = filepath.ToSlash(relativeToFile)

		return "![" + altText + "](" + relativeToFile + ")"
	})

	return processed
}

// processBase64Images 处理内容中的base64图片，保存到本地并替换为相对路径
func (n *Note) processBase64Images(filePath, content string) (string, error) {
	// 正则匹配base64图片
	base64Pattern := regexp.MustCompile(`!\[(.*?)\]\(data:image/(.*?);base64,(.*?)\)`)
	matches := base64Pattern.FindAllStringSubmatch(content, -1)

	if len(matches) == 0 {
		return content, nil
	}

	// 获取当前文件所在目录和文件名（不含扩展名）
	dir := filepath.Dir(filePath)
	baseName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	assetsDir := filepath.Join(dir, baseName+".assets")

	// 创建assets目录
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		return "", fmt.Errorf("创建图片目录失败: %v", err)
	}

	// 替换每个base64图片
	for _, match := range matches {
		if len(match) < 4 {
			continue
		}

		altText := match[1]
		imageType := match[2]
		base64Data := match[3]

		// 生成唯一文件名
		hash := md5.Sum([]byte(base64Data))
		hashStr := hex.EncodeToString(hash[:])

		var filename string
		var fileExt string
		switch imageType {
		case "png":
			fileExt = "png"
		case "jpeg", "jpg":
			fileExt = "jpg"
		case "gif":
			fileExt = "gif"
		case "svg+xml":
			fileExt = "svg"
		default:
			fileExt = "png"
		}
		filename = hashStr + "." + fileExt

		// 图片保存路径
		imagePath := filepath.Join(assetsDir, filename)

		// 解码base64数据并保存
		imageBytes, err := base64.StdEncoding.DecodeString(base64Data)
		if err != nil {
			continue
		}

		if err := ioutil.WriteFile(imagePath, imageBytes, 0644); err != nil {
			continue
		}

		// 构建相对路径
		relativePath := baseName + ".assets/" + filename

		// 替换内容中的base64为相对路径
		oldStr := match[0]
		newStr := fmt.Sprintf("![%s](%s)", altText, relativePath)
		content = strings.Replace(content, oldStr, newStr, 1)
	}

	return content, nil
}

// SaveImage 保存粘贴的图片，返回HTTP URL用于预览
func (n *Note) SaveImage(filePath string, imageBase64 string) (string, error) {
	// 解析base64数据
	parts := strings.Split(imageBase64, ",")
	if len(parts) != 2 {
		return "", errors.New("无效的base64图片数据")
	}

	// 获取MIME类型
	mimeType := strings.TrimSuffix(strings.Split(parts[0], ";")[0], "data:")

	// 获取当前文件所在目录和文件名（不含扩展名）
	dir := filepath.Dir(filePath)
	baseName := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))
	assetsDir := filepath.Join(dir, baseName+".assets")

	// 创建assets目录
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		return "", fmt.Errorf("创建图片目录失败: %v", err)
	}

	// 生成唯一文件名
	hash := md5.Sum([]byte(parts[1]))
	hashStr := hex.EncodeToString(hash[:])

	var filename string
	var fileExt string
	switch mimeType {
	case "image/png":
		fileExt = "png"
	case "image/jpeg":
		fileExt = "jpg"
	case "image/gif":
		fileExt = "gif"
	case "image/svg+xml":
		fileExt = "svg"
	default:
		fileExt = "png"
	}
	filename = hashStr + "." + fileExt

	// 图片保存路径
	imagePath := filepath.Join(assetsDir, filename)

	// 解码base64数据并保存
	imageBytes, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("解码base64失败: %v", err)
	}

	if err := ioutil.WriteFile(imagePath, imageBytes, 0644); err != nil {
		return "", fmt.Errorf("保存图片失败: %v", err)
	}

	baseURL := n.getFileServerURL()

	// 返回HTTP URL用于预览
	if baseURL != "" {
		// 计算相对于notesDir的相对路径
		relPath, err := filepath.Rel(n.notesDir, imagePath)
		if err != nil {
			return "", err
		}
		relPath = filepath.ToSlash(relPath)
		return baseURL + "/" + relPath, nil
	} else {
		// 如果文件服务器没有运行，返回相对路径
		return baseName + ".assets/" + filename, nil
	}
}

// WatchDirectory 开始监听目录变化
func (n *Note) WatchDirectory(dirPath string) error {
	if n.watcher == nil {
		return errors.New("文件监听器未初始化")
	}

	// 移除之前的监听
	if len(n.watcher.WatchList()) > 0 {
		for _, path := range n.watcher.WatchList() {
			n.watcher.Remove(path)
		}
	}

	// 添加新目录监听
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return n.watcher.Add(path)
		}
		return nil
	})

	return err
}

// OpenDirectory 打开文件夹选择对话框
func (n *Note) OpenDirectory() (string, error) {
	dir, err := os.UserHomeDir()
	if err != nil {
		dir = ""
	}

	result, err := runtime.OpenDirectoryDialog(n.ctx, runtime.OpenDialogOptions{
		Title:            "选择备忘录文件夹",
		DefaultDirectory: dir,
	})
	if err != nil {
		return "", err
	}
	if result == "" {
		return "", errors.New("未选择文件夹")
	}

	// 启动文件服务器
	if err := n.startFileServer(result); err != nil {
		fmt.Printf("启动文件服务器失败: %v\n", err)
	}

	// 开始监听目录
	if err := n.WatchDirectory(result); err != nil {
		fmt.Printf("开始监听目录失败: %v\n", err)
	}

	return result, nil
}

// GetNotesDir 返回笔记根目录
func (n *Note) GetNotesDir() (string, error) {
	baseDir := n.getAppPath()
	notes := filepath.Join(baseDir, "notes")
	if err := os.MkdirAll(notes, 0755); err != nil {
		return "", err
	}
	return notes, nil
}

// GetFiles 获取指定目录下的文件列表
func (n *Note) GetFiles(dirPath string) ([]FileInfo, error) {
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var fileList []FileInfo
	for _, file := range files {
		// 只显示文件夹和.md文件
		if file.IsDir() || strings.HasSuffix(file.Name(), ".md") {
			fileList = append(fileList, FileInfo{
				Name:      file.Name(),
				Path:      filepath.Join(dirPath, file.Name()),
				IsDir:     file.IsDir(),
				Size:      file.Size(),
				Modified:  file.ModTime(),
				Extension: filepath.Ext(file.Name()),
			})
		}
	}
	return fileList, nil
}

// CreateFile 创建新文件
func (n *Note) CreateFile(dirPath, filename string) error {
	fullPath := filepath.Join(dirPath, filename)
	if !strings.HasSuffix(filename, ".md") {
		fullPath += ".md"
	}
	if _, err := os.Stat(fullPath); err == nil {
		return errors.New("文件已存在")
	}
	return ioutil.WriteFile(fullPath, []byte("# 新备忘录\n\n开始记录..."), 0644)
}

// CreateFolder 创建新文件夹
func (n *Note) CreateFolder(dirPath, foldername string) error {
	fullPath := filepath.Join(dirPath, foldername)
	if _, err := os.Stat(fullPath); err == nil {
		return errors.New("文件夹已存在")
	}
	return os.Mkdir(fullPath, 0755)
}

// DeleteItem 删除文件或文件夹
func (n *Note) DeleteItem(path string) error {
	return os.RemoveAll(path)
}

// RenameItem 重命名文件或文件夹
func (n *Note) RenameItem(oldPath, newName string) error {
	dir := filepath.Dir(oldPath)
	newPath := filepath.Join(dir, newName)
	return os.Rename(oldPath, newPath)
}

// watchFiles 监听文件变化
func (n *Note) watchFiles() {
	if n.watcher == nil {
		return
	}

	var lastEventTime time.Time

	for {
		select {
		case event, ok := <-n.watcher.Events:
			if !ok {
				return
			}

			// 忽略临时文件和频繁事件
			if strings.HasSuffix(event.Name, ".tmp") ||
				strings.Contains(event.Name, "~") ||
				time.Since(lastEventTime) < 500*time.Millisecond {
				continue
			}

			lastEventTime = time.Now()

			// 只处理创建、删除、重命名事件
			var changeType string
			if event.Op&fsnotify.Create == fsnotify.Create {
				changeType = "create"
			} else if event.Op&fsnotify.Remove == fsnotify.Remove {
				changeType = "delete"
			} else if event.Op&fsnotify.Rename == fsnotify.Rename {
				changeType = "rename"
			} else {
				continue
			}

			// 发送事件到前端
			if n.ctx != nil {
				fileEvent := FileChangeEvent{
					Type: changeType,
					Path: event.Name,
					Name: filepath.Base(event.Name),
				}
				eventData, _ := json.Marshal(fileEvent)
				runtime.EventsEmit(n.ctx, "fileChange", string(eventData))
			}

		case err, ok := <-n.watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("文件监听错误: %v\n", err)
		}
	}
}

// 关闭监听器和服务器
func (n *Note) Shutdown() {
	if n.watcher != nil {
		n.watcher.Close()
	}
	if n.server != nil {
		n.server.Close()
	}
}
