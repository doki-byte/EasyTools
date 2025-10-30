package extractor

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// 解压配置
type ExtractConfig struct {
	FS           embed.FS // 嵌入的文件系统
	TargetDir    string   // 目标目录（相对于应用基目录）
	Description  string   // 描述信息
	CheckSymlink bool     // 是否检查符号链接
	ExcludeExts  []string // 要排除的文件扩展名
	IncludeExts  []string // 要包含的文件扩展名（如果为空则包含所有）
	ExcludeDirs  []string // 要排除的目录
}

// 解压管理器
type Extractor struct {
	configs     map[string]*ExtractConfig
	appBaseDir  string
	initialized bool
	mu          sync.Mutex
}

var (
	globalExtractor *Extractor
	once            sync.Once
)

// 获取全局解压器实例
func GetExtractor() *Extractor {
	once.Do(func() {
		globalExtractor = &Extractor{
			configs:    make(map[string]*ExtractConfig),
			appBaseDir: getAppBaseDir(),
		}
	})
	return globalExtractor
}

// 获取应用基础目录
func getAppBaseDir() string {
	if runtime.GOOS == "darwin" {
		appName := "EasyTools"
		homeDir, err := os.UserHomeDir()
		if err != nil {
			panic("获取用户主目录失败: " + err.Error())
		}
		return filepath.Join(homeDir, "Library", "Application Support", appName)
	}

	currentPath, err := os.Getwd()
	if err != nil {
		panic("获取当前路径失败: " + err.Error())
	}
	return filepath.Join(currentPath, "EasyToolsFiles")
}

// 注册解压配置
func (e *Extractor) Register(name string, config *ExtractConfig) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.configs[name] = config
}

// 批量注册解压配置
func (e *Extractor) RegisterBatch(configs map[string]*ExtractConfig) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for name, config := range configs {
		e.configs[name] = config
	}
}

// 检查文件是否应该被包含
func (e *Extractor) shouldIncludeFile(path string, config *ExtractConfig) bool {
	// 检查目录排除
	for _, excludeDir := range config.ExcludeDirs {
		if strings.Contains(path, excludeDir) {
			return false
		}
	}

	ext := strings.ToLower(filepath.Ext(path))

	// 检查排除的扩展名
	for _, excludeExt := range config.ExcludeExts {
		if strings.ToLower(excludeExt) == ext {
			return false
		}
	}

	// 检查包含的扩展名
	if len(config.IncludeExts) > 0 {
		for _, includeExt := range config.IncludeExts {
			if strings.ToLower(includeExt) == ext {
				return true
			}
		}
		return false
	}

	return true
}

// 解压单个资源
func (e *Extractor) ExtractOne(name string) error {
	e.mu.Lock()
	config, exists := e.configs[name]
	e.mu.Unlock()

	if !exists {
		return fmt.Errorf("未找到解压配置: %s", name)
	}

	targetDir := filepath.Join(e.appBaseDir, config.TargetDir)

	// 检查目标目录是否已存在
	if _, err := os.Stat(targetDir); err == nil {
		log.Printf("%s 资源文件夹已存在，跳过解压", config.Description)
		return nil
	}

	log.Printf("开始解压 %s 资源...", config.Description)
	if err := e.extractFS(config, targetDir); err != nil {
		return fmt.Errorf("解压 %s 失败: %w", config.Description, err)
	}

	log.Printf("%s 资源解压完成", config.Description)
	return nil
}

// 解压所有注册的资源
func (e *Extractor) ExtractAll() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	var errors []string

	for _, config := range e.configs {
		targetDir := filepath.Join(e.appBaseDir, config.TargetDir)

		if _, err := os.Stat(targetDir); err == nil {
			log.Printf("%s 资源文件夹已存在，跳过解压", config.Description)
			continue
		}

		log.Printf("开始解压 %s 资源...", config.Description)
		if err := e.extractFS(config, targetDir); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", config.Description, err))
			continue
		}
		log.Printf("%s 资源解压完成", config.Description)
	}

	if len(errors) > 0 {
		return fmt.Errorf("解压过程中发生错误: %s", strings.Join(errors, "; "))
	}

	e.initialized = true
	return nil
}

// 提取并写入文件到目标目录
func (e *Extractor) extractResource(fsEmbed embed.FS, resourcePath, targetDir string) (string, error) {
	resourcePath = strings.ReplaceAll(resourcePath, "\\", "/")
	outputPath := filepath.Join(targetDir, resourcePath)

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return "", fmt.Errorf("创建目录失败: %w", err)
	}

	data, err := fsEmbed.ReadFile(resourcePath)
	if err != nil {
		return "", fmt.Errorf("无法读取资源文件 %s: %w", resourcePath, err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return "", fmt.Errorf("无法写入文件 %s: %w", outputPath, err)
	}

	return outputPath, nil
}

// 解压整个文件系统
func (e *Extractor) extractFS(config *ExtractConfig, targetDir string) error {
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	var fileCount int
	err := fs.WalkDir(config.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || path == "." {
			return nil
		}

		relPath := strings.TrimPrefix(path, "./")
		if relPath == "" {
			return nil
		}

		// 检查文件是否应该被包含
		if !e.shouldIncludeFile(relPath, config) {
			log.Printf("跳过文件: %s", relPath)
			return nil
		}

		_, err = e.extractResource(config.FS, relPath, targetDir)
		if err != nil && config.CheckSymlink && strings.Contains(err.Error(), "cannot find") {
			return e.tryCreateSymlink(relPath, targetDir)
		}
		if err != nil {
			return fmt.Errorf("处理文件 %s 失败: %w", path, err)
		}

		fileCount++
		//log.Printf("已提取: %s", relPath)
		return nil
	})

	if err != nil {
		return fmt.Errorf("遍历嵌入文件失败: %w", err)
	}

	log.Printf("成功提取 %d 个文件到: %s", fileCount, targetDir)
	return nil
}

// 尝试创建符号链接
func (e *Extractor) tryCreateSymlink(relPath, targetDir string) error {
	fullPath := filepath.Join(targetDir, relPath)
	dir := filepath.Dir(fullPath)
	filename := filepath.Base(fullPath)

	if strings.HasPrefix(filename, "_") {
		normalName := strings.TrimPrefix(filename, "_")
		normalPath := filepath.Join(dir, normalName)

		if _, err := os.Stat(normalPath); err == nil {
			if err := os.Symlink(normalName, fullPath); err == nil {
				log.Printf("已创建符号链接: %s -> %s", relPath, normalName)
				return nil
			}
		}
	}

	return fmt.Errorf("无法创建符号链接: %s", relPath)
}

// 获取应用基础目录
func (e *Extractor) GetAppBaseDir() string {
	return e.appBaseDir
}

// 检查是否已初始化
func (e *Extractor) IsInitialized() bool {
	return e.initialized
}
