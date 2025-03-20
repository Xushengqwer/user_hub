package dependencies

import (
	"fmt"
	"user_hub/common/config"
	"user_hub/common/core"

	"time"

	"context"
	"github.com/go-redis/redis/v8"
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

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.Ping(ctx).Result()
	if err != nil {
		logger.Error("无法连接到 Redis", zap.Error(err))
		return nil, fmt.Errorf("无法连接到 Redis: %w", err)
	}

	logger.Info("成功连接到 Redis")
	return client, nil
}
