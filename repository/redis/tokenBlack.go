package redis

import (
	"context"
	"fmt" // 引入 fmt 包用于错误包装
	"time"

	// 使用 go-redis/v9
	"github.com/redis/go-redis/v9"

	"github.com/Xushengqwer/user_hub/constants" // 引入常量包获取前缀
)

// TODO: 考虑为令牌黑名单使用单独的、启用持久化（RDB+AOF）的 Redis 实例。
// 原因: 防止 Redis 实例重启导致黑名单数据丢失。如果黑名单丢失，
// 已加入黑名单的令牌（尤其是刷新令牌）可能会再次变得有效，导致安全风险（用户无法真正退出）。
// 使用单独实例可避免影响其他不需要持久化的缓存数据。

// TokenBlackRepo 定义了基于 JTI (JWT ID) 的令牌黑名单仓库接口。
// - 提供将 JTI 加入黑名单和检查其状态的操作。
type TokenBlackRepo interface {
	// AddJtiToBlacklist 将指定的 JTI 加入 Redis 黑名单，并设置其过期时间。
	// - jti: 要加入黑名单的 JWT ID。
	// - ttl: 该 JTI 在黑名单中的存活时间，应等于其对应令牌的剩余有效时间。
	// - 如果 Redis 操作失败，则返回包装后的错误。
	AddJtiToBlacklist(ctx context.Context, jti string, ttl time.Duration) error

	// IsJtiBlacklisted 检查指定的 JTI 是否存在于 Redis 黑名单中。
	// - jti: 要检查的 JWT ID。
	// - 返回: bool 值表示是否存在于黑名单，以及可能的查询错误。
	// - 注意：此方法不返回 commonerrors.ErrRepoNotFound，因为 JTI 不存在于黑名单是预期情况，返回 false, nil。
	IsJtiBlacklisted(ctx context.Context, jti string) (bool, error)
}

// tokenBlackRepo 是 TokenBlackRepo 接口基于 go-redis/v9 的实现。
type tokenBlackRepo struct {
	client *redis.Client // client 是 Redis v9 客户端实例
}

// NewTokenBlacklistRepo 创建一个新的 tokenBlackRepo 实例。
// - 依赖注入 Redis v9 客户端。
func NewTokenBlacklistRepo(client *redis.Client) TokenBlackRepo {
	return &tokenBlackRepo{client: client}
}

// buildBlacklistKey 根据 JTI 生成用于 Redis 黑名单操作的键名。
// - 使用常量中定义的前缀和 "jti:" 来标识。
func (r *tokenBlackRepo) buildBlacklistKey(jti string) string {
	// 示例键: "blacklist:jti:xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
	return constants.BlacklistKeyPrefix + ":jti:" + jti
}

// AddJtiToBlacklist 实现接口方法，将 JTI 加入黑名单。
func (r *tokenBlackRepo) AddJtiToBlacklist(ctx context.Context, jti string, ttl time.Duration) error {
	// 确保 TTL 是有效的正数，防止永久加入黑名单或立即过期
	if ttl <= 0 {
		// 可以选择记录警告或返回错误，这里选择记录并跳过（不加入黑名单）
		fmt.Printf("警告: AddJtiToBlacklist 收到无效的 TTL (%v)，JTI '%s' 未加入黑名单\n", ttl, jti)
		// 或者返回错误: return fmt.Errorf("无效的 TTL: %v", ttl)
		return nil // 或者根据策略返回错误
	}

	key := r.buildBlacklistKey(jti)
	// 使用 SET 命令将 JTI 加入黑名单，值为 "blacklisted" (或任何非空值)，并设置过期时间
	// Set 命令的 NX (Not Exists) 或 XX (Exists) 选项在这里通常不需要
	if err := r.client.Set(ctx, key, "blacklisted", ttl).Err(); err != nil {
		// 包装 Redis SET 操作错误，添加中文上下文
		return fmt.Errorf("tokenBlackRepo.AddJtiToBlacklist: 将 JTI 加入黑名单失败 (JTI: %s): %w", jti, err)
	}
	// 操作成功，返回 nil
	return nil
}

// IsJtiBlacklisted 实现接口方法，检查 JTI 是否在黑名单中。
func (r *tokenBlackRepo) IsJtiBlacklisted(ctx context.Context, jti string) (bool, error) {
	key := r.buildBlacklistKey(jti)
	// 使用 EXISTS 命令检查 key 是否存在即可，比 GET 更高效
	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		// 包装 Redis EXISTS 操作错误，添加中文上下文
		return false, fmt.Errorf("tokenBlackRepo.IsJtiBlacklisted: 检查 JTI 黑名单失败 (JTI: %s): %w", jti, err)
	}
	// Exists 返回 1 表示存在（在黑名单中），返回 0 表示不存在
	return exists == 1, nil

	/*
	   // 或者使用 GET 的方式（如果需要获取值，但这里不需要）:
	   _, err := r.client.Get(ctx, key).Result()
	   if err != nil {
	       if errors.Is(err, redis.Nil) {
	           // Key 不存在，表示 JTI 不在黑名单中
	           return false, nil
	       }
	       // 包装其他 Redis GET 操作错误
	       return false, fmt.Errorf("tokenBlackRepo.IsJtiBlacklisted: 检查 JTI 黑名单失败 (JTI: %s): %w", jti, err)
	   }
	   // Key 存在，表示 JTI 在黑名单中
	   return true, nil
	*/
}
