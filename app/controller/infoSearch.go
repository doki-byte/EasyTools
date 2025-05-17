package controller

import (
	"fmt"
	"strings"
)

// Antivirus 控制器
type InfoSearch struct {
	Base
}

// AntivirusItem 杀软表结构体
type AntivirusItem struct {
	ID          int    `json:"id"`
	ProcessName string `json:"process_name"` // 对应 tasklist 的进程名
	Description string `json:"description"`  // 杀软名称
}

// GoogleQuery 结构体
type GoogleQuery struct {
	ID          int    `json:"id"`
	Category    string `json:"category"`    // 查询分类
	Description string `json:"description"` // 说明
	Command     string `json:"command"`     // 查询命令
}

// PasswordData 密码数据结构体
type PasswordData struct {
	ID       int    `json:"id" gorm:"column:id"`             // 映射数据库 id
	Name     string `json:"name" gorm:"column:name"`         // 映射数据库 name
	Method   string `json:"method" gorm:"column:method"`     // 映射数据库 method
	UserID   string `json:"userId" gorm:"column:userId"`     // 映射数据库 userId
	Password string `json:"password" gorm:"column:password"` // 映射数据库 password
	Level    string `json:"level" gorm:"column:level"`       // 映射数据库 level
}

// TableName 指定表名
func (PasswordData) TableName() string {
	return "password_data"
}

// TableName 指定表名
func (AntivirusItem) TableName() string {
	return "antivirus_list"
}

// TableName 指定表名
func (GoogleQuery) TableName() string {
	return "google_query"
}

// NewAntivirus 创建新的 Antivirus 控制器
func NewInfoSearch() *InfoSearch {
	return &InfoSearch{}
}

// QueryAntivirusProcesses 根据用户输入模糊查询杀软进程
func (a *InfoSearch) QueryAntivirusProcesses(tasklistOutput string) ([]map[string]string, error) {
	// 定义返回结果
	var results []map[string]string

	// 获取数据库连接
	db := a.db()
	if db == nil {
		return nil, fmt.Errorf("数据库连接未初始化")
	}

	a.log("开始处理用户输入的 tasklist 输出内容...")

	// 将用户输入按行分割，并提取每行的程序名（假设第一个字段为程序名）
	lines := strings.Split(tasklistOutput, "\n")
	userProcesses := make([]string, 0)
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			userProcesses = append(userProcesses, fields[0]) // 提取程序名
		}
	}

	if len(userProcesses) == 0 {
		return nil, fmt.Errorf("未能从输入中提取任何程序名")
	}

	// a.log(fmt.Sprintf("提取到 %d 个程序名，开始进行模糊匹配查询...", len(userProcesses)))

	// 查询数据库中的杀软进程列表
	var antivirusList []AntivirusItem
	err := db.Find(&antivirusList).Error
	if err != nil {
		// a.log(fmt.Sprintf("查询数据库失败: %v", err))
		return nil, fmt.Errorf("查询数据库失败: %v", err)
	}

	// a.log(fmt.Sprintf("从数据库中加载了 %d 个杀软进程", len(antivirusList)))

	// 对用户输入的进程逐行进行模糊匹配
	for _, userProcess := range userProcesses {
		for _, antivirus := range antivirusList {
			// 判断用户输入是否包含数据库中的杀软进程名
			if strings.Contains(strings.ToLower(userProcess), strings.ToLower(antivirus.ProcessName)) {
				results = append(results, map[string]string{
					"program":     userProcess,           // 用户输入的程序名
					"match":       antivirus.ProcessName, // 匹配到的杀软进程名
					"description": antivirus.Description, // 杀软描述
				})
			}
		}
	}

	// a.log(fmt.Sprintf("模糊匹配完成，识别到 %d 个结果", len(results)))
	return results, nil
}

// QueryGoogleQueries 根据域名生成查询语法
func (a *InfoSearch) QueryGoogleQueries(googleDomain string) ([]map[string]interface{}, error) {
	// 定义返回结果
	var results []map[string]interface{}

	// 获取数据库连接
	db := a.db()
	if db == nil {
		return nil, fmt.Errorf("数据库连接未初始化")
	}

	// 查询数据库中的 Google 查询列表
	var googleQueries []GoogleQuery
	err := db.Find(&googleQueries).Error
	if err != nil {
		return nil, fmt.Errorf("查询数据库失败: %v", err)
	}

	// 遍历查询结果并替换命令中的 ${googleDomain}
	for _, query := range googleQueries {
		// 替换命令中的 ${googleDomain}
		modifiedCommand := strings.ReplaceAll(query.Command, "${googleDomain}", googleDomain)

		// 将修改后的命令添加到返回结果中
		result := map[string]interface{}{
			"category":    query.Category,
			"description": query.Description,
			"command":     modifiedCommand,
		}
		results = append(results, result)
	}

	return results, nil
}

// QueryPasswords 分页查询全部数据
func (a *InfoSearch) QueryPasswords(page, pageSize int) ([]PasswordData, int64, error) {
	// 获取数据库连接
	db := a.db()
	if db == nil {
		return nil, 0, fmt.Errorf("数据库连接未初始化")
	}

	var passwordDataList []PasswordData
	var total int64

	// 执行查询，获取所有数据，不带查询条件
	err := db.Model(&PasswordData{}).
		Count(&total). // 获取总数据数
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&passwordDataList).Error

	if err != nil {
		return nil, 0, fmt.Errorf("查询数据库失败: %v", err)
	}

	return passwordDataList, total, nil
}

// QueryPasswordsWithQuery 分页查询带查询条件的数据
func (a *InfoSearch) QueryPasswordsWithQuery(page, pageSize int, query string) ([]PasswordData, int64, error) {
	// 获取数据库连接
	db := a.db()
	if db == nil {
		return nil, 0, fmt.Errorf("数据库连接未初始化")
	}

	var passwordDataList []PasswordData
	var total int64

	// 构造查询条件
	queryString := "%" + query + "%"

	// 执行查询，带上查询条件
	err := db.Model(&PasswordData{}).
		Where("name LIKE ? OR method LIKE ? OR userId LIKE ? OR password LIKE ? OR level LIKE ?", queryString, queryString, queryString, queryString, queryString).
		Count(&total). // 获取总数据数
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&passwordDataList).Error

	if err != nil {
		return nil, 0, fmt.Errorf("查询数据库失败: %v", err)
	}

	return passwordDataList, total, nil
}

// QueryPasswordsAPI 分页查询接口
func (a *InfoSearch) QueryPasswordsAPI(page, pageSize int, query string) (map[string]interface{}, error) {
	var data []PasswordData
	var total int64
	var err error

	// 如果没有查询条件，执行默认查询
	if query == "" {
		data, total, err = a.QueryPasswords(page, pageSize)
	} else {
		// 如果有查询条件，执行带条件的查询
		data, total, err = a.QueryPasswordsWithQuery(page, pageSize, query)
	}

	if err != nil {
		return nil, fmt.Errorf("查询失败: %v", err)
	}

	// 如果数据为空，返回一个空数组而不是 nil
	if data == nil {
		data = []PasswordData{}
	}

	// 返回查询结果和总数
	return map[string]interface{}{
		"data":  data,
		"total": total,
	}, nil
}
