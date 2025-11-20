package redis

import (
	"context"
	"testing"
	"time"

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
