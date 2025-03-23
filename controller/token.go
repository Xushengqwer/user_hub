package controller

import (
	"github.com/gin-gonic/gin"
	"user_hub/common/code"
	"user_hub/common/dependencies"
	"user_hub/common/response"
	"user_hub/middleware"
	"user_hub/models/dto"
	"user_hub/models/enums"
	"user_hub/models/vo"
	"user_hub/service/token"

	"net/http"
	"strings"
)

// TokenController 处理令牌相关的 HTTP 请求
type TokenController struct {
	tokenService token.TokenService               // 令牌服务实例
	jwtUtil      dependencies.JWTUtilityInterface // JWT依赖
}

// NewTokenController 创建 TokenController 实例
// 参数:
// - tokenService: 令牌服务实例，用于处理退出登录和刷新令牌的逻辑
func NewTokenController(tokenService token.TokenService, jwtUtil dependencies.JWTUtilityInterface) *TokenController {
	return &TokenController{
		tokenService: tokenService,
		jwtUtil:      jwtUtil,
	}
}

// Logout 退出登录
// 该接口将当前用户的访问令牌加入黑名单，使其失效
// @Summary 退出登录
// @Description 将当前用户的访问令牌加入黑名单，阻止后续访问
// @Tags auth
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer 访问令牌"
// @Success 200 {object} response.APIResponse[vo.Empty] "退出成功"
// @Failure 400 {object} response.APIResponse[string] "请求头中缺少 Authorization 或格式错误"
// @Failure 500 {object} response.APIResponse[string] "服务器内部错误"
// @Router /api/v1/auth/logout [post]
func (ctrl *TokenController) Logout(ctx *gin.Context) {
	// 第一步：从请求头中获取 Authorization
	authHeader := ctx.GetHeader("Authorization")
	if authHeader == "" {
		response.RespondError(ctx, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "请求头中缺少 Authorization")
		return
	}

	// 第二步：解析 Authorization，提取令牌
	// 假设 Authorization 格式为 "Bearer <token>"
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	if tokenString == authHeader { // 如果没有 "Bearer " 前缀，说明格式错误
		response.RespondError(ctx, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "Authorization 格式错误")
		return
	}

	// 第三步：将令牌加入黑名单
	// tokenService 会处理令牌的失效逻辑，例如存储到 Redis 并设置过期时间
	if err := ctrl.tokenService.Logout(ctx, tokenString); err != nil {
		response.RespondError(ctx, http.StatusInternalServerError, code.ErrCodeServerInternal, "退出登录失败")
		return
	}

	// 第四步：返回成功响应
	response.RespondSuccess[vo.Empty](ctx, vo.Empty{}, "退出成功")
}

// RefreshToken 刷新令牌
// 该接口使用刷新令牌获取新的访问令牌和刷新令牌，支持从请求体或 Cookie 中获取刷新令牌
// @Summary 刷新令牌
// @Description 使用刷新令牌获取新的访问令牌和刷新令牌，支持 JSON 请求体或 HttpOnly Cookie
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RefreshTokenRequest false "请求体，包含刷新令牌（可选，若使用 Cookie 则可省略）"
// @Success 200 {object} response.APIResponse[vo.TokenPair] "刷新成功，返回新的令牌对"
// @Failure 400 {object} response.APIResponse[string] "请求参数错误，例如未提供刷新令牌"
// @Failure 401 {object} response.APIResponse[string] "刷新令牌无效或已过期"
// @Failure 500 {object} response.APIResponse[string] "服务器内部错误"
// @Router /api/v1/auth/refresh-token [post]
func (ctrl *TokenController) RefreshToken(ctx *gin.Context) {
	// 第一步：尝试从请求体中解析刷新令牌
	var req dto.RefreshTokenRequest
	var refreshToken string
	if err := ctx.ShouldBindJSON(&req); err == nil && req.RefreshToken != "" {
		// 如果请求体中成功解析到刷新令牌，则使用它
		refreshToken = req.RefreshToken
	} else {
		// 第二步：如果请求体中没有刷新令牌，尝试从 Cookie 中获取
		cookieToken, err := ctx.Cookie("refresh_token")
		if err == nil && cookieToken != "" {
			// 如果 Cookie 中存在刷新令牌，则使用它
			refreshToken = cookieToken
		} else {
			// 如果两者都没有提供有效的刷新令牌，返回错误
			response.RespondError(ctx, http.StatusBadRequest, code.ErrCodeClientInvalidInput, "未提供有效的刷新令牌")
			return
		}
	}

	// 第三步：刷新令牌
	// tokenService 会验证刷新令牌并生成新的令牌对，同时将旧刷新令牌加入黑名单
	newTokenPair, err := ctrl.tokenService.RefreshToken(ctx, refreshToken)
	if err != nil {
		// 如果刷新失败（例如令牌无效或已过期），返回 401 错误
		response.RespondError(ctx, http.StatusUnauthorized, code.ErrCodeClientUnauthorized, "刷新令牌失败")
		return
	}

	// 第四步：返回新的令牌对
	// 成功刷新后，返回新的访问令牌和刷新令牌对
	response.RespondSuccess(ctx, newTokenPair, "刷新成功")
}

// RegisterRoutes 注册 TokenController 的路由
// 该方法将所有与令牌管理相关的路由注册到指定的路由组中
// - 输入: group *gin.RouterGroup，路由组实例，用于注册路由
// - 意图: 为退出登录添加认证和权限中间件，为刷新令牌不加中间件，依赖服务层验证
func (ctrl *TokenController) RegisterRoutes(group *gin.RouterGroup) {
	// 第一步：创建 auth 子路由组
	// - 意图: 处理令牌相关的操作（如退出登录、刷新令牌），路径前缀为 /auth，与认证相关路由保持一致
	// - 路径: /api/v1/auth
	auth := group.Group("/auth")
	{
		// 第二步：注册退出登录路由
		// - 意图: 允许认证后的管理员或用户退出登录，需要验证访问令牌并限制为 Admin 或 User 角色
		// - 方法: POST /api/v1/auth/logout
		// - 中间件: AuthMiddleware 验证令牌，PermissionMiddleware 允许 Admin 和 User
		auth.POST("/logout", middleware.AuthMiddleware(ctrl.jwtUtil), middleware.PermissionMiddleware(enums.Admin, enums.User), ctrl.Logout)

		// 第三步：注册刷新令牌路由
		// - 意图: 允许使用刷新令牌获取新令牌对，无需访问令牌认证，依赖服务层验证刷新令牌
		// - 方法: POST /api/v1/auth/refresh-token
		// - 中间件: 无，直接交给 tokenService.RefreshToken 处理验证逻辑
		auth.POST("/refresh-token", ctrl.RefreshToken)
	}
}
