package crack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"EasyTools/app/controller/system"
	"EasyTools/app/model"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gorm.io/gorm"
)

// TaskProgress 任务进度信息
type TaskProgress struct {
	TaskID        string    `json:"task_id"`
	TotalTargets  int       `json:"total_targets"`  // 总任务数
	CurrentTarget int       `json:"current_target"` // 已完成任务数
	StartTime     time.Time `json:"start_time"`
	Status        string    `json:"status"` // "running", "completed", "stopped"
}

// CrackController 密码爆破控制器
type CrackController struct {
	ctx          context.Context
	mu           sync.Mutex
	taskProgress map[string]*TaskProgress      // 任务进度跟踪
	taskCancels  map[string]context.CancelFunc // 任务取消函数
	system.Base
}

func NewCrackController() *CrackController {
	return &CrackController{
		taskProgress: make(map[string]*TaskProgress),
		taskCancels:  make(map[string]context.CancelFunc),
	}
}

func (c *CrackController) Startup(ctx context.Context) {
	c.ctx = ctx
	// 确保字典文件目录存在
	if err := EnsureDictFilesExist(); err != nil {
		c.sendLog("warning", fmt.Sprintf("创建字典目录失败: %v", err))
	}
	// 初始化数据库表
	db := c.Db()
	if db != nil {
		_ = db.AutoMigrate(&model.VulnerabilityInfo{})
		_ = db.AutoMigrate(&model.ProtocolConnection{})
	}
}

// saveCrackResult 保存密码爆破结果到数据库
func (c *CrackController) saveCrackResult(taskId, templateId, vulnName, vulnURL, severity, protocol, extract, request, response string) error {
	db := c.Db()
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}

	vuln := model.VulnerabilityInfo{
		TaskId:       taskId,
		TemplateID:   templateId,
		VulnName:     vulnName,
		VulnURL:      vulnURL,
		Severity:     severity,
		Protocol:     protocol,
		Extract:      extract,
		Request:      request,
		Response:     response,
		Description:  fmt.Sprintf("%s弱口令漏洞", protocol),
		Reference:    "",
		ResponseTime: "",
	}

	if err := db.Create(&vuln).Error; err != nil {
		c.sendLog("error", fmt.Sprintf("保存密码爆破结果到数据库失败: %v", err))
		return err
	}

	return nil
}

// 统一日志事件发送：通过 log_event 推到前端全局日志
func (c *CrackController) sendLog(level, message string) {
	if c.ctx == nil {
		return
	}
	eventData := map[string]interface{}{
		"level":   level,
		"message": message,
		"time":    time.Now().Format("15:04:05"),
		"source":  "密码爆破",
	}
	runtime.EventsEmit(c.ctx, "log_event", eventData)
}

// CrackConfig 爆破配置参数
type CrackConfig struct {
	Threads        int  `json:"threads"`          // 并发线程数
	Timeout        int  `json:"timeout"`          // 超时时间（毫秒）
	OnlyOneAccount bool `json:"only_one_account"` // 仅破解一个账户
}

