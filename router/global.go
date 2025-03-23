package router

import (
	"user_hub/common/config"
	"user_hub/common/core"
	"user_hub/common/dependencies"
	"user_hub/constants"
	"user_hub/controller"
	"user_hub/initialization"
	"user_hub/middleware"

	"github.com/gin-gonic/gin"
)

// SetupRouter 初始化并配置 Gin 路由器
// 该函数负责创建路由实例，注册全局中间件，设置统一的 API 前缀，并整合所有控制器的路由
// - 输入: logger 日志实例，用于记录请求和错误
// - 输入: cfg 配置实例，用于速率限制等中间件
// - 输入: smsClient 短信服务客户端，用于 AuthController
// - 输入: jwtUtil JWT 工具实例，用于认证中间件
// - 输入: appServices 应用服务结构体，包含所有服务实例
// - 输出: *gin.Engine 配置完成的路由器实例
func SetupRouter(
	logger *core.ZapLogger,
	cfg *config.RateLimitConfig,
	smsClient dependencies.SMSClient,
	jwtUtil dependencies.JWTUtilityInterface,
	appServices *initialization.AppServices, // 使用 AppServices 结构体
) *gin.Engine {
	// 第一步：创建 Gin 路由器实例
	router := gin.Default()

	// 第二步：注册全局中间件（按指定顺序）
	router.Use(middleware.ErrorHandlingMiddleware(logger))
	router.Use(middleware.RequestIDMiddleware())
	router.Use(middleware.RequestLoggerMiddleware(logger))
	router.Use(middleware.CorsMiddleware())
	router.Use(middleware.RateLimitMiddleware(logger, cfg))
	router.Use(middleware.RequestTimeoutMiddleware(logger, constants.RequestTimeout))

	// 第三步：创建统一的 API 前缀分组
	v1 := router.Group("/api/v1")

	// 第四步：初始化所有控制器并注册路由
	// 4.1 初始化 AuthController 并注册路由
	authCtrl := controller.NewAuthController(smsClient, appServices.CodeRepo)
	authCtrl.RegisterRoutes(v1)

	// 4.2 初始化 IdentityController 并注册路由
	identityCtrl := controller.NewIdentityController(appServices.IdentityService, jwtUtil)
	identityCtrl.RegisterRoutes(v1)

	// 4.3 初始化 ProfileController 并注册路由
	profileCtrl := controller.NewProfileController(appServices.ProfileService, jwtUtil)
	profileCtrl.RegisterRoutes(v1)

	// 4.4 初始化 TokenController 并注册路由
	tokenCtrl := controller.NewTokenController(appServices.TokenService, jwtUtil)
	tokenCtrl.RegisterRoutes(v1)

	// 4.5 初始化 QueryController 并注册路由
	queryCtrl := controller.NewQueryController(appServices.QueryService, jwtUtil)
	queryCtrl.RegisterRoutes(v1)

	// 4.6 初始化 UserController 并注册路由
	userCtrl := controller.NewUserController(appServices.UserService, jwtUtil)
	userCtrl.RegisterRoutes(v1)

	// 4.7 初始化 WechatMiniProgramController 并注册路由
	wechatCtrl := controller.NewWechatMiniProgramController(appServices.WechatMiniProgram)
	wechatCtrl.RegisterRoutes(v1)

	// 4.8 初始化 AccountController 并注册路由
	accountCtrl := controller.NewAccountController(appServices.Account)
	accountCtrl.RegisterRoutes(v1)

	// 4.9 初始化 PhoneController 并注册路由
	phoneCtrl := controller.NewPhoneController(appServices.Phone)
	phoneCtrl.RegisterRoutes(v1)

	// 第五步：返回配置完成的路由器
	return router
}
