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

// PhoneController 手机号控制器
type PhoneController struct {
	phoneService auth.Phone // 手机号服务实例
}

// NewPhoneController 创建 PhoneController 实例
// - 输入: phoneService 手机号服务实例
// - 输出: *PhoneController 控制器实例
// - 意图: 通过依赖注入初始化控制器，确保解耦和可测试性
func NewPhoneController(phoneService auth.Phone) *PhoneController {
	return &PhoneController{
		phoneService: phoneService,
	}
}

// LoginOrRegisterHandler 处理手机号登录或注册请求
// @Summary 手机号登录或注册
// @Description 使用手机号和验证码进行登录或注册
// @Tags 手机号认证
// @Accept json
// @Produce json
// @Param body body  dto.PhoneLoginOrRegisterData true "手机号和验证码"
// @Param X-Platform header string true "平台类型" Enums(web, wechat, app)
// @Success 200 {object} response.APIResponse[vo.LoginResponse] "登录或注册成功"
// @Failure 400 {object} response.APIResponse[any] "输入参数无效"
// @Failure 500 {object} response.APIResponse[any] "服务器内部错误"
// @Router /api/v1/phone/login [post]
func (ctrl *PhoneController) LoginOrRegisterHandler(c *gin.Context) {
	// 1. 绑定请求数据
	// - 意图: 将请求体中的 JSON 数据绑定到 DTO 结构体，若失败则返回 400 错误
	var phoneLoginOrRegisterData dto.PhoneLoginOrRegisterData
	if err := c.ShouldBindJSON(&phoneLoginOrRegisterData); err != nil {
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

	// 3. 调用服务层执行登录或注册
	// - 意图: 将请求数据和平台类型传递给服务层处理，返回用户信息和 token，若失败则根据错误类型返回对应状态码
	userInfo, tokenPair, err := ctrl.phoneService.LoginOrRegister(c.Request.Context(), phoneLoginOrRegisterData, platform)
	if err != nil {
		switch {
		case errors.Is(err, userError.ErrCaptchaExpired), errors.Is(err, userError.ErrCaptchaInvalid):
			response.RespondError(c, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "验证码错误")
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
	response.RespondSuccess(c, responseData, "登录或注册成功")
}

// RegisterRoutes 注册 PhoneController 的路由
// 该方法将手机号相关的路由注册到指定的路由组中
// - 输入: group *gin.RouterGroup，路由组实例，用于注册路由
// - 意图: 将手机号登录或注册的路由挂载到 /phone/login 路径下
func (ctrl *PhoneController) RegisterRoutes(group *gin.RouterGroup) {
	// 第一步：注册手机号登录或注册路由
	// - 意图: 处理手机号的登录或注册请求，路径为 /api/v1/phone/login
	// - 方法: POST
	group.POST("/phone/login", ctrl.LoginOrRegisterHandler)
}