// NewCrackScanenr 启动密码爆破扫描
// taskId: 任务ID
// target: 目标URL，格式如 "ssh://192.168.1.1:22" 或 "mysql://192.168.1.1:3306"
// usernames: 用户名字典
// passwords: 密码字典
// sshConfigJSON: SSH隧道配置JSON字符串（可选，如果为空则不使用SSH隧道）
// configJSON: 爆破配置JSON字符串（可选，包含threads、timeout、onlyOneAccount）
func (c *CrackController) NewCrackScanenr(taskId, target string, usernames, passwords []string, sshConfigJSON string, configJSON string) error {
	if c.ctx == nil {
		return fmt.Errorf("控制器未初始化")
	}

	// 解析爆破配置
	config := CrackConfig{
		Threads:        5,    // 默认5个并发
		Timeout:        3000, // 默认3秒超时
		OnlyOneAccount: true, // 默认仅破解一个账户
	}
	if configJSON != "" {
		if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
			c.sendLog("warning", fmt.Sprintf("解析爆破配置失败，使用默认配置: %v", err))
		}
	}
	// 参数校验和限制
	if config.Threads <= 0 {
		config.Threads = 5
	}
	if config.Threads > 100 {
		config.Threads = 100 // 最大100个并发
	}
	if config.Timeout <= 0 {
		config.Timeout = 3000
	}
	if config.Timeout > 30000 {
		config.Timeout = 30000 // 最大30秒
	}

	// 初始化任务进度（如果不存在）
	c.mu.Lock()
	if _, exists := c.taskProgress[taskId]; !exists {
		c.taskProgress[taskId] = &TaskProgress{
			TaskID:        taskId,
			TotalTargets:  0,
			CurrentTarget: 0,
			StartTime:     time.Now(),
			Status:        "running",
		}
	}
	// 增加总任务数
	c.taskProgress[taskId].TotalTargets++
	c.mu.Unlock()

	// 解析SSH配置
	var sshConfig *SSHConfig
	if sshConfigJSON != "" {
		sshConfig = &SSHConfig{}
		if err := json.Unmarshal([]byte(sshConfigJSON), sshConfig); err != nil {
			c.sendLog("warning", fmt.Sprintf("解析SSH配置失败，将使用直连方式: %v", err))
			sshConfig = nil
		} else if sshConfig.Enabled {
			c.sendLog("info", fmt.Sprintf("使用SSH隧道: %s@%s:%s", sshConfig.Username, sshConfig.Address, sshConfig.Port))
		}
	}

	// 解析URL
	u, err := url.Parse(target)
	if err != nil {
		c.sendLog("error", fmt.Sprintf("解析目标URL失败: %s, 错误: %v", target, err))
		return fmt.Errorf("解析目标URL失败: %v", err)
	}

	scheme := u.Scheme
	host := u.Host

	// 如果host中没有端口，根据协议添加默认端口
	if u.Port() == "" {
		defaultPorts := map[string]string{
			"ftp":           "21",
			"ssh":           "22",
			"telnet":        "23",
			"ldap":          "389",
			"smb":           "445",
			"rdp":           "3389",
			"mysql":         "3306",
			"mssql":         "1433",
			"postgresql":    "5432",
			"oracle":        "1521",
			"redis":         "6379",
			"mongodb":       "27017",
			"vnc":           "5900",
			"socks5":        "1080",
			"mqtt":          "1883",
			"memcached":     "11211",
			"activemq":      "61616",
			"rsync":         "873",
			"kafka":         "9092",
			"rmi":           "1099",
			"jdwp":          "5005",
			"adb":           "5555",
			"zookeeper":     "2181",
			"rabbitmq":      "5672",
			"elasticsearch": "9200",
			"etcd":          "2379",
		}
		if port, ok := defaultPorts[scheme]; ok {
			host = fmt.Sprintf("%s:%s", host, port)
		}
	}

	c.sendLog("info", fmt.Sprintf("开始爆破目标: %s://%s, 用户名数量=%d, 密码数量=%d, 并发数=%d, 超时=%dms", scheme, host, len(usernames), len(passwords), config.Threads, config.Timeout))

	// 根据协议调用对应的爆破函数
	go func() {
		defer func() {
			// 发送完成事件
			runtime.EventsEmit(c.ctx, fmt.Sprintf("crackDone::%s", target), nil)
			c.sendLog("info", fmt.Sprintf("爆破任务完成: %s://%s", scheme, host))
			// 更新任务进度
			c.updateTaskProgress(taskId, 1)
			// 清理取消函数
			c.mu.Lock()
			delete(c.taskCancels, fmt.Sprintf("%s::%s", taskId, target))
			c.mu.Unlock()
		}()

		// 创建控制上下文（用于停止爆破）
		ctrlCtx, cancel := context.WithCancel(c.ctx)
		// 存储取消函数，用于停止任务
		c.mu.Lock()
		c.taskCancels[fmt.Sprintf("%s::%s", taskId, target)] = cancel
		c.mu.Unlock()
		defer cancel()

		if scanFunc, ok := crackScanners[scheme]; ok {
			// 传递配置参数给爆破函数
			scanFunc(c.ctx, ctrlCtx, taskId, host, usernames, passwords, sshConfig, &config)
		} else {
			c.sendLog("warning", fmt.Sprintf("不支持的协议: %s，目标: %s", scheme, target))
			// 不支持的协议也更新进度
			c.updateTaskProgress(taskId, 1)
		}
	}()

	return nil
}

// crackFunc 定义爆破函数类型
type crackFunc func(ctx, ctrlCtx context.Context, taskId, host string, usernames, passwords []string, sshConfig *SSHConfig, config *CrackConfig)

