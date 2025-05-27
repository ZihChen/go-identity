package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
)

type Redis struct {
	Client *redis.Client
}

// NewRedis 創建新的Redis
func NewRedis(cfg *config.Config) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Domain, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// 測試連接
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return client, nil
}

// Close 關閉Redis
func (r *Redis) Close() error {
	if err := r.Client.Close(); err != nil {
		return fmt.Errorf("failed to close redis connection: %w", err)
	}

	return nil
}
