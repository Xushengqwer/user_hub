package controller

import (
	"github.com/gin-gonic/gin"
	"user_hub/common/code"
	"user_hub/common/response"
	"user_hub/models/dto"
	"user_hub/service/token"

	"net/http"
	"strings"
)

// TokenController 处理令牌相关的 HTTP 请求
type TokenController struct {
	tokenService token.TokenService // 令牌服务，用于处理令牌的业务逻辑
}

// NewTokenController 创建 TokenController 实例
// 参数:
// - tokenService: 令牌服务实例，用于处理退出登录和刷新令牌的逻辑
func NewTokenController(tokenService token.TokenService) *TokenController {
	return &TokenController{
		tokenService: tokenService,
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
// @Success 200 {object} response.APIResponse[interface{}] "退出成功"
// @Failure 400 {object} response.APIResponse[string] "请求头中缺少 Authorization 或格式错误"
// @Failure 500 {object} response.APIResponse[string] "服务器内部错误"
// @Router /auth/logout [post]
func (c *TokenController) Logout(ctx *gin.Context) {
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
	if err := c.tokenService.Logout(ctx, tokenString); err != nil {
		response.RespondError(ctx, http.StatusInternalServerError, code.ErrCodeServerInternal, "退出登录失败")
		return
	}

	// 第四步：返回成功响应
	response.RespondSuccess[interface{}](ctx, nil, "退出成功")
}

// RefreshToken 刷新令牌
// 该接口使用刷新令牌获取新的访问令牌和刷新令牌，支持从请求体或 Cookie 中获取刷新令牌
// @Summary 刷新令牌
// @Description 使用刷新令牌获取新的访问令牌和刷新令牌，支持 JSON 请求体或 HttpOnly Cookie
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RefreshTokenRequest false "请求体，包含刷新令牌（可选，若使用 Cookie 则可省略）"
// @Param refresh_token cookie string false "HttpOnly Cookie 中的刷新令牌（可选，若使用请求体则忽略）"
// @Success 200 {object} response.APIResponse[vo.TokenPair] "刷新成功，返回新的令牌对"
// @Failure 400 {object} response.APIResponse[string] "请求参数错误，例如未提供刷新令牌"
// @Failure 401 {object} response.APIResponse[string] "刷新令牌无效或已过期"
// @Failure 500 {object} response.APIResponse[string] "服务器内部错误"
// @Router /auth/refresh-token [post]
func (c *TokenController) RefreshToken(ctx *gin.Context) {
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
	newTokenPair, err := c.tokenService.RefreshToken(ctx, refreshToken)
	if err != nil {
		// 如果刷新失败（例如令牌无效或已过期），返回 401 错误
		response.RespondError[interface{}](ctx, http.StatusUnauthorized, code.ErrCodeClientUnauthorized, "刷新令牌失败")
		return
	}

	// 第四步：返回新的令牌对
	// 成功刷新后，返回新的访问令牌和刷新令牌对
	response.RespondSuccess[interface{}](ctx, newTokenPair, "刷新成功")
}
