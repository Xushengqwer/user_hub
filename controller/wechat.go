package controller

import (
	"errors"
	"net/http"

	"github.com/Xushengqwer/go-common/commonerrors" // 引入公共错误包
	"github.com/Xushengqwer/go-common/core"         // 引入日志包
	"github.com/Xushengqwer/go-common/models/enums"
	"github.com/Xushengqwer/go-common/response"
	"github.com/Xushengqwer/user_hub/models/dto"
	"github.com/Xushengqwer/user_hub/models/vo"
	"github.com/Xushengqwer/user_hub/service/login/oAuth" // Corrected import path
	"github.com/gin-gonic/gin"
	"go.uber.org/zap" // 引入 zap 用于日志字段
)

// WechatAuthController 处理与微信小程序认证相关的 HTTP 请求。
// 依赖于 oAuth.WechatMiniProgramService 来执行核心业务逻辑。
type WechatAuthController struct {
	wechatService oAuth.WechatMiniProgramService // wechatService: 微信小程序认证服务的实例。
	logger        *core.ZapLogger                // logger: 日志记录器。
}

// NewWechatAuthController 创建一个新的 WechatAuthController 实例。
// 设计目的:
//   - 通过依赖注入传入 wechatService 和 logger，增强了代码的可测试性和模块化。
//
// 参数:
//   - wechatService: 实现了 oAuth.WechatMiniProgramService 接口的服务实例。
//   - logger: 日志记录器实例。
//
// 返回:
//   - *WechatAuthController: 初始化完成的控制器实例。
func NewWechatAuthController(
	wechatService oAuth.WechatMiniProgramService,
	logger *core.ZapLogger, // 注入 logger
) *WechatAuthController {
	return &WechatAuthController{
		wechatService: wechatService,
		logger:        logger, // 存储 logger
	}
}

// LoginOrRegisterHandler 处理微信小程序用户使用 code 进行登录或自动注册的请求。
// @Summary 微信小程序登录或注册
// @Description 用户通过提供微信小程序 wx.login() 获取的 code，进行登录或（如果首次登录）自动注册账户。
// @Tags 微信小程序认证
// @Accept json
// @Produce json
// @Param body body dto.WechatMiniProgramLoginData true "包含微信小程序 code 的请求体"
// @Param X-Platform header string true "客户端平台类型" Enums(web, wechat, app) default(wechat)
// @Success 200 {object} docs.SwaggerAPILoginResponse "登录或注册成功，返回用户信息及访问和刷新令牌"
// @Failure 400 {object} docs.SwaggerAPIErrorResponseString "请求参数无效 (如JSON格式错误、code为空、平台类型无效) 或 业务逻辑错误 (如微信 code 无效或已过期、用户状态异常)"
// @Failure 500 {object} docs.SwaggerAPIErrorResponseString "系统内部错误 (如调用微信API失败、数据库操作失败、令牌生成失败)"
// @Router /api/v1/user-hub/wechat/login [post] // <--- 已更新路径
func (ctrl *WechatAuthController) LoginOrRegisterHandler(c *gin.Context) {
	const operation = "WechatAuthController.LoginOrRegisterHandler"

	// 1. 绑定并校验请求体数据。
	var wechatLoginData dto.WechatMiniProgramLoginData
	if err := c.ShouldBindJSON(&wechatLoginData); err != nil {
		ctrl.logger.Warn("微信登录/注册请求参数绑定失败", zap.String("operation", operation), zap.Error(err))
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "输入参数无效")
		return
	}
	// code 的有效性由服务层调用微信 API 时校验。

	// 2. 获取并验证请求头中的 X-Platform 参数。
	platformStr := c.GetHeader("X-Platform")
	platform, err := enums.PlatformFromString(platformStr)
	if err != nil {
		ctrl.logger.Warn("无效的平台类型",
			zap.String("operation", operation),
			zap.String("platformHeader", platformStr),
			zap.String("code", wechatLoginData.Code), // 记录关联的 code
			zap.Error(err),
		)
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "无效的平台类型")
		return
	}

	// 3. 调用服务层执行登录或注册逻辑。
	//    服务层会处理 code 换取 openid、用户查找/创建、状态检查和令牌生成。
	userInfo, tokenPair, err := ctrl.wechatService.LoginOrRegister(c.Request.Context(), wechatLoginData, platform)
	if err != nil {
		// 根据服务层返回的错误类型记录日志并响应。
		if errors.Is(err, commonerrors.ErrSystemError) {
			ctrl.logger.Error("微信登录/注册服务返回系统错误",
				zap.String("operation", operation),
				zap.String("code", wechatLoginData.Code),
				zap.Any("platform", platform),
				zap.Error(err),
			)
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, commonerrors.ErrSystemError.Error())
		} else {
			// 业务逻辑错误（例如，微信 code 无效、用户状态异常）。
			ctrl.logger.Warn("微信登录/注册服务返回业务错误",
				zap.String("operation", operation),
				zap.String("code", wechatLoginData.Code),
				zap.Any("platform", platform),
				zap.Error(err), // 记录具体的业务错误信息
			)
			// 将服务层返回的、对用户友好的错误信息直接展示给用户。
			response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, err.Error())
		}
		return
	}

	// 4. 登录或注册成功，构造响应数据。
	responseData := vo.LoginResponse{
		User:  userInfo,
		Token: tokenPair,
	}

	// 5. 记录日志并返回成功响应。
	ctrl.logger.Info("微信登录/注册成功",
		zap.String("operation", operation),
		zap.String("userID", userInfo.UserID),
		zap.Any("platform", platform),
	)
	response.RespondSuccess(c, responseData, "登录/注册成功")
}

// RegisterRoutes 注册与微信小程序认证相关的路由到指定的 Gin 路由组。
// 设计目的:
//   - 集中管理此控制器的 API 端点。
//
// 参数:
//   - group: Gin 的路由组实例。如果 group 是 router.Group("/user-hub/api/v1")，
//     那么这里注册的 "/wechat/login" 实际路径将是 "/user-hub/api/v1/wechat/login"。
func (ctrl *WechatAuthController) RegisterRoutes(group *gin.RouterGroup) {
	// 注册微信小程序 code 登录/注册接口
	// - 路径: /wechat/login (相对于 group 的基础路径)
	//   完整路径将是 group的基础路径 + "/wechat/login"
	// - 方法: POST
	// - 此接口通常不需要用户认证即可访问。
	group.POST("/wechat/login", ctrl.LoginOrRegisterHandler)
}
