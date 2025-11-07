package controller

import (
	"EasyTools/app/controller/system"
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

// User 系统API
type User struct {
	system.Base
}

// UserItem 工具结构体
type UserItem struct {
	Id       int    `json:"id"`
	UserName string `json:"username" gorm:"column:username"`
	PassWord string `json:"password" gorm:"column:password"`
}

type UpdateUserItem struct {
	UserName    string `json:"username" gorm:"column:username"`
	PassWord    string `json:"password" gorm:"column:password"`
	OldPassWord string `json:"OldPassword"`
}

func NewUser() *User {
	return &User{}
}

// TableName 指定表名
func (UserItem) TableName() string {
	return "user"
}

// Login 用户登录验证
func (s *User) Login(username, password string) error {
	db := s.Db()
	if db == nil {
		return fmt.Errorf("数据库连接未初始化")
	}

	var user UserItem
	if err := db.Where("id = 1").First(&user).Error; err != nil {
		// 统一返回相同错误信息防止用户枚举
		return fmt.Errorf("用户名或密码错误")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PassWord), []byte(password)); err != nil {
		return fmt.Errorf("用户名或密码错误")
	}

	return nil
}

// UpdateUser 修改用户信息（包含密码修改）
func (s *User) UpdateUser(username string, updatedUser UpdateUserItem) error {
	db := s.Db()
	if db == nil {
		return fmt.Errorf("数据库连接未初始化")
	}

	// 检查要修改的用户是否存在
	var existingUser UserItem
	if err := db.Where("id = 1").First(&existingUser).Error; err != nil {
		return fmt.Errorf("用户不存在")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(existingUser.PassWord), []byte(updatedUser.OldPassWord)); err != nil {
		return fmt.Errorf("原密码错误")
	}

	// 如果更新密码则进行哈希处理
	if updatedUser.PassWord != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(updatedUser.PassWord), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("密码加密失败: %v", err)
		}
		updatedUser.PassWord = string(hashedPassword)
	}

	// 使用Select选择要更新的字段，避免更新主键
	result := db.Model(&existingUser).
		Updates(updatedUser)

	if result.Error != nil {
		return fmt.Errorf("修改用户信息失败: %v", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("未修改任何信息")
	}

	return nil
}
