//go:build windows

package unwxapp

import (
	"EasyTools/app/controller/proxy"
	"EasyTools/app/controller/system"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"

	"EasyTools/app/model"
)

// 使用 model 包中的类型定义
type WechatConfig = model.WechatConfig
type MiniAppInfo = model.MiniAppInfo
type VersionTask = model.VersionTask

type VersionTaskStatus struct {
	Number          string   `json:"Number"`
	DecompileStatus string   `json:"DecompileStatus"`
	MatchStatus     string   `json:"MatchStatus"`
	Message         string   `json:"Message"`
	WxapkgTags      []string `json:"WxapkgTags"`
}

type InfoToFront struct {
	AppID      string               `json:"AppID"`
	UpdateDate string               `json:"UpdateDate"`
	Info       *MiniAppInfo         `json:"Info"`
	Versions   []*VersionTaskStatus `json:"Versions"`
}

type UnWxapp struct {
	system.Base
	http                *http.Client
	autoDecompile       bool
	mutex               sync.Mutex
	autoDecompileTicker *time.Ticker
	stopAutoDecompile   chan bool
	appidQuotaEmitState int32 // 由 appid_query 用 atomic：本批 GetAllMiniApp 内额度提示仅发一次
	appidKeyStatusMu    sync.Mutex
	appidKeyStatuses    []appidKeyRuntimeStatus // 与配置中有效密钥行一一对应，由 appid_query 更新
	appidMPEmbedMu      sync.Mutex
	appidMPEmbedLast    time.Time
	// 单次 GetAllMiniApp 批次内：内嵌 MP 连续失败达阈值后跳过后续 searchWxaembedded（仍可走 Open API）
	mpEmbedBatchConsecutiveFails atomic.Int32
	mpEmbedBatchSkipMP          atomic.Bool
}

func NewUnWxapp() *UnWxapp {
	app := &UnWxapp{
		http: proxy.GlobalProxyManager.GetHTTPClient(),
	}
	return app
}

// 获取配置
func (u *UnWxapp) getConfig() (*WechatConfig, error) {
	db := u.Db()
	if db == nil {
		return nil, errors.New("数据库连接失败")
	}

	var config WechatConfig
	if err := db.First(&config, 1).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

// 保存配置
func (u *UnWxapp) saveConfig(config *WechatConfig) error {
	db := u.Db()
	if db == nil {
		return errors.New("数据库连接失败")
	}
	return db.Save(config).Error
}

// ============ 公开接口 ============

func (u *UnWxapp) InitCheck() bool {
	if u.checkNodeInPath() {
		return true
	}

	if u.checkCommonNodePaths() {
		return true
	}

	return false
}

func (u *UnWxapp) checkNodeInPath() bool {
	if _, err := exec.LookPath("node"); err == nil {
		log.Println("检测到 Node.js")
		return true
	}
	return false
}

func (u *UnWxapp) checkCommonNodePaths() bool {
	commonPaths := []string{
		filepath.Join(os.Getenv("ProgramFiles"), "nodejs", "node.exe"),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "nodejs", "node.exe"),
		filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "nodejs", "node.exe"),
		filepath.Join(os.Getenv("APPDATA"), "npm", "node.exe"),
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			log.Printf("✓ 在 %s 检测到 Node.js\n", path)
			return true
		}
	}
	return false
}

func (u *UnWxapp) SetAppletPath(path string) error {
	config, err := u.getConfig()
	if err != nil {
		return err
	}
	config.AppletPath = path
	return u.saveConfig(config)
}

func (u *UnWxapp) GetAppletPath() (string, error) {
	config, err := u.getConfig()
	if err != nil {
		// 返回一个空字符串作为默认值，并返回具体的错误
		return "", fmt.Errorf("获取配置失败: %v", err)
	}
	// 成功时，返回结果和 nil 表示没有错误
	return config.AppletPath, nil
}

func (u *UnWxapp) GetWechatRules() ([]string, error) {
	config, err := u.getConfig()
	if err != nil {
		return nil, err
	}

	var rules []string
	if err := json.Unmarshal([]byte(config.Rules), &rules); err != nil {
		return nil, err
	}
	return rules, nil
}

func (u *UnWxapp) SaveWechatRules(rules []string) error {
	config, err := u.getConfig()
	if err != nil {
		return err
	}

	rulesJSON, err := json.Marshal(rules)
	if err != nil {
		return err
	}

	config.Rules = string(rulesJSON)
	return u.saveConfig(config)
}

func (u *UnWxapp) AutoDecompile(enable bool) error {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	u.autoDecompile = enable

	// 同时保存到数据库
	config, err := u.getConfig()
	if err != nil {
		return err
	}
	config.AutoDecompile = enable

	if enable {
		// 启动自动反编译循环
		u.startAutoDecompile()
	} else {
		// 停止自动反编译循环
		u.stopAutoDecompileLoop()
	}

	return u.saveConfig(config)
}

