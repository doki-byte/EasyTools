package middleware

import (
	"EasyTools/app/controller/connect/ftp/app/config"
	"EasyTools/app/controller/connect/ftp/app/model"

	"github.com/gin-gonic/gin"
)

func DbCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 如果 DB 为空，尝试初始化
		if model.Db == nil {
			if err := model.DbMigrate(config.DefaultConfig.DbType, config.DefaultConfig.DbDsn); err != nil {
				c.Abort()
				c.JSON(500, gin.H{"code": 500, "msg": "数据库初始化失败: " + err.Error()})
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
				c.JSON(500, gin.H{"code": 500, "msg": "数据库重新连接失败: " + err.Error()})
				return
			}
		}

		// 检查关键表是否存在，如果不存在则重新迁移
		var count int64
		checkTable := model.Db.Raw("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name = 'ftp_config'").Scan(&count)
		if checkTable.Error != nil || count < 1 {
			// 表不存在，重新迁移
			if err := model.DbMigrate(config.DefaultConfig.DbType, config.DefaultConfig.DbDsn); err != nil {
				c.Abort()
				c.JSON(500, gin.H{"code": 500, "msg": "数据库表初始化失败: " + err.Error()})
				return
			}
		}

		c.Next()
	}
}
