package dependencies

import (
	"fmt"
	"time"

	"github.com/Xushengqwer/go-common/core"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/Xushengqwer/user_hub/config"
	"github.com/Xushengqwer/user_hub/models/entities"
)

// InitMySQL 初始化 MySQL 连接并返回 *gorm.DB
func InitMySQL(cfg *config.UserHubConfig, logger *core.ZapLogger) (*gorm.DB, error) {
	if cfg.MySQLConfig.DSN == "" {
		logger.Error("MySQL DSN is not configured")
		return nil, fmt.Errorf("MySQL DSN is empty in configuration")
	}

	// 配置 GORM 日志模式， 创建 GormLogger 适配器
	gormLogger := core.NewGormLogger(logger, cfg.GormLogConfig) // 假设 cfg.GormLogConfig 仍然存在且适用

	gormConfig := &gorm.Config{
		Logger: gormLogger, // 使用 GormLogger 作为 GORM 的日志接口
	}

	// 连接数据库
	// 重试逻辑
	var db *gorm.DB
	var err error
	maxRetries := 5
	retryInterval := 2 * time.Second

	logger.Info("Attempting to connect to MySQL using DSN", zap.String("dsn_preview", previewDSN(cfg.MySQLConfig.DSN))) // 记录DSN预览，注意隐藏密码

	for i := 0; i < maxRetries; i++ {
		db, err = gorm.Open(mysql.Open(cfg.MySQLConfig.DSN), gormConfig)
		if err == nil {
			// 验证连接是否可用
			sqlDB, pingErr := db.DB()
			if pingErr == nil {
				if err = sqlDB.Ping(); err == nil {
					break // 连接成功，跳出重试循环
				}
			} else {
				err = pingErr // db.DB() 可能出错
			}
		}
		logger.Warn("无法连接到 MySQL，尝试重试",
			zap.Int("retry", i+1),
			zap.Int("maxRetries", maxRetries),
			zap.Error(err),
			zap.String("dsn_preview", previewDSN(cfg.MySQLConfig.DSN)),
		)
		if i < maxRetries-1 { // 最后一次失败时不再等待
			time.Sleep(retryInterval)
		}
	}

	if err != nil {
		logger.Error("无法连接到数据库 (DSN)",
			zap.Error(err),
			zap.String("dsn_preview", previewDSN(cfg.MySQLConfig.DSN)),
		)
		return nil, fmt.Errorf("无法连接到数据库 (DSN: %s): %w", previewDSN(cfg.MySQLConfig.DSN), err)
	}

	// 获取通用数据库对象 sql.DB 以便进行底层操作
	sqlDB, err := db.DB()
	if err != nil {
		logger.Error("无法获取数据库对象", zap.Error(err))
		return nil, fmt.Errorf("无法获取数据库对象: %w", err)
	}

	// 配置连接池
	sqlDB.SetMaxIdleConns(cfg.MySQLConfig.MaxIdleConn)
	sqlDB.SetMaxOpenConns(cfg.MySQLConfig.MaxOpenConn)
	sqlDB.SetConnMaxLifetime(time.Hour) // 建议这个值也加入配置

	// 自动迁移数据库表结构
	// 注意：确保你的 GORM 版本与 entities 定义兼容
	err = db.AutoMigrate(
		&entities.User{},
		&entities.UserIdentity{},
		&entities.UserProfile{},
	)
	if err != nil {
		logger.Error("数据库迁移失败", zap.Error(err))
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	logger.Info("成功连接到 MySQL 数据库 (使用DSN) 并完成自动迁移")
	return db, nil
}

// previewDSN 返回一个用于日志记录的DSN预览版本，隐藏密码。
// 这是一个简单的实现，你可能需要根据你的DSN格式进行调整。
func previewDSN(dsn string) string {
	// 尝试找到 "@" 符号，密码通常在它之前
	atIndex := -1
	for i := len(dsn) - 1; i >= 0; i-- {
		if dsn[i] == '@' {
			atIndex = i
			break
		}
	}

	if atIndex == -1 { // 没有找到 "@"，可能DSN格式不同或不含密码
		return dsn
	}

	// 尝试找到密码开始的位置，通常在第一个 ":" 之后，但在 "@" 之前
	passwordStartIndex := -1
	for i := 0; i < atIndex; i++ {
		if dsn[i] == ':' {
			passwordStartIndex = i + 1
			break
		}
	}

	if passwordStartIndex == -1 || passwordStartIndex >= atIndex { // 没有找到密码的明确分隔符
		return dsn
	}

	return dsn[:passwordStartIndex] + "****" + dsn[atIndex:]
}
