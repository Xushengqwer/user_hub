package redis

import (
	"context"
	"errors"
	"github.com/go-redis/redis/v8"
	"user_hub/constants"
)

// todo  这里可以考虑单独连接一个 Redis 实例，启用 Redis 的持久化：防止数据丢失导致的安全问题，推荐结合 RDB 和 AOF 使用，
// 原因如下: 如果黑名单丢失，退出登录时加入黑名单的刷新令牌会变得“有效”，客户端又可以用它自动登录，导致用户无法真正退出。
//         单独连接一个 Redis 实例是因为一旦开启持久化的机制，会导致该实例上的所有数据都开启持久化，会牵连到不必要持久化的数据

// TokenBlackRepo 定义令牌黑名单仓库接口
// - 提供将令牌加入黑名单和检查黑名单状态的操作
type TokenBlackRepo interface {
	// AddTokensToBlacklist 将令牌加入黑名单
	// - 输入: ctx 上下文, refreshToken 刷新令牌
	// - 输出: error 操作错误
	AddTokensToBlacklist(ctx context.Context, refreshToken string) error

	// IsBlacklisted 检查令牌是否在黑名单中
	// - 输入: ctx 上下文, token 令牌
	// - 输出: bool 是否在黑名单, error 操作错误
	IsBlacklisted(ctx context.Context, token string) (bool, error)
}

// tokenBlackRepo 实现 TokenBlackRepo 接口的结构体
type tokenBlackRepo struct {
	client *redis.Client // Redis 客户端实例
}

// NewTokenBlacklistRepo 创建 TokenBlackRepo 实例
// - 输入: client Redis 客户端实例
// - 输出: TokenBlackRepo 接口实现
func NewTokenBlacklistRepo(client *redis.Client) TokenBlackRepo {
	return &tokenBlackRepo{client: client}
}

// AddTokensToBlacklist 将令牌加入黑名单
// - 输入: ctx 上下文, refreshToken 刷新令牌
// - 输出: error 操作错误
// - Redis: SET blacklist:<refreshToken> "blacklisted" EX <RefreshTokenBlacklistTTL_seconds>
func (r *tokenBlackRepo) AddTokensToBlacklist(ctx context.Context, refreshToken string) error {
	// 1. 生成键名
	// - 使用 constants.BlacklistKeyPrefix 前缀拼接刷新令牌作为键
	refreshKey := constants.BlacklistKeyPrefix + refreshToken

	// 2. 将刷新令牌加入黑名单
	// - 调用 Redis SET 命令，存储值 "blacklisted" 并设置过期时间
	err := r.client.Set(ctx, refreshKey, "blacklisted", constants.RefreshTokenBlacklistTTL).Err()
	return err
}

// IsBlacklisted 检查令牌是否在黑名单中
// - 输入: ctx 上下文, token 令牌
// - 输出: bool 是否在黑名单, error 操作错误
// - Redis: GET blacklist:<token>
func (r *tokenBlackRepo) IsBlacklisted(ctx context.Context, token string) (bool, error) {
	// 1. 生成键名
	// - 使用 "blacklist:" 前缀拼接令牌作为键
	key := "blacklist:" + token

	// 2. 获取黑名单状态
	// - 调用 Redis GET 命令，读取指定键的值
	val, err := r.client.Get(ctx, key).Result()

	// 3. 处理查询结果
	// - 如果键不存在（redis.Nil），返回 false 表示不在黑名单
	// - 如果发生其他错误，返回错误
	// - 如果值存在，检查是否为 "blacklisted"
	if errors.Is(err, redis.Nil) {
		return false, nil // key 不存在，表示不在黑名单
	} else if err != nil {
		return false, err // Redis 错误
	}
	return val == "blacklisted", nil
}
