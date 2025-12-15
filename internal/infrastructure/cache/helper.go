package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

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
		fmt.Printf("Cache data format error for %s key %s, continuing with database query\n", entityName, cacheKey)
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
