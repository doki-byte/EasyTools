package model

import (
	"time"

	"gorm.io/gorm"
)

const TableNameVersionTask = "version_task"

type VersionTask struct {
	ID              int       `gorm:"primarykey" json:"id"`
	AppID           string    `json:"app_id" gorm:"column:app_id;index"`
	Version         string    `json:"version" gorm:"column:version"`
	DecompileStatus string    `json:"decompile_status" gorm:"column:decompile_status"`
	MatchStatus     string    `json:"match_status" gorm:"column:match_status"`
	Message         string    `json:"message" gorm:"column:message"`
	Matched         string    `json:"matched" gorm:"column:matched;type:text"`
	UpdateDate      string    `json:"update_date" gorm:"column:update_date"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (*VersionTask) TableName() string {
	return TableNameVersionTask
}

func (v *VersionTask) Initialize(db *gorm.DB) {
	// VersionTask 不需要初始化默认数据
}
