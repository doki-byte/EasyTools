package model

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var Db *gorm.DB

// DbMigrate 打开 SQLite3 数据库文件并自动迁移模型。
func DbMigrate(dbType, dsn string) error {
	defer func() {
		if err := recover(); err != nil {
			slog.Error("DbMigrate panic", "err", err)
		}
	}()

	if dbType != "sqlite3" {
		return fmt.Errorf("unsupported db type: %s", dbType)
	}

	dir := filepath.Dir(dsn)
	if dir != "." {
		if err := os.MkdirAll(dir, 755); err != nil {
			return fmt.Errorf("failed to create directory %q: %v", dir, err)
		}
	}

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn), // 提高日志级别便于调试
	})
	if err != nil {
		return fmt.Errorf("failed to connect sqlite database %q: %v", dsn, err)
	}
	Db = db

	sqlDB, err := Db.DB()
	if err != nil {
		return fmt.Errorf("failed to get generic database object: %v", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %v", err)
	}

	// 打印数据库文件信息
	if info, err := os.Stat(dsn); err == nil {
		slog.Info("Opened database file", "path", dsn, "size", info.Size())
	}

	// 自动迁移所有模型表
	if err := Db.AutoMigrate(
		&SshConf{},
		&CmdNote{},
	); err != nil {
		slog.Error("AutoMigrate error", "err", err)
		return err
	}

	slog.Info("AutoMigrate completed successfully")
	return nil
}
