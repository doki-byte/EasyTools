package model

import (
	"EasyTools/app/controller/connect/dbmanager/app/utils"
	"log/slog"
	"gorm.io/gorm"
)

const TableNameDbConfig = "dbmanager_config"

type DbConfig struct {
	ID           uint   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Type         string `gorm:"column:type;not null;size:32" json:"type" binding:"required"`
	Name         string `gorm:"column:name;not null;size:128" json:"name" binding:"required"`
	// 对于 MySQL/PostgreSQL/Oracle 等网络数据库，Host/Port/Username 必填
	// 但对于 SQLite，这些字段并不需要，因此这里不使用全局的 binding:"required"
	// 在业务层针对不同类型做校验，避免 SQLite 也被要求填写这些字段
	Host         string `gorm:"column:host;not null;size:128" json:"host"`
	Port         string `gorm:"column:port;not null;size:16" json:"port"`
	Username     string `gorm:"column:username;not null;size:128" json:"username"`
	Password     string `gorm:"column:password;not null;size:4096;default:''" json:"password"`
	Charset      string `gorm:"column:charset;size:32" json:"charset"`
	Timeout      uint64 `gorm:"column:timeout;default:10" json:"timeout"`
	ReadTimeout  uint64 `gorm:"column:read_timeout;default:30" json:"read_timeout"`
	WriteTimeout uint64 `gorm:"column:write_timeout;default:30" json:"write_timeout"`
	DbFile       string `gorm:"column:db_file;size:512" json:"db_file"` // SQLite 文件路径
	// 连接方式：'direct'（直连）、'ssh'（SSH隧道）或 ''（空字符串，未设置）
	ConnectionType string `gorm:"column:connection_type;size:16;default:''" json:"connection_type"`
	// SSH 隧道配置
	SSHAddr     string `gorm:"column:ssh_addr;size:128" json:"ssh_addr"`
	SSHPort     string `gorm:"column:ssh_port;size:16" json:"ssh_port"`
	SSHUsername string `gorm:"column:ssh_username;size:128" json:"ssh_username"`
	SSHPassword string `gorm:"column:ssh_password;size:4096" json:"ssh_password"`
}

func (DbConfig) TableName() string {
	return TableNameDbConfig
}

func (c *DbConfig) Create(conf *DbConfig) error {
	return Db.Create(conf).Error
}

func (c *DbConfig) FindByID(id uint) (DbConfig, error) {
	var conf DbConfig
	err := Db.First(&conf, "id = ?", id).Error
	return conf, err
}

func (c *DbConfig) FindAll() ([]DbConfig, error) {
	var list []DbConfig
	err := Db.Order("id desc").Find(&list).Error
	return list, err
}

func (c *DbConfig) UpdateById(id uint, conf *DbConfig) error {
	return Db.Model(&DbConfig{}).Where("id = ?", id).Updates(conf).Error
}

// UpdateByIdSelective 选择性更新字段（使用map）
func (c *DbConfig) UpdateByIdSelective(id uint, updates map[string]interface{}) error {
	return Db.Model(&DbConfig{}).Where("id = ?", id).Updates(updates).Error
}

// EncodePassword base64编码密码的公共方法
func EncodePassword(password string) string {
	if password == "" {
		return ""
	}
	return utils.Base64Encode(password)
}

func (c *DbConfig) DeleteByID(id uint) error {
	return Db.Delete(&DbConfig{}, id).Error
}

// BeforeCreate Hook - base64编码密码（简单编码，不加密）
func (c *DbConfig) BeforeCreate(_ *gorm.DB) error {
	if c.Password != "" {
		c.Password = utils.Base64Encode(c.Password)
	}
	if c.SSHPassword != "" {
		c.SSHPassword = utils.Base64Encode(c.SSHPassword)
	}
	return nil
}

// BeforeUpdate Hook - base64编码密码（简单编码，不加密）
func (c *DbConfig) BeforeUpdate(_ *gorm.DB) error {
	if c.Password != "" {
		// 如果不是base64编码的字符串，则进行编码
		if !utils.IsBase64(c.Password) {
			c.Password = utils.Base64Encode(c.Password)
		}
	}
	if c.SSHPassword != "" {
		// 如果不是base64编码的字符串，则进行编码
		if !utils.IsBase64(c.SSHPassword) {
			c.SSHPassword = utils.Base64Encode(c.SSHPassword)
		}
	}
	return nil
}

// AfterFind Hook - base64解码密码
func (c *DbConfig) AfterFind(_ *gorm.DB) error {
	if c.Password != "" {
		decoded, err := utils.Base64Decode(c.Password)
		if err != nil {
			// 解码失败可能是旧数据或已经是明文，直接使用原值
			slog.Warn("密码解码失败，使用原值", "config_id", c.ID, "err", err)
		} else {
			c.Password = decoded
		}
	}
	if c.SSHPassword != "" {
		decoded, err := utils.Base64Decode(c.SSHPassword)
		if err != nil {
			// 解码失败可能是旧数据或已经是明文，直接使用原值
			slog.Warn("SSH密码解码失败，使用原值", "config_id", c.ID, "err", err)
		} else {
			c.SSHPassword = decoded
		}
	}
	return nil
}

