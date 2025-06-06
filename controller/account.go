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

// AccountController 处理与账号密码认证相关的 HTTP 请求。
// 依赖于 auth.AccountService 来执行核心业务逻辑。
type AccountController struct {
	accountService auth.AccountService // accountService: 账号密码认证服务的实例。
	logger         *core.ZapLogger     // logger: 日志记录器。
	cookieConfig   config.CookieConfig // 新增：存储 Cookie 配置
}

// NewAccountController 创建一个新的 AccountController 实例。
// 设计目的:
//   - 通过依赖注入传入 accountService 和 logger，增强了代码的可测试性和模块化。
//
// 参数:
//   - accountService: 实现了 auth.AccountService 接口的服务实例。
//   - logger: 日志记录器实例。
//   - cookieCfg: Cookie 配置。
//
// 返回:
//   - *AccountController: 初始化完成的控制器实例。
func NewAccountController(
	accountService auth.AccountService,
	logger *core.ZapLogger, // 注入 logger
	cookieCfg config.CookieConfig, // 新增：接收 Cookie 配置
) *AccountController {
	return &AccountController{
		accountService: accountService,
		logger:         logger,    // 存储 logger
		cookieConfig:   cookieCfg, // 存储 Cookie 配置
	}
}

// RegisterHandler 处理用户使用账号密码进行注册的请求。
// @Summary 账号密码注册
// @Description 用户通过提供账号、密码和确认密码来创建新账户。
// @Tags 账号密码认证
// @Accept json
// @Produce json
// @Param body body dto.AccountRegisterData true "注册信息 (账号、密码、确认密码)"
// @Success 200 {object} docs.SwaggerAPIUserinfoResponse "注册成功，返回用户信息（通常只有用户ID）"
// @Failure 400 {object} docs.SwaggerAPIErrorResponseString "请求参数无效 (如JSON格式错误、必填项缺失) 或 业务逻辑错误 (如账号已存在、密码不一致)"
// @Failure 500 {object} docs.SwaggerAPIErrorResponseString "系统内部错误 (如数据库操作失败、密码加密失败)"
// @Router /api/v1/user-hub/account/register [post]
func (ctrl *AccountController) RegisterHandler(c *gin.Context) {
	const operation = "AccountController.RegisterHandler" // 操作标识，用于日志

	// 1. 绑定并校验请求体中的 JSON 数据到 DTO 结构体。
	var accountRegisterData dto.AccountRegisterData
	if err := c.ShouldBindJSON(&accountRegisterData); err != nil {
		ctrl.logger.Warn("注册请求参数绑定失败",
			zap.String("operation", operation),
			zap.Error(err), // 记录具体的绑定错误
		)
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "输入参数无效")
		return
	}

	// 2. 调用服务层执行注册逻辑。
	userInfo, err := ctrl.accountService.Register(c.Request.Context(), accountRegisterData)
	if err != nil {
		// 根据服务层返回的错误类型记录日志并响应。
		if errors.Is(err, commonerrors.ErrSystemError) {
			// 服务层已经记录了详细的系统错误日志，控制器只需记录服务调用失败即可。
			ctrl.logger.Error("账号注册服务返回系统错误",
				zap.String("operation", operation),
				zap.String("account", accountRegisterData.Account), // 注意脱敏
				zap.Error(err), // 记录服务层返回的错误（虽然是 ErrSystemError，但可能包含包装信息）
			)
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, commonerrors.ErrSystemError.Error())
		} else {
			// 业务逻辑错误，记录警告级别日志。
			ctrl.logger.Warn("账号注册服务返回业务错误",
				zap.String("operation", operation),
				zap.String("account", accountRegisterData.Account), // 注意脱敏
				zap.Error(err), // 记录具体的业务错误信息
			)
			response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, err.Error())
		}
		return
	}

	// 3. 注册成功，记录日志并返回用户信息。
	ctrl.logger.Info("账号注册成功",
		zap.String("operation", operation),
		zap.String("userID", userInfo.UserID),
		zap.String("account", accountRegisterData.Account), // 注意脱敏
	)
	response.RespondSuccess(c, userInfo, "注册成功")
}

