package redis

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

// CodeRepo 定义 Redis 验证码数据仓库接口
// - 提供验证码的存储、获取和删除操作
type CodeRepo interface {
	// SetCaptcha 设置验证码并指定过期时间
	// - 输入: ctx 上下文, phone 手机号, captcha 验证码, expire 过期时间
	// - 输出: error 操作错误
	SetCaptcha(ctx context.Context, phone string, captcha string, expire time.Duration) error

	// GetCaptcha 根据手机号获取验证码
	// - 输入: ctx 上下文, phone 手机号
	// - 输出: string 验证码, error 操作错误
	GetCaptcha(ctx context.Context, phone string) (string, error)

	// DeleteCaptcha 删除指定手机号的验证码
	// - 输入: ctx 上下文, phone 手机号
	// - 输出: error 操作错误
	DeleteCaptcha(ctx context.Context, phone string) error
	// 可扩展其他操作，例如缓存管理
}

// codeRepo 实现 CodeRepo 接口的结构体
type codeRepo struct {
	client *redis.Client // Redis 客户端实例
}

// NewCodeRepo 创建 CodeRepo 实例
// - 输入: client Redis 客户端实例
// - 输出: CodeRepo 接口实现
func NewCodeRepo(client *redis.Client) CodeRepo {
	return &codeRepo{client: client}
}

// SetCaptcha 设置验证码并指定过期时间
// - 输入: ctx 上下文, phone 手机号, captcha 验证码, expire 过期时间
// - 输出: error 操作错误
// - Redis: SET captcha:<phone> <captcha> EX <expire_seconds>
func (r *codeRepo) SetCaptcha(ctx context.Context, phone string, captcha string, expire time.Duration) error {
	// 1. 生成键名
	// - 使用 "captcha:" 前缀拼接手机号作为键
	key := "captcha:" + phone

	// 2. 设置验证码
	// - 调用 Redis SET 命令，存储验证码并设置过期时间
	return r.client.Set(ctx, key, captcha, expire).Err()
}

// GetCaptcha 根据手机号获取验证码
// - 输入: ctx 上下文, phone 手机号
// - 输出: string 验证码, error 操作错误
// - Redis: GET captcha:<phone>
func (r *codeRepo) GetCaptcha(ctx context.Context, phone string) (string, error) {
	// 1. 生成键名
	// - 使用 "captcha:" 前缀拼接手机号作为键
	key := "captcha:" + phone

	// 2. 获取验证码
	// - 调用 Redis GET 命令，读取指定键的值
	return r.client.Get(ctx, key).Result()
}

// DeleteCaptcha 删除指定手机号的验证码
// - 输入: ctx 上下文, phone 手机号
// - 输出: error 操作错误
// - Redis: DEL captcha:<phone>
func (r *codeRepo) DeleteCaptcha(ctx context.Context, phone string) error {
	// 1. 生成键名
	// - 使用 "captcha:" 前缀拼接手机号作为键
	key := "captcha:" + phone

	// 2. 删除验证码
	// - 调用 Redis DEL 命令，删除指定键
	return r.client.Del(ctx, key).Err()
}
