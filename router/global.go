package router

import (
	// 引入公共模块和项目包
	"github.com/Xushengqwer/go-common/core" // 引入日志包
	"github.com/Xushengqwer/user_hub/config"
	"github.com/Xushengqwer/user_hub/constants"
	swaggerFiles "github.com/swaggo/files"     // swagger-files 包
	ginSwagger "github.com/swaggo/gin-swagger" // gin-swagger 包
	"time"

	commonMiddleware "github.com/Xushengqwer/go-common/middleware"

	"github.com/gin-gonic/gin"

	otelgin "go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

	"github.com/Xushengqwer/user_hub/controller"
	"github.com/Xushengqwer/user_hub/dependencies"
	_ "github.com/Xushengqwer/user_hub/docs" // 引入 docs 包以注册 Swagger 信息
	"github.com/Xushengqwer/user_hub/initialization"
)

// SetupRouter 初始化并配置 Gin 引擎，注册所有中间件和路由。
// 设计目的:
//   - 作为应用路由配置的统一入口点。
//   - 应用全局中间件，处理通用逻辑如日志、错误恢复、超时、限流、CORS等。
//   - 创建 API 版本分组（/api/v1）。
//   - 实例化所有控制器，并将它们的路由注册到相应的分组下。
//
// 注意: 认证和权限校验预期由上游网关处理，此服务不再包含相关中间件。
// 参数:
//   - logger: Zap 日志记录器实例，用于中间件和控制器。
//   - cfg: 应用的全局配置 (UserHubConfig)，用于获取 RateLimitConfig 等。
//   - jwtUtil: JWT 工具实例，传递给需要它的控制器。
//   - appServices: 包含所有已初始化服务实例的结构体。
//
// 返回:
//   - *gin.Engine: 配置完成的 Gin 引擎实例，可以直接运行。
func SetupRouter(
	logger *core.ZapLogger,
	cfg *config.UserHubConfig, // 传入完整的 UserHubConfig
	jwtUtil dependencies.JWTTokenInterface,
	appServices *initialization.AppServices,
	appDeps *initialization.AppDependencies, // <-- 传入 AppDependencies 包含了 DB
) *gin.Engine {
	logger.Info("开始设置 Gin 路由...")

	// 1. 创建 Gin 引擎实例
	//    使用 gin.Default() 包含 Logger 和 Recovery 中间件。Recovery 是有用的。
	router := gin.Default()

	// 1. OTel Middleware (最先，处理追踪上下文和 Span)
	router.Use(otelgin.Middleware(constants.ServiceName))

	// 2. Panic Recovery (捕获后续中间件和 handler 的 panic)
	router.Use(commonMiddleware.ErrorHandlingMiddleware(logger))

	// 3. Request Logger (记录访问日志，需要 TraceID)
	// 注意：你的 RequestLoggerMiddleware 需要 *zap.Logger，而你注入的是 *core.ZapLogger
	// 你需要将 core.ZapLogger 适配一下，或者修改中间件接收 core.ZapLogger
	// 假设你的 core.ZapLogger 有一个方法 .Logger() 返回底层的 *zap.Logger
	if baseLogger := logger.Logger(); baseLogger != nil {
		router.Use(commonMiddleware.RequestLoggerMiddleware(baseLogger))
	} else {
		logger.Warn("无法获取底层的 *zap.Logger，跳过 RequestLoggerMiddleware 注册")
	}

	// 4. Request Timeout (超时控制)
	// 假设配置中的 RequestTimeout 是秒数
	requestTimeout := time.Duration(cfg.ServerConfig.RequestTimeout) * time.Second
	router.Use(commonMiddleware.RequestTimeoutMiddleware(logger, requestTimeout))

	// 5. User Context (提取用户信息)
	router.Use(commonMiddleware.UserContextMiddleware())
	// 3. 创建 API 版本分组 /api/v1
	v1 := router.Group("api/v1/user-hub")
	logger.Info("API 路由将注册到 api/v1/user-hub 分组下")

	// 4. 初始化所有控制器 (使用更新后的名称和依赖)
	accountCtrl := controller.NewAccountController(appServices.Account, logger, cfg.CookieConfig)
	authCtrl := controller.NewAuthController(appServices.SMS, appServices.CodeRepo, logger) // AuthController 依赖 SMS, CodeRepo, Logger
	identityCtrl := controller.NewIdentityController(appServices.IdentityService, jwtUtil, logger)
	phoneCtrl := controller.NewPhoneAuthController(appServices.Phone, logger, cfg.CookieConfig) // 使用更新后的名称和依赖
	profileCtrl := controller.NewUserProfileController(appServices.ProfileService, jwtUtil, logger, appDeps.DB)
	tokenCtrl := controller.NewAuthTokenController(appServices.TokenService, jwtUtil, logger, cfg.CookieConfig)
	userCtrl := controller.NewUserController(appServices.UserService, jwtUtil, logger)
	userListQueryCtrl := controller.NewUserListQueryController(appServices.QueryService, jwtUtil, logger)
	wechatCtrl := controller.NewWechatAuthController(appServices.WechatMiniProgram, logger) // 使用更新后的名称和依赖

	// 5. 注册每个控制器的路由到 /api/v1 分组
	accountCtrl.RegisterRoutes(v1)
	authCtrl.RegisterRoutes(v1)
	identityCtrl.RegisterRoutes(v1)
	phoneCtrl.RegisterRoutes(v1)
	profileCtrl.RegisterRoutes(v1)
	tokenCtrl.RegisterRoutes(v1)
	userCtrl.RegisterRoutes(v1)
	userListQueryCtrl.RegisterRoutes(v1)
	wechatCtrl.RegisterRoutes(v1)

	logger.Info("所有业务路由已成功注册")

	// 6. 配置 Swagger UI 路由
	//    确保已在 main.go 或此处导入 _ "user_hub/docs"
	//    访问路径通常是 /swagger/index.html
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	logger.Info("Swagger UI 路由已注册，访问路径: /swagger/index.html")

	// 7. 返回配置好的 Gin 引擎
	return router
}
