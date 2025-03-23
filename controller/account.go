package controller

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"user_hub/common/code"
	"user_hub/common/response"
	"user_hub/models/dto"
	"user_hub/models/enums"
	"user_hub/models/vo"
	"user_hub/service/auth"
	"user_hub/userError"
)

// AccountController 账号密码控制器
type AccountController struct {
	accountService auth.Account // 账号密码服务实例
}

// NewAccountController 创建 AccountController 实例
// - 输入: accountService 账号密码服务实例
// - 输出: *AccountController 控制器实例
// - 意图: 通过依赖注入初始化控制器，确保解耦和可测试性
func NewAccountController(accountService auth.Account) *AccountController {
	return &AccountController{
		accountService: accountService,
	}
}

// RegisterHandler 处理账号密码注册请求
// @Summary 账号密码注册
// @Description 使用账号和密码进行注册
// @Tags 账号密码认证
// @Accept json
// @Produce json
// @Param body body dto.AccountRegisterData true "注册信息"
// @Success 200 {object} response.APIResponse[vo.Userinfo] "注册成功"
// @Failure 400 {object} response.APIResponse[any] "输入参数无效"
// @Failure 500 {object} response.APIResponse[any] "服务器内部错误"
// @Router /api/v1/account/register [post]
func (ctrl *AccountController) RegisterHandler(c *gin.Context) {
	// 1. 绑定请求数据
	// - 意图: 将请求体中的 JSON 数据绑定到 DTO 结构体，若失败则返回 400 错误
	var accountRegisterData dto.AccountRegisterData
	if err := c.ShouldBindJSON(&accountRegisterData); err != nil {
		response.RespondError(c, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "输入参数无效")
		return
	}

	// 2. 调用服务层执行注册
	// - 意图: 将注册数据传递给服务层处理，返回用户信息，若失败则根据错误类型返回对应状态码
	userInfo, err := ctrl.accountService.Register(c.Request.Context(), accountRegisterData)
	if err != nil {
		switch {
		case errors.Is(err, userError.ErrServerInternal):
			response.RespondError(c, http.StatusInternalServerError, code.ErrCodeServerInternal, "服务器内部错误")
		default:
			response.RespondError(c, http.StatusBadRequest, code.ErrCodeClientInvalidInput, err.Error())
		}
		return
	}

	// 3. 返回成功响应
	// - 意图: 使用统一的成功响应格式返回用户信息
	response.RespondSuccess(c, userInfo, "注册成功")
}

// LoginHandler 处理账号密码登录请求
// @Summary 账号密码登录
// @Description 使用账号和密码进行登录
// @Tags 账号密码认证
// @Accept json
// @Produce json
// @Param body body dto.AccountLoginData true "登录信息"
// @Param X-Platform header string true "平台类型" Enums(web, wechat, app)
// @Success 200 {object} response.APIResponse[vo.LoginResponse] "登录成功"
// @Failure 400 {object} response.APIResponse[any] "输入参数无效"
// @Failure 500 {object} response.APIResponse[any] "服务器内部错误"
// @Router /api/v1/account/login [post]
func (ctrl *AccountController) LoginHandler(c *gin.Context) {
	// 1. 绑定请求数据
	// - 意图: 将请求体中的 JSON 数据绑定到 DTO 结构体，若失败则返回 400 错误
	var accountLoginData dto.AccountLoginData
	if err := c.ShouldBindJSON(&accountLoginData); err != nil {
		response.RespondError(c, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "输入参数无效")
		return
	}

	// 2. 获取并验证平台参数
	// - 意图: 从请求头 X-Platform 获取平台类型，并转换为 enums.Platform 类型，若失败则返回 400 错误
	platformStr := c.GetHeader("X-Platform")
	platform, err := enums.PlatformFromString(platformStr)
	if err != nil {
		response.RespondError(c, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "无效的平台类型")
		return
	}

	// 3. 调用服务层执行登录
	// - 意图: 将登录数据和平台类型传递给服务层处理，返回用户信息和 token，若失败则根据错误类型返回对应状态码
	userInfo, tokenPair, err := ctrl.accountService.Login(c.Request.Context(), accountLoginData, platform)
	if err != nil {
		switch {
		case errors.Is(err, userError.ErrServerInternal):
			response.RespondError(c, http.StatusInternalServerError, code.ErrCodeServerInternal, "服务器内部错误")
		default:
			response.RespondError(c, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "未知错误")
		}
		return
	}

	// 4. 构造响应数据
	// - 意图: 将用户信息和 token 封装成结构化的响应数据
	responseData := vo.LoginResponse{
		User:  userInfo,
		Token: tokenPair,
	}

	// 5. 返回成功响应
	// - 意图: 使用统一的成功响应格式返回数据
	response.RespondSuccess(c, responseData, "登录成功")
}

// RegisterRoutes 注册 AccountController 的路由
// 该方法将账号密码相关的路由注册到指定的路由组中
// - 输入: group *gin.RouterGroup，路由组实例，用于注册路由
// - 意图: 将账号密码的注册和登录路由挂载到 /account/register 和 /account/login 路径下
func (ctrl *AccountController) RegisterRoutes(group *gin.RouterGroup) {
	// 第一步：注册账号密码注册路由
	// - 意图: 处理账号密码的注册请求，路径为 /api/v1/account/register
	// - 方法: POST
	group.POST("/account/register", ctrl.RegisterHandler)

	// 第二步：注册账号密码登录路由
	// - 意图: 处理账号密码的登录请求，路径为 /api/v1/account/login
	// - 方法: POST
	group.POST("/account/login", ctrl.LoginHandler)
}
