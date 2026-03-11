package mocks

import (
	"context"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/redis/go-redis/v9"
)

// NilCacheManager 實作 CacheManager 接口的 nil 對象模式
type NilCacheManager struct{}

// NewNilCacheManager 創建一個不執行任何操作的快取管理器
func NewNilCacheManager() infrastructure.CacheManager {
	return &NilCacheManager{}
}

func (n *NilCacheManager) Connect(ctx context.Context) error { return nil }
func (n *NilCacheManager) Close() error                      { return nil }
func (n *NilCacheManager) Get(ctx context.Context, key string) (string, error) {
	return "", redis.Nil
}
func (n *NilCacheManager) Set(
	ctx context.Context,
	key string,
	value interface{},
	expiration time.Duration,
) (string, error) {
	return "OK", nil
}
func (n *NilCacheManager) Del(ctx context.Context, key string) error { return nil }
func (n *NilCacheManager) SetNX(
	ctx context.Context,
	key string,
	value interface{},
	expiration time.Duration,
) (bool, error) {
	return true, nil
}
func (n *NilCacheManager) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	return nil, nil
}
func (n *NilCacheManager) HealthCheck(ctx context.Context) error { return nil }

func (n *NilCacheManager) BatchSet(
	ctx context.Context,
	entries []entity.CacheSetEntry,
	ttl time.Duration,
) error {
	return nil
}
func (n *NilCacheManager) BatchDelete(ctx context.Context, keys []string) error {
	return nil
}
