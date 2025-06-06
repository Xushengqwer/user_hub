package controller

import (
	"errors"
	"net/http"

	"github.com/Xushengqwer/go-common/commonerrors" // 引入公共错误包
	"github.com/Xushengqwer/go-common/core"         // 引入日志包
	"github.com/Xushengqwer/go-common/models/enums"
	"github.com/Xushengqwer/go-common/response"
	"github.com/Xushengqwer/user_hub/config"
	"github.com/Xushengqwer/user_hub/constants"
	"github.com/Xushengqwer/user_hub/models/dto"
	"github.com/Xushengqwer/user_hub/models/vo"
	"github.com/Xushengqwer/user_hub/service/login/auth"
	"github.com/Xushengqwer/user_hub/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap" // 引入 zap 用于日志字段
)

// PhoneAuthController 处理与手机号+验证码认证相关的 HTTP 请求。
// 依赖于 auth.PhoneAuthService 来执行核心业务逻辑。
type PhoneAuthController struct {
	phoneService auth.PhoneAuthService // phoneService: 手机号认证服务的实例。
	logger       *core.ZapLogger       // logger: 日志记录器。
	cookieConfig config.CookieConfig   // 新增：存储 Cookie 配置
}

// NewPhoneAuthController 创建一个新的 PhoneAuthController 实例。
// 设计目的:
//   - 通过依赖注入传入 phoneService 和 logger，增强了代码的可测试性和模块化。
//
// 参数:
//   - phoneService: 实现了 auth.PhoneAuthService 接口的服务实例。
//   - logger: 日志记录器实例。
//   - cookieCfg: Cookie 配置。
//
// 返回:
//   - *PhoneAuthController: 初始化完成的控制器实例。
func NewPhoneAuthController(
	phoneService auth.PhoneAuthService,
	logger *core.ZapLogger, // 注入 logger
	cookieCfg config.CookieConfig, // 新增：接收 Cookie 配置
) *PhoneAuthController {
	return &PhoneAuthController{
		phoneService: phoneService,
		logger:       logger,    // 存储 logger
		cookieConfig: cookieCfg, // 存储 Cookie 配置
	}
}

