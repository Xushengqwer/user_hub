package constants

import (
	"time"
)

const (
	// 认证令牌和刷新令牌的过期时间

	AccessTokenTTL = 15 * time.Minute // 认证令牌（Access Token）的有效期

	RefreshTokenTTL = 10 * 24 * time.Hour // 刷新令牌（Refresh Token）的有效期

	// 认证令牌和刷新令牌的黑名单过期时间

	AccessTokenBlacklistTTL = 15 * time.Minute // 认证令牌黑名单的有效期

	RefreshTokenBlacklistTTL = 10 * 24 * time.Hour // 刷新令牌黑名单的有效期

	// redis 键的前缀

	BlacklistKeyPrefix = "blacklist"

	// 认证中间件存储的上下文的键值对

	UserContextKey   = "userID"
	RoleContextKey   = "role"
	StatusContextKey = "status"

	// RequestIDKey 作为上下文存储和后续获取时使用的键名
	RequestIDKey = "RequestID"
)
