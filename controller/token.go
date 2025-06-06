package controller

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Xushengqwer/go-common/commonerrors" // 引入公共错误包
	"github.com/Xushengqwer/go-common/core"         // 引入日志包
	"github.com/Xushengqwer/go-common/models/enums"
	"github.com/Xushengqwer/go-common/response"
	"github.com/Xushengqwer/user_hub/config"
	"github.com/Xushengqwer/user_hub/constants"
	"github.com/Xushengqwer/user_hub/dependencies"
	// "user_hub/docs" // 如果您的 linter/IDE 需要，可以导入 docs 包，swag 通常会自动处理
	"github.com/Xushengqwer/user_hub/models/dto"
	"github.com/Xushengqwer/user_hub/models/vo"
	"github.com/Xushengqwer/user_hub/service/token" // 假设 service/token 包下有 AuthTokenService
	"github.com/Xushengqwer/user_hub/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap" // 引入 zap 用于日志字段
)

// AuthTokenController 处理与认证令牌（Access Token, Refresh Token）管理相关的 HTTP 请求。
// 例如：用户退出登录（吊销令牌）、刷新令牌。
type AuthTokenController struct {
	tokenService token.AuthTokenService         // tokenService: 令牌管理服务的实例。
	jwtUtil      dependencies.JWTTokenInterface // jwtUtil: JWT 工具，用于认证中间件。
	logger       *core.ZapLogger                // logger: 日志记录器。
	cookieConfig config.CookieConfig            // 新增：存储 Cookie 配置
}

// NewAuthTokenController 创建一个新的 AuthTokenController 实例。
// 设计目的:
//   - 通过依赖注入传入所需的服务实例、JWT工具和日志记录器。
//
// 参数:
//   - tokenService: 实现了 token.AuthTokenService 接口的服务实例。
//   - jwtUtil: JWT工具实例。
//   - logger: 日志记录器实例。
//   - cookieCfg: Cookie 配置。
//
// 返回:
//   - *AuthTokenController: 初始化完成的控制器实例。
func NewAuthTokenController(
	tokenService token.AuthTokenService,
	jwtUtil dependencies.JWTTokenInterface,
	logger *core.ZapLogger, // 注入 logger
	cookieCfg config.CookieConfig, // 新增：接收 Cookie 配置
) *AuthTokenController {
	return &AuthTokenController{
		tokenService: tokenService,
		jwtUtil:      jwtUtil,
		logger:       logger,    // 存储 logger
		cookieConfig: cookieCfg, // 存储 Cookie 配置
	}
}

