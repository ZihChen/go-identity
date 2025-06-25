// 文件: tests/redis_test.go
package tests

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 使用miniredis模拟Redis服务器来进行测试
func setupMiniRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client, func()) {
	// 启动模拟的Redis服务器
	mr, err := miniredis.Run()
	require.NoError(t, err, "Failed to start mini redis")

	// 创建Redis客户端连接到模拟服务器
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	// 返回清理函数
	cleanup := func() {
		client.Close()
		mr.Close()
	}

	return mr, client, cleanup
}

func TestRedisConnectionWithMiniRedis(t *testing.T) {
	// 使用模拟Redis进行测试
	_, client, cleanup := setupMiniRedis(t)
	defer cleanup()

	// 测试Redis连接
	ctx := context.Background()
	pong, err := client.Ping(ctx).Result()
	assert.NoError(t, err, "Should ping Redis without error")
	assert.Equal(t, "PONG", pong, "Should receive PONG from Redis")
}

func TestRedisReadWriteWithMiniRedis(t *testing.T) {
	// 使用模拟Redis进行测试
	_, client, cleanup := setupMiniRedis(t)
	defer cleanup()

	// 测试上下文
	ctx := context.Background()

	// 测试键和值
	testKey := "test:key:" + time.Now().Format("20060102150405")
	testValue := "test value"

	// 写入值
	err := client.Set(ctx, testKey, testValue, 5*time.Minute).Err()
	assert.NoError(t, err, "Should set value without error")

	// 读取值
	value, err := client.Get(ctx, testKey).Result()
	assert.NoError(t, err, "Should get value without error")
	assert.Equal(t, testValue, value, "Retrieved value should match the set value")

	// 删除测试键
	err = client.Del(ctx, testKey).Err()
	assert.NoError(t, err, "Should delete key without error")

	// 验证删除
	exists, err := client.Exists(ctx, testKey).Result()
	assert.NoError(t, err, "Should check existence without error")
	assert.Equal(t, int64(0), exists, "Key should not exist after deletion")
}
