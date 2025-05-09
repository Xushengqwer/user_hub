package controller

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Xushengqwer/go-common/commonerrors" // 引入公共错误包
	"github.com/Xushengqwer/go-common/core"         // 引入日志包
	"github.com/Xushengqwer/go-common/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap" // 引入 zap 用于日志字段

	"user_hub/dependencies"
	// "user_hub/docs" // 如果您的 linter/IDE 需要，可以导入 docs 包，swag 通常会自动处理
	"user_hub/models/dto"
	"user_hub/models/vo"
	"user_hub/service/token" // 假设 service/token 包下有 AuthTokenService
)

// AuthTokenController 处理与认证令牌（Access Token, Refresh Token）管理相关的 HTTP 请求。
// 例如：用户退出登录（吊销令牌）、刷新令牌。
type AuthTokenController struct {
	tokenService token.AuthTokenService         // tokenService: 令牌管理服务的实例。
	jwtUtil      dependencies.JWTTokenInterface // jwtUtil: JWT 工具，用于认证中间件。
	logger       *core.ZapLogger                // logger: 日志记录器。
}

// NewAuthTokenController 创建一个新的 AuthTokenController 实例。
// 设计目的:
//   - 通过依赖注入传入所需的服务实例、JWT工具和日志记录器。
//
// 参数:
//   - tokenService: 实现了 token.AuthTokenService 接口的服务实例。
//   - jwtToken: JWT工具实例。
//   - logger: 日志记录器实例。
//
// 返回:
//   - *AuthTokenController: 初始化完成的控制器实例。
func NewAuthTokenController(
	tokenService token.AuthTokenService,
	jwtUtil dependencies.JWTTokenInterface,
	logger *core.ZapLogger, // 注入 logger
) *AuthTokenController {
	return &AuthTokenController{
		tokenService: tokenService,
		jwtUtil:      jwtUtil,
		logger:       logger, // 存储 logger
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
// @Router /api/v1/auth/logout [post]
func (ctrl *AuthTokenController) Logout(c *gin.Context) {
	const operation = "AuthTokenController.Logout"

	// 1. 从请求头中获取 Authorization。
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		ctrl.logger.Warn("退出登录请求缺少 Authorization 头", zap.String("operation", operation))
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "请求头中缺少 Authorization")
		return
	}

	// 2. 解析 Authorization，提取令牌字符串。
	//    期望格式为 "Bearer <token>"。
	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		ctrl.logger.Warn("退出登录请求 Authorization 头格式错误",
			zap.String("operation", operation),
			zap.String("authHeader", authHeader), // 注意：生产环境可能需要隐藏部分令牌内容
		)
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "Authorization 格式错误，应为 Bearer token")
		return
	}
	tokenToRevoke := strings.TrimPrefix(authHeader, bearerPrefix)
	if tokenToRevoke == "" {
		ctrl.logger.Warn("退出登录请求 Bearer token 为空", zap.String("operation", operation))
		response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "提供的令牌为空")
		return
	}

	// 3. 调用服务层执行退出登录（吊销令牌）逻辑。
	//    服务层会解析令牌，获取 JTI，并将其加入黑名单。
	err := ctrl.tokenService.Logout(c.Request.Context(), tokenToRevoke)
	if err != nil {
		// 根据服务层返回的错误类型进行响应。
		// 注意：根据我们之前的约定，服务层的 Logout 即使内部 Redis 失败也倾向于返回 nil。
		// 但以防万一服务层逻辑改变，或者返回了其他错误，我们还是处理一下。
		if errors.Is(err, commonerrors.ErrSystemError) {
			// 服务层明确指示发生了系统错误。
			ctrl.logger.Error("退出登录服务返回系统错误",
				zap.String("operation", operation),
				// zap.String("tokenPrefix", tokenToRevoke[:min(10, len(tokenToRevoke))]), // 只记录部分前缀
				zap.Error(err),
			)
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, commonerrors.ErrSystemError.Error())
		} else {
			// 其他未预期的错误。
			ctrl.logger.Error("退出登录服务返回未知错误",
				zap.String("operation", operation),
				zap.Error(err),
			)
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, "退出登录时发生未知错误")
		}
		return
	}

	// 4. 返回成功响应。
	ctrl.logger.Info("用户退出登录成功",
		zap.String("operation", operation),
		// 可以考虑记录 UserID (如果 AuthMiddleware 注入了的话)
		// zap.String("userID", c.GetString(constants.UserContextKey)),
	)
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
// @Router /api/v1/auth/refresh-token [post]
func (ctrl *AuthTokenController) RefreshToken(c *gin.Context) {
	const operation = "AuthTokenController.RefreshToken"

	// 1. 尝试从请求体或 Cookie 中获取 Refresh Token。
	var refreshToken string
	var req dto.RefreshTokenRequest

	// 尝试从 JSON Body 获取
	if err := c.ShouldBindJSON(&req); err == nil && req.RefreshToken != "" {
		refreshToken = req.RefreshToken
		ctrl.logger.Debug("从请求体获取到 Refresh Token", zap.String("operation", operation))
	} else {
		// 如果 Body 中没有或解析失败，尝试从 Cookie 获取
		cookieToken, cookieErr := c.Cookie("refresh_token") // 假设 Cookie 名为 refresh_token
		if cookieErr == nil && cookieToken != "" {
			refreshToken = cookieToken
			ctrl.logger.Debug("从 Cookie 获取到 Refresh Token", zap.String("operation", operation))
		} else {
			// 如果 Body 和 Cookie 都没有提供有效的 Refresh Token
			ctrl.logger.Warn("刷新令牌请求未提供有效的 Refresh Token (Body 或 Cookie)", zap.String("operation", operation))
			response.RespondError(c, http.StatusBadRequest, response.ErrCodeClientInvalidInput, "未提供有效的刷新令牌")
			return
		}
	}

	// 2. 调用服务层执行令牌刷新逻辑。
	//    服务层会验证 Refresh Token 的有效性（签名、过期、黑名单）、获取用户信息、生成新令牌对、并将旧 Token 加入黑名单。
	newTokenPair, err := ctrl.tokenService.RefreshToken(c.Request.Context(), refreshToken)
	if err != nil {
		// 根据服务层返回的错误类型进行响应。
		if errors.Is(err, commonerrors.ErrSystemError) {
			// 服务层已记录详细错误。
			ctrl.logger.Error("刷新令牌服务返回系统错误",
				zap.String("operation", operation),
				// zap.String("tokenPrefix", refreshToken[:min(10, len(refreshToken))]), // 记录部分前缀
				zap.Error(err),
			)
			response.RespondError(c, http.StatusInternalServerError, response.ErrCodeServerInternal, commonerrors.ErrSystemError.Error())
		} else {
			// 其他错误视为业务逻辑/认证失败错误（例如，令牌无效、过期、被吊销、用户状态异常）。
			ctrl.logger.Warn("刷新令牌服务返回业务/认证错误",
				zap.String("operation", operation),
				// zap.String("tokenPrefix", refreshToken[:min(10, len(refreshToken))]),
				zap.Error(err), // 记录具体的业务错误信息
			)
			// 对于刷新令牌失败，返回 401 Unauthorized 更符合语义。
			response.RespondError(c, http.StatusUnauthorized, response.ErrCodeClientUnauthorized, err.Error())
		}
		return
	}

	// 3. 刷新成功，返回新的令牌对。
	//    注意：如果需要通过 Set-Cookie 返回新的 Refresh Token，可以在这里设置。
	//    c.SetCookie("refresh_token", newTokenPair.RefreshToken, ...)
	ctrl.logger.Info("成功刷新令牌",
		zap.String("operation", operation),
		// 可以考虑记录 UserID (如果服务层返回了或能从新令牌解析)
	)
	response.RespondSuccess(c, newTokenPair, "刷新成功")
}

// RegisterRoutes 注册与令牌管理相关的路由到指定的 Gin 路由组。
// 设计目的:
//   - 集中管理退出登录和刷新令牌的 API 端点。
//   - 注意：认证和基础权限校验预期由上游网关服务处理。此处的注释仅说明接口设计的权限意图。
//
// 参数:
//   - group: Gin 的路由组实例，所有路由将基于此组的路径前缀。
func (ctrl *AuthTokenController) RegisterRoutes(group *gin.RouterGroup) {
	// 创建 /auth 子路由组，用于令牌管理操作
	authRoutes := group.Group("/auth")
	{
		// 注册退出登录路由
		// - 场景: 用户主动退出登录，需要吊销当前会话的令牌。
		// - 预期权限: 需要用户已认证（由网关处理），允许所有已认证角色（Admin, User）调用。
		authRoutes.POST("/logout", ctrl.Logout) // 移除中间件

		// 注册刷新令牌路由
		// - 场景: Access Token 过期后，客户端使用 Refresh Token 获取新的令牌对。
		// - 预期权限: 无需认证（因为 Refresh Token 本身就是一种认证凭证），服务层会校验其有效性。
		authRoutes.POST("/refresh-token", ctrl.RefreshToken) // 无需中间件
	}
}
