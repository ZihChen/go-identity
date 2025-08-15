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
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"github.com/redis/go-redis/v9"
)

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
		return nil // Redis connection 還存在
	}

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled while trying to connect to Redis: %w", ctx.Err())
		default:
			client := redis.NewClient(&redis.Options{
				Addr:     fmt.Sprintf("%s:%d", m.config.Redis.Domain, m.config.Redis.Port),
				Password: m.config.Redis.Password,
				DB:       m.config.Redis.DB,
			})

			// 初始化 redsync 互斥鎖
			pool := goredis.NewPool(client)
			rs := redsync.New(pool)

			// 測試連接
			if err := client.Ping(ctx).Err(); err != nil {
				log.Printf("Failed to connect to Redis: %v, retrying in 3 seconds...", err)
				_ = client.Close()
				time.Sleep(3 * time.Second) // 等待三秒重新連線
				continue
			}

			// 連接成功
			m.client, m.redsync, m.isClosed = client, rs, false
			log.Printf(
				"Successfully connected to Redis at %s:%d",
				m.config.Redis.Domain,
				m.config.Redis.Port,
			)
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
func (m *Manager) Pipeline() redis.Pipeliner {
	client, err := m.GetClient()
	if err != nil {
		return nil
	}
	return client.Pipeline()
}

// MGet 批次獲取多個 key 的值
func (m *Manager) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	client, err := m.GetClient()
	if err != nil {
		return nil, err
	}
	return client.MGet(ctx, keys...).Result()
}