func (u *UnWxapp) ClearApplet() error {
	config, err := u.getConfig()
	if err != nil {
		return err
	}

	if config.AppletPath == "" {
		return errors.New("Applet路径未设置")
	}

	appletPath, err := filepath.Abs(config.AppletPath)
	if err != nil {
		return fmt.Errorf("获取绝对路径失败: %v", err)
	}

	log.Printf("开始清空Applet目录: %s", appletPath)

	// 检查路径是否是users目录（新版本路径结构）
	normalizedPath := strings.ToLower(strings.TrimRight(appletPath, "\\/"))
	baseName := strings.ToLower(filepath.Base(appletPath))
	isUsersDir := baseName == "users" || strings.HasSuffix(normalizedPath, "users")

	if isUsersDir {
		// log.Printf("检测到users目录，只删除packages目录内容，保留数据库文件")

		// 遍历所有用户目录，只删除packages目录下的内容
		userEntries, err := os.ReadDir(appletPath)
		if err != nil {
			return fmt.Errorf("读取users目录失败: %v", err)
		}

		var deleteErrors []string
		for _, userEntry := range userEntries {
			if !userEntry.IsDir() {
				continue
			}

			userDirName := userEntry.Name()
			// 构建 packages 目录路径
			packagesPath := filepath.Join(appletPath, userDirName, "applet", "packages")

			// 检查packages目录是否存在
			if isDir, err := u.isDir(packagesPath); err == nil && isDir {
				log.Printf("删除用户 %s 的packages目录: %s", userDirName, packagesPath)
				// 只删除packages目录下的内容，不删除整个目录结构
				entries, err := os.ReadDir(packagesPath)
				if err != nil {
					log.Printf("读取packages目录失败: %s, %v", packagesPath, err)
					continue
				}

				for _, entry := range entries {
					entryPath := filepath.Join(packagesPath, entry.Name())
					if err := os.RemoveAll(entryPath); err != nil {
						// 记录错误但继续处理其他文件
						deleteErrors = append(deleteErrors, fmt.Sprintf("%s: %v", entryPath, err))
						log.Printf("删除失败（可能被占用）: %s, %v", entryPath, err)
					}
				}
			}
		}

		if len(deleteErrors) > 0 {
			log.Printf("部分文件删除失败（可能被其他进程占用）: %d 个错误", len(deleteErrors))
			// 不返回错误，因为部分文件可能被微信占用，这是正常的
		}
	} else {
		// 旧版本路径结构，直接删除整个目录
		// log.Printf("使用旧版本路径结构，删除整个目录: %s", appletPath)
		if err := os.RemoveAll(appletPath); err != nil {
			// 如果删除失败，尝试只删除目录下的内容
			// log.Printf("删除整个目录失败，尝试删除目录内容: %v", err)
			entries, err := os.ReadDir(appletPath)
			if err == nil {
				for _, entry := range entries {
					entryPath := filepath.Join(appletPath, entry.Name())
					if err := os.RemoveAll(entryPath); err != nil {
						log.Printf("删除失败（可能被占用）: %s, %v", entryPath, err)
					}
				}
			}
		} else {
			// 重新创建空目录
			if err := os.MkdirAll(appletPath, 0755); err != nil {
				log.Printf("重新创建目录失败: %v", err)
			}
		}
	}

	// 清空任务表
	db := u.Db()
	if db == nil {
		return errors.New("数据库连接失败")
	}

	if err := db.Exec("DELETE FROM version_task").Error; err != nil {
		log.Printf("清空任务表失败: %v", err)
	}

	// 清空小程序信息表（可选，根据需求决定）
	if err := db.Exec("DELETE FROM mini_app_info").Error; err != nil {
		log.Printf("清空小程序信息表失败: %v", err)
	}

	log.Printf("Applet目录清空完成")
	return nil
}

func (u *UnWxapp) ClearDecompiled() error {
	config, err := u.getConfig()
	if err != nil {
		return err
	}

	if config.AppletPath == "" {
		return errors.New("Applet路径未设置")
	}

	appletPath, err := filepath.Abs(config.AppletPath)
	if err != nil {
		return fmt.Errorf("获取绝对路径失败: %v", err)
	}

	log.Printf("开始清空反编译文件: %s", appletPath)

	// 检查路径是否是users目录（新版本路径结构）
	normalizedPath := strings.ToLower(strings.TrimRight(appletPath, "\\/"))
	baseName := strings.ToLower(filepath.Base(appletPath))
	isUsersDir := baseName == "users" || strings.HasSuffix(normalizedPath, "users")

	var packagesDirs []string

	if isUsersDir {
		// log.Printf("检测到users目录，遍历所有用户目录下的packages目录")

		// 遍历所有用户目录，找到packages目录
		userEntries, err := os.ReadDir(appletPath)
		if err != nil {
			return fmt.Errorf("读取users目录失败: %v", err)
		}

		for _, userEntry := range userEntries {
			if !userEntry.IsDir() {
				continue
			}

			userDirName := userEntry.Name()
			// 构建 packages 目录路径
			packagesPath := filepath.Join(appletPath, userDirName, "applet", "packages")

			// 检查packages目录是否存在
			if isDir, err := u.isDir(packagesPath); err == nil && isDir {
				// log.Printf("找到用户 %s 的packages目录: %s", userDirName, packagesPath)
				packagesDirs = append(packagesDirs, packagesPath)
			}
		}
	} else {
		// 旧版本路径结构，直接使用传入的路径
		// log.Printf("使用旧版本路径结构: %s", appletPath)
		packagesDirs = append(packagesDirs, appletPath)
	}

	if len(packagesDirs) == 0 {
		// log.Printf("未找到任何packages目录")
		return nil
	}

	// 遍历所有packages目录，删除反编译文件
	for _, packagesDir := range packagesDirs {
		// log.Printf("处理packages目录: %s", packagesDir)

		// 获取所有小程序目录
		entries, err := os.ReadDir(packagesDir)
		if err != nil {
			log.Printf("读取packages目录失败: %s, %v", packagesDir, err)
			continue
		}

		// 遍历所有小程序目录
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			appID := entry.Name()
			if !strings.HasPrefix(appID, "wx") {
				continue
			}

			appPath := filepath.Join(packagesDir, appID)

			// 遍历版本目录
			versionEntries, err := os.ReadDir(appPath)
			if err != nil {
				log.Printf("读取版本目录失败: %s, %v", appPath, err)
				continue
			}

			for _, versionEntry := range versionEntries {
				if !versionEntry.IsDir() {
					continue
				}

				version := versionEntry.Name()
				versionPath := filepath.Join(appPath, version)

				// 获取动态包名
				packageNames, err := u.getPackageNames(versionPath)
				if err != nil {
					log.Printf("获取包名失败: %s, %v", versionPath, err)
					continue
				}

				// 删除所有包名的反编译输出目录
				for _, pkgName := range packageNames {
					decompiledDir := filepath.Join(versionPath, pkgName)
					if _, err := os.Stat(decompiledDir); err == nil {
						log.Printf("删除反编译目录: %s", decompiledDir)
						if err := os.RemoveAll(decompiledDir); err != nil {
							log.Printf("删除反编译目录失败: %s, %v", decompiledDir, err)
						}
					}
				}
			}
		}
	}

	// 清空任务表
	db := u.Db()
	if db == nil {
		return errors.New("数据库连接失败")
	}

	if err := db.Exec("DELETE FROM version_task").Error; err != nil {
		log.Printf("清空任务表失败: %v", err)
	}

	// 同时清空匹配结果
	if err := db.Exec("UPDATE version_task SET matched = ''").Error; err != nil {
		log.Printf("清空匹配结果失败: %v", err)
	}

	log.Printf("反编译文件清空完成")
	return nil
}

