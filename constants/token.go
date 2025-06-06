package constants

import (
	"time"
)

const (
	// 认证令牌和刷新令牌的过期时间

	AccessTokenTTL = 15 * time.Minute // 认证令牌（Access Token）的有效期

	RefreshTokenTTL = 10 * 24 * time.Hour // 刷新令牌（Refresh Token）的有效期
)