// crackScanners 协议到爆破函数的映射
var crackScanners = map[string]crackFunc{
	"ftp":           FtpScan,
	"ssh":           SshScan,
	"telnet":        TelnetScan,
	"ldap":          LdapScan,
	"smb":           SmbScan,
	"rdp":           RdpScan,
	"mysql":         MysqlScan,
	"mssql":         MssqlScan,
	"postgresql":    PostgresScan,
	"oracle":        OracleScan,
	"redis":         RedisScan,
	"mongodb":       MongodbScan,
	"vnc":           VncScan,
	"socks5":        Socks5Scan,
	"mqtt":          MqttScan,
	"memcached":     MemcachedScan,
	"activemq":      ActiveMQScan,
	"rsync":         RsyncScan,
	"kafka":         KafkaScan,
	"rmi":           RmiScan,
	"jdwp":          JdwpScan,
	"adb":           AdbScan,
	"zookeeper":     ZookeeperScan,
	"rabbitmq":      RabbitMQScan,
	"elasticsearch": ElasticsearchScan,
	"etcd":          EtcdScan,
}

// LoadUsernameDictFromFile 从文件加载用户名字典
// protocol: 协议名，如 "ftp", "ssh" 等
// 返回: 用户名列表和错误
func (c *CrackController) LoadUsernameDictFromFile(protocol string) ([]string, error) {
	return LoadUsernameDict(protocol)
}

// LoadPasswordDictFromFile 从文件加载密码字典
// 返回: 密码列表和错误
func (c *CrackController) LoadPasswordDictFromFile() ([]string, error) {
	return LoadPasswordDict()
}

// GetUsernameDictPath 获取用户名字典文件路径
// protocol: 协议名
// 返回: 字典文件路径
func (c *CrackController) GetUsernameDictPath(protocol string) string {
	return GetUsernameDictPath(protocol)
}

// GetPasswordDictPath 获取密码字典文件路径
// 返回: 字典文件路径
func (c *CrackController) GetPasswordDictPath() string {
	return GetPasswordDictPath()
}

// ListAvailableUsernameDicts 列出所有可用的用户名字典文件
// 返回: 协议名列表
func (c *CrackController) ListAvailableUsernameDicts() ([]string, error) {
	return ListAvailableUsernameDicts()
}

// SaveUsernameDictToFile 保存用户名字典到文件
// protocol: 协议名，如 "ftp", "ssh" 等
// usernames: 用户名列表
// 返回: 错误
func (c *CrackController) SaveUsernameDictToFile(protocol string, usernames []string) error {
	return SaveUsernameDict(protocol, usernames)
}

// SavePasswordDictToFile 保存密码字典到文件
// passwords: 密码列表
// 返回: 错误
func (c *CrackController) SavePasswordDictToFile(passwords []string) error {
	return SavePasswordDict(passwords)
}

// ResetUsernameDictToDefault 恢复指定协议的用户名字典为默认值
// protocol: 协议名
// 返回: 错误
func (c *CrackController) ResetUsernameDictToDefault(protocol string) error {
	return ResetUsernameDictToDefault(protocol)
}

// ResetPasswordDictToDefault 恢复密码字典为默认值
// 返回: 错误
func (c *CrackController) ResetPasswordDictToDefault() error {
	return ResetPasswordDictToDefault()
}