func (u *UnWxapp) GetAllMiniApp() ([]InfoToFront, error) {
	config, err := u.getConfig()
	if err != nil {
		return nil, err
	}

	if config.AppletPath == "" {
		return []InfoToFront{}, nil
	}

	miniPrograms, err := u.getAllMiniApp(config.AppletPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []InfoToFront{}, nil
		}
		return nil, err
	}

	u.resetAppidQueryQuotaNoticeForBatch()
	u.beginAppidKeyStatusBatch(len(splitAppidQueryAPIKeys(config.AppidQueryAPIKeys)))
	u.resetMPEmbedBatchGate()

	var result []InfoToFront
	for _, mp := range miniPrograms {
		versionsTask := u.createVersionTasks(mp)
		versionStatuses, err := u.handleVersionTasks(mp.AppID, versionsTask)
		if err != nil {
			log.Printf("处理版本任务失败: %v", err)
			continue
		}

		for _, st := range versionStatuses {
			pkgPath, err := u.findActualPackagePath(config.AppletPath, mp.AppID, st.Number)
			if err != nil {
				log.Printf("wxapkg 标签: 解析路径失败 %s %s: %v", mp.AppID, st.Number, err)
				continue
			}
			st.WxapkgTags = wxapkgTagsFromDir(pkgPath)
		}

		info, err := u.findOrCreateInfo(mp.AppID)
		if err != nil {
			log.Printf("获取小程序信息失败: %v", err)
			continue
		}

		frontItem := InfoToFront{
			AppID:      mp.AppID,
			UpdateDate: mp.UpdateDate,
			Info:       info,
			Versions:   versionStatuses,
		}

		result = append(result, frontItem)
	}

	return result, nil
}

func (u *UnWxapp) Decompile(item InfoToFront) error {

	// 参数验证
	if item.AppID == "" {
		return fmt.Errorf("AppID 不能为空")
	}

	if len(item.Versions) == 0 {
		return fmt.Errorf("版本列表不能为空")
	}

	for _, version := range item.Versions {
		if version.Number == "" {
			return fmt.Errorf("版本号不能为空")
		}
	}

	for _, version := range item.Versions {
		go func(appid, versionNum string) {

			task, err := u.findVersionTask(appid, versionNum)
			if err != nil {
				return
			}

			// 检查任务状态
			if !u.checkVersionTaskStatus(task, false) {
				return
			}

			// 使用Node.js工具进行反编译
			_, err = u.decompileWithNode(task)
			if err != nil {
				return
			}

		}(item.AppID, version.Number)
	}

	return nil
}

func (u *UnWxapp) GetMatchedString(appid, version string) ([]string, error) {
	task, err := u.findVersionTask(appid, version)
	if err != nil {
		return nil, err
	}
	return strings.Split(task.Matched, "\n"), nil
}

func (u *UnWxapp) SelectDirectory() (string, error) {
	return runtime.OpenDirectoryDialog(u.Ctx, runtime.OpenDialogOptions{
		Title: "选择小程序目录",
	})
}

// GetPackagePath 获取实际的package路径（用于前端打开文件夹等功能）
func (u *UnWxapp) GetPackagePath(appID, version string) (string, error) {
	config, err := u.getConfig()
	if err != nil {
		return "", fmt.Errorf("获取配置失败: %v", err)
	}

	if config.AppletPath == "" {
		return "", errors.New("Applet路径未设置")
	}

	// 使用findActualPackagePath查找实际的package路径
	packagePath, err := u.findActualPackagePath(config.AppletPath, appID, version)
	if err != nil {
		return "", err
	}

	return packagePath, nil
}

func (u *UnWxapp) ExtractSensitiveInfo(appID, version string) error {

	task, err := u.findVersionTask(appID, version)
	if err != nil {
		return fmt.Errorf("查找版本任务失败: %v", err)
	}

	// 检查反编译是否已完成
	if task.DecompileStatus != "Stopped" {
		return fmt.Errorf("请先完成反编译再提取敏感信息")
	}

	// 获取反编译后的文件列表
	config, err := u.getConfig()
	if err != nil {
		return fmt.Errorf("获取配置失败: %v", err)
	}

	// 查找实际的package路径（支持新版本users目录和旧版本路径）
	packagePath, err := u.findActualPackagePath(config.AppletPath, task.AppID, task.Version)
	if err != nil {
		return fmt.Errorf("查找package路径失败: %v", err)
	}

	log.Printf("提取敏感信息，使用package路径: %s", packagePath)

	// 动态获取包名
	packageNames, err := u.getPackageNames(packagePath)
	if err != nil {
		return fmt.Errorf("获取包名失败: %v", err)
	}

	if len(packageNames) == 0 {
		return fmt.Errorf("未找到有效的包名")
	}

	var files []string

	// 遍历所有包目录
	for _, pkgName := range packageNames {
		outputDir := filepath.Join(packagePath, pkgName)
		if _, err := os.Stat(outputDir); os.IsNotExist(err) {
			continue
		}

		if err := filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				files = append(files, path)
			}
			return nil
		}); err != nil {
			return fmt.Errorf("遍历反编译文件失败: %v", err)
		}
	}

	if len(files) == 0 {
		return fmt.Errorf("未找到反编译文件")
	}

	// 开始提取敏感信息

	go u.extractInfo(task, files)

	return nil
}