// LoginOrRegisterHandler 处理用户使用手机号和验证码进行登录或注册的请求。
// @Summary 手机号登录或注册
// @Description 用户通过提供手机号和接收到的短信验证码来登录或自动注册账户。
// @Tags 手机号认证
// @Accept json
// @Produce json
// @Param body body dto.PhoneLoginOrRegisterData true "登录/注册信息 (手机号、验证码)"
// @Param X-Platform header string true "客户端平台类型" Enums(web, wechat, app) default(web)
// @Success 200 {object} docs.SwaggerAPILoginResponse "登录或注册成功，返回用户信息及访问和刷新令牌"
// @Failure 400 {object} docs.SwaggerAPIErrorResponseString "请求参数无效 (如JSON格式错误、平台类型无效) 或 业务逻辑错误 (如验证码错误或过期、用户状态异常)"
// @Failure 500 {object} docs.SwaggerAPIErrorResponseString "系统内部错误 (如数据库操作失败、令牌生成失败、Redis操作失败)"
// @Router /api/v1/user-hub/phone/login [post] // <--- 已更新路径
func (ctrl *PhoneAuthController) LoginOrRegisterHandler(c *gin.Context) {
	const operation = "PhoneAuthController.LoginOrRegisterHandler"

	// 1. 绑定并校验请求体数据。
	var phoneLoginOrRegisterData dto.PhoneLoginOrRegisterData
	if err := c.ShouldBindJSON(&phoneLoginOrRegisterData); err != nil {
		ctrl.logger.Warn("手机号登录/注册请求参数绑定失败",
			zap.String("operation", operation),
			zap.Error(err),
		)
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "输入参数无效")
		return
	}

	// 2. 获取并验证请求头中的 X-Platform 参数。
	platformStr := c.GetHeader("X-Platform")
	platform, err := enums.PlatformFromString(platformStr)
	if err != nil {
		ctrl.logger.Warn("无效的平台类型",
			zap.String("operation", operation),
			zap.String("platformHeader", platformStr),
			zap.String("phone", phoneLoginOrRegisterData.Phone), // 记录关联手机号
			zap.Error(err),
		)
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "无效的平台类型")
		return
	}

	// 3. 调用服务层执行登录或注册逻辑。
	//    服务层会处理验证码校验、用户查找/创建、状态检查和令牌生成。
	userInfo, tokenPair, err := ctrl.phoneService.LoginOrRegister(c.Request.Context(), phoneLoginOrRegisterData, platform)
	if err != nil {
		// 根据服务层返回的错误类型记录日志并响应。
		if errors.Is(err, commonerrors.ErrSystemError) {
			ctrl.logger.Error("手机号登录/注册服务返回系统错误",
				zap.String("operation", operation),
				zap.String("phone", phoneLoginOrRegisterData.Phone), // 注意脱敏
				zap.Any("platform", platform),
				zap.Error(err),
			)
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, commonerrors.ErrSystemError.Error())
		} else {
			// 业务逻辑错误（例如，验证码错误、用户状态异常）。
			ctrl.logger.Warn("手机号登录/注册服务返回业务错误",
				zap.String("operation", operation),
				zap.String("phone", phoneLoginOrRegisterData.Phone), // 注意脱敏
				zap.Any("platform", platform),
				zap.Error(err), // 记录具体的业务错误信息
			)
			// 将服务层返回的、对用户友好的错误信息直接展示给用户。
			response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, err.Error())
		}
		return
	}

	// 4. 根据平台处理令牌响应
	if platform == enums.PlatformWeb {
		rtMaxAge := int(constants.RefreshTokenTTL.Seconds())
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     ctrl.cookieConfig.RefreshTokenName,
			Value:    tokenPair.RefreshToken,
			MaxAge:   rtMaxAge,
			Path:     ctrl.cookieConfig.Path,
			Domain:   ctrl.cookieConfig.Domain,
			Secure:   ctrl.cookieConfig.Secure,
			HttpOnly: ctrl.cookieConfig.HttpOnly,
			SameSite: utils.ParseSameSiteString(ctrl.cookieConfig.SameSite),
		})
		responseData := vo.LoginResponse{
			User:  userInfo,
			Token: vo.TokenPair{AccessToken: tokenPair.AccessToken},
		}
		ctrl.logger.Info("手机号登录/注册成功 (Web平台，RT已设置到Cookie)", zap.String("operation", operation), zap.String("userID", userInfo.UserID), zap.String("phone", phoneLoginOrRegisterData.Phone), zap.Any("platform", platform))
		response.RespondSuccess(c, responseData, "登录/注册成功")
	} else {
		responseData := vo.LoginResponse{
			User:  userInfo,
			Token: tokenPair,
		}
		ctrl.logger.Info("手机号登录/注册成功", zap.String("operation", operation), zap.String("userID", userInfo.UserID), zap.String("phone", phoneLoginOrRegisterData.Phone), zap.Any("platform", platform))
		response.RespondSuccess(c, responseData, "登录/注册成功")
	}
}

// RegisterRoutes 注册与手机号认证相关的路由到指定的 Gin 路由组。
func (ctrl *PhoneAuthController) RegisterRoutes(group *gin.RouterGroup) {
	// 注册手机号登录或注册接口
	// - 路径: /phone/login (相对于 group 的基础路径)
	//   完整路径将是 group的基础路径 + "/phone/login"
	//   例如: "/api/v1/user-hub/phone/login"
	// - 方法: POST
	// - 此接口通常不需要用户认证即可访问。
	group.POST("/phone/login", ctrl.LoginOrRegisterHandler)
}
