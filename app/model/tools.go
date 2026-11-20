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
	Path     string `json:"path"`
	Desc     string `json:"desc"`
	Icon     string `json:"icon,omitempty"` // 可选字段
	Terminal int    `json:"terminal"`       // 新增字段，1: 需要终端, 0: 不需要
	CateSort int `json:"catesort"` // 新增分类排序字段
	CmdSort  int `json:"cmdsort"`  // 新增分类排序字段
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
			{Category: "命令测试", Cmd: "echo", Param: "\"右键菜单增加工具\"", Name: "右键菜单增加工具", Path: "", Desc: "右键菜单增加工具", Icon: "", Terminal: 1, CateSort: 0, CmdSort: 0},
			{Category: "命令测试", Cmd: "echo", Param: "\"支持拖拽软件进入工具仓库进行添加哦\"", Name: "支持拖拽增加工具", Path: "", Desc: "支持拖拽软件进入工具仓库进行添加哦", Icon: "", Terminal: 1, CateSort: 0, CmdSort: 0},
		}
		if err := db.Create(&defaultTools).Error; err != nil {
			log.Fatalf("Error inserting default data: %v", err)
		}
	}
}
