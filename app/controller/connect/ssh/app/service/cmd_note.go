package service

import (
	"EasyTools/app/controller/connect/ssh/app/model"
	gin2 "EasyTools/app/controller/connect/ssh/gin"
	"strconv"
)

func CmdNoteCreate(c *gin2.Context) {
	var cmd model.CmdNote
	if err := c.ShouldBind(&cmd); err != nil {
		c.JSON(200, gin2.H{"code": 1, "msg": err.Error()})
		return
	}
	cmd.Uid = c.GetUint("uid")
	err := cmd.Create(&cmd)
	if err != nil {
		c.JSON(200, gin2.H{"code": 2, "msg": err.Error()})
		return
	}
	CmdNoteFindAll(c)
}

func CmdNoteFindByID(c *gin2.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(200, gin2.H{"code": 2, "msg": err.Error()})
		return
	}
	var cmd model.CmdNote
	data, err := cmd.FindByID(uint(id), c.GetUint("uid"))
	if err != nil {
		c.JSON(200, gin2.H{"code": 2, "msg": err.Error()})
		return
	}
	c.JSON(200, gin2.H{"code": 0, "msg": "ok", "data": data})
}

func CmdNoteFindAll(c *gin2.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10000"))
	if err != nil {
		c.JSON(200, gin2.H{"code": 2, "msg": err.Error()})
		return
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil {
		c.JSON(200, gin2.H{"code": 2, "msg": err.Error()})
		return
	}

	var cmd model.CmdNote
	data, err := cmd.FindAll(offset, limit, c.GetUint("uid"))
	if err != nil {
		c.JSON(200, gin2.H{"code": 2, "msg": err.Error()})
		return
	}
	c.JSON(200, gin2.H{"code": 0, "msg": "ok", "data": data})
}

func CmdNoteUpdateById(c *gin2.Context) {
	var cmd model.CmdNote
	if err := c.ShouldBind(&cmd); err != nil {
		c.JSON(200, gin2.H{"code": 1, "msg": err.Error()})
		return
	}
	err := cmd.UpdateById(cmd.ID, c.GetUint("uid"), &cmd)
	if err != nil {
		c.JSON(200, gin2.H{"code": 2, "msg": err.Error()})
		return
	}
	CmdNoteFindAll(c)
}

func CmdNoteDeleteById(c *gin2.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(200, gin2.H{"code": 2, "msg": err.Error()})
		return
	}
	var cmd model.CmdNote
	err = cmd.DeleteByID(uint(id), c.GetUint("uid"))
	if err != nil {
		c.JSON(200, gin2.H{"code": 2, "msg": err.Error()})
		return
	}
	CmdNoteFindAll(c)
}
