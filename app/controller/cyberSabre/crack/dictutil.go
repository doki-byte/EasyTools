package crack

import (
	"EasyTools/app/controller/system"
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// 默认用户名字典（来自slack，并补充其他协议）
var defaultUserDict = map[string][]string{
	"ftp":        {"ftp", "admin", "www", "web", "root", "db", "wwwroot", "data"},
	"mysql":      {"root", "mysql"},
	"mssql":      {"sa", "sql"},
	"smb":        {"administrator", "admin", "guest"},
	"rdp":        {"administrator", "admin", "guest"},
	"postgresql": {"postgres", "admin"},
	"ssh":        {"root", "admin"},
	"mongodb":    {"root", "admin"},
	"oracle":     {"sys", "system", "admin", "test", "web", "orcl"},
	"ldap":       {"admin", "administrator", "root"},
	"socks5":     {"admin", "administrator"},
	"mqtt":       {"admin", "administrator"},
	"vnc":        {"admin", "administrator", "root"},
	"telnet":     {"root", "admin"},
	"activemq":   {"admin", "root", "activemq", "system", "user"},
	"kafka":      {"admin", "kafka", "root", "test"},
	"rsync":      {"rsync", "root", "admin", "backup"},
	"redis":      {"redis", "root", "admin", "default"}, // Redis通常不需要用户名，但某些配置可能需要
	"memcached":  {"memcached", "root", "admin"},        // Memcached通常不需要用户名
	"rmi":        {"rmi", "admin", "root"},              // RMI通常不需要用户名
	"jdwp":       {"jdwp", "admin", "root"},             // JDWP通常不需要用户名
	"adb":        {"adb", "root", "shell", "admin"},     // ADB通常不需要用户名
	"elasticsearch": {"elastic", "admin", "kibana", "logstash", "apm_system", "beats_system", "remote_monitoring_user"}, // Elasticsearch默认用户名
	"etcd":       {"root", "admin", "etcd"},             // Etcd通常不需要用户名，但某些配置可能需要
	"zookeeper":  {"admin", "zookeeper", "root"},        // Zookeeper通常不需要用户名，但某些配置可能需要
	"rabbitmq":   {"guest", "admin", "rabbitmq", "root"}, // RabbitMQ默认用户名
}

// 默认密码字典（来自slack，并补充常见服务默认密码）
var defaultPasswords = []string{
	"123456", "admin", "admin123", "root", "", "pass123", "pass@123", "password", "123123", "654321", "111111", "123", "1", "admin@123", "Admin@123", "admin123!@#", "{user}", "{user}1", "{user}111", "{user}123", "{user}@123", "{user}_123", "{user}#123", "{user}@111", "{user}@2019", "{user}@123#4", "P@ssw0rd!", "P@ssw0rd", "Passw0rd", "qwe123", "12345678", "test", "test123", "123qwe", "123qwe!@#", "123456789", "123321", "666666", "a123456.", "123456~a", "123456!a", "000000", "1234567890", "8888888", "!QAZ2wsx", "1qaz2wsx", "abc123", "abc123456", "1qaz@WSX", "a11111", "a12345", "Aa1234", "Aa1234.", "Aa12345", "a123456", "a123123", "Aa123123", "Aa123456", "Aa12345.", "sysadmin", "system", "1qaz!QAZ", "2wsx@WSX", "qwe123!@#", "Aa123456!", "A123456s!", "sa123456", "1q2w3e", "Charge123", "Aa123456789",
	// Elasticsearch常见默认密码
	"changeme", "elastic", "kibana", "logstash",
	// RabbitMQ常见默认密码
	"guest",
	// 其他常见服务默认密码
	"etcd", "zookeeper", "rabbitmq",
}

// LoadUsernameDict 加载用户名字典文件
// protocol: 协议名，如 "ftp", "ssh", "telnet" 等
// 返回: 用户名列表和错误
func LoadUsernameDict(protocol string) ([]string, error) {
	dictPath := system.GetUsernameDictPath(protocol)

	// 如果文件不存在，返回空列表（不报错）
	if _, err := os.Stat(dictPath); os.IsNotExist(err) {
		return []string{}, nil
	}

	file, err := os.Open(dictPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var usernames []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// 跳过空行和注释行
		if line != "" && !strings.HasPrefix(line, "#") {
			usernames = append(usernames, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return usernames, nil
}

// LoadPasswordDict 加载密码字典文件
// 返回: 密码列表和错误
func LoadPasswordDict() ([]string, error) {
	dictPath := system.GetPasswordDictPath()

	// 如果文件不存在，返回空列表（不报错）
	if _, err := os.Stat(dictPath); os.IsNotExist(err) {
		return []string{}, nil
	}

	file, err := os.Open(dictPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var passwords []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// 跳过空行和注释行
		if line != "" && !strings.HasPrefix(line, "#") {
			passwords = append(passwords, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return passwords, nil
}

// LoadUsernameDictWithFallback 加载用户名字典，如果文件不存在则返回默认值
// protocol: 协议名
// defaultUsernames: 如果文件不存在时返回的默认用户名列表
func LoadUsernameDictWithFallback(protocol string, defaultUsernames []string) []string {
	usernames, err := LoadUsernameDict(protocol)
	if err != nil || len(usernames) == 0 {
		return defaultUsernames
	}
	return usernames
}

// LoadPasswordDictWithFallback 加载密码字典，如果文件不存在则返回默认值
// defaultPasswords: 如果文件不存在时返回的默认密码列表
func LoadPasswordDictWithFallback(defaultPasswords []string) []string {
	passwords, err := LoadPasswordDict()
	if err != nil || len(passwords) == 0 {
		return defaultPasswords
	}
	return passwords
}

// EnsureDictFilesExist 确保字典文件目录存在，并创建默认字典文件
func EnsureDictFilesExist() error {
	// 确保目录存在
	if err := system.EnsureDictDir(); err != nil {
		return err
	}

	baseDir := system.GetAppBaseDir()
	usernameDictDir := filepath.Join(baseDir, "tools", "dict", "username")
	passwordDictDir := filepath.Join(baseDir, "tools", "dict")

	// 创建用户名字典文件
	for protocol, usernames := range defaultUserDict {
		dictPath := filepath.Join(usernameDictDir, protocol+".txt")
		// 如果文件不存在，创建默认字典文件
		if _, err := os.Stat(dictPath); os.IsNotExist(err) {
			content := strings.Join(usernames, "\n")
			if err := os.WriteFile(dictPath, []byte(content), 0644); err != nil {
				return err
			}
		}
	}

	// 创建密码字典文件
	passwordDictPath := filepath.Join(passwordDictDir, "password.txt")
	if _, err := os.Stat(passwordDictPath); os.IsNotExist(err) {
		content := strings.Join(defaultPasswords, "\n")
		if err := os.WriteFile(passwordDictPath, []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
}

// GetUsernameDictPath 获取用户名字典文件路径（供外部调用）
func GetUsernameDictPath(protocol string) string {
	return system.GetUsernameDictPath(protocol)
}

// GetPasswordDictPath 获取密码字典文件路径（供外部调用）
func GetPasswordDictPath() string {
	return system.GetPasswordDictPath()
}

// SaveUsernameDict 保存用户名字典到文件
// protocol: 协议名
// usernames: 用户名列表
func SaveUsernameDict(protocol string, usernames []string) error {
	// 确保目录存在
	if err := EnsureDictFilesExist(); err != nil {
		return err
	}

	dictPath := system.GetUsernameDictPath(protocol)
	content := strings.Join(usernames, "\n")
	return os.WriteFile(dictPath, []byte(content), 0644)
}

// SavePasswordDict 保存密码字典到文件
// passwords: 密码列表
func SavePasswordDict(passwords []string) error {
	// 确保目录存在
	if err := EnsureDictFilesExist(); err != nil {
		return err
	}

	dictPath := system.GetPasswordDictPath()
	content := strings.Join(passwords, "\n")
	return os.WriteFile(dictPath, []byte(content), 0644)
}

// ListAvailableUsernameDicts 列出所有可用的用户名字典文件
// 返回: 协议名列表
func ListAvailableUsernameDicts() ([]string, error) {
	baseDir := system.GetAppBaseDir()
	usernameDictDir := filepath.Join(baseDir, "tools", "dict", "username")

	// 如果目录不存在，返回空列表
	if _, err := os.Stat(usernameDictDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	entries, err := os.ReadDir(usernameDictDir)
	if err != nil {
		return nil, err
	}

	var protocols []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".txt") {
			// 移除 .txt 后缀，获取协议名
			protocol := strings.TrimSuffix(entry.Name(), ".txt")
			protocols = append(protocols, protocol)
		}
	}

	return protocols, nil
}

// ResetUsernameDictToDefault 恢复指定协议的用户名字典为默认值
// protocol: 协议名
func ResetUsernameDictToDefault(protocol string) error {
	// 确保目录存在
	if err := EnsureDictFilesExist(); err != nil {
		return err
	}

	// 获取默认用户名字典
	defaultUsernames, exists := defaultUserDict[protocol]
	if !exists {
		return fmt.Errorf("协议 %s 没有默认字典", protocol)
	}

	dictPath := system.GetUsernameDictPath(protocol)
	content := strings.Join(defaultUsernames, "\n")
	return os.WriteFile(dictPath, []byte(content), 0644)
}

// ResetPasswordDictToDefault 恢复密码字典为默认值
func ResetPasswordDictToDefault() error {
	// 确保目录存在
	if err := EnsureDictFilesExist(); err != nil {
		return err
	}

	dictPath := system.GetPasswordDictPath()
	content := strings.Join(defaultPasswords, "\n")
	return os.WriteFile(dictPath, []byte(content), 0644)
}