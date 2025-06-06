package redis

import (
	"context"
	"errors"
	"fmt" // 引入 fmt 包用于错误包装
	"time"

	// 使用 go-redis/v9
	"github.com/redis/go-redis/v9"
	// 引入你的公共错误包
	"github.com/Xushengqwer/go-common/commonerrors"
)

// CodeRepo 定义了与 Redis 中存储验证码相关的操作接口。
// - 它封装了 Redis 的具体命令，提供标准化的验证码管理方法。
type CodeRepo interface {
	// SetCaptcha 在 Redis 中设置验证码，并指定其有效时间。
	// - 接收应用上下文、手机号（作为键的一部分）、验证码本身以及过期时长。
	// - 如果 Redis 操作失败，则返回包装后的错误。
	SetCaptcha(ctx context.Context, phone string, captcha string, expire time.Duration) error

	// GetCaptcha 从 Redis 中根据手机号检索对应的验证码。
	// - 如果验证码不存在（可能已过期或未设置），将返回 commonerrors.ErrRepoNotFound。
	// - 其他 Redis 查询错误将被包装后返回。
	GetCaptcha(ctx context.Context, phone string) (string, error)

	// DeleteCaptcha 从 Redis 中删除指定手机号的验证码。
	// - 通常在验证码成功使用后调用，防止重复使用。
	// - 如果 Redis 操作失败，则返回包装后的错误。
	DeleteCaptcha(ctx context.Context, phone string) error
}

// codeRepo 是 CodeRepo 接口基于 go-redis/v9 的实现。
type codeRepo struct {
	// 注意：字段类型改为 *redis.Client (v9 版本)
	client *redis.Client // client 是 Redis v9 客户端实例
}

// NewCodeRepo 创建一个新的 codeRepo 实例。
// - 依赖注入 Redis v9 客户端。
func NewCodeRepo(client *redis.Client) CodeRepo {
	return &codeRepo{client: client}
}

// buildKey 根据手机号生成用于 Redis 操作的键名。
// - 使用 "captcha:" 作为统一前缀，方便管理和识别。
func (r *codeRepo) buildKey(phone string) string {
	// 考虑对 phone 进行清洗或验证，防止注入非法字符到 key 中（虽然 Redis key 通常比较灵活）
	// 但基本的前缀拼接是常见的
	return "captcha:" + phone
}

// SetCaptcha 实现接口方法，在 Redis 中存储验证码。
func (r *codeRepo) SetCaptcha(ctx context.Context, phone string, captcha string, expire time.Duration) error {
	key := r.buildKey(phone)
	// 执行 Redis SET 命令，带过期时间 (EX)
	// v9 的 Set 方法签名与 v8 相同
	if err := r.client.Set(ctx, key, captcha, expire).Err(); err != nil {
		// 包装 Redis SET 操作错误，添加中文上下文
		return fmt.Errorf("codeRepo.SetCaptcha: 设置验证码失败 (手机号: %s): %w", phone, err)
	}
	// 操作成功，返回 nil
	return nil
}

// GetCaptcha 实现接口方法，从 Redis 中获取验证码。
func (r *codeRepo) GetCaptcha(ctx context.Context, phone string) (string, error) {
	key := r.buildKey(phone)
	// 执行 Redis GET 命令
	// v9 的 Get 方法签名与 v8 相同
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		// 检查是否是 Redis 的 "key not found" 错误 (redis.Nil 在 v9 中依然可用)
		if errors.Is(err, redis.Nil) {
			// 验证码不存在或已过期，返回约定的公共错误
			return "", commonerrors.ErrRepoNotFound // 使用正确的公共错误包
		}
		// 包装其他 Redis GET 操作错误，添加中文上下文
		return "", fmt.Errorf("codeRepo.GetCaptcha: 获取验证码失败 (手机号: %s): %w", phone, err)
	}
	// 查询成功，返回获取到的验证码和 nil 错误
	return val, nil
}

// DeleteCaptcha 实现接口方法，从 Redis 中删除验证码。
func (r *codeRepo) DeleteCaptcha(ctx context.Context, phone string) error {
	key := r.buildKey(phone)
	// 执行 Redis DEL 命令
	// v9 的 Del 方法签名与 v8 相同
	if err := r.client.Del(ctx, key).Err(); err != nil {
		// 包装 Redis DEL 操作错误，添加中文上下文
		// 注意：即使 key 不存在，DEL 通常也会成功返回 0 或 1（取决于版本和模式），Err() 返回 nil。
		// 主要捕获连接错误等非 Nil 错误。
		// if !errors.Is(err, redis.Nil) { // 通常不需要检查 Nil
		return fmt.Errorf("codeRepo.DeleteCaptcha: 删除验证码失败 (手机号: %s): %w", phone, err)
		// }
	}
	// 操作成功（或 key 本就不存在），返回 nil
	return nil
}