// AddPasswordCrackResult 添加密码爆破结果
func (c *CrackController) AddPasswordCrackResult(connectionJSON string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var conn model.ProtocolConnection
	if err := json.Unmarshal([]byte(connectionJSON), &conn); err != nil {
		return fmt.Errorf("解析连接数据失败: %v", err)
	}

	// 如果 Credential 为空，尝试从 Username 和 Password 生成
	if conn.Credential == "" && conn.Username != "" && conn.Password != "" {
		conn.Credential = fmt.Sprintf("%s/%s", conn.Username, conn.Password)
	}

	// 设置默认值
	if conn.Status == "" {
		conn.Status = "success"
	}
	if conn.VulnURL == "" && conn.Host != "" && conn.Port != "" {
		conn.VulnURL = fmt.Sprintf("%s://%s:%s", strings.ToLower(conn.Protocol), conn.Host, conn.Port)
	}
	if conn.Message == "" {
		conn.Message = "密码爆破成功"
	}

	db := c.Db()
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}

	// 检查是否已存在相同的连接（避免重复）
	var existing model.ProtocolConnection
	result := db.Where("protocol = ? AND host = ? AND port = ? AND username = ? AND password = ?",
		conn.Protocol, conn.Host, conn.Port, conn.Username, conn.Password).First(&existing)

	if result.Error == nil {
		// 已存在，更新记录
		conn.ID = existing.ID
		if err := db.Model(&existing).Updates(&conn).Error; err != nil {
			c.sendLog("error", fmt.Sprintf("更新密码爆破结果失败: %v", err))
			return err
		}
		c.sendLog("info", fmt.Sprintf("已更新密码爆破结果: %s://%s:%s", conn.Protocol, conn.Host, conn.Port))
	} else if result.Error == gorm.ErrRecordNotFound {
		// 不存在，创建新记录
		if err := db.Create(&conn).Error; err != nil {
			c.sendLog("error", fmt.Sprintf("保存密码爆破结果失败: %v", err))
			return err
		}
		c.sendLog("info", fmt.Sprintf("已保存密码爆破结果: %s://%s:%s", conn.Protocol, conn.Host, conn.Port))
	} else {
		// 数据库错误
		return result.Error
	}

	return nil
}

// GetAllPasswordCrackResults 获取所有密码爆破结果
func (c *CrackController) GetAllPasswordCrackResults() ([]model.ProtocolConnection, error) {
	db := c.Db()
	if db == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}

	var connections []model.ProtocolConnection
	if err := db.Order("created_at DESC").Find(&connections).Error; err != nil {
		return nil, err
	}

	return connections, nil
}

// GetPasswordCrackResultsByProtocol 根据协议类型获取结果
func (c *CrackController) GetPasswordCrackResultsByProtocol(protocol string) ([]model.ProtocolConnection, error) {
	db := c.Db()
	if db == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}

	var connections []model.ProtocolConnection
	if err := db.Where("protocol = ?", protocol).Order("created_at DESC").Find(&connections).Error; err != nil {
		return nil, err
	}

	return connections, nil
}

// DeletePasswordCrackResult 删除密码爆破结果
func (c *CrackController) DeletePasswordCrackResult(id int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	db := c.Db()
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}

	if err := db.Delete(&model.ProtocolConnection{}, id).Error; err != nil {
		return err
	}

	c.sendLog("info", fmt.Sprintf("已删除密码爆破结果 ID: %d", id))
	return nil
}

// BatchDeletePasswordCrackResults 批量删除密码爆破结果
func (c *CrackController) BatchDeletePasswordCrackResults(ids []int) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	db := c.Db()
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}

	if len(ids) == 0 {
		return nil
	}

	if err := db.Delete(&model.ProtocolConnection{}, ids).Error; err != nil {
		return err
	}

	c.sendLog("info", fmt.Sprintf("已批量删除 %d 条密码爆破结果", len(ids)))
	return nil
}

// ClearAllPasswordCrackResults 清空所有密码爆破结果并重置自增ID
func (c *CrackController) ClearAllPasswordCrackResults() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	db := c.Db()
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}

	// 删除所有记录
	if err := db.Exec("DELETE FROM protocol_connections").Error; err != nil {
		return err
	}

	// 重置SQLite的自增序列（如果是SQLite数据库）
	// 注意：这只会重置SQLite的自增ID，其他数据库可能需要不同的方法
	if err := db.Exec("DELETE FROM sqlite_sequence WHERE name = 'protocol_connections'").Error; err != nil {
		// 如果表不存在或不是SQLite，忽略错误（可能是其他数据库类型）
		c.sendLog("info", "重置自增ID（可能不是SQLite数据库，已忽略）")
	} else {
		c.sendLog("info", "已重置自增ID")
	}

	c.sendLog("info", "已清空所有密码爆破结果并重置ID")
	return nil
}

// UpdatePasswordCrackResult 更新密码爆破结果
func (c *CrackController) UpdatePasswordCrackResult(id int, connectionJSON string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var conn model.ProtocolConnection
	if err := json.Unmarshal([]byte(connectionJSON), &conn); err != nil {
		return fmt.Errorf("解析连接数据失败: %v", err)
	}

	db := c.Db()
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}

	conn.ID = id
	if err := db.Model(&model.ProtocolConnection{}).Where("id = ?", id).Updates(&conn).Error; err != nil {
		return err
	}

	c.sendLog("info", fmt.Sprintf("已更新密码爆破结果 ID: %d", id))
	return nil
}