// CheckDecompileStatus 添加方法检查反编译状态
func (u *UnWxapp) CheckDecompileStatus(appID, version string) (string, error) {
	task, err := u.findVersionTask(appID, version)
	if err != nil {
		return "", err
	}

	config, err := u.getConfig()
	if err != nil {
		return "", err
	}

	// 查找实际的package路径（支持新版本users目录和旧版本路径）
	packagePath, err := u.findActualPackagePath(config.AppletPath, task.AppID, task.Version)
	if err != nil {
		return "", fmt.Errorf("查找package路径失败: %v", err)
	}

	// 动态获取包名
	packageNames, err := u.getPackageNames(packagePath)
	if err != nil {
		return "", err
	}

	// 统计文件数量
	var fileCount int

	// 遍历所有包目录
	for _, pkgName := range packageNames {
		outputDir := filepath.Join(packagePath, pkgName)
		if _, err := os.Stat(outputDir); os.IsNotExist(err) {
			continue
		}

		filepath.Walk(outputDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				fileCount++
			}
			return nil
		})
	}

	return fmt.Sprintf("数据库状态: %s, 文件数量: %d", task.DecompileStatus, fileCount), nil
}

// GetTaskStatus 获取单个任务的状态（用于轮询，避免刷新整个列表）
func (u *UnWxapp) GetTaskStatus(appID, version string) (*VersionTaskStatus, error) {
	task, err := u.findVersionTask(appID, version)
	if err != nil {
		return nil, err
	}

	status := &VersionTaskStatus{
		Number:          task.Version,
		DecompileStatus: task.DecompileStatus,
		MatchStatus:     task.MatchStatus,
		Message:         task.Message,
	}
	if config, err := u.getConfig(); err == nil && config.AppletPath != "" {
		if pkgPath, err := u.findActualPackagePath(config.AppletPath, appID, version); err == nil {
			status.WxapkgTags = wxapkgTagsFromDir(pkgPath)
		}
	}
	return status, nil
}

// ============ 内部方法 ============

type MiniProgram struct {
	AppID      string
	UpdateDate string
	Versions   []*Version
}

type Version struct {
	Number     string
	UpdateDate string
}

