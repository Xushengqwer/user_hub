package dependencies

import (
	"fmt"
	"user_hub/common/config"
	"user_hub/common/core"

	"time"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// InitMySQL 初始化 MySQL 连接并返回 *gorm.DB
func InitMySQL(cfg *config.MySQLConfig, logger *core.ZapLogger) (*gorm.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Database,
		cfg.Charset,
		cfg.ParseTime,
		cfg.Loc,
	)

	// 配置 GORM 日志模式， 创建 GormLogger 适配器
	gormLogger := core.NewGormLogger(logger)

	gormConfig := &gorm.Config{
		Logger: gormLogger, // 使用 GormLogger 作为 GORM 的日志接口
	}

	// 连接数据库
	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		logger.Error("无法连接到数据库", zap.Error(err))
		return nil, fmt.Errorf("无法连接到数据库: %w", err)
	}

	// 获取通用数据库对象 sql.DB 以便进行底层操作
	sqlDB, err := db.DB()
	if err != nil {
		logger.Error("无法获取数据库对象", zap.Error(err))
		return nil, fmt.Errorf("无法获取数据库对象: %w", err)
	}

	// 配置连接池
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConn)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConn)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 自动迁移数据库表结构
	err = db.AutoMigrate()
	if err != nil {
		logger.Error("数据库迁移失败", zap.Error(err))
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	logger.Info("成功连接到 MySQL 数据库并完成自动迁移")
	return db, nil
}
