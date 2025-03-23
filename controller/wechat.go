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
	"user_hub/service/auth/oAuth"
	"user_hub/userError"
)

// WechatMiniProgramController 微信小程序控制器
type WechatMiniProgramController struct {
	wechatService oAuth.WechatMiniProgram // 微信小程序服务实例
}

// NewWechatMiniProgramController 创建 WechatMiniProgramController 实例
// - 输入: wechatService 微信小程序服务实例
// - 输出: *WechatMiniProgramController 控制器实例
// - 意图: 通过依赖注入初始化控制器，确保解耦和可测试性
func NewWechatMiniProgramController(wechatService oAuth.WechatMiniProgram) *WechatMiniProgramController {
	return &WechatMiniProgramController{
		wechatService: wechatService,
	}
}

// LoginOrRegisterHandler 处理微信小程序登录或注册请求
// @Summary 微信小程序登录或注册
// @Description 使用微信小程序授权码进行登录或注册
// @Tags 微信小程序认证
// @Accept json
// @Produce json
// @Param  body body dto.WechatMiniProgramLoginData true "授权码"
// @Param X-Platform header string true "平台类型" Enums(web, wechat, app)
// @Success 200 {object} response.APIResponse[vo.LoginResponse] "登录或注册成功"
// @Failure 400 {object} response.APIResponse[any] "输入参数无效"
// @Failure 500 {object} response.APIResponse[any] "服务器内部错误"
// @Router /api/v1/wechat/login [post]
func (ctrl *WechatMiniProgramController) LoginOrRegisterHandler(c *gin.Context) {
	// 1. 绑定请求数据
	// - 意图: 将请求体中的 JSON 数据绑定到 DTO 结构体，若失败则返回 400 错误
	var wechatMiniProgramLoginData dto.WechatMiniProgramLoginData
	if err := c.ShouldBindJSON(&wechatMiniProgramLoginData); err != nil {
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
	userInfo, tokenPair, err := ctrl.wechatService.LoginOrRegister(c.Request.Context(), wechatMiniProgramLoginData, platform)
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
	response.RespondSuccess(c, responseData, "登录或注册成功")
}

// RegisterRoutes 注册 WechatMiniProgramController 的路由
// 该方法将微信小程序相关的路由注册到指定的路由组中
// - 输入: group *gin.RouterGroup，路由组实例，用于注册路由
// - 意图: 将微信小程序登录或注册的路由挂载到 /wechat/login 路径下
func (ctrl *WechatMiniProgramController) RegisterRoutes(group *gin.RouterGroup) {
	// 第一步：注册微信小程序登录或注册路由
	// - 意图: 处理微信小程序的登录或注册请求，路径为 /api/v1/wechat/login
	// - 方法: POST
	group.POST("/wechat/login", ctrl.LoginOrRegisterHandler)
}