func (u *UnWxapp) getAllMiniApp(appletPath string) ([]*MiniProgram, error) {
	applet, err := filepath.Abs(appletPath)
	if err != nil {
		log.Printf("获取绝对路径失败: %v", err)
		return nil, err
	}

	// log.Printf("开始处理路径: %s", applet)

	var items []*MiniProgram
	var packagesDirs []string

	// 检查路径是否是users目录（新版本路径结构）
	// 路径格式: D:\Users\doki\AppData\Roaming\Tencent\xwechat\radium\users
	// 用户只需要输入到users目录，程序自动遍历所有用户目录并拼接applet\packages
	normalizedPath := strings.ToLower(strings.TrimRight(applet, "\\/"))
	baseName := strings.ToLower(filepath.Base(applet))
	isUsersDir := baseName == "users" || strings.HasSuffix(normalizedPath, "users")

	if isUsersDir {
		log.Printf("✓ 检测到users目录，自动遍历所有用户目录: %s", applet)

		// 读取users目录下的所有子目录（用户目录）
		userEntries, err := os.ReadDir(applet)
		if err != nil {
			log.Printf("✗ 读取users目录失败: %v", err)
			return nil, fmt.Errorf("读取users目录失败: %v", err)
		}

		log.Printf("找到 %d 个用户目录，开始自动拼接路径...", len(userEntries))

		// 先对用户目录进行排序，确保遍历顺序稳定
		sort.Slice(userEntries, func(i, j int) bool {
			return userEntries[i].Name() < userEntries[j].Name()
		})

		// 遍历所有用户目录，自动拼接 applet\packages 路径
		for _, userEntry := range userEntries {
			if !userEntry.IsDir() {
				continue
			}

			userDirName := userEntry.Name()
			// 自动拼接完整路径: users\用户目录\applet\packages
			userPackagesPath := filepath.Join(applet, userDirName, "applet", "packages")

			log.Printf("  检查用户: %s -> 拼接路径: %s", userDirName, userPackagesPath)

			// 检查packages目录是否存在
			if isDir, err := u.isDir(userPackagesPath); err == nil && isDir {
				log.Printf("  ✓ 找到packages目录: %s", userPackagesPath)
				packagesDirs = append(packagesDirs, userPackagesPath)
			} else {
				if err != nil {
					log.Printf("  ✗ 路径不存在或无法访问: %s (错误: %v)", userPackagesPath, err)
				} else {
					log.Printf("  ✗ packages目录不存在: %s", userPackagesPath)
				}
			}
		}

		log.Printf("✓ 遍历完成，共找到 %d 个有效的packages目录", len(packagesDirs))

		// 对packages目录列表进行排序，确保处理顺序稳定
		sort.Strings(packagesDirs)
	} else {
		// 旧版本路径结构，直接使用传入的路径（兼容旧版本）
		// log.Printf("使用旧版本路径结构（直接指向packages目录）: %s", applet)
		packagesDirs = append(packagesDirs, applet)
	}

	// 如果没找到任何packages目录，返回空列表
	if len(packagesDirs) == 0 {
		log.Printf("警告: 未找到任何packages目录，返回空列表")
		return items, nil
	}

	// 用于去重，因为不同用户目录可能包含相同的小程序
	appMap := make(map[string]*MiniProgram)

	// 遍历所有找到的packages目录
	for _, packagesDir := range packagesDirs {
		// log.Printf("处理第 %d/%d 个packages目录: %s", idx+1, len(packagesDirs), packagesDir)
		entries, err := os.ReadDir(packagesDir)
		if err != nil {
			log.Printf("✗ 读取packages目录失败: %s (错误: %v)", packagesDir, err)
			continue
		}
		// log.Printf("packages目录中有 %d 个条目", len(entries))

		// 按修改时间排序
		sort.Slice(entries, func(i, j int) bool {
			infoI, _ := entries[i].Info()
			infoJ, _ := entries[j].Info()
			return infoI.ModTime().After(infoJ.ModTime())
		})

		appCount := 0
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			appid := entry.Name()
			if !strings.HasPrefix(appid, "wx") {
				continue
			}

			versionsDir := filepath.Join(packagesDir, appid)
			if isDir, err := u.isDir(versionsDir); err != nil || !isDir {
				log.Printf("跳过，版本目录不存在或不是目录: %s", versionsDir)
				continue
			}

			versionEntries, err := os.ReadDir(versionsDir)
			if err != nil {
				log.Printf("读取版本目录失败: %s (错误: %v)", versionsDir, err)
				continue
			}
			appCount++

			info, err := entry.Info()
			if err != nil {
				continue
			}

			// 按修改时间排序
			sort.Slice(versionEntries, func(i, j int) bool {
				infoI, _ := versionEntries[i].Info()
				infoJ, _ := versionEntries[j].Info()
				return infoI.ModTime().After(infoJ.ModTime())
			})

			var versions []*Version
			for _, versionEntry := range versionEntries {
				versionPath := filepath.Join(versionsDir, versionEntry.Name())

				// 动态检查包文件
				hasWxapkg, err := u.hasWxapkgFiles(versionPath)
				if err != nil {
					continue
				}

				if hasWxapkg {
					info, err := versionEntry.Info()
					if err != nil {
						continue
					}
					version := &Version{
						Number:     info.Name(),
						UpdateDate: info.ModTime().Format("2006/01/02 15:04"),
					}
					versions = append(versions, version)
				}
			}

			// 如果已存在该小程序，合并版本列表（保留最新的修改时间）
			if existingApp, exists := appMap[appid]; exists {
				// 合并版本，去重
				versionMap := make(map[string]*Version)
				for _, v := range existingApp.Versions {
					versionMap[v.Number] = v
				}
				for _, v := range versions {
					if existingVersion, ok := versionMap[v.Number]; ok {
						// 如果版本已存在，解析日期字符串比较，保留更新日期更晚的
						existingTime, _ := time.Parse("2006/01/02 15:04", existingVersion.UpdateDate)
						newTime, _ := time.Parse("2006/01/02 15:04", v.UpdateDate)
						if newTime.After(existingTime) {
							versionMap[v.Number] = v
						}
					} else {
						versionMap[v.Number] = v
					}
				}
				// 重新构建版本列表
				mergedVersions := make([]*Version, 0, len(versionMap))
				for _, v := range versionMap {
					mergedVersions = append(mergedVersions, v)
				}
				// 按修改时间排序，如果时间相同则按版本号排序（确保排序稳定）
				sort.Slice(mergedVersions, func(i, j int) bool {
					timeI, _ := time.Parse("2006/01/02 15:04", mergedVersions[i].UpdateDate)
					timeJ, _ := time.Parse("2006/01/02 15:04", mergedVersions[j].UpdateDate)
					if !timeI.Equal(timeJ) {
						return timeI.After(timeJ)
					}
					// 如果时间相同，按版本号排序（确保排序稳定）
					return mergedVersions[i].Number < mergedVersions[j].Number
				})
				existingApp.Versions = mergedVersions
				// 更新修改时间为最新的
				existingTime, _ := time.Parse("2006/01/02 15:04", existingApp.UpdateDate)
				if info.ModTime().After(existingTime) {
					existingApp.UpdateDate = info.ModTime().Format("2006/01/02 15:04")
				}
			} else {
				// 新小程序，添加到map
				appMap[appid] = &MiniProgram{
					AppID:      appid,
					UpdateDate: info.ModTime().Format("2006/01/02 15:04"),
					Versions:   versions,
				}
			}
		}

		// log.Printf("从packages目录 %s 中找到 %d 个小程序", packagesDir, appCount)
	}

	// 将map转换为slice
	for _, app := range appMap {
		items = append(items, app)
	}

	// log.Printf("总共找到 %d 个不重复的小程序", len(items))

	// 按小程序目录最近修改时间降序，最新在上；相同时用 AppID 稳定排序
	if len(items) > 0 {
		sort.Slice(items, func(i, j int) bool {
			timeI, errI := time.Parse("2006/01/02 15:04", items[i].UpdateDate)
			timeJ, errJ := time.Parse("2006/01/02 15:04", items[j].UpdateDate)
			if errI != nil || errJ != nil {
				return items[i].AppID < items[j].AppID
			}
			if !timeI.Equal(timeJ) {
				return timeI.After(timeJ)
			}
			return items[i].AppID < items[j].AppID
		})
	}

	return items, nil
}

// 检查目录是否有wxapkg文件
func (u *UnWxapp) hasWxapkgFiles(dirPath string) (bool, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".wxapkg") {
			return true, nil
		}
	}
	return false, nil
}

// 获取包名列表
func (u *UnWxapp) getPackageNames(packagePath string) ([]string, error) {
	var packageNames []string

	// 默认包名
	defaultPackages := []string{"__APP__", "__PLUGINCODE__", "__WITHOUT_MULTI_PLUGINCODE__"}

	// 首先检查是否存在默认包名的wxapkg文件
	for _, pkg := range defaultPackages {
		wxapkgFile := filepath.Join(packagePath, pkg+".wxapkg")
		if u.fileExist(wxapkgFile) {
			packageNames = append(packageNames, pkg)
		}
	}

	// 如果找到了默认包名，直接返回
	if len(packageNames) > 0 {
		return packageNames, nil
	}

	// 扫描目录查找符合模式的大写字母包名
	entries, err := os.ReadDir(packagePath)
	if err != nil {
		return nil, err
	}

	// 正则表达式匹配 __大写字母__ 模式
	pattern := regexp.MustCompile(`^__[A-Z_]+__$`)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasSuffix(name, ".wxapkg") {
			// 去掉.wxapkg后缀
			baseName := strings.TrimSuffix(name, ".wxapkg")

			// 检查是否符合模式
			if pattern.MatchString(baseName) {
				packageNames = append(packageNames, baseName)
			}
		}
	}

	// 如果没有找到任何包，返回默认包名（即使文件不存在）
	if len(packageNames) == 0 {
		return defaultPackages, nil
	}

	return packageNames, nil
}