// ExportPasswordCrackResults 导出密码爆破结果到JSON文件
func (c *CrackController) ExportPasswordCrackResults(protocol string) (string, error) {
	var connections []model.ProtocolConnection
	var err error

	if protocol != "" && protocol != "all" {
		connections, err = c.GetPasswordCrackResultsByProtocol(protocol)
	} else {
		connections, err = c.GetAllPasswordCrackResults()
	}

	if err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(connections, "", "  ")
	if err != nil {
		return "", err
	}

	// 保存到文件
	exportDir := filepath.Join(c.GetAppPath(), "file")
	_ = os.MkdirAll(exportDir, 0755)

	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("password_crack_results_%s.json", timestamp)
	if protocol != "" && protocol != "all" {
		filename = fmt.Sprintf("password_crack_results_%s_%s.json", protocol, timestamp)
	}

	filepath := filepath.Join(exportDir, filename)
	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return "", err
	}

	c.sendLog("info", fmt.Sprintf("已导出 %d 条密码爆破结果到: %s", len(connections), filepath))
	return filepath, nil
}

// GetPasswordCrackStatistics 获取密码爆破统计信息
func (c *CrackController) GetPasswordCrackStatistics() (map[string]interface{}, error) {
	db := c.Db()
	if db == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}

	var total int64
	var success int64
	var failed int64

	db.Model(&model.ProtocolConnection{}).Count(&total)
	db.Model(&model.ProtocolConnection{}).Where("status = ?", "success").Count(&success)
	db.Model(&model.ProtocolConnection{}).Where("status = ?", "failed").Count(&failed)

	// 按协议类型统计
	type ProtocolCount struct {
		Protocol string
		Count    int64
	}
	var protocolCounts []ProtocolCount
	db.Model(&model.ProtocolConnection{}).
		Select("protocol, COUNT(*) as count").
		Group("protocol").
		Scan(&protocolCounts)

	stats := map[string]interface{}{
		"total":       total,
		"success":     success,
		"failed":      failed,
		"by_protocol": protocolCounts,
	}

	return stats, nil
}

