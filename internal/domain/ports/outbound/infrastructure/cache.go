package infrastructure

import (
	"context"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/redis/go-redis/v9"
)

// CacheManager 通用緩存管理器介面 (基於現有功能設計)
type CacheManager interface {
	// 連接管理
	Connect(ctx context.Context) error
	Close() error

	// 基本操作
	Get(ctx context.Context, key string) (string, error)
	Set(
		ctx context.Context,
		key string,
		value interface{},
		expiration time.Duration,
	) (string, error)
	SetNX(
		ctx context.Context,
		key string,
		value interface{},
		expiration time.Duration,
	) (bool, error)
	MGet(ctx context.Context, keys ...string) ([]interface{}, error)

	// Pipeline操作 (安全版本)
	Pipeline() (redis.Pipeliner, error)

	// 低層級客戶端存取
	GetClient() (*redis.Client, error)

	// 健康檢查
	HealthCheck(ctx context.Context) error

	// 分布式鎖支援
	GetMutex(key string, expireTime time.Duration) (*redsync.Mutex, error)
	GetMutexWithOption(key string, options ...redsync.Option) (*redsync.Mutex, error)
	GetRedsync() (*redsync.Redsync, error)
}
