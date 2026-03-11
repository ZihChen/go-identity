package redis

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/go-redsync/redsync/v4/redis/goredis/v9"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"github.com/redis/go-redis/v9"
)

// 確保Manager實現CacheManager介面 (編譯時檢查)
var _ infrastructure.CacheManager = (*Manager)(nil)

type Manager struct {
	client   *redis.Client
	redsync  *redsync.Redsync
	config   *config.Config
	mu       sync.RWMutex
	isClosed bool
}

func NewRedisManager(cfg *config.Config) *Manager {
	return &Manager{
		config: cfg,
	}
}

func (m *Manager) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client != nil {
		return nil // Redis connection 已存在
	}

	retryCount := 0
	maxRetries := 5

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled while trying to connect to Redis: %w", ctx.Err())
		default:
			client := redis.NewClient(&redis.Options{
				Addr:            fmt.Sprintf("%s:%d", m.config.Redis.Domain, m.config.Redis.Port),
				Password:        m.config.Redis.Password,
				DB:              m.config.Redis.DB,
				PoolSize:        m.config.Redis.PoolSize,     // 連線池大小
				MinIdleConns:    m.config.Redis.MinIdleConns, // 最小空閒連線數
				MaxRetries:      m.config.Redis.MaxRetries,   // 最大重試次數
				DialTimeout:     m.config.Redis.DialTimeout,  // 連線超時
				ReadTimeout:     m.config.Redis.ReadTimeout,  // 讀取超時
				WriteTimeout:    m.config.Redis.WriteTimeout, // 寫入超時
				PoolTimeout:     m.config.Redis.PoolTimeout,  // 連線池等待超時
				ConnMaxIdleTime: m.config.Redis.IdleTimeout,  // 連線最大空閒時間
				ConnMaxLifetime: m.config.Redis.MaxConnAge,   // 連線最大生命週期
			})

			// 初始化 redsync 互斥鎖
			pool := goredis.NewPool(client)
			rs := redsync.New(pool)

			// 測試連接
			if err := client.Ping(ctx).Err(); err != nil {
				retryCount++
				if retryCount >= maxRetries {
					_ = client.Close()
					return fmt.Errorf(
						"failed to connect to Redis after %d attempts: %w",
						maxRetries,
						err,
					)
				}

				// 指數退避策略，最大30秒
				backoff := time.Duration(1<<uint(retryCount)) * time.Second
				if backoff > 30*time.Second {
					backoff = 30 * time.Second
				}

				log.Printf("Failed to connect to Redis (attempt %d/%d): %v, retrying in %v...",
					retryCount, maxRetries, err, backoff)

				_ = client.Close()
				time.Sleep(backoff)
				continue
			}

			// 連接成功
			m.client, m.redsync, m.isClosed = client, rs, false
			log.Printf("Successfully connected to Redis at %s:%d",
				m.config.Redis.Domain, m.config.Redis.Port)
			return nil
		}
	}
}

func (m *Manager) GetClient() (*redis.Client, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.isClosed {
		return nil, errors.New("redis connection is closed")
	}
	if m.client == nil {
		return nil, errors.New("redis client not initialized")
	}
	return m.client, nil
}

func (m *Manager) GetRedsync() (*redsync.Redsync, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.isClosed {
		return nil, errors.New("redis connection is closed")
	}
	if m.redsync == nil {
		return nil, errors.New("redsync not initialized")
	}
	return m.redsync, nil
}

func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isClosed || m.client == nil {
		return nil
	}

	err := m.client.Close()
	if err != nil {
		return fmt.Errorf("failed to close redis connection: %w", err)
	}

	m.client, m.isClosed = nil, true
	return nil
}

func (m *Manager) SetNX(
	ctx context.Context,
	key string,
	value interface{},
	expiration time.Duration,
) (bool, error) {
	client, err := m.GetClient()
	if err != nil {
		return false, err
	}
	success, err := client.SetNX(ctx, key, value, expiration).Result()
	if err != nil {
		return false, err
	}
	return success, nil
}

func (m *Manager) Set(
	ctx context.Context,
	key string,
	value interface{},
	expiration time.Duration,
) (string, error) {
	client, err := m.GetClient()
	if err != nil {
		return "", err
	}
	status, err := client.Set(ctx, key, value, expiration).Result()
	if err != nil {
		return status, err
	}
	return status, nil
}

func (m *Manager) GetMutex(key string, expireTime time.Duration) (*redsync.Mutex, error) {
	r, err := m.GetRedsync()
	if err != nil {
		return nil, err
	}
	return r.NewMutex(key, redsync.WithExpiry(expireTime)), nil
}

func (m *Manager) GetMutexWithOption(
	key string,
	options ...redsync.Option,
) (*redsync.Mutex, error) {
	r, err := m.GetRedsync()
	if err != nil {
		return nil, err
	}
	return r.NewMutex(key, options...), nil
}

// Pipeline 返回 Redis Pipeline 用於批次操作
func (m *Manager) Pipeline() (redis.Pipeliner, error) {
	client, err := m.GetClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get redis client for pipeline: %w", err)
	}
	return client.Pipeline(), nil
}

// Get 獲取單個 key 的值
func (m *Manager) Get(ctx context.Context, key string) (string, error) {
	client, err := m.GetClient()
	if err != nil {
		return "", err
	}
	return client.Get(ctx, key).Result()
}

// MGet 批次獲取多個 key 的值
func (m *Manager) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	client, err := m.GetClient()
	if err != nil {
		return nil, err
	}
	return client.MGet(ctx, keys...).Result()
}

// HealthCheck 檢查Redis連接健康狀況
func (m *Manager) HealthCheck(ctx context.Context) error {
	client, err := m.GetClient()
	if err != nil {
		return err
	}
	return client.Ping(ctx).Err()
}

// Del 刪除 key 的值
func (m *Manager) Del(ctx context.Context, key string) error {
	client, err := m.GetClient()
	if err != nil {
		return err
	}
	return client.Del(ctx, key).Err()
}

// BatchSet 使用 Pipeline 批次寫入多個 key-value 對。
func (m *Manager) BatchSet(ctx context.Context, entries []entity.CacheSetEntry, ttl time.Duration) error {
	client, err := m.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get redis client for batch set: %w", err)
	}
	pipe := client.Pipeline()
	for _, e := range entries {
		pipe.Set(ctx, e.Key, e.Value, ttl)
	}
	_, err = pipe.Exec(ctx)
	return err
}

// BatchDelete 使用 Pipeline 批次刪除多個 key。
func (m *Manager) BatchDelete(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	client, err := m.GetClient()
	if err != nil {
		return fmt.Errorf("failed to get redis client for batch delete: %w", err)
	}
	pipe := client.Pipeline()
	for _, key := range keys {
		pipe.Del(ctx, key)
	}
	_, err = pipe.Exec(ctx)
	return err
}
