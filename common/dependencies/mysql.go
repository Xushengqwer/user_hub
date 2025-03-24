package dependencies

import (
	"fmt"
	"user_hub/common/config"
	"user_hub/common/core"
	"user_hub/models/entities"

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
	// 重试逻辑
	var db *gorm.DB
	var err error
	maxRetries := 5
	retryInterval := 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		db, err = gorm.Open(mysql.Open(dsn), gormConfig)
		if err == nil {
			// 验证连接是否可用
			sqlDB, err := db.DB()
			if err == nil && sqlDB.Ping() == nil {
				break // 连接成功，跳出重试循环
			}
		}
		logger.Warn("无法连接到 MySQL，尝试重试", zap.Int("retry", i+1), zap.Int("maxRetries", maxRetries), zap.Error(err))
		if i < maxRetries-1 { // 最后一次失败时不再等待
			time.Sleep(retryInterval)
		}
	}

	if err != nil {
		logger.Error("无法连接到数据库", zap.Error(err))
		return nil, fmt.Errorf("无法连接到数据库: %w", err)
	}

	// todo  为适应微信云托管的MySQL5.7版本，临时调整 sql_mode,如果部署到其他云厂商可以升级MySQL版本
	err = db.Exec("SET SESSION sql_mode = 'NO_ENGINE_SUBSTITUTION'").Error
	if err != nil {
		logger.Error("无法设置 sql_mode", zap.Error(err))
		return nil, fmt.Errorf("无法设置 sql_mode: %w", err)
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
	err = db.AutoMigrate(
		&entities.User{},
		&entities.UserIdentity{},
		&entities.UserProfile{},
	)
	if err != nil {
		logger.Error("数据库迁移失败", zap.Error(err))
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	logger.Info("成功连接到 MySQL 数据库并完成自动迁移")
	return db, nil
}
