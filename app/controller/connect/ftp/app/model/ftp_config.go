package model

import (
	"EasyTools/app/controller/connect/ftp/app/utils"
	"encoding/json"
	"log/slog"
	"strconv"
	"gorm.io/gorm"
)

const TableNameFtpConfig = "ftp_config"

type FtpConfig struct {
	ID           uint   `gorm:"column:id;primaryKey;autoIncrement" json:"id"`
	Name         string `gorm:"column:name;not null;size:128" json:"name" binding:"required"`
	Host         string `gorm:"column:host;not null;size:128" json:"host" binding:"required"`
	Port         string `gorm:"column:port;not null;size:16" json:"port" binding:"required"`
	Username     string `gorm:"column:username;not null;size:128" json:"username"`
	Password     string `gorm:"column:password;not null;size:4096;default:''" json:"password"`
	// 连接方式：'direct'（直连）、'ssh'（SSH隧道）或 ''（空字符串，未设置）
	ConnectionType string `gorm:"column:connection_type;size:16;default:''" json:"connection_type"`
	// SSH 隧道配置
	SSHAddr     string `gorm:"column:ssh_addr;size:128" json:"ssh_addr"`
	SSHPort     string `gorm:"column:ssh_port;size:16" json:"ssh_port"`
	SSHUsername string `gorm:"column:ssh_username;size:128" json:"ssh_username"`
	SSHPassword string `gorm:"column:ssh_password;size:4096" json:"ssh_password"`
}

func (FtpConfig) TableName() string {
	return TableNameFtpConfig
}

// UnmarshalJSON 自定义JSON反序列化，处理id字段可以是字符串或数字的情况
func (c *FtpConfig) UnmarshalJSON(data []byte) error {
	// 使用别名类型避免递归调用
	type Alias FtpConfig
	aux := &struct {
		ID interface{} `json:"id"` // 使用interface{}来接收任意类型
		*Alias
	}{
		Alias: (*Alias)(c),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// 处理id字段：支持字符串和数字两种类型
	if aux.ID != nil {
		switch v := aux.ID.(type) {
		case float64:
			// JSON数字类型会解析为float64
			c.ID = uint(v)
		case string:
			// 字符串类型需要转换
			if v != "" {
				id, err := strconv.ParseUint(v, 10, 32)
				if err != nil {
					return err
				}
				c.ID = uint(id)
			}
		case nil:
			// nil值，保持默认值0
			c.ID = 0
		default:
			// 其他类型，尝试转换为uint
			if idFloat, ok := v.(float64); ok {
				c.ID = uint(idFloat)
			}
		}
	}

	return nil
}

func (c *FtpConfig) Create(conf *FtpConfig) error {
	return Db.Create(conf).Error
}

func (c *FtpConfig) FindByID(id uint) (FtpConfig, error) {
	var conf FtpConfig
	err := Db.First(&conf, "id = ?", id).Error
	return conf, err
}

func (c *FtpConfig) FindAll() ([]FtpConfig, error) {
	var list []FtpConfig
	err := Db.Order("id desc").Find(&list).Error
	return list, err
}

func (c *FtpConfig) UpdateById(id uint, conf *FtpConfig) error {
	return Db.Model(&FtpConfig{}).Where("id = ?", id).Updates(conf).Error
}

// UpdateByIdSelective 选择性更新字段（使用map）
func (c *FtpConfig) UpdateByIdSelective(id uint, updates map[string]interface{}) error {
	return Db.Model(&FtpConfig{}).Where("id = ?", id).Updates(updates).Error
}

// EncodePassword base64编码密码的公共方法
func EncodePassword(password string) string {
	if password == "" {
		return ""
	}
	return utils.Base64Encode(password)
}

func (c *FtpConfig) DeleteByID(id uint) error {
	return Db.Delete(&FtpConfig{}, id).Error
}

// BeforeCreate Hook - base64编码密码（简单编码，不加密）
func (c *FtpConfig) BeforeCreate(_ *gorm.DB) error {
	if c.Password != "" {
		c.Password = utils.Base64Encode(c.Password)
	}
	if c.SSHPassword != "" {
		c.SSHPassword = utils.Base64Encode(c.SSHPassword)
	}
	return nil
}

// BeforeUpdate Hook - base64编码密码（简单编码，不加密）
func (c *FtpConfig) BeforeUpdate(_ *gorm.DB) error {
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
func (c *FtpConfig) AfterFind(_ *gorm.DB) error {
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


