package initialization

import (
	"fmt"

	"github.com/Xushengqwer/go-common/core"
	"github.com/redis/go-redis/v9" // 使用 v9 版本
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Xushengqwer/user_hub/config"
	// 直接导入本地的 dependencies 包 (假设其包声明为 package dependencies)
	"github.com/Xushengqwer/user_hub/dependencies"
	"github.com/Xushengqwer/user_hub/utils"
)

// AppDependencies 封装了应用运行所需的所有基础依赖项。
// 设计目的:
//   - 将各个独立的依赖（数据库连接、Redis客户端、配置、日志等）聚合到一个结构体中。
//   - 方便在应用的不同层（如服务层、控制器层）之间传递这些共享的依赖。
type AppDependencies struct {
	Config       *config.UserHubConfig           // Config: 应用的全局配置。
	Logger       *core.ZapLogger                 // Logger: Zap 日志记录器实例。
	DB           *gorm.DB                        // DB: GORM 数据库连接实例 (通常是原始连接，非事务性)。
	RedisClient  *redis.Client                   // RedisClient: Redis v9 客户端实例。
	JwtToken     dependencies.JWTTokenInterface  // JWTUtil: JWT 工具实例。
	WechatClient dependencies.WechatClient       // WechatClient: 微信 API 客户端实例。
	SMSClient    dependencies.SMSClient          // SMSClient: 短信服务客户端实例。
	COSClient    dependencies.COSClientInterface // 新增 COS 客户端接口
}

// SetupDependencies 初始化应用所需的所有基础依赖项。
// 设计目的:
//   - 按正确的顺序创建和配置各个依赖组件（日志、数据库、Redis、外部客户端等）。
//   - 处理初始化过程中可能出现的错误。
//   - 返回一个包含所有已初始化依赖的 AppDependencies 结构体。
//
// 参数:
//   - cfg: *config.UserHubConfig，应用的全局配置。
//   - logger: *core.ZapLogger，已初始化的日志记录器实例。
//
// 返回:
//   - *AppDependencies: 包含所有成功初始化的依赖项的结构体指针。
//   - error: 如果任何关键依赖项初始化失败，则返回相应的错误。
func SetupDependencies(cfg *config.UserHubConfig, logger *core.ZapLogger) (*AppDependencies, error) {
	// 初始化 AppDependencies 结构体，用于存储所有依赖项
	var deps AppDependencies
	deps.Config = cfg    // 直接使用传入的配置
	deps.Logger = logger // 直接使用传入的日志记录器

	// 1. 注册自定义验证器
	//    - 这是应用启动时需要完成的基础设置。
	if err := utils.RegisterCustomValidators(); err != nil {
		// 如果注册失败，这是一个严重问题，应阻止应用启动。
		// 返回错误而不是直接 Fatal，让 main 函数处理退出。
		return nil, fmt.Errorf("注册自定义验证器失败: %w", err)
	}
	logger.Info("自定义验证器注册成功")

	// 2. 初始化数据库连接 (MySQL)
	//    - 依赖配置中的 MySQLConfig 和 logger。
	db, err := dependencies.InitMySQL(cfg, logger) // 直接使用包名调用
	if err != nil {
		// 数据库连接失败是关键错误。
		return nil, fmt.Errorf("初始化数据库失败: %w", err) // 保持错误包装
	}
	deps.DB = db
	logger.Info("数据库连接初始化成功")

	// 3. 初始化 Redis 连接
	//    - 依赖配置中的 RedisConfig 和 logger。
	//    - 注意：InitRedis 应返回 *redis.Client (v9 版本)
	redisClient, err := dependencies.InitRedis(&cfg.RedisConfig, logger) // 直接使用包名调用
	if err != nil {
		// Redis 连接失败也是关键错误。
		return nil, fmt.Errorf("初始化 Redis 失败: %w", err) // 保持错误包装
	}
	deps.RedisClient = redisClient
	logger.Info("Redis 连接初始化成功")

	// 4. 初始化 JWT 工具
	//    - 依赖配置中的 JWTConfig。
	deps.JwtToken = dependencies.NewJWTUtility(&cfg.JWTConfig) // 直接使用包名调用
	logger.Info("JWT 工具初始化成功")

	// 5. 初始化微信客户端工具
	//    - 依赖配置中的 WechatConfig。
	deps.WechatClient = dependencies.NewWechatClient(&cfg.WechatConfig) // 直接使用包名调用
	logger.Info("微信客户端初始化成功")

	// 6. 初始化短信服务客户端 (微信云托管)
	//    - 依赖配置中的 SMSConfig 和 logger。
	//    - NewSMSClient 内部会进行配置校验并可能返回错误。
	logger.Info("准备初始化短信服务客户端", zap.Any("smsConfig", cfg.SMSConfig)) // 打印配置用于调试
	smsClient, err := dependencies.NewSMSClient(&cfg.SMSConfig)      // 直接使用包名调用
	if err != nil {
		// 短信服务初始化失败可能是配置问题或依赖问题。
		// 根据业务重要性，决定是否将其视为启动失败。当前视为失败。
		logger.Error("初始化短信服务客户端失败", zap.Error(err)) // 记录错误详情
		return nil, fmt.Errorf("初始化短信服务失败: %w", err) // 返回包装后的错误
	}
	deps.SMSClient = smsClient // 字段名改为 SMSClient
	logger.Info("短信服务客户端初始化成功")

	// 7. 初始化 COS 客户端
	//    - 依赖配置中的 COSConfig 和 logger
	cosClient, err := dependencies.InitCOS(&cfg.COSConfig, logger)
	if err != nil {
		logger.Error("初始化 COS 客户端失败", zap.Error(err))
		return nil, fmt.Errorf("初始化 COS 客户端失败: %w", err)
	}
	deps.COSClient = cosClient
	logger.Info("COS 客户端初始化成功")

	// 8. 所有依赖项初始化成功，返回包含它们的结构体 (序号可能需要调整)
	logger.Info("所有基础依赖项初始化完成")
	return &deps, nil
}
