package utils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/go-redsync/redsync/v4"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/redis/go-redis/v9"
)

// QueryWithCache 使用泛型的安全快取查詢函數
func QueryWithCache[T any](
	ctx context.Context,
	cache infrastructure.CacheManager,
	cacheKey string,
	ttl time.Duration,
	entityName string,
	dbQuery func(ctx context.Context) (T, error),
) (T, error) {
	var zeroValue T

	// 1. 先嘗試從快取獲取
	if cachedData, err := cache.Get(ctx, cacheKey); err == nil {
		var entity T
		if unmarshalErr := json.Unmarshal([]byte(cachedData), &entity); unmarshalErr == nil {
			return entity, nil
		}
		// 快取數據格式錯誤，繼續查詢資料庫
		fmt.Printf(
			"Cache data format error for %s key %s, continuing with database query\n",
			entityName,
			cacheKey,
		)
	} else if !errors.Is(err, redis.Nil) {
		// 快取服務錯誤（非 key 不存在），記錄但不中斷，繼續查詢資料庫
		fmt.Printf("Cache get error for %s key %s: %v\n", entityName, cacheKey, err)
	}

	// 2. 從資料庫查詢
	entity, err := dbQuery(ctx)
	if err != nil {
		return zeroValue, err
	}

	// 3. 更新快取（異步進行，不影響主流程）
	go func() {
		if entityData, err := json.Marshal(entity); err == nil {
			if _, err := cache.Set(ctx, cacheKey, string(entityData), ttl); err != nil {
				fmt.Printf("Cache set error for %s key %s: %v\n", entityName, cacheKey, err)
			}
		}
	}()

	return entity, nil
}

// ExecuteWithLock 使用分佈式鎖執行函數的共用方法
func ExecuteWithLock(
	ctx context.Context,
	cache infrastructure.CacheManager,
	logger infrastructure.Logger,
	mutexKey string,
	entityID uint64,
	entityType string,
	fn func() error,
) error {
	// 智能重試機制：3次嘗試，每次使用遞增參數
	maxAttempts := 3
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		// 智能遞增參數策略
		expiry := time.Duration(10+attempt*5) * time.Second
		tries := 5 + attempt*2
		baseDelay := time.Duration(50*attempt) * time.Millisecond

		mutex, err := cache.GetMutexWithOption(mutexKey,
			redsync.WithExpiry(expiry),
			redsync.WithTries(tries),
			redsync.WithRetryDelay(baseDelay),
		)
		if err != nil {
			logger.ErrorWithContext(ctx, "Failed to create mutex",
				logger.String("mutex_key", mutexKey),
				logger.String("entity_type", entityType),
				logger.UInt64("entity_id", entityID),
				logger.Int("attempt", attempt),
				logger.Error("err", err),
			)
			if attempt < maxAttempts {
				// 指數退避策略: 500ms → 2s → 4.5s
				backoffTime := time.Duration(attempt*attempt) * 500 * time.Millisecond
				time.Sleep(backoffTime)
				continue
			}
			return fmt.Errorf(
				"failed to create mutex for %s %d after %d attempts: %w",
				entityType,
				entityID,
				maxAttempts,
				err,
			)
		}

		if err = mutex.Lock(); err != nil {
			logger.WarnWithContext(ctx, "Failed to acquire lock",
				logger.String("mutex_key", mutexKey),
				logger.String("entity_type", entityType),
				logger.UInt64("entity_id", entityID),
				logger.Int("attempt", attempt),
				logger.String("expiry", expiry.String()),
				logger.Int("tries", tries),
				logger.Error("err", err),
			)

			if attempt < maxAttempts {
				// 指數退避策略
				backoffTime := time.Duration(attempt*attempt) * 500 * time.Millisecond
				logger.InfoWithContext(ctx, "Retrying after backoff",
					logger.String("entity_type", entityType),
					logger.UInt64("entity_id", entityID),
					logger.String("backoff_time", backoffTime.String()),
					logger.Int("next_attempt", attempt+1),
				)
				time.Sleep(backoffTime)
				continue
			}
			return fmt.Errorf(
				"failed to acquire lock for %s %d after %d attempts: %w",
				entityType,
				entityID,
				maxAttempts,
				err,
			)
		}

		// 成功獲取鎖
		logger.InfoWithContext(ctx, "Successfully acquired lock",
			logger.String("entity_type", entityType),
			logger.UInt64("entity_id", entityID),
			logger.Int("attempt", attempt),
			logger.String("expiry", expiry.String()),
			logger.Int("tries", tries),
		)

		defer func() {
			ok, unlockErr := mutex.Unlock()
			if !ok || unlockErr != nil {
				logger.ErrorWithContext(ctx, "Failed to unlock mutex",
					logger.String("mutex_key", mutexKey),
					logger.String("entity_type", entityType),
					logger.UInt64("entity_id", entityID),
					logger.Error("err", unlockErr),
					logger.Bool("unlock_success", ok),
				)
			}
		}()

		return fn()
	}

	return fmt.Errorf(
		"failed to execute with lock for %s %d after %d attempts",
		entityType,
		entityID,
		maxAttempts,
	)
}
