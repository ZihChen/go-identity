package redis

import (
	"context"
	"errors"
	"fmt"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"github.com/redis/go-redis/v9"
	"log"
	"sync"
	"time"
)

type Manager struct {
	client   *redis.Client
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

			// 測試連接
			if err := client.Ping(ctx).Err(); err != nil {
				log.Printf("Failed to connect to Redis: %v, retrying in 3 seconds...", err)
				_ = client.Close()
				time.Sleep(3 * time.Second) // 等待三秒重新連線
				continue
			}

			// 連接成功
			m.client, m.isClosed = client, false
			log.Printf("Successfully connected to Redis at %s:%d", m.config.Redis.Domain, m.config.Redis.Port)
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
