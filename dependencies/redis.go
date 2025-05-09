package dependencies

import (
	"fmt"
	"github.com/Xushengqwer/go-common/core"
	"github.com/Xushengqwer/user_hub/config"
	"github.com/redis/go-redis/v9"

	"time"

	"context"

	"go.uber.org/zap"
)

// InitRedis 初始化 Redis 连接
func InitRedis(cfg *config.RedisConfig, logger *core.ZapLogger) (*redis.Client, error) {
	addr := fmt.Sprintf("%s:%d", cfg.Address, cfg.Port)
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  cfg.DialTimeout,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		PoolSize:     10,
		MinIdleConns: 3,
	})

	// 重试逻辑
	maxRetries := 5
	retryInterval := 2 * time.Second
	var err error

	for i := 0; i < maxRetries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err = client.Ping(ctx).Result()
		cancel()
		if err == nil {
			break // 连接成功，跳出重试循环
		}
		logger.Warn("无法连接到 Redis，尝试重试", zap.Int("retry", i+1), zap.Int("maxRetries", maxRetries), zap.Error(err))
		if i < maxRetries-1 { // 最后一次失败时不再等待
			time.Sleep(retryInterval)
		}
	}

	if err != nil {
		logger.Error("无法连接到 Redis", zap.Error(err))
		return nil, fmt.Errorf("无法连接到 Redis: %w", err)
	}

	logger.Info("成功连接到 Redis")
	return client, nil
}