func (u *UnWxapp) createVersionTasks(miniProgram *MiniProgram) []*VersionTask {
	var tasks []*VersionTask
	for _, version := range miniProgram.Versions {
		task := &VersionTask{
			AppID:           miniProgram.AppID,
			Version:         version.Number,
			DecompileStatus: "Waiting",
			MatchStatus:     "Waiting",
			UpdateDate:      version.UpdateDate,
		}
		//log.Printf("创建版本任务: AppID=%s, Version=%s", task.AppID, task.Version)
		tasks = append(tasks, task)
	}
	return tasks
}

func (u *UnWxapp) handleVersionTasks(appID string, versionsTask []*VersionTask) ([]*VersionTaskStatus, error) {
	db := u.Db()
	if db == nil {
		return nil, errors.New("数据库连接失败")
	}

	// 查找现有任务
	var existingTasks []VersionTask
	if err := db.Where("app_id = ?", appID).Find(&existingTasks).Error; err != nil {
		return nil, err
	}

	existingMap := make(map[string]*VersionTask)
	for i := range existingTasks {
		existingMap[existingTasks[i].Version] = &existingTasks[i]
	}

	var newTasks []*VersionTask
	var statuses []*VersionTaskStatus

	for _, task := range versionsTask {
		if existing, ok := existingMap[task.Version]; ok {
			statuses = append(statuses, &VersionTaskStatus{
				Number:          existing.Version,
				DecompileStatus: existing.DecompileStatus,
				MatchStatus:     existing.MatchStatus,
				Message:         existing.Message,
			})
		} else {
			newTasks = append(newTasks, task)
			statuses = append(statuses, &VersionTaskStatus{
				Number:          task.Version,
				DecompileStatus: "Waiting",
				MatchStatus:     "Waiting",
			})
		}
	}

	// 保存新任务
	if len(newTasks) > 0 {
		if err := db.Create(newTasks).Error; err != nil {
			return nil, err
		}
	}

	return statuses, nil
}

func (u *UnWxapp) findOrCreateInfo(appID string) (*MiniAppInfo, error) {
	db := u.Db()
	if db == nil {
		return nil, errors.New("数据库连接失败")
	}

	var info MiniAppInfo
	if err := db.Where("app_id = ?", appID).First(&info).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 同步查询小程序信息，而不是异步
			newInfo, err := u.queryMimiAPPInfo(appID)
			if err != nil {
				if errors.Is(err, ErrAppidQueryQuotaExhausted) {
					log.Printf("AppidQuery 全部 API Key 不可用，appid %s 以原始 ID 展示", appID)
					u.maybeEmitAppidQueryQuotaNotice()
					return &MiniAppInfo{AppID: appID}, nil
				}
				if errors.Is(err, errAppidQueryNoAPIKeys) {
					return &MiniAppInfo{AppID: appID}, nil
				}
				// API查询失败（可能是API服务器问题），返回基本信息对象，不影响功能使用
				log.Printf("从API查询小程序信息失败（这是API服务器端的问题，不影响本地功能）: %v", err)
				return &MiniAppInfo{
					AppID: appID,
				}, nil
			}

			// 尝试保存到数据库，但失败不影响返回结果
			if err := db.Create(newInfo).Error; err != nil {
				// 即使保存失败，也返回查询到的信息
			}

			return newInfo, nil
		}
		return &MiniAppInfo{
			AppID: appID,
		}, nil
	}

	return &info, nil
}

func (u *UnWxapp) findVersionTask(appID, version string) (*VersionTask, error) {
	if appID == "" || version == "" {
		return nil, fmt.Errorf("appID 或 version 不能为空")
	}

	db := u.Db()
	if db == nil {
		return nil, errors.New("数据库连接失败")
	}

	var task VersionTask
	result := db.Where("app_id = ? AND version = ?", appID, version).First(&task)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			// 如果记录不存在，创建一个新的任务记录
			newTask := &VersionTask{
				AppID:           appID,
				Version:         version,
				DecompileStatus: "Waiting",
				MatchStatus:     "Waiting",
				Message:         "任务尚未开始",
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}

			if err := db.Create(newTask).Error; err != nil {

				return nil, fmt.Errorf("创建新任务失败: %v", err)
			}

			return newTask, nil
		}

		return nil, result.Error
	}

	return &task, nil
}

func (u *UnWxapp) updateVersionTask(task *VersionTask) error {
	db := u.Db()
	if db == nil {
		return errors.New("数据库连接失败")
	}

	// 确保更新时间
	task.UpdatedAt = time.Now()

	result := db.Save(task)
	if result.Error != nil {

		return result.Error
	}

	return nil
}

func (u *UnWxapp) checkVersionTaskStatus(task *VersionTask, allowedStoppedStatus bool) bool {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	// 如果任务正在运行，返回false
	if task.DecompileStatus == "Running" || task.MatchStatus == "Running" {

		return false
	}

	// 如果允许停止状态，则Stopped状态也可以执行
	if allowedStoppedStatus {
		result := task.DecompileStatus == "Stopped" || task.DecompileStatus == "Waiting" || task.DecompileStatus == "Error"
		return result
	}

	// 默认允许 Waiting 状态执行
	result := task.DecompileStatus == "Waiting" || task.DecompileStatus == "Error"
	return result
}

// findActualPackagePath 根据AppID和Version查找实际的package路径
// 支持新版本（users目录）和旧版本路径结构
func (u *UnWxapp) findActualPackagePath(appletPath, appID, version string) (string, error) {
	applet, err := filepath.Abs(appletPath)
	if err != nil {
		return "", fmt.Errorf("获取绝对路径失败: %v", err)
	}

	// 检查路径是否是users目录（新版本路径结构）
	normalizedPath := strings.ToLower(strings.TrimRight(applet, "\\/"))
	baseName := strings.ToLower(filepath.Base(applet))
	isUsersDir := baseName == "users" || strings.HasSuffix(normalizedPath, "users")

	if isUsersDir {
		// 遍历所有用户目录，找到包含该AppID和Version的packages路径
		userEntries, err := os.ReadDir(applet)
		if err != nil {
			return "", fmt.Errorf("读取users目录失败: %v", err)
		}

		for _, userEntry := range userEntries {
			if !userEntry.IsDir() {
				continue
			}

			userDirName := userEntry.Name()
			// 构建完整的package路径: users\用户目录\applet\packages\AppID\Version
			packagePath := filepath.Join(applet, userDirName, "applet", "packages", appID, version)

			// 检查路径是否存在
			if _, err := os.Stat(packagePath); err == nil {
				log.Printf("找到package路径: %s", packagePath)
				return packagePath, nil
			}
		}

		return "", fmt.Errorf("未找到AppID=%s, Version=%s的package路径", appID, version)
	} else {
		// 旧版本路径结构，直接拼接
		packagePath := filepath.Join(applet, appID, version)
		return packagePath, nil
	}
}

