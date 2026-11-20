package assistive

import (
	"fmt"

	"gorm.io/gorm"
)

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

// QueryPasswords 分页查询全部数据
func (a *Assistive) QueryPasswords(page, pageSize int) ([]PasswordData, int64, error) {
	db := a.Db()
	if db == nil {
		return nil, 0, fmt.Errorf("数据库连接未初始化")
	}

	var passwordDataList []PasswordData
	var total int64

	err := db.Model(&PasswordData{}).
		Count(&total).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&passwordDataList).Error

	if err != nil {
		return nil, 0, fmt.Errorf("查询数据库失败: %v", err)
	}

	return passwordDataList, total, nil
}

// QueryPasswordsWithQuery 分页查询带查询条件的数据
func (a *Assistive) QueryPasswordsWithQuery(page, pageSize int, query string) ([]PasswordData, int64, error) {
	db := a.Db()
	if db == nil {
		return nil, 0, fmt.Errorf("数据库连接未初始化")
	}

	var passwordDataList []PasswordData
	var total int64

	queryString := "%" + query + "%"

	err := db.Model(&PasswordData{}).
		Where("name LIKE ? OR method LIKE ? OR userId LIKE ? OR level LIKE ?", queryString, queryString, queryString, queryString).
		Count(&total).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&passwordDataList).Error

	if err != nil {
		return nil, 0, fmt.Errorf("查询数据库失败: %v", err)
	}

	return passwordDataList, total, nil
}

// QueryPasswordsAPI 分页查询接口
func (a *Assistive) QueryPasswordsAPI(page, pageSize int, query string) (map[string]interface{}, error) {
	var data []PasswordData
	var total int64
	var err error

	if query == "" {
		data, total, err = a.QueryPasswords(page, pageSize)
	} else {
		data, total, err = a.QueryPasswordsWithQuery(page, pageSize, query)
	}

	if err != nil {
		return nil, fmt.Errorf("查询失败: %v", err)
	}

	if data == nil {
		data = []PasswordData{}
	}

	return map[string]interface{}{
		"data":  data,
		"total": total,
	}, nil
}

// CreatePassword 创建密码记录
func (a *Assistive) CreatePassword(data PasswordData) (PasswordData, error) {
	db := a.Db()
	if db == nil {
		return PasswordData{}, fmt.Errorf("数据库连接未初始化")
	}

	err := db.Create(&data).Error
	if err != nil {
		return PasswordData{}, fmt.Errorf("创建密码记录失败: %v", err)
	}

	return data, nil
}

// UpdatePassword 更新密码记录
func (a *Assistive) UpdatePassword(data PasswordData) (PasswordData, error) {
	db := a.Db()
	if db == nil {
		return PasswordData{}, fmt.Errorf("数据库连接未初始化")
	}

	err := db.Save(&data).Error
	if err != nil {
		return PasswordData{}, fmt.Errorf("更新密码记录失败: %v", err)
	}

	return data, nil
}

// DeletePassword 删除密码记录
func (a *Assistive) DeletePassword(id int) error {
	db := a.Db()
	if db == nil {
		return fmt.Errorf("数据库连接未初始化")
	}

	err := db.Where("id = ?", id).Delete(&PasswordData{}).Error
	if err != nil {
		return fmt.Errorf("删除密码记录失败: %v", err)
	}

	return nil
}

// GetPasswordByID 根据ID获取密码记录
func (a *Assistive) GetPasswordByID(id int) (PasswordData, error) {
	db := a.Db()
	if db == nil {
		return PasswordData{}, fmt.Errorf("数据库连接未初始化")
	}

	var data PasswordData
	err := db.Where("id = ?", id).First(&data).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return PasswordData{}, fmt.Errorf("记录不存在")
		}
		return PasswordData{}, fmt.Errorf("查询记录失败: %v", err)
	}

	return data, nil
}