// Logout 处理用户退出登录的请求。
// @Summary 退出登录
// @Description 用户请求吊销其当前的认证令牌（通常是 Refresh Token），使其失效。客户端应在调用此接口后清除本地存储的令牌。
// @Tags 认证管理 (Auth Management)
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer <需要吊销的令牌>" example("Bearer eyJhbGciOiJI...")
// @Success 200 {object} docs.SwaggerAPIEmptyResponse "退出登录成功"
// @Failure 400 {object} docs.SwaggerAPIErrorResponseString "请求格式错误 (如缺少 Authorization 头或格式非 Bearer)"
// @Failure 401 {object} docs.SwaggerAPIErrorResponseString "认证失败 (通常由 AuthMiddleware 处理，此接口本身逻辑较少触发)"
// @Failure 500 {object} docs.SwaggerAPIErrorResponseString "系统内部错误 (如 Redis 操作失败)"
// @Router /api/v1/user-hub/auth/logout [post] // <--- 已更新路径
func (ctrl *AuthTokenController) Logout(c *gin.Context) {
	const operation = "AuthTokenController.Logout"

	// 1. 从请求头中获取 Authorization (通常是 Access Token)。这个逻辑可以保留，用于吊销 AT。
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		const bearerPrefix = "Bearer "
		if strings.HasPrefix(authHeader, bearerPrefix) {
			tokenToRevoke := strings.TrimPrefix(authHeader, bearerPrefix)
			if tokenToRevoke != "" {
				// 调用服务层吊销这个令牌 (AT) 的 JTI
				err := ctrl.tokenService.Logout(c.Request.Context(), tokenToRevoke) // Logout 服务应能处理 AT 或 RT
				if err != nil {
					// 记录错误，但通常不应阻止退出流程的其他部分（如清除Cookie）
					ctrl.logger.Error("退出登录时吊销Authorization头中的令牌失败", zap.String("operation", operation), zap.Error(err))
				} else {
					ctrl.logger.Info("已尝试吊销Authorization头中的令牌", zap.String("operation", operation))
				}
			}
		}
	} else {
		ctrl.logger.Info("退出登录请求未提供Authorization头，跳过AT吊销", zap.String("operation", operation))
	}

	// 2. 根据平台清除 Refresh Token Cookie (如果是 Web 平台)
	platformStr := c.GetHeader("X-Platform")
	platform, _ := enums.PlatformFromString(platformStr) // 错误可忽略，如果平台无效，不清除cookie也行，或者默认行为

	if platform == enums.PlatformWeb {
		ctrl.logger.Info("Web平台退出登录，尝试清除RT Cookie", zap.String("operation", operation))
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     ctrl.cookieConfig.RefreshTokenName,
			Value:    "",
			MaxAge:   -1, // Expire immediately
			Path:     ctrl.cookieConfig.Path,
			Domain:   ctrl.cookieConfig.Domain,
			Secure:   ctrl.cookieConfig.Secure,
			HttpOnly: ctrl.cookieConfig.HttpOnly,
			SameSite: utils.ParseSameSiteString(ctrl.cookieConfig.SameSite),
		})
	}
	// 对于非 Web 平台，客户端应自行删除本地存储的 RT。
	// 如果你的 tokenService.Logout 也依赖于从请求中获取RT来吊销（除了AT），
	// 那么非Web平台客户端仍需发送其RT（例如在body中），然后你可以调用tokenService.Logout(rtFromNonWebClient)。
	// 但通常，清除Web Cookie 和 吊销AT 就足够了。

	ctrl.logger.Info("用户退出登录操作完成", zap.String("operation", operation))
	response.RespondSuccess[vo.Empty](c, vo.Empty{}, "退出成功")
}

