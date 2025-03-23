package initialization

import (
	"fmt"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"user_hub/common/config"
	"user_hub/common/core"
	"user_hub/common/dependencies"
	"user_hub/utils"
)

type AppDependencies struct {
	Config       *config.GlobalConfig
	Logger       *core.ZapLogger
	DB           *gorm.DB
	RedisClient  *redis.Client
	JwtToken     dependencies.JWTUtilityInterface
	WechatClient dependencies.WechatClient
	SMS          dependencies.SMSClient
}

func SetupDependencies(cfg *config.GlobalConfig, logger *core.ZapLogger) (*AppDependencies, error) {
	var deps AppDependencies

	// 1. 注册自定义验证器
	if err := utils.RegisterCustomValidators(); err != nil {
		logger.Fatal("注册自定义验证器失败", zap.Error(err))
	}

	// 2. 初始化数据库
	db, err := dependencies.InitMySQL(&cfg.MySQLConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("初始化数据库失败: %w", err)
	}
	deps.DB = db

	// 3. 初始化 Redis
	redisClient, err := dependencies.InitRedis(&cfg.RedisConfig, logger)
	if err != nil {
		return nil, fmt.Errorf("初始化 Redis 失败: %w", err)
	}
	deps.RedisClient = redisClient

	// 4. 初始化JWT生成工具
	deps.JwtToken = dependencies.NewJWTUtility(&cfg.JWTConfig)

	// 5. 初始化微信客户端工具
	deps.WechatClient = dependencies.NewWechatClient(&cfg.WechatConfig)

	// 6. 检查 SMSConfig 是否正确加载
	logger.Info("SMSConfig 检查", zap.Any("smsConfig", cfg.SMSConfig))
	if cfg.SMSConfig.AppID == "" || cfg.SMSConfig.Secret == "" || cfg.SMSConfig.Endpoint == "" {
		logger.Error("SMSConfig 字段缺失", zap.Any("smsConfig", cfg.SMSConfig))
	}

	// 6. 初始化sms工具 --- 微信云托管
	smsClient, err := dependencies.NewSMSClient(&cfg.SMSConfig)
	if err != nil {
		return nil, err
	}
	deps.SMS = smsClient

	return &deps, nil
}
