package token

import (
	"context"
	"user_hub/common/dependencies"
	"user_hub/models/vo"
	"user_hub/repository/mysql"
	"user_hub/repository/redis"
	"user_hub/userError"
)

// TokenService 定义令牌管理服务接口
// - 所有登录方式（账号密码、微信小程序、手机号等）共享此服务，用于统一处理令牌操作
type TokenService interface {
	// Logout 退出登录，将刷新令牌加入黑名单
	// - 输入: ctx 用于上下文控制，refreshToken 是客户端传递的刷新令牌
	// - 输出: error 表示退出是否成功，成功时返回 nil
	// - 注意: 前端需自行清除 access_token，安全性依赖客户端配合
	Logout(ctx context.Context, refreshToken string) error

	// RefreshToken 使用刷新令牌续期认证令牌
	// - 输入: ctx 用于上下文控制，refreshToken 是客户端传递的旧刷新令牌
	// - 输出: TokenPair 返回新的访问令牌和刷新令牌，error 表示可能的错误
	RefreshToken(ctx context.Context, refreshToken string) (vo.TokenPair, error)
}

// tokenService 实现 TokenService 接口的结构体
type tokenService struct {
	tokenBlackRepo redis.TokenBlackRepo             // 用于管理令牌黑名单
	userRepo       mysql.UserRepository             // 用于查询用户信息
	jwtUtil        dependencies.JWTUtilityInterface // 用于生成和解析 JWT 令牌
}

// NewTokenService 创建 TokenService 实例，通过依赖注入初始化
// - 输入: 各仓库和工具的实例
// - 输出: TokenService 接口实例
func NewTokenService(
	tokenBlackRepo redis.TokenBlackRepo,
	userRepo mysql.UserRepository,
	jwtUtil dependencies.JWTUtilityInterface,
) TokenService {
	return &tokenService{
		tokenBlackRepo: tokenBlackRepo,
		userRepo:       userRepo,
		jwtUtil:        jwtUtil,
	}
}

// Logout 退出登录，将刷新令牌加入黑名单
func (t *tokenService) Logout(ctx context.Context, refreshToken string) error {
	// 1. 将 refreshToken 加入黑名单
	// - 调用 TokenBlackRepo 的 AddTokensToBlacklist 方法
	err := t.tokenBlackRepo.AddTokensToBlacklist(ctx, refreshToken)
	if err != nil {
		// - 如果加入黑名单失败，返回自定义错误
		return userError.ErrTokenBlacklistFailed
	}

	// 2. 成功退出，返回 nil
	// - 前端需自行清除 access_token，服务端只负责标记 refreshToken 失效
	return nil
}

// RefreshToken 使用刷新令牌续期认证令牌
func (t *tokenService) RefreshToken(ctx context.Context, refreshToken string) (vo.TokenPair, error) {
	// 1. 解析 refreshToken，获取自定义声明
	// - 使用 JWTUtility 解析令牌，提取 UserID 和 Platform 等信息
	claims, err := t.jwtUtil.ParseRefreshToken(refreshToken)
	if err != nil {
		// - 解析失败，返回无效令牌错误
		return vo.TokenPair{}, userError.ErrInvalidRefreshToken
	}

	// 2. 检查 refreshToken 是否在黑名单中
	// - 调用 TokenBlackRepo 的 IsTokenBlacklisted 方法
	isBlacklisted, err := t.tokenBlackRepo.IsBlacklisted(ctx, refreshToken)
	if err != nil {
		// - 检查黑名单时发生错误，返回服务器内部错误
		return vo.TokenPair{}, userError.ErrServerInternal
	}
	if isBlacklisted {
		// - 如果令牌在黑名单中，返回失效错误
		return vo.TokenPair{}, userError.ErrRefreshTokenExpired
	}

	// 3. 从数据库获取用户最新信息
	// - 根据 UserID 查询 User 表，确保用户状态有效
	user, err := t.userRepo.GetUserByID(ctx, claims.UserID)
	if err != nil {
		// - 查询失败或用户不存在，返回服务器错误
		return vo.TokenPair{}, userError.ErrServerInternal
	}

	// 4. 生成新的访问令牌和刷新令牌
	// - 使用 JWTUtility 生成新令牌，平台保持与旧令牌一致
	newAccessToken, err := t.jwtUtil.GenerateAccessToken(user.UserID, user.UserRole, user.Status, claims.Platform)
	if err != nil {
		// - 生成访问令牌失败，返回服务器错误
		return vo.TokenPair{}, userError.ErrServerInternal
	}
	newRefreshToken, err := t.jwtUtil.GenerateRefreshToken(user.UserID, claims.Platform)
	if err != nil {
		// - 生成刷新令牌失败，返回服务器错误
		return vo.TokenPair{}, userError.ErrServerInternal
	}

	// 5. 返回新的令牌对
	// - 返回新的 TokenPair 给客户端，包含新访问令牌和刷新令牌
	return vo.TokenPair{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	}, nil
}
