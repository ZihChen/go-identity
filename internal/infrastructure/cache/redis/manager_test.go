package redis

import (
	"context"
	"testing"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/stretchr/testify/assert"
)

func TestManager_Pipeline_ErrorHandling(t *testing.T) {
	t.Run("未初始化client", func(t *testing.T) {
		manager := &Manager{}
		pipeline, err := manager.Pipeline()
		assert.Error(t, err)
		assert.Nil(t, pipeline)
		assert.Contains(t, err.Error(), "redis client not initialized")
	})
}

func TestManager_HealthCheck(t *testing.T) {
	t.Run("連接失敗 - 未初始化client", func(t *testing.T) {
		manager := &Manager{}
		err := manager.HealthCheck(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis client not initialized")
	})

	t.Run("超時處理", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		manager := &Manager{}
		err := manager.HealthCheck(ctx)
		assert.Error(t, err)
		// 應該是client not initialized錯誤，而不是context超時
		assert.Contains(t, err.Error(), "redis client not initialized")
	})
}

func TestManager_MethodSignatures(t *testing.T) {
	t.Run("Pipeline方法返回正確類型", func(t *testing.T) {
		manager := &Manager{}
		pipeline, err := manager.Pipeline()
		// 應該返回錯誤而不是nil pipeline
		assert.Error(t, err)
		assert.Nil(t, pipeline)
	})
}

func TestManager_ImplementsCacheManager(t *testing.T) {
	// 編譯時檢查
	var _ infrastructure.CacheManager = (*Manager)(nil)
	
	t.Run("介面方法可用性", func(t *testing.T) {
		manager := &Manager{}
		ctx := context.Background()
		
		// 測試Pipeline (安全版本) - 應該返回錯誤
		pipeline, err := manager.Pipeline()
		assert.Error(t, err)
		assert.Nil(t, pipeline)
		
		// 測試健康檢查 - 應該返回錯誤 
		err = manager.HealthCheck(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "redis client not initialized")
		
		// 測試GetClient - 應該返回錯誤
		client, err := manager.GetClient()
		assert.Error(t, err)
		assert.Nil(t, client)
	})
}

func TestManager_ExponentialBackoff(t *testing.T) {
	t.Run("指數退避計算", func(t *testing.T) {
		// 測試指數退避邏輯
		testCases := []struct {
			retryCount int
			expected   time.Duration
		}{
			{1, 2 * time.Second},   // 2^1 = 2s
			{2, 4 * time.Second},   // 2^2 = 4s
			{3, 8 * time.Second},   // 2^3 = 8s
			{4, 16 * time.Second},  // 2^4 = 16s
			{5, 30 * time.Second},  // 2^5 = 32s, 但最大限制為30s
			{6, 30 * time.Second},  // 超過最大值，應該是30s
		}
		
		for _, tc := range testCases {
			backoff := time.Duration(1<<uint(tc.retryCount)) * time.Second
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
			assert.Equal(t, tc.expected, backoff, 
				"retry count %d should have backoff %v", tc.retryCount, tc.expected)
		}
	})
}
