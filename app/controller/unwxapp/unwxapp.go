//go:build linux || darwin

package unwxapp

import (
	"EasyTools/app/controller/system"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tidwall/gjson"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
)

var ansiRegex = regexp.MustCompile(`\x1B\[[0-9;]*[mKJH]`)

// 微信配置表
type WechatConfig struct {
	ID            int    `gorm:"primarykey" json:"id"`
	AppletPath    string `json:"applet_path" gorm:"column:applet_path"`
	Rules         string `json:"rules" gorm:"column:rules"` // JSON格式存储规则
	AutoDecompile bool   `json:"auto_decompile" gorm:"column:auto_decompile"`
}

func (*WechatConfig) TableName() string {
	return "wechat_config"
}

// 小程序信息表
type MiniAppInfo struct {
	ID            int       `gorm:"primarykey" json:"id"`
	AppID         string    `json:"app_id" gorm:"column:app_id;index"`
	Nickname      string    `json:"nickname" gorm:"column:nickname"`
	Username      string    `json:"username" gorm:"column:username"`
	Description   string    `json:"description" gorm:"column:description"`
	Avatar        string    `json:"avatar" gorm:"column:avatar"`
	UsesCount     string    `json:"uses_count" gorm:"column:uses_count"`
	PrincipalName string    `json:"principal_name" gorm:"column:principal_name"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (*MiniAppInfo) TableName() string {
	return "mini_app_info"
}

// 版本任务表
type VersionTask struct {
	ID              int       `gorm:"primarykey" json:"id"`
	AppID           string    `json:"app_id" gorm:"column:app_id;index"`
	Version         string    `json:"version" gorm:"column:version"`
	DecompileStatus string    `json:"decompile_status" gorm:"column:decompile_status"`
	MatchStatus     string    `json:"match_status" gorm:"column:match_status"`
	Message         string    `json:"message" gorm:"column:message"`
	Matched         string    `json:"matched" gorm:"column:matched;type:text"`
	UpdateDate      string    `json:"update_date" gorm:"column:update_date"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (*VersionTask) TableName() string {
	return "version_task"
}

type VersionTaskStatus struct {
	Number          string `json:"Number"`
	DecompileStatus string `json:"DecompileStatus"`
	MatchStatus     string `json:"MatchStatus"`
	Message         string `json:"Message"`
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
}

func NewUnWxapp() *UnWxapp {
	app := &UnWxapp{
		http: &http.Client{},
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

	if u.checkViaShell() {
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
		"/usr/local/bin/node",
		"/opt/homebrew/bin/node",
		"/usr/bin/node",
		"/opt/local/bin/node", // MacPorts
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			log.Printf("✓ 在 %s 检测到 Node.js\n", path)
			return true
		}
	}
	return false
}

func (u *UnWxapp) checkViaShell() bool {
	// 获取当前用户
	_, err := user.Current()
	if err != nil {
		return false
	}

	shells := []struct {
		name string
		rc   string
	}{
		{"zsh", ".zshrc"},
		{"bash", ".bash_profile"},
		{"bash", ".bashrc"},
	}
	home, _ := os.UserHomeDir()

	for _, shell := range shells {
		cmd := exec.Command(shell.name, "-c", "source "+filepath.Join(home, shell.rc)+" 2>/dev/null; which node")
		if output, err := cmd.Output(); err == nil {
			path := strings.TrimSpace(string(output))
			if path != "" && !strings.Contains(path, "not found") && !strings.Contains(path, "no ") {
				log.Printf("✓ 通过 %s 检测到 Node.js: %s\n", shell.name, path)
				return true
			}
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

	log.Printf("开始清空Applet目录: %s", config.AppletPath)

	// 删除整个小程序目录
	if err := os.RemoveAll(config.AppletPath); err != nil {
		return fmt.Errorf("删除Applet目录失败: %v", err)
	}

	// 重新创建空目录
	if err := os.MkdirAll(config.AppletPath, 0755); err != nil {
		return fmt.Errorf("重新创建Applet目录失败: %v", err)
	}

	// 清空任务表
	db := u.Db()
	if db == nil {
		return errors.New("数据库连接失败")
	}

	if err := db.Exec("DELETE FROM version_task").Error; err != nil {
		return err
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

	// 获取所有小程序目录
	entries, err := os.ReadDir(config.AppletPath)
	if err != nil {
		return err
	}

	// 遍历所有小程序目录
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		appID := entry.Name()
		appPath := filepath.Join(config.AppletPath, appID)

		// 遍历版本目录
		versionEntries, err := os.ReadDir(appPath)
		if err != nil {
			continue
		}

		for _, versionEntry := range versionEntries {
			if !versionEntry.IsDir() {
				continue
			}

			version := versionEntry.Name()
			// 获取动态包名
			packageNames, err := u.getPackageNames(filepath.Join(appPath, version))
			if err != nil {
				continue
			}

			// 删除所有包名的反编译输出目录
			for _, pkgName := range packageNames {
				decompiledDir := filepath.Join(appPath, version, pkgName)
				if _, err := os.Stat(decompiledDir); err == nil {
					log.Printf("删除反编译目录: %s", decompiledDir)
					if err := os.RemoveAll(decompiledDir); err != nil {
						log.Printf("删除反编译目录失败: %s, %v", decompiledDir, err)
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
		return err
	}

	// 同时清空匹配结果
	if err := db.Exec("UPDATE version_task SET matched = ''").Error; err != nil {
		log.Printf("清空匹配结果失败: %v", err)
	}

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

	var result []InfoToFront
	for _, mp := range miniPrograms {
		versionsTask := u.createVersionTasks(mp)
		versionStatuses, err := u.handleVersionTasks(mp.AppID, versionsTask)
		if err != nil {
			log.Printf("处理版本任务失败: %v", err)
			continue
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

	// 动态获取包名
	packagePath := filepath.Join(config.AppletPath, task.AppID, task.Version)
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

	// 动态获取包名
	packagePath := filepath.Join(config.AppletPath, task.AppID, task.Version)
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
		return nil, err
	}

	var items []*MiniProgram
	entries, err := os.ReadDir(applet)
	if err != nil {
		return nil, err
	}

	// 按修改时间排序
	sort.Slice(entries, func(i, j int) bool {
		infoI, _ := entries[i].Info()
		infoJ, _ := entries[j].Info()
		return infoI.ModTime().After(infoJ.ModTime())
	})

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		appid := entry.Name()
		if !strings.HasPrefix(appid, "wx") {
			continue
		}

		versionsDir := filepath.Join(applet, appid)
		if isDir, err := u.isDir(versionsDir); err != nil || !isDir {
			continue
		}

		versionEntries, err := os.ReadDir(versionsDir)
		if err != nil {

			continue
		}

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

		items = append(items, &MiniProgram{
			AppID:      appid,
			UpdateDate: info.ModTime().Format("2006/01/02 15:04"),
			Versions:   versions,
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
				log.Printf("查询小程序信息失败: %v", err)
				// 即使查询失败也返回一个基本的信息对象
				return &MiniAppInfo{
					AppID: appID,
				}, nil
			}

			// 保存到数据库
			if err := db.Create(newInfo).Error; err != nil {
				log.Printf("保存小程序信息失败: %v", err)
			}

			return newInfo, nil
		}
		return nil, err
	}

	return &info, nil
}

func (u *UnWxapp) queryMiniAPPInfoAsync(appID string) {
	info, err := u.queryMimiAPPInfo(appID)
	if err != nil {
		log.Printf("查询小程序信息失败: %v", err)
		return
	}

	db := u.Db()
	if db == nil {
		return
	}

	// 使用事务和锁来避免并发插入
	err = db.Transaction(func(tx *gorm.DB) error {
		// 先检查是否已存在
		var existing MiniAppInfo
		if err := tx.Where("app_id = ?", appID).First(&existing).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				// 不存在则创建
				return tx.Create(info).Error
			}
			return err
		}
		// 已存在则更新
		return tx.Model(&MiniAppInfo{}).Where("app_id = ?", appID).Updates(info).Error
	})

	if err != nil {
		log.Printf("保存小程序信息失败: %v", err)
	}
}

func (u *UnWxapp) queryMimiAPPInfo(appid string) (*MiniAppInfo, error) {
	req, err := http.NewRequest("POST", "https://kainy.cn/api/weapp/info/", strings.NewReader("appid="+appid))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	response, err := u.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	bytes, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	code := gjson.Get(string(bytes), "code").Int()
	msg := gjson.Get(string(bytes), "error").String()
	if code != 0 && code != 2 {
		return nil, errors.New(msg)
	}

	return &MiniAppInfo{
		AppID:         appid,
		Nickname:      gjson.Get(string(bytes), "data.nickname").String(),
		Username:      gjson.Get(string(bytes), "data.username").String(),
		Description:   gjson.Get(string(bytes), "data.description").String(),
		Avatar:        gjson.Get(string(bytes), "data.avatar").String(),
		UsesCount:     gjson.Get(string(bytes), "data.uses_count").String(),
		PrincipalName: gjson.Get(string(bytes), "data.principal_name").String(),
	}, nil
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

	// 构建Node.js工具参数
	packagePath := filepath.Join(config.AppletPath, task.AppID, task.Version)

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
		// 方法2: 使用常见路径
		possiblePaths := []string{
			"/usr/local/bin/node",
			"/opt/homebrew/bin/node", // Apple Silicon Homebrew路径
			"/usr/bin/node",
		}

		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				nodePath = path
				break
			}
		}

		if nodePath == "" {
			return nil, fmt.Errorf("无法找到node可执行文件")
		}
	}

	cmd := exec.Command(nodePath, args...)
	cmd.Dir = unwxappDir

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
	config, _ := u.getConfig()
	baseDir := filepath.Join(config.AppletPath, task.AppID, task.Version)

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

// 按文件名对结果进行排序
func (u *UnWxapp) sortResultsByFilename(results []string) []string {
	sort.Slice(results, func(i, j int) bool {
		// 提取文件名部分进行比较
		fileI := strings.Split(results[i], " -> ")[0]
		fileJ := strings.Split(results[j], " -> ")[0]
		return fileI < fileJ
	})
	return results
}

// ============ 工具方法 ============

func stripANSI(input string) string {
	return ansiRegex.ReplaceAllString(input, "")
}

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

func (u *UnWxapp) removeEmptyAndDuplicateString(slice []string) []string {
	keys := make(map[string]bool)
	var result []string
	for _, item := range slice {
		item = strings.TrimSpace(item)
		if item != "" && !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}
	return result
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
