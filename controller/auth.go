package controller

import (
	"github.com/gin-gonic/gin"
	"user_hub/common/code"
	"user_hub/common/dependencies"
	"user_hub/common/response"
	"user_hub/models/dto"
	"user_hub/repository/redis"
	"user_hub/utils"

	"net/http"
	"time"
)

// AuthController 处理认证相关的 HTTP 请求
// 该控制器负责管理验证码发送、用户登录等操作
type AuthController struct {
	smsClient dependencies.SMSClient // 短信服务客户端，用于发送验证码
	codeRepo  redis.CodeRepo         // Redis 仓库，用于存储验证码
}

// NewAuthController 创建 AuthController 实例，通过依赖注入初始化
// - smsClient: 短信服务客户端，用于发送短信
// - codeRepo: Redis 验证码存储仓库，用于管理和验证验证码
func NewAuthController(smsClient dependencies.SMSClient, codeRepo redis.CodeRepo) *AuthController {
	return &AuthController{
		smsClient: smsClient,
		codeRepo:  codeRepo,
	}
}

// SendCaptcha 向指定手机号发送验证码
// 该接口生成一个 6 位随机验证码，通过短信发送给用户，并在 Redis 中存储，设置 5 分钟有效期
// 响应中不包含验证码本身，仅返回成功消息，以确保安全性
//
// @Summary 发送短信验证码
// @Description 向指定手机号发送一个 6 位随机数字验证码，有效期为 5 分钟
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.SendCaptchaRequest true "请求体，包含手机号"
// @Success 200 {object} response.APIResponse[any] "验证码发送成功"
// @Failure 400 {object} response.APIResponse[string] "请求参数错误，例如手机号格式不正确"
// @Failure 500 {object} response.APIResponse[string] "服务器内部错误，例如短信发送失败或 Redis 存储失败"
// @Router /api/v1/auth/send-captcha [post]
func (c *AuthController) SendCaptcha(ctx *gin.Context) {
	// 第一步：解析并验证请求参数
	// - 从请求体中提取手机号，并通过 DTO 进行验证
	var req dto.SendCaptchaRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		// - 如果参数绑定失败（例如手机号为空或格式错误），返回 400 错误
		response.RespondError(ctx, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "无效的输入参数")
		return
	}

	// 第二步：生成 6 位随机验证码
	// - 使用 auth 包中的工具函数生成验证码
	captcha := utils.GenerateCaptcha()

	// 第三步：通过短信服务发送验证码
	// - 调用 SMS 客户端将验证码发送到用户手机号，假设服务端已内置频率限制
	if err := c.smsClient.SendCode(ctx, req.Phone, captcha); err != nil {
		// - 如果发送失败（例如网络问题或服务商限制），返回 500 错误
		response.RespondError(ctx, http.StatusInternalServerError, code.ErrCodeServerInternal, "验证码发送失败")
		return
	}

	// 第四步：在 Redis 中存储验证码
	// - 设置 5 分钟过期时间，以便后续验证使用
	expire := 5 * time.Minute
	if err := c.codeRepo.SetCaptcha(ctx, req.Phone, captcha, expire); err != nil {
		// - 如果存储失败（例如 Redis 连接问题），返回 500 错误
		response.RespondError(ctx, http.StatusInternalServerError, code.ErrCodeServerInternal, "验证码存储失败")
		return
	}

	// 第五步：返回成功响应
	// - 使用 Message 字段返回成功消息，Data 留空（nil），避免泄露验证码
	response.RespondSuccess[interface{}](ctx, nil, "验证码发送成功")
}

// RegisterRoutes 注册 AuthController 的路由
// 该方法将所有与认证相关的路由注册到指定的路由组中
// - 输入: group *gin.RouterGroup，路由组实例，用于注册路由
func (c *AuthController) RegisterRoutes(group *gin.RouterGroup) {
	// 第一步：创建 auth 子路由组
	// - 所有认证相关的路由都挂载在 /auth 下
	auth := group.Group("/auth")

	// 第二步：注册发送验证码路由
	// - 该路由无需认证，任何用户均可访问
	// - 方法: POST /api/v1/auth/send-captcha
	auth.POST("/send-captcha", c.SendCaptcha)

	// 注意：Logout 方法未在此处，因为它在 TokenController 中
	// 如果后续需要在此添加其他路由（如登录），可以继续扩展
}