// ExecuteCommandOnConnection 针对指定的爆破结果执行命令，并保存执行结果
// id: ProtocolConnection 记录ID
// command: 要执行的命令（例如 whoami、id 等）
// sshConfigJSON: SSH隧道配置JSON字符串（可选，如果为空则不使用SSH隧道）
// 返回：命令执行输出内容
func (c *CrackController) ExecuteCommandOnConnection(id int, command string, sshConfigJSON string) (string, error) {
	if c.ctx == nil {
		return "", fmt.Errorf("控制器未初始化")
	}

	command = strings.TrimSpace(command)
	if command == "" {
		return "", fmt.Errorf("命令不能为空")
	}

	db := c.Db()
	if db == nil {
		return "", fmt.Errorf("数据库未初始化")
	}

	var conn model.ProtocolConnection
	if err := db.First(&conn, id).Error; err != nil {
		return "", fmt.Errorf("未找到爆破结果记录 (id=%d): %v", id, err)
	}

	// 解析SSH配置（如果提供）
	var sshConfig *SSHConfig
	if sshConfigJSON != "" {
		sshConfig = &SSHConfig{}
		if err := json.Unmarshal([]byte(sshConfigJSON), sshConfig); err != nil {
			c.sendLog("warning", fmt.Sprintf("解析SSH配置失败，将使用直连方式: %v", err))
			sshConfig = nil
		} else if sshConfig.Enabled {
			c.sendLog("info", fmt.Sprintf("使用SSH隧道执行命令: %s@%s:%s", sshConfig.Username, sshConfig.Address, sshConfig.Port))
		}
	}

	// 根据协议执行命令
	proto := strings.ToLower(conn.Protocol)
	var result string
	var err error

	switch proto {
	case "ssh":
		// SSH协议应该通过"发送到简练助手"功能处理，不支持在这里执行命令
		// SSH支持密钥登录，需要通过简练助手进行连接
		return "", errors.New("SSH 协议请使用发送到简练助手功能进行连接和命令执行")

	case "docker":
		if conn.Host == "" || conn.Port == "" {
			return "", errors.New("Docker 连接信息不完整，无法执行命令")
		}
		result, err = ExecuteDockerCommand(conn.Host, conn.Port, sshConfig, command)

	case "kubernetes", "k8s":
		if conn.Host == "" || conn.Port == "" {
			return "", errors.New("Kubernetes 连接信息不完整，无法执行命令")
		}
		result, err = ExecuteKubernetesCommand(conn.Host, conn.Port, sshConfig, command)

	case "zookeeper":
		if conn.Host == "" || conn.Port == "" {
			return "", errors.New("Zookeeper 连接信息不完整，无法执行命令")
		}
		result, err = ExecuteZookeeperCommand(conn.Host, conn.Port, sshConfig, command)

	case "elasticsearch":
		if conn.Host == "" {
			return "", errors.New("Elasticsearch 连接信息不完整，无法执行命令")
		}
		result, err = ExecuteElasticsearchCommand(conn.Host, conn.Port, sshConfig, command)

	case "etcd":
		if conn.Host == "" {
			return "", errors.New("Etcd 连接信息不完整，无法执行命令")
		}
		result, err = ExecuteEtcdCommand(conn.Host, conn.Port, sshConfig, command)

	case "rabbitmq":
		if conn.Host == "" {
			return "", errors.New("RabbitMQ 连接信息不完整，无法执行命令")
		}
		// 这里使用爆破出的账号密码作为管理 API BasicAuth（如端口为 5672，会默认切到 15672）
		// 对于未授权访问的情况，password应该是空字符串，而不是"未授权访问"
		password := conn.Password
		if password == "未授权访问" {
			password = ""
		}
		result, err = ExecuteRabbitMQCommand(conn.Host, conn.Port, conn.Username, password, sshConfig, command)

	case "wmi":
		// WMI 仅支持 Windows 本机执行 wmic 远程查询，不走 SSH 隧道
		if conn.Host == "" {
			return "", errors.New("WMI 连接信息不完整，无法执行命令")
		}
		result, err = ExecuteWMICommand(conn.Host, conn.Username, conn.Password, command)

	case "mqtt":
		if conn.Host == "" {
			return "", errors.New("MQTT 连接信息不完整，无法执行命令")
		}
		result, err = ExecuteMqttCommand(conn.Host, conn.Port, conn.Username, conn.Password, sshConfig, command)

	case "telnet":
		if conn.Host == "" {
			return "", errors.New("Telnet 连接信息不完整，无法执行命令")
		}
		result, err = ExecuteTelnetCommand(conn.Host, conn.Port, conn.Username, conn.Password, sshConfig, command)

	default:
		return "", fmt.Errorf("当前协议暂不支持命令执行: %s", conn.Protocol)
	}

	if err != nil {
		return "", err
	}

	// 保存最近一次命令执行结果
	conn.CommandResult = result
	if saveErr := db.Save(&conn).Error; saveErr != nil {
		c.sendLog("warning", fmt.Sprintf("保存命令执行结果失败 (id=%d): %v", id, saveErr))
	}

	c.sendLog("info", fmt.Sprintf("命令执行成功: 协议=%s 主机=%s:%s 命令=%s", conn.Protocol, conn.Host, conn.Port, command))
	return result, nil
}

// CheckPortAlive 检查端口是否存活（用于密码爆破前的存活检测）
// host: 主机地址
// port: 端口号
// timeout: 超时时间（秒）
// 返回: 是否存活
func (c *CrackController) CheckPortAlive(host string, port int, timeout int) (bool, error) {
	if timeout <= 0 {
		timeout = 3 // 默认3秒
	}
	if timeout > 30 {
		timeout = 30 // 最大30秒
	}

	address := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", address, time.Duration(timeout)*time.Second)
	if err != nil {
		return false, nil // 连接失败，但不返回错误
	}
	defer conn.Close()
	return true, nil
}

