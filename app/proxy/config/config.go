package config

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

type Config struct {
	Email     string
	FofaKey   string
	HunterKey string
	QuakeKey  string
	Country   string
	Maxpage   string

	CoroutineCount int
	LiveProxies    int
	AllProxies     int
	LiveProxyLists []string
	Timeout        string
	SocksAddress   string
	FilePath       string

	Status int

	Code        int
	Error       string
	GlobalProxy string
}

// 获取数据库路径
func getDBPath() string {
	optSys := runtime.GOOS
	dbName := "config.db"

	if optSys == "windows" {
		return filepath.Join("EasyToolsFiles", dbName)
	}
	return filepath.Join("EasyToolsFiles", dbName)
}

// 初始化数据库和表
func initDB() (*sql.DB, error) {
	dbPath := getDBPath()
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %v", err)
	}

	// 创建配置表
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS proxyconfig (
		key TEXT PRIMARY KEY,
		value TEXT
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		return nil, fmt.Errorf("failed to create table: %v", err)
	}

	// 插入默认配置
	defaultConfig := map[string]string{
		"Timeout":        "10",
		"GlobalProxy":    "0",
		"Country":        "0",
		"Email":          "",
		"FofaKey":        "",
		"HunterKey":      "",
		"QuakeKey":       "",
		"Maxpage":        "10",
		"CoroutineCount": "200",
		"SocksAddress":   "socks5://127.0.0.1:1080",
	}

	// 使用事务批量插入
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %v", err)
	}

	stmt, err := tx.Prepare("INSERT OR IGNORE INTO proxyconfig (key, value) VALUES (?, ?)")
	if err != nil {
		return nil, fmt.Errorf("failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	for key, value := range defaultConfig {
		_, err = stmt.Exec(key, value)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to insert default config: %v", err)
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %v", err)
	}

	return db, nil
}

func GetConfig() *Config {
	c := &Config{}

	// 初始化数据库
	db, err := initDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// 查询所有配置项
	rows, err := db.Query("SELECT key, value FROM proxyconfig")
	if err != nil {
		log.Fatalf("Failed to query config: %v", err)
	}
	defer rows.Close()

	// 将结果存储到map中
	configMap := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			log.Printf("Failed to scan row: %v", err)
			continue
		}
		configMap[key] = value
	}

	// 使用反射设置结构体字段
	cr := reflect.ValueOf(c).Elem()
	for key, value := range configMap {
		field := cr.FieldByName(key)
		if !field.IsValid() {
			log.Printf("Warning: Unknown config key: %s", key)
			continue
		}

		switch field.Kind() {
		case reflect.String:
			field.SetString(value)
		case reflect.Int:
			intValue, err := strconv.Atoi(value)
			if err != nil {
				log.Printf("Warning: Invalid integer value for key %s: %s", key, value)
				continue
			}
			field.SetInt(int64(intValue))
		case reflect.Slice:
			if field.Type().Elem().Kind() == reflect.String {
				field.Set(reflect.ValueOf(strings.Split(value, ",")))
			}
		}
	}

	return c
}

func (p *Config) SaveConfig() error {
	db, err := sql.Open("sqlite3", getDBPath())
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}
	defer db.Close()

	// 准备更新语句
	stmt, err := db.Prepare("REPLACE INTO proxyconfig (key, value) VALUES (?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	// 需要保存的字段
	fields := map[string]interface{}{
		"Email":          p.Email,
		"FofaKey":        p.FofaKey,
		"HunterKey":      p.HunterKey,
		"QuakeKey":       p.QuakeKey,
		"Maxpage":        p.Maxpage,
		"CoroutineCount": p.CoroutineCount,
		"Timeout":        p.Timeout,
		"SocksAddress":   p.SocksAddress,
		"Country":        p.Country,
		"GlobalProxy":    p.GlobalProxy,
	}

	// 更新数据库
	for key, value := range fields {
		var strValue string

		switch v := value.(type) {
		case string:
			strValue = v
		case int:
			strValue = strconv.Itoa(v)
		default:
			log.Printf("Unsupported type for key %s", key)
			continue
		}

		_, err := stmt.Exec(key, strValue)
		if err != nil {
			return fmt.Errorf("failed to update config for key %s: %v", key, err)
		}
	}

	return nil
}

var globalConfig *Config

func init() {
	globalConfig = GetConfig()
}

// 以下方法保持不变...
func (p *Config) GetProfile() Config {
	return *globalConfig
}

func (p *Config) GetCoroutineCount() int {
	return p.CoroutineCount
}

func (p *Config) GetLiveProxies() int {
	return p.LiveProxies
}

func (p *Config) SetAllProxies(datasets []string) {
	p.AllProxies = len(datasets)
	p.LiveProxyLists = datasets
}

func (p *Config) SetLiveProxies(datasets []string) {
	p.LiveProxyLists = datasets
	p.LiveProxies = len(datasets)
}

func (p *Config) GetTimeout() string { return p.Timeout }

func (p *Config) GetSocksAddress() string {
	return p.SocksAddress
}

func (p *Config) GetStatus() int {
	return p.Status
}

func (p *Config) SetStatus(i int) {
	p.Status = i
}
