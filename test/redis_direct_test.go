package tests

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisDirectManagerSync(t *testing.T) {
	// 加載配置
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "Should load config without error")

	// 創建Redis配置
	redisAddr := fmt.Sprintf("%s:%d", cfg.Redis.Domain, cfg.Redis.Port)
	redisOpt := asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	// 創建Redis客戶端
	client := asynq.NewClient(redisOpt)
	defer client.Close()

	// 創建管理員同步事件
	managerEvent := createManagerSyncEvent()

	// 序列化事件
	eventBytes, err := json.Marshal(managerEvent)
	require.NoError(t, err, "Should marshal manager event without error")

	// 創建任務
	task := asynq.NewTask(queue.TypeManagerSync, eventBytes)

	// 添加任務到Redis隊列
	info, err := client.Enqueue(task)
	require.NoError(t, err, "Should enqueue manager sync task without error")

	t.Logf("Successfully enqueued manager sync task to Redis, task ID: %s, queue: %s",
		info.ID, info.Queue)

	// 驗證任務是否在隊列中
	inspector := asynq.NewInspector(redisOpt)
	tasks, err := inspector.ListPendingTasks(info.Queue)
	require.NoError(t, err, "Should list pending tasks without error")

	// 檢查任務是否存在
	found := false
	for _, t := range tasks {
		if t.ID == info.ID {
			found = true
			break
		}
	}

	assert.True(t, found, "Should find the task in the queue")
}

func TestRedisDirectPlayerSync(t *testing.T) {
	// 加載配置
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "Should load config without error")

	// 創建Redis配置
	redisAddr := fmt.Sprintf("%s:%d", cfg.Redis.Domain, cfg.Redis.Port)
	redisOpt := asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	// 創建Redis客戶端
	client := asynq.NewClient(redisOpt)
	defer client.Close()

	// 創建玩家同步事件
	playerEvent := createPlayerSyncEvent()

	// 序列化事件
	eventBytes, err := json.Marshal(playerEvent)
	require.NoError(t, err, "Should marshal player event without error")

	// 創建任務
	task := asynq.NewTask(queue.TypePlayerSync, eventBytes)

	// 添加任務到Redis隊列
	info, err := client.Enqueue(task)
	require.NoError(t, err, "Should enqueue player sync task without error")

	t.Logf("Successfully enqueued player sync task to Redis, task ID: %s, queue: %s",
		info.ID, info.Queue)

	// 驗證任務是否在隊列中
	inspector := asynq.NewInspector(redisOpt)
	tasks, err := inspector.ListPendingTasks(info.Queue)
	require.NoError(t, err, "Should list pending tasks without error")

	// 檢查任務是否存在
	found := false
	for _, t := range tasks {
		if t.ID == info.ID {
			found = true
			break
		}
	}

	assert.True(t, found, "Should find the task in the queue")
}

func TestRedisDirectMerchantSync(t *testing.T) {
	// 加載配置
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "Should load config without error")

	// 創建Redis配置
	redisAddr := fmt.Sprintf("%s:%d", cfg.Redis.Domain, cfg.Redis.Port)
	redisOpt := asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	// 創建Redis客戶端
	client := asynq.NewClient(redisOpt)
	defer client.Close()

	// 創建商戶同步事件
	merchantEvent := createMerchantSyncEvent()

	// 序列化事件
	eventBytes, err := json.Marshal(merchantEvent)
	require.NoError(t, err, "Should marshal merchant event without error")

	// 創建任務
	task := asynq.NewTask(queue.TypeMerchantSync, eventBytes)

	// 添加任務到Redis隊列
	info, err := client.Enqueue(task)
	require.NoError(t, err, "Should enqueue merchant sync task without error")

	t.Logf("Successfully enqueued merchant sync task to Redis, task ID: %s, queue: %s",
		info.ID, info.Queue)

	// 驗證任務是否在隊列中
	inspector := asynq.NewInspector(redisOpt)
	tasks, err := inspector.ListPendingTasks(info.Queue)
	require.NoError(t, err, "Should list pending tasks without error")

	// 檢查任務是否存在
	found := false
	for _, t := range tasks {
		if t.ID == info.ID {
			found = true
			break
		}
	}

	assert.True(t, found, "Should find the task in the queue")
}