// LoginHandler 处理用户使用账号密码进行登录的请求。
// @Summary 账号密码登录
// @Description 用户通过提供账号和密码来获取认证令牌。
// @Tags 账号密码认证
// @Accept json
// @Produce json
// @Param body body dto.AccountLoginData true "登录信息 (账号、密码)"
// @Param X-Platform header string true "客户端平台类型" Enums(web, wechat, app) default(web)
// @Success 200 {object} docs.SwaggerAPILoginResponse "登录成功，返回用户信息及访问和刷新令牌"
// @Failure 400 {object} docs.SwaggerAPIErrorResponseString "请求参数无效 (如JSON格式错误、平台类型无效) 或 业务逻辑错误 (如账号不存在、密码错误、用户状态异常)"
// @Failure 500 {object} docs.SwaggerAPIErrorResponseString "系统内部错误 (如数据库操作失败、令牌生成失败)"
// @Router /api/v1/user-hub/account/login [post] // <--- 已更新路径
func (ctrl *AccountController) LoginHandler(c *gin.Context) {
	const operation = "AccountController.LoginHandler"

	// 1. 绑定并校验请求体中的 JSON 数据。
	var accountLoginData dto.AccountLoginData
	if err := c.ShouldBindJSON(&accountLoginData); err != nil {
		ctrl.logger.Warn("登录请求参数绑定失败",
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
			zap.Error(err),
		)
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "无效的平台类型")
		return
	}

	// 3. 调用服务层执行登录逻辑。
	userInfo, tokenPair, err := ctrl.accountService.Login(c.Request.Context(), accountLoginData, platform)
	if err != nil {
		// 根据服务层返回的错误类型记录日志并响应。
		if errors.Is(err, commonerrors.ErrSystemError) {
			ctrl.logger.Error("账号登录服务返回系统错误",
				zap.String("operation", operation),
				zap.String("account", accountLoginData.Account), // 注意脱敏
				zap.Any("platform", platform),
				zap.Error(err),
			)
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, commonerrors.ErrSystemError.Error())
		} else {
			ctrl.logger.Warn("账号登录服务返回业务错误",
				zap.String("operation", operation),
				zap.String("account", accountLoginData.Account), // 注意脱敏
				zap.Any("platform", platform),
				zap.Error(err), // 记录具体的业务错误信息
			)
			response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, err.Error())
		}
		return
	}

	// 4. 根据平台处理令牌响应
	if platform == enums.PlatformWeb { // 假设 enums.PlatformWeb 是你定义的 web 平台枚举值
		// Web 平台: RT 在 HttpOnly Cookie, AT 在 JSON
		rtMaxAge := int(constants.RefreshTokenTTL.Seconds())
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     ctrl.cookieConfig.RefreshTokenName, // 使用注入的配置
			Value:    tokenPair.RefreshToken,
			MaxAge:   rtMaxAge,
			Path:     ctrl.cookieConfig.Path,
			Domain:   ctrl.cookieConfig.Domain,
			Secure:   ctrl.cookieConfig.Secure,
			HttpOnly: ctrl.cookieConfig.HttpOnly,
			SameSite: utils.ParseSameSiteString(ctrl.cookieConfig.SameSite), // 使用你的辅助函数
		})

		// 准备只包含 AccessToken 的 JSON 响应
		responseData := vo.LoginResponse{
			User:  userInfo,
			Token: vo.TokenPair{AccessToken: tokenPair.AccessToken}, // RefreshToken 为空
		}
		ctrl.logger.Info("账号登录成功 (Web平台，RT已设置到Cookie)", zap.String("operation", operation), zap.String("userID", userInfo.UserID), zap.Any("platform", platform))
		response.RespondSuccess(c, responseData, "登录成功")
	} else {
		// 其他平台: AT 和 RT 都在 JSON (维持原样)
		responseData := vo.LoginResponse{
			User:  userInfo,
			Token: tokenPair,
		}
		ctrl.logger.Info("账号登录成功", zap.String("operation", operation), zap.String("userID", userInfo.UserID), zap.Any("platform", platform))
		response.RespondSuccess(c, responseData, "登录成功")
	}
}

// RegisterRoutes 注册与账号密码认证相关的路由到指定的 Gin 路由组。
// 设计目的:
//   - 将此控制器的所有路由集中定义和注册，便于管理。
//
// 参数:
//   - group: Gin 的路由组实例，所有路由将基于此组的路径前缀。
//     例如，如果 group 是 router.Group("/user-hub/api/v1")，
//     那么这里注册的 "/account/register" 实际路径将是 "/user-hub/api/v1/account/register"。
func (ctrl *AccountController) RegisterRoutes(group *gin.RouterGroup) {
	// 注册账号密码注册接口
	// - 路径: /api/v1/user-hub/account/register (相对于 group 的基础路径)
	// - 方法: POST
	group.POST("/account/register", ctrl.RegisterHandler)

	// 注册账号密码登录接口
	// - 路径: /api/v1/user-hub/account/login (相对于 group 的基础路径)
	// - 方法: POST
	group.POST("/account/login", ctrl.LoginHandler)
}
