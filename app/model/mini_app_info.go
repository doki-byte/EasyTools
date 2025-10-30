package model

import (
	"time"

	"gorm.io/gorm"
)

const TableNameMiniAppInfo = "mini_app_info"

type MiniAppInfo struct {
	ID            int       `gorm:"primarykey" json:"id"`
	AppID         string    `json:"app_id" gorm:"column:app_id;index"`
	Nickname      string    `json:"nickname" gorm:"column:nickname"`
	Username      string    `json:"username" gorm:"column:username"`
	Description   string    `json:"description" gorm:"column:description"`
	Avatar        string    `json:"avatar" gorm:"column:avatar"`
	UsesCount     string    `json:"uses_count" gorm:"column:uses_count"`
	PrincipalName string    `json:"principal_name" gorm:"column:principal_name"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (*MiniAppInfo) TableName() string {
	return TableNameMiniAppInfo
}

func (m *MiniAppInfo) Initialize(db *gorm.DB) {
	// MiniAppInfo 不需要初始化默认数据
}
