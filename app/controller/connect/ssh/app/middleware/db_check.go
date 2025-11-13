package middleware

import (
	"EasyTools/app/controller/connect/ssh/app/config"
	"EasyTools/app/controller/connect/ssh/app/model"
	gin2 "EasyTools/app/controller/connect/ssh/gin"
)

func DbCheck() gin2.HandlerFunc {
	return func(c *gin2.Context) {
		// 如果 DB 为空，尝试初始化
		if model.Db == nil {
			if err := model.DbMigrate(config.DefaultConfig.DbType, config.DefaultConfig.DbDsn); err != nil {
				c.Abort()
				c.JSON(500, gin2.H{"code": 500, "msg": "数据库初始化失败: " + err.Error()})
				return
			}
			c.Next()
			return
		}

		// 检查是否还能正常连接数据库
		tx := model.Db.Exec("SELECT 1")
		if tx.Error != nil {
			if err := model.DbMigrate(config.DefaultConfig.DbType, config.DefaultConfig.DbDsn); err != nil {
				c.Abort()
				c.JSON(500, gin2.H{"code": 500, "msg": "数据库重新连接失败: " + err.Error()})
				return
			}
		}

		c.Next()
	}
}
