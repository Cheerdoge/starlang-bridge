package server

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"starlang-bridge/model"
)

// InitDB 初始化 MySQL 连接、连接池并执行表结构迁移
func InitDB(cfg MySQLConfig) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
		Logger:         logger.Default.LogMode(parseLogLevel(cfg.LogLevel)),
		TranslateError: true,
	})
	if err != nil {
		return nil, fmt.Errorf("连接 MySQL 失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层连接池失败: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("MySQL Ping 失败: %w", err)
	}

	if err := db.AutoMigrate(
		&model.User{},
		&model.ChildProfile{},
		&model.TrainingSession{},
		&model.TrainingTurn{},
		&model.TaskCard{},
	); err != nil {
		return nil, fmt.Errorf("自动迁移表结构失败: %w", err)
	}

	return db, nil
}

// CloseDB 优雅关闭数据库连接
func CloseDB(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func parseLogLevel(level string) logger.LogLevel {
	switch level {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "info":
		return logger.Info
	default:
		return logger.Warn
	}
}