// RefreshToken 处理使用 Refresh Token 刷新认证令牌的请求。
// @Summary 刷新令牌
// @Description 使用有效的 Refresh Token 获取一对新的 Access Token 和 Refresh Token。支持从请求体或 Cookie 中获取 Refresh Token。
// @Tags 认证管理 (Auth Management)
// @Accept json
// @Produce json
// @Param request body dto.RefreshTokenRequest false "请求体 (可选)，包含 refresh_token 字段"
// @Success 200 {object} docs.SwaggerAPITokenPairResponse "刷新成功，返回新的令牌对"
// @Failure 400 {object} docs.SwaggerAPIErrorResponseString "请求参数错误 (如未提供有效的 Refresh Token)"
// @Failure 401 {object} docs.SwaggerAPIErrorResponseString "认证失败 (Refresh Token 无效、已过期、已被吊销或用户状态异常)"
// @Failure 500 {object} docs.SwaggerAPIErrorResponseString "系统内部错误 (如数据库操作失败、令牌生成失败、Redis 操作失败)"
// @Router /api/v1/user-hub/auth/refresh-token [post] // <--- 已更新路径
func (ctrl *AuthTokenController) RefreshToken(c *gin.Context) {
	const operation = "AuthTokenController.RefreshToken"
	var refreshTokenString string // 改为 refreshTokenString 以明确是字符串

	// 1. 获取平台信息
	platformStr := c.GetHeader("X-Platform")
	platform, err := enums.PlatformFromString(platformStr) // 使用 go-common 的 enums
	if err != nil {
		// 如果平台头缺失或无效，可以根据你的策略决定是拒绝还是默认为某个平台（例如非 Web）
		// 这里假设如果解析失败，则按非 Web 平台处理，尝试从 Body 读取
		ctrl.logger.Warn("刷新令牌请求平台类型无效或缺失，尝试从Body读取RT", zap.String("operation", operation), zap.String("platformHeader", platformStr), zap.Error(err))
		platform = enums.PlatformApp // 或其他非 Web 的默认值，或者直接报错
	}

	// 2. 根据平台获取 Refresh Token
	if platform == enums.PlatformWeb {
		cookieRT, err := c.Cookie(ctrl.cookieConfig.RefreshTokenName)
		if err != nil {
			ctrl.logger.Warn("Web平台刷新令牌请求：Cookie中未找到RT", zap.String("operation", operation), zap.Error(err))
			response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "未提供有效的刷新令牌")
			return
		}
		refreshTokenString = cookieRT
		ctrl.logger.Debug("从Cookie获取到Refresh Token (Web平台)", zap.String("operation", operation))
	} else {
		// 非 Web 平台 (App, WeChat, etc.)，尝试从请求体获取
		var req dto.RefreshTokenRequest
		if err := c.ShouldBindJSON(&req); err == nil && req.RefreshToken != "" {
			refreshTokenString = req.RefreshToken
			ctrl.logger.Debug("从请求体获取到Refresh Token (非Web平台)", zap.String("operation", operation))
		} else {
			ctrl.logger.Warn("非Web平台刷新令牌请求：请求体中未提供有效RT", zap.String("operation", operation), zap.Error(err))
			response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "未提供有效的刷新令牌")
			return
		}
	}

	// 3. 调用服务层执行令牌刷新逻辑。
	newTokenPair, err := ctrl.tokenService.RefreshToken(c.Request.Context(), refreshTokenString)
	if err != nil {
		// ... (错误处理逻辑不变) ...
		if errors.Is(err, commonerrors.ErrSystemError) {
			ctrl.logger.Error("刷新令牌服务返回系统错误", zap.String("operation", operation), zap.Error(err))
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, commonerrors.ErrSystemError.Error())
		} else {
			ctrl.logger.Warn("刷新令牌服务返回业务/认证错误", zap.String("operation", operation), zap.Error(err))
			response.RespondError(c, http.StatusUnauthorized, response.ErrCodeClientUnauthorized, err.Error())
		}
		return
	}

	// 4. 根据平台处理新令牌的响应
	if platform == enums.PlatformWeb {
		rtMaxAge := int(constants.RefreshTokenTTL.Seconds())
		http.SetCookie(c.Writer, &http.Cookie{
			Name:     ctrl.cookieConfig.RefreshTokenName,
			Value:    newTokenPair.RefreshToken,
			MaxAge:   rtMaxAge,
			Path:     ctrl.cookieConfig.Path,
			Domain:   ctrl.cookieConfig.Domain,
			Secure:   ctrl.cookieConfig.Secure,
			HttpOnly: ctrl.cookieConfig.HttpOnly,
			SameSite: utils.ParseSameSiteString(ctrl.cookieConfig.SameSite),
		})
		ctrl.logger.Info("成功刷新令牌 (Web平台，新RT已设置到Cookie)", zap.String("operation", operation))
		response.RespondSuccess(c, vo.TokenPair{AccessToken: newTokenPair.AccessToken}, "刷新成功")
	} else {
		ctrl.logger.Info("成功刷新令牌 (非Web平台)", zap.String("operation", operation))
		response.RespondSuccess(c, newTokenPair, "刷新成功")
	}
}

// RegisterRoutes 注册与令牌管理相关的路由到指定的 Gin 路由组。
// 设计目的:
//   - 集中管理退出登录和刷新令牌的 API 端点。
//   - 注意：认证和基础权限校验预期由上游网关服务处理。此处的注释仅说明接口设计的权限意图。
//
// 参数:
//   - group: Gin 的路由组实例，所有路由将基于此组的路径前缀。
//     例如，如果 group 是 router.Group("/user-hub/api/v1")，
//     那么这里注册的 "/auth" 子分组的完整基础路径将是 "/user-hub/api/v1/auth"。
func (ctrl *AuthTokenController) RegisterRoutes(group *gin.RouterGroup) {
	// 创建 /auth 子路由组，用于令牌管理操作
	authRoutes := group.Group("/auth")
	{
		// 注册退出登录路由
		// - 场景: 用户主动退出登录，需要吊销当前会话的令牌。
		// - 预期权限: 需要用户已认证（由网关处理），允许所有已认证角色（Admin, User）调用。
		authRoutes.POST("/logout", ctrl.Logout)

		// 注册刷新令牌路由
		// - 场景: Access Token 过期后，客户端使用 Refresh Token 获取新的令牌对。
		// - 预期权限: 无需认证（因为 Refresh Token 本身就是一种认证凭证），服务层会校验其有效性。
		authRoutes.POST("/refresh-token", ctrl.RefreshToken)
	}
}
