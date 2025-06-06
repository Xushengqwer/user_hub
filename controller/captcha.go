package controller

import (
	"net/http"
	"time"

	"github.com/Xushengqwer/go-common/commonerrors" // 引入公共错误包
	"github.com/Xushengqwer/go-common/core"         // 引入日志包
	"github.com/Xushengqwer/go-common/response"
	"github.com/Xushengqwer/user_hub/dependencies"
	"github.com/Xushengqwer/user_hub/models/dto"
	"github.com/Xushengqwer/user_hub/repository/redis"
	"github.com/Xushengqwer/user_hub/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap" // 引入 zap 用于日志字段
)

// AuthController 处理与认证辅助功能相关的 HTTP 请求，例如发送验证码。
// 注意：登录、注册、登出、刷新令牌等核心认证流程由其他控制器（如 AccountController, TokenController）处理。
type AuthController struct {
	smsClient dependencies.SMSClient // smsClient: 短信服务客户端，用于实际发送短信。
	codeRepo  redis.CodeRepo         // codeRepo: Redis 验证码仓库，用于存储和验证验证码。
	logger    *core.ZapLogger        // logger: 日志记录器。
}

// NewAuthController 创建一个新的 AuthController 实例。
// 设计目的:
//   - 通过依赖注入传入所需的服务和仓库实例，以及日志记录器。
//
// 参数:
//   - smsClient: 实现了 dependencies.SMSClient 接口的短信服务实例。
//   - codeRepo: 实现了 redis.CodeRepo 接口的验证码仓库实例。
//   - logger: 日志记录器实例。
//
// 返回:
//   - *AuthController: 初始化完成的控制器实例。
func NewAuthController(
	smsClient dependencies.SMSClient,
	codeRepo redis.CodeRepo,
	logger *core.ZapLogger, // 注入 logger
) *AuthController {
	return &AuthController{
		smsClient: smsClient,
		codeRepo:  codeRepo,
		logger:    logger, // 存储 logger
	}
}

// SendCaptcha 处理发送短信验证码的请求。
// 流程: 校验手机号 -> 生成验证码 -> 调用短信服务发送 -> 将验证码存入 Redis (设置过期时间)。
// @Summary 发送短信验证码
// @Description 向用户指定的手机号发送一个6位随机数字验证码，该验证码在5分钟内有效。
// @Tags 认证辅助 (Auth Helper)
// @Accept json
// @Produce json
// @Param request body dto.SendCaptchaRequest true "请求体，包含目标手机号"
// @Success 200 {object} docs.SwaggerAPIEmptyResponse "验证码发送成功（响应体中不包含验证码）"
// @Failure 400 {object} docs.SwaggerAPIErrorResponseString "请求参数无效 (如JSON格式错误、手机号格式不正确)"
// @Failure 500 {object} docs.SwaggerAPIErrorResponseString "系统内部错误 (如短信服务发送失败、Redis存储失败)"
// @Router /api/v1/user-hub/auth/send-captcha [post] // <--- 已更新路径
func (ctrl *AuthController) SendCaptcha(c *gin.Context) {
	const operation = "AuthController.SendCaptcha" // 操作标识，用于日志

	// 1. 绑定并校验请求体数据。
	var req dto.SendCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ctrl.logger.Warn("发送验证码请求参数绑定失败",
			zap.String("operation", operation),
			zap.Error(err), // 记录绑定错误
		)
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "无效的输入参数")
		return
	}
	// TODO: 可以在DTO的binding标签或此处添加更严格的手机号格式校验逻辑，
	//       例如使用自定义validator。当前假设dto.SendCaptchaRequest已有基础校验。

	// 2. 生成6位随机验证码。
	captcha := utils.GenerateCaptcha()
	ctrl.logger.Info("已生成验证码",
		zap.String("operation", operation),
		zap.String("phone", req.Phone), // 注意：生产环境日志中手机号可能需要脱敏
		// 不记录验证码本身到常规日志，除非是调试模式下的特定日志级别
	)

	// 3. 调用短信服务发送验证码。
	//    此处假设短信服务本身会处理发送频率限制等问题。
	if err := ctrl.smsClient.SendCode(c.Request.Context(), req.Phone, captcha); err != nil {
		ctrl.logger.Error("调用短信服务发送验证码失败",
			zap.String("operation", operation),
			zap.String("phone", req.Phone),
			zap.Error(err), // 记录短信服务返回的原始错误
		)
		// 短信发送失败是系统层面问题，返回通用系统错误。
		response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, commonerrors.ErrSystemError.Error())
		return
	}
	ctrl.logger.Info("短信服务发送验证码成功", zap.String("operation", operation), zap.String("phone", req.Phone))

	// 4. 在 Redis 中存储验证码，并设置5分钟过期时间。
	//    这是为了后续用户使用验证码登录/注册时进行校验。
	expire := 5 * time.Minute
	if err := ctrl.codeRepo.SetCaptcha(c.Request.Context(), req.Phone, captcha, expire); err != nil {
		ctrl.logger.Error("将验证码存入 Redis 失败",
			zap.String("operation", operation),
			zap.String("phone", req.Phone),
			zap.Error(err), // 记录 Redis 操作错误
		)
		// Redis 存储失败是系统层面问题，返回通用系统错误。
		response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, commonerrors.ErrSystemError.Error())
		return
	}
	ctrl.logger.Info("验证码成功存入 Redis",
		zap.String("operation", operation),
		zap.String("phone", req.Phone),
		zap.Duration("expire", expire),
	)

	// 5. 返回成功响应。
	//    响应体中不应包含验证码本身，以确保安全。
	response.RespondSuccess[interface{}](c, nil, "验证码发送成功，请注意查收")
}

// RegisterRoutes 注册与认证辅助功能相关的路由到指定的 Gin 路由组。
// 设计目的:
//   - 集中管理此控制器的路由。
//
// 参数:
//   - group: Gin 的路由组实例。如果 group 是 router.Group("/user-hub/api/v1")，
//     那么这里注册的 "/auth" 子分组的完整基础路径将是 "/user-hub/api/v1/auth"。
func (ctrl *AuthController) RegisterRoutes(group *gin.RouterGroup) {
	// 创建 /auth 子路由组，用于挂载所有认证辅助相关的接口
	authRoutes := group.Group("/auth")
	{
		// 注册发送短信验证码的接口
		// - 路径: /api/v1/user-hub/send-captcha (相对于 authRoutes 的基础路径)
		//   完整路径将是 group的基础路径 + "/auth" + "/send-captcha"
		//   例如: "/user-hub/api/v1/auth/send-captcha"
		// - 方法: POST
		// - 此接口通常不需要用户认证即可访问。
		authRoutes.POST("/send-captcha", ctrl.SendCaptcha)
	}
	// 注意：核心的登录、注册、登出、刷新令牌等接口通常在其他专门的控制器中定义和注册。
}