func (u *UnWxapp) decompileWithNode(task *VersionTask) ([]string, error) {

	// 更新任务状态为运行中
	task.DecompileStatus = "Running"
	task.Message = "反编译中"
	if err := u.updateVersionTask(task); err != nil {

		return nil, err
	}

	config, err := u.getConfig()
	if err != nil {

		task.DecompileStatus = "Error"
		task.Message = "获取配置失败"
		u.updateVersionTask(task)
		return nil, err
	}

	// 查找实际的package路径（支持新版本users目录和旧版本路径）
	packagePath, err := u.findActualPackagePath(config.AppletPath, task.AppID, task.Version)
	if err != nil {
		task.DecompileStatus = "Error"
		task.Message = fmt.Sprintf("查找package路径失败: %v", err)
		u.updateVersionTask(task)
		return nil, fmt.Errorf("查找package路径失败: %v", err)
	}

	log.Printf("使用package路径: %s", packagePath)

	baseDir := u.GetAppPath()
	unwxappDir := filepath.Join(baseDir, "tools", "Unwxapp")

	args := []string{"index.js", "wx"}
	args = append(args, packagePath)
	if task.AppID != "" {
		args = append(args, "-i", task.AppID)
	}
	args = append(args, "-f") // 启用格式化

	nodePath, err := exec.LookPath("node")
	if err != nil {
		// 尝试常见路径
		possiblePaths := []string{
			filepath.Join(os.Getenv("ProgramFiles"), "nodejs", "node.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "nodejs", "node.exe"),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Programs", "nodejs", "node.exe"),
			filepath.Join(os.Getenv("APPDATA"), "npm", "node.exe"),
		}

		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				nodePath = path
				break
			}
		}

		if nodePath == "" {
			task.DecompileStatus = "Error"
			task.Message = "无法找到node可执行文件"
			u.updateVersionTask(task)
			return nil, fmt.Errorf("无法找到node可执行文件")
		}
	}

	cmd := exec.Command(nodePath, args...)
	cmd.Dir = unwxappDir
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	output, err := cmd.CombinedOutput()
	_ = string(output)

	if err != nil {

		task.DecompileStatus = "Error"
		task.Message = fmt.Sprintf("反编译失败: %v", err)
		if err := u.updateVersionTask(task); err != nil {
			log.Printf("更新错误状态失败: %v", err)
		}
		return nil, err
	}

	// 动态获取包名
	packageNames, err := u.getPackageNames(packagePath)
	if err != nil {
		task.DecompileStatus = "Error"
		task.Message = "获取包名失败"
		if err := u.updateVersionTask(task); err != nil {
		}
		return nil, fmt.Errorf("获取包名失败: %v", err)
	}

	if len(packageNames) == 0 {
		task.DecompileStatus = "Error"
		task.Message = "未找到有效的包名"
		if err := u.updateVersionTask(task); err != nil {
		}
		return nil, fmt.Errorf("未找到有效的包名")
	}

	// 检查反编译输出目录是否存在
	var dirsToWalk []string
	for _, pkgName := range packageNames {
		outputDir := filepath.Join(packagePath, pkgName)
		if _, err := os.Stat(outputDir); !os.IsNotExist(err) {
			dirsToWalk = append(dirsToWalk, outputDir)
		}
	}

	// 如果所有目录都不存在，则报错
	if len(dirsToWalk) == 0 {
		task.DecompileStatus = "Error"
		task.Message = "反编译输出目录不存在"
		if err := u.updateVersionTask(task); err != nil {
		}
		return nil, fmt.Errorf("反编译输出目录不存在，检查了包名: %v", packageNames)
	}

	// 先对反编译结果中的内联 style 属性做一次规整，方便后续敏感信息提取
	for _, dir := range dirsToWalk {
		if _, err := u.compactInlineStylesInDir(dir); err != nil {
			log.Printf("规整内联样式失败（忽略继续）: %v", err)
		}
	}

	// 统计文件数量
	var files []string
	var fileCount int

	// 遍历存在的目录
	for _, dir := range dirsToWalk {
		if err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				files = append(files, path)
				fileCount++
			}
			return nil
		}); err != nil {
			// 即使遍历失败，也继续处理
		}
	}

	// 更新状态为完成
	task.DecompileStatus = "Stopped"
	task.Message = fmt.Sprintf("反编译完成，生成 %d 个文件", fileCount)
	if err := u.updateVersionTask(task); err != nil {

		return files, err
	}

	// 反编译完成后自动触发敏感信息提取
	if len(files) > 0 {

		go u.extractInfo(task, files)
	} else {

	}

	return files, nil
}