// CheckPortsAlive 批量检查多个端口是否存活
// targets: 目标列表，格式为 "host:port" 或 "host"
// ports: 端口列表（如果targets中的host没有端口，则使用此列表）
// timeout: 超时时间（秒），使用前端传来的值，不再使用默认值
// 返回: 存活的目标列表（格式为 "host:port"）
func (c *CrackController) CheckPortsAlive(targets []string, ports []int, timeout int) ([]string, error) {
	// 使用前端传来的值，只做边界检查
	if timeout <= 0 {
		timeout = 1 // 最小1秒
	}
	if timeout > 30 {
		timeout = 30 // 最大30秒
	}

	aliveTargets := make([]string, 0)
	timeoutDuration := time.Duration(timeout) * time.Second

	// 创建通道用于并发检测
	type checkResult struct {
		target string
		alive  bool
	}
	resultChan := make(chan checkResult, len(targets)*len(ports))

	// 并发检测所有目标和端口
	var wg sync.WaitGroup
	for _, target := range targets {
		target = strings.TrimSpace(target)
		if target == "" {
			continue
		}

		// 解析目标（可能包含端口）
		var host string
		var targetPorts []int

		if strings.Contains(target, ":") {
			// 包含端口
			parts := strings.Split(target, ":")
			host = parts[0]
			if len(parts) > 1 {
				if p, err := strconv.Atoi(parts[1]); err == nil {
					targetPorts = []int{p}
				} else {
					continue // 端口格式错误，跳过
				}
			}
		} else {
			// 不包含端口，使用提供的端口列表
			host = target
			targetPorts = ports
		}

		if len(targetPorts) == 0 {
			continue
		}

		// 对每个端口进行检测
		for _, port := range targetPorts {
			wg.Add(1)
			go func(h string, p int) {
				defer wg.Done()
				address := net.JoinHostPort(h, strconv.Itoa(p))
				conn, err := net.DialTimeout("tcp", address, timeoutDuration)
				alive := err == nil
				if alive {
					conn.Close()
				}
				resultChan <- checkResult{
					target: fmt.Sprintf("%s:%d", h, p),
					alive:  alive,
				}
			}(host, port)
		}
	}

	// 等待所有检测完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集存活的目标
	for result := range resultChan {
		if result.alive {
			aliveTargets = append(aliveTargets, result.target)
		}
	}

	return aliveTargets, nil
}

// updateTaskProgress 更新任务进度
// taskId: 任务ID
// increment: 完成的任务数增量（通常为1）
func (c *CrackController) updateTaskProgress(taskId string, increment int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if progress, exists := c.taskProgress[taskId]; exists {
		progress.CurrentTarget += increment
		// 如果已完成所有任务，标记为完成
		if progress.TotalTargets > 0 && progress.CurrentTarget >= progress.TotalTargets {
			progress.Status = "completed"
			// 发送任务完成事件
			if c.ctx != nil {
				runtime.EventsEmit(c.ctx, "crackTaskCompleted", map[string]interface{}{
					"task_id": taskId,
					"status":  "completed",
				})
			}
		}
	}
}

// GetTaskProgress 获取任务进度信息
// taskId: 任务ID
// 返回: 任务进度信息，如果任务不存在则返回nil
func (c *CrackController) GetTaskProgress(taskId string) (*TaskProgress, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if progress, exists := c.taskProgress[taskId]; exists {
		// 返回副本，避免并发问题
		return &TaskProgress{
			TaskID:        progress.TaskID,
			TotalTargets:  progress.TotalTargets,
			CurrentTarget: progress.CurrentTarget,
			StartTime:     progress.StartTime,
			Status:        progress.Status,
		}, nil
	}

	return nil, fmt.Errorf("任务不存在: %s", taskId)
}

// ClearTaskProgress 清除任务进度（用于清理已完成的任务）
// taskId: 任务ID
func (c *CrackController) ClearTaskProgress(taskId string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.taskProgress, taskId)
	// 清理相关的取消函数
	for key := range c.taskCancels {
		if strings.HasPrefix(key, taskId+"::") {
			delete(c.taskCancels, key)
		}
	}
}

// StopCrackTask 停止指定的爆破任务
// taskId: 任务ID
func (c *CrackController) StopCrackTask(taskId string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 更新任务状态
	if progress, exists := c.taskProgress[taskId]; exists {
		progress.Status = "stopped"
	}

	// 取消所有相关的任务
	cancelled := 0
	for key, cancel := range c.taskCancels {
		if strings.HasPrefix(key, taskId+"::") {
			cancel()
			delete(c.taskCancels, key)
			cancelled++
		}
	}

	if cancelled > 0 {
		c.sendLog("info", fmt.Sprintf("已停止任务 %s 的 %d 个子任务", taskId, cancelled))
		return nil
	}

	return fmt.Errorf("任务 %s 不存在或已停止", taskId)
}
