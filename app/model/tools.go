package model

import (
	"log"

	"gorm.io/gorm"
)

const TableNameTools = "tools"

type Tools struct {
	ID       int    `json:"id"`
	Category string `json:"category"`
	Cmd      string `json:"cmd"`
	Param    string `json:"param"`
	Name     string `json:"name"`
	Desc     string `json:"desc"`
	Icon     string `json:"icon,omitempty"` // 可选字段
	Terminal int    `json:"terminal"`       // 新增字段，1: 需要终端, 0: 不需要
	CateSort int    `json:"catesort"`       // 新增分类排序字段
	CmdSort  int    `json:"cmdsort"`        // 新增分类排序字段
}

func (*Tools) TableName() string {
	return TableNameTools
}

func (s *Tools) Initialize(db *gorm.DB) {
	var count int64
	if err := db.Model(&Tools{}).Count(&count).Error; err != nil {
		log.Fatalf("Error counting records: %v", err)
	}

	if count == 0 {
		defaultTools := []Tools{
			{Category: "命令测试", Cmd: "whoami", Param: "", Name: "whoami", Desc: "终端命令显示", Icon: "", Terminal: 1, CateSort: 0, CmdSort: 0},
			{Category: "命令测试", Cmd: "calc", Param: "", Name: "弹计算器", Desc: "不显示终端框框", Icon: "", Terminal: 0, CateSort: 0, CmdSort: 0},
			{Category: "数据库管理", Cmd: "cd EasyToolsFiles\\tools\\HeidiSQL &", Param: "heidisql.exe", Name: "HeidiSQL", Desc: "便携数据库连接工具", Icon: "", Terminal: 0, CateSort: 0, CmdSort: 0},
		}
		if err := db.Create(&defaultTools).Error; err != nil {
			log.Fatalf("Error inserting default data: %v", err)
		}
	}
}
