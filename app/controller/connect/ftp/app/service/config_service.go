package service

import (
	"EasyTools/app/controller/connect/ftp/app/model"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
)

func ConfigCreate(c *gin.Context) {
	var config model.FtpConfig
	if err := c.ShouldBind(&config); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "参数绑定失败: " + err.Error()})
		return
	}

	// 字段校验
	if err := validateFtpConfig(&config); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	// 规范化 connection_type
	if config.ConnectionType == "" {
		config.ConnectionType = ""
	}

	// 如果连接方式不是SSH，清空SSH配置
	if config.ConnectionType != "ssh" {
		config.SSHAddr = ""
		config.SSHPort = ""
		config.SSHUsername = ""
		config.SSHPassword = ""
	} else {
		// SSH模式下，如果端口为空，设置默认值
		if config.SSHPort == "" {
			config.SSHPort = "22"
		}
	}

	err := config.Create(&config)
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": "创建配置失败: " + err.Error()})
		return
	}
	ConfigFindAll(c)
}

func ConfigFindByID(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	var config model.FtpConfig
	data, err := config.FindByID(uint(id))
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "msg": "ok", "data": data})
}

func ConfigFindAll(c *gin.Context) {
	var config model.FtpConfig
	data, err := config.FindAll()
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	c.JSON(200, gin.H{"code": 0, "msg": "ok", "data": data})
}

func ConfigUpdateById(c *gin.Context) {
	var config model.FtpConfig
	if err := c.ShouldBind(&config); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": "参数绑定失败: " + err.Error()})
		return
	}

	// 检查ID是否存在
	if config.ID == 0 {
		c.JSON(200, gin.H{"code": 1, "msg": "配置ID不能为空"})
		return
	}

	// 更新时同样做一次基础校验，避免保存无效配置
	if err := validateFtpConfig(&config); err != nil {
		c.JSON(200, gin.H{"code": 1, "msg": err.Error()})
		return
	}

	// 如果密码为空，表示用户不想更新密码，需要排除密码字段
	updates := make(map[string]interface{})
	updates["name"] = config.Name
	updates["host"] = config.Host
	updates["port"] = config.Port
	updates["username"] = config.Username

	// 更新连接方式：'direct'（直连）、'ssh'（SSH隧道）或 ''（空字符串）
	// 无论是什么值（包括空字符串），都要更新，确保能正确保存和清空
	if config.ConnectionType == "" {
		updates["connection_type"] = ""
	} else {
		updates["connection_type"] = config.ConnectionType
	}

	// 更新SSH隧道配置
	// 如果连接方式不是SSH，清空所有SSH配置
	if config.ConnectionType != "ssh" {
		updates["ssh_addr"] = ""
		updates["ssh_port"] = ""
		updates["ssh_username"] = ""
		updates["ssh_password"] = ""
	} else {
		// SSH模式下，更新SSH字段
		if config.SSHAddr != "" {
			updates["ssh_addr"] = config.SSHAddr
			if config.SSHPort != "" {
				updates["ssh_port"] = config.SSHPort
			} else {
				updates["ssh_port"] = "22" // 默认SSH端口
			}
			updates["ssh_username"] = config.SSHUsername
			// SSH密码：如果为空字符串，表示用户想清空；如果不为空，则更新
			if config.SSHPassword != "" {
				updates["ssh_password"] = model.EncodePassword(config.SSHPassword)
			} else {
				// 如果SSH密码为空，也更新为空字符串（清空SSH密码）
				updates["ssh_password"] = ""
			}
		} else {
			// SSH地址为空，清空所有SSH配置
			updates["ssh_addr"] = ""
			updates["ssh_port"] = ""
			updates["ssh_username"] = ""
			updates["ssh_password"] = ""
		}
	}

	// 只有当密码不为空时才更新密码
	if config.Password != "" {
		// base64编码密码
		updates["password"] = model.EncodePassword(config.Password)
	}

	// 使用Updates方法，只更新提供的字段
	err := config.UpdateByIdSelective(config.ID, updates)
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": "更新配置失败: " + err.Error()})
		return
	}
	ConfigFindAll(c)
}

// validateFtpConfig 校验FTP配置必填字段
func validateFtpConfig(conf *model.FtpConfig) error {
	if conf == nil {
		return fmt.Errorf("配置不能为空")
	}

	if conf.Name == "" {
		return fmt.Errorf("连接名称不能为空")
	}
	if conf.Host == "" {
		return fmt.Errorf("主机不能为空")
	}
	if conf.Port == "" {
		return fmt.Errorf("端口不能为空")
	}

	return nil
}

func ConfigDeleteById(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	var config model.FtpConfig
	err = config.DeleteByID(uint(id))
	if err != nil {
		c.JSON(200, gin.H{"code": 2, "msg": err.Error()})
		return
	}
	ConfigFindAll(c)
}
