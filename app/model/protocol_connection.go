package model

import (
	"gorm.io/gorm"
	"time"
)

const TableNameProtocolConnection = "protocol_connections"

// ProtocolConnection 协议连接信息（用于保存密码爆破结果）
type ProtocolConnection struct {
	ID            int       `gorm:"primaryKey;autoIncrement" json:"id"`
	Protocol      string    `gorm:"column:protocol;index" json:"protocol"`            // 协议类型：SSH, FTP, MySQL等
	Host          string    `gorm:"column:host" json:"host"`                          // 主机地址
	Port          string    `gorm:"column:port" json:"port"`                          // 端口
	Username      string    `gorm:"column:username" json:"username"`                  // 用户名
	Password      string    `gorm:"column:password" json:"password"`                  // 密码
	Status        string    `gorm:"column:status;default:pending" json:"status"`      // 状态：success, failed, pending
	Message       string    `gorm:"column:message;type:text" json:"message"`          // 连接结果消息
	Credential    string    `gorm:"column:credential" json:"credential"`              // 凭证信息（格式：username/password）
	VulnURL       string    `gorm:"column:vuln_url" json:"vuln_url"`                  // 漏洞URL
	Description   string    `gorm:"column:description;type:text" json:"description"`  // 描述信息
	CommandResult string    `gorm:"column:command_result;type:text" json:"command_result"` // 最近一次命令执行结果
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

func (*ProtocolConnection) TableName() string {
	return TableNameProtocolConnection
}

// Initialize 初始化数据库表
func (p *ProtocolConnection) Initialize(db *gorm.DB) error {
	return db.AutoMigrate(&ProtocolConnection{})
}
