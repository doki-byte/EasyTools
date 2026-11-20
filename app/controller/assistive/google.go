package assistive

import (
	"fmt"
	"strings"
)

// GoogleQuery 结构体
type GoogleQuery struct {
	ID          int    `json:"id"`
	Category    string `json:"category"`    // 查询分类
	Description string `json:"description"` // 说明
	Command     string `json:"command"`     // 查询命令
}

// TableName 指定表名
func (GoogleQuery) TableName() string {
	return "google_query"
}

// QueryGoogleQueries 根据域名生成查询语法
func (a *Assistive) QueryGoogleQueries(googleDomain string) ([]map[string]interface{}, error) {
	// 定义返回结果
	var results []map[string]interface{}

	// 获取数据库连接
	db := a.Db()
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

// GetAllGoogleQueries 获取所有Google语法
func (a *Assistive) GetAllGoogleQueries() ([]GoogleQuery, error) {
	db := a.Db()
	if db == nil {
		return nil, fmt.Errorf("数据库连接未初始化")
	}

	var queries []GoogleQuery
	err := db.Order("id ASC").Find(&queries).Error
	if err != nil {
		return nil, fmt.Errorf("查询数据库失败: %v", err)
	}

	return queries, nil
}

// AddGoogleQuery 新增Google语法
func (a *Assistive) AddGoogleQuery(query map[string]interface{}) error {
	db := a.Db()
	if db == nil {
		return fmt.Errorf("数据库连接未初始化")
	}

	newQuery := GoogleQuery{
		Category:    query["category"].(string),
		Description: query["description"].(string),
		Command:     query["command"].(string),
	}

	err := db.Create(&newQuery).Error
	if err != nil {
		return fmt.Errorf("新增语法失败: %v", err)
	}

	return nil
}

// UpdateGoogleQuery 更新Google语法
func (a *Assistive) UpdateGoogleQuery(query map[string]interface{}) error {
	db := a.Db()
	if db == nil {
		return fmt.Errorf("数据库连接未初始化")
	}

	updateData := GoogleQuery{
		ID:          int(query["id"].(float64)),
		Category:    query["category"].(string),
		Description: query["description"].(string),
		Command:     query["command"].(string),
	}

	err := db.Save(&updateData).Error
	if err != nil {
		return fmt.Errorf("更新语法失败: %v", err)
	}

	return nil
}

// DeleteGoogleQuery 删除Google语法
func (a *Assistive) DeleteGoogleQuery(id int) error {
	db := a.Db()
	if db == nil {
		return fmt.Errorf("数据库连接未初始化")
	}

	err := db.Where("id = ?", id).Delete(&GoogleQuery{}).Error
	if err != nil {
		return fmt.Errorf("删除语法失败: %v", err)
	}

	return nil
}
