package model

import (
	"log"

	"gorm.io/gorm"
)

const TableNameUser = "user"

type User struct {
	Id       int    `json:"id"`
	UserName string `json:"username" gorm:"column:username"`
	PassWord string `json:"password" gorm:"column:password"`
}

func (*User) TableName() string {
	return TableNameUser
}

func (s *User) Initialize(db *gorm.DB) {
	var count int64
	if err := db.Model(&User{}).Count(&count).Error; err != nil {
		log.Fatalf("Error counting records: %v", err)
	}

	if count == 0 {
		defaultSites := []User{
			{Id: 1, UserName: "EasyTools", PassWord: "$2a$10$LiTjOhhgG82UobTfIenpMe6PzZS.QtV1NEaVF6cQ0rVEi/a/Uf8tC"},
		}
		if err := db.Create(&defaultSites).Error; err != nil {
			log.Fatalf("Error inserting default data: %v", err)
		}
	}
}
