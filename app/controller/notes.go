package controller

import (
	"errors"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Note struct {
	Base
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

// NewNote 创建新的 Note 控制器
func NewNote() *Note {
	return &Note{}
}

// GetNotesDir 返回默认的笔记目录路径（不弹出对话框）
func (n *Note) GetNotesDir() (string, error) {
	notes := filepath.Join("EasyToolsFiles", "notes")
	// 确保目录存在
	if err := os.MkdirAll(notes, 0755); err != nil {
		return "", err
	}
	return notes, nil
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
	return result, nil
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

// ReadFile 读取文件内容
func (n *Note) ReadFile(filePath string) (string, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SaveFile 保存文件内容
func (n *Note) SaveFile(filePath, content string) error {
	return ioutil.WriteFile(filePath, []byte(content), 0644)
}
