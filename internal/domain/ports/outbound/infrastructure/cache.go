package infrastructure

import (
	"context"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
)

// CacheManager 通用緩存管理器介面。
// 所有方法只使用標準庫型別或 domain entity 型別，不依賴任何 Redis 具體型別。
type CacheManager interface {
	// 連接管理
	Connect(ctx context.Context) error
	Close() error

	// 基本單筆操作
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) (string, error)
	Del(ctx context.Context, key string) error
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error)
	MGet(ctx context.Context, keys ...string) ([]interface{}, error)

	// 批次操作（取代 Pipeline）
	BatchSet(ctx context.Context, entries []entity.CacheSetEntry, ttl time.Duration) error
	BatchDelete(ctx context.Context, keys []string) error

	// 健康檢查
	HealthCheck(ctx context.Context) error
}