func (u *UnWxapp) extractInfo(task *VersionTask, files []string) {
	// 更新任务状态为运行中
	task.MatchStatus = "Running"
	task.Message = "敏感信息提取中"
	if err := u.updateVersionTask(task); err != nil {

		return
	}

	var results []string
	filteredFileExt := []string{".png", ".jpg", ".jpeg", ".wxapkg", ".br", ".gif", ".ico", ".svg", ".webp", ".woff", ".ttf"}

	// 获取规则
	rules, err := u.GetWechatRules()
	if err != nil {
		task.MatchStatus = "Error"
		task.Message = "获取规则失败: " + err.Error()
		u.updateVersionTask(task)
		return
	}

	// 预编译所有正则表达式
	var compiledRules []*regexp.Regexp
	for _, rule := range rules {
		re, err := regexp.Compile(rule)
		if err != nil {

			continue
		}
		compiledRules = append(compiledRules, re)
	}

	if len(compiledRules) == 0 {
		task.MatchStatus = "Error"
		task.Message = "所有规则编译失败"
		u.updateVersionTask(task)
		return
	}

	// 获取输出目录的基础路径
	config, err := u.getConfig()
	if err != nil {
		task.MatchStatus = "Error"
		task.Message = "获取配置失败: " + err.Error()
		u.updateVersionTask(task)
		return
	}

	// 查找实际的package路径（支持新版本users目录和旧版本路径）
	baseDir, err := u.findActualPackagePath(config.AppletPath, task.AppID, task.Version)
	if err != nil {
		task.MatchStatus = "Error"
		task.Message = "查找package路径失败: " + err.Error()
		u.updateVersionTask(task)
		return
	}

	// 按文件类型分类统计
	fileStats := make(map[string]int)

	// 处理每个文件
	processedFiles := 0
	for _, file := range files {
		// 跳过指定后缀文件
		ext := strings.ToLower(filepath.Ext(file))
		if u.stringSliceContain(filteredFileExt, ext) {
			continue
		}

		// 统计文件类型
		fileStats[ext]++

		// 获取相对路径（相对于基础目录）
		relativePath := file
		if rel, err := filepath.Rel(baseDir, file); err == nil {
			relativePath = rel
		}

		// 读取文件内容
		bytes, err := os.ReadFile(file)
		if err != nil {

			continue
		}

		content := string(bytes)
		fileHasMatches := false

		// 使用所有规则进行匹配
		for ruleIndex, re := range compiledRules {
			matches := re.FindAllString(content, -1)
			if len(matches) > 0 {
				fileHasMatches = true

				for _, match := range matches {
					match = strings.TrimSpace(match)
					if match != "" {
						// 创建详细的结果项，包含规则索引
						resultItem := fmt.Sprintf("[规则%d] %s -> %s", ruleIndex+1, relativePath, match)

						if !u.containsResult(results, resultItem) {
							results = append(results, resultItem)
						}
					}
				}
			}
		}

		// 如果文件有匹配项，添加文件分隔符以便阅读
		if fileHasMatches && len(results) > 0 && !strings.HasPrefix(results[len(results)-1], "====") {
			results = append(results, "====")
		}

		processedFiles++

		// 每处理10个文件更新一次进度
		if processedFiles%10 == 0 {
			task.Message = fmt.Sprintf("正在提取敏感信息 (%d/%d)", processedFiles, len(files))
			u.updateVersionTask(task)
		}
	}

	// 添加文件统计信息到结果开头
	if len(results) > 0 {
		statsHeader := []string{
			"=== 提取统计信息 ===",
			fmt.Sprintf("扫描文件总数: %d", len(files)),
			fmt.Sprintf("发现匹配的文件: %d", processedFiles),
		}

		// 添加文件类型统计
		for ext, count := range fileStats {
			statsHeader = append(statsHeader, fmt.Sprintf("%s 文件: %d", ext, count))
		}

		statsHeader = append(statsHeader, "====================", "")
		results = append(statsHeader, results...)
	}

	task.Matched = strings.Join(results, "\n")
	task.MatchStatus = "Stopped"
	task.Message = fmt.Sprintf("敏感信息提取完成，共找到 %d 个匹配项", len(results))

	if err := u.updateVersionTask(task); err != nil {
	}
}

// 检查结果是否已存在（基于相同文件相同内容的去重）
func (u *UnWxapp) containsResult(results []string, newResult string) bool {
	for _, result := range results {
		if result == newResult {
			return true
		}
	}
	return false
}

// ============ 工具方法 ============

func (u *UnWxapp) fileExist(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func (u *UnWxapp) isDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

func (u *UnWxapp) stringSliceContain(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// 启动自动反编译
func (u *UnWxapp) startAutoDecompile() {
	if u.autoDecompileTicker != nil {
		return // 已经在运行
	}

	u.autoDecompileTicker = time.NewTicker(10 * time.Second) // 每10秒检查一次
	u.stopAutoDecompile = make(chan bool)

	go func() {
		for {
			select {
			case <-u.autoDecompileTicker.C:
				u.autoDecompileCheck()
			case <-u.stopAutoDecompile:
				u.autoDecompileTicker.Stop()
				u.autoDecompileTicker = nil
				return
			}
		}
	}()
}

// 停止自动反编译
func (u *UnWxapp) stopAutoDecompileLoop() {
	if u.stopAutoDecompile != nil {
		u.stopAutoDecompile <- true
	}
}

// 自动反编译检查
func (u *UnWxapp) autoDecompileCheck() {
	u.mutex.Lock()
	defer u.mutex.Unlock()

	if !u.autoDecompile {
		return
	}

	config, err := u.getConfig()
	if err != nil {

		return
	}

	if config.AppletPath == "" {
		return
	}

	// 获取所有小程序
	miniPrograms, err := u.getAllMiniApp(config.AppletPath)
	if err != nil {
		return
	}

	// 处理每个小程序的最新版本
	for _, mp := range miniPrograms {
		if len(mp.Versions) == 0 {
			continue
		}

		// 获取最新版本（按时间排序的第一个）
		latestVersion := mp.Versions[0]

		// 检查任务状态
		task, err := u.findVersionTask(mp.AppID, latestVersion.Number)
		if err != nil {

			continue
		}

		// 如果状态是 Waiting，则自动开始反编译
		if task.DecompileStatus == "Waiting" {

			// 更新任务状态
			task.DecompileStatus = "Running"
			task.Message = "自动反编译中"
			if err := u.updateVersionTask(task); err != nil {

				continue
			}

			// 开始反编译
			go func(appID, version string) {
				_, err := u.decompileWithNode(task)
				if err != nil {

					return
				}
			}(mp.AppID, latestVersion.Number)
		}
	}
}
