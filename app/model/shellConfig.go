package model

import (
	"log"

	"gorm.io/gorm"
)

const TableNameConfig = "config"

type ShellConfig struct {
	Id   int  `json:"id"`
	Init bool `json:"init" gorm:"column:init"`
}

func (*ShellConfig) TableName() string {
	return TableNameConfig
}

func (s *ShellConfig) Initialize(db *gorm.DB) {
	var count int64
	if err := db.Model(&ShellConfig{}).Count(&count).Error; err != nil {
		log.Fatalf("Error counting records: %v", err)
	}

	if count == 0 {
		defaultShellConfig := []ShellConfig{
			{Id: 1, Init: false},
		}
		if err := db.Create(&defaultShellConfig).Error; err != nil {
			log.Fatalf("Error inserting default data: %v", err)
		}
	}
}
