package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jvdiamondtech/ms-identity-cat/internal/adapter/handler/worker"
	"github.com/jvdiamondtech/ms-identity-cat/internal/di"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/infraport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/cache/redis"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/database"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/models"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type resultWriterCtxKey struct{}

type fakeResultWriter struct{}

func (f *fakeResultWriter) WriteResult([]byte) error { return nil }
func (f *fakeResultWriter) FailResult([]byte) error  { return nil }
func (f *fakeResultWriter) TaskID() string           { return "test-task-id" }

func processTaskDirectly(
	t *testing.T,
	handler *worker.WorkerHandler,
	taskType string,
	payload []byte,
) error {
	task := asynq.NewTask(taskType, payload)

	ctx := context.WithValue(context.Background(), resultWriterCtxKey{}, &fakeResultWriter{})

	switch taskType {
	case queue.TypeMerchantSync:
		return handler.HandleMerchantSync(ctx, task)
	case queue.TypePlayerSync:
		return handler.HandlePlayerSync(ctx, task)
	case queue.TypeManagerSync:
		return handler.HandleManagerSync(ctx, task)
	default:
		return fmt.Errorf("unsupported task type: %s", taskType)
	}
}

func setupWorkerComponents(
	t *testing.T,
) (*di.WorkerComponents, *config.Config, infraport.Logger, func()) {
	// 讀取配置
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "Should load config without error")

	// 建立 ServiceLogger
	logger := new(MockLogger)

	// 建立 Redis Manager
	redisManager := redis.NewRedisManager(cfg)
	err = redisManager.Connect(context.Background())
	require.NoError(t, err, "Should connect to Redis without error")

	// 初始化資料庫連接
	db, err := database.NewDatabase(cfg)
	require.NoError(t, err, "Should connect to database without error")

	// 初始化 Worker 組件
	workerComponents, err := di.InitializeWorkerComponents(
		cfg,
		logger,
		redisManager,
		db.DB,
	)
	require.NoError(t, err, "Should initialize worker components without error")

	// 清理函數
	cleanup := func() {
		// 關閉資料庫連接
		sqlDB, _ := db.DB.DB()
		if sqlDB != nil {
			err = sqlDB.Close()
			require.NoError(t, err)
		}
		// 關閉 Redis 連接
		err = redisManager.Close()
		require.NoError(t, err)
	}

	return workerComponents, cfg, logger, cleanup
}

func setupRedisClient(t *testing.T) (*asynq.Client, *config.Config, func()) {
	// 讀取配置
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "Should load config without error")

	// 建立 Redis 連接
	redisAddr := fmt.Sprintf("%s:%d", cfg.Redis.Domain, cfg.Redis.Port)
	redisOpt := asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	// 建立 Redis 客戶端
	client := asynq.NewClient(redisOpt)

	// 測試 Redis 連接
	inspector := asynq.NewInspector(redisOpt)
	_, err = inspector.Queues()
	require.NoError(t, err, "Should connect to Redis without error")

	// 清理函數
	cleanup := func() {
		err = client.Close()
		require.NoError(t, err)
	}

	return client, cfg, cleanup
}

func setupKinesisClient(t *testing.T) (*kinesis.Client, string, *config.Config, func()) {
	// 讀取配置
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "Should load config without error")

	// 載入 AWS 配置
	awsConfig, err := cfg.LoadAWSConfig(context.Background())
	require.NoError(t, err, "Should load AWS config without error")

	// 建立 Kinesis 客戶端
	client := kinesis.NewFromConfig(awsConfig)

	// 從 ARN 中提取stream名稱
	streamARN := cfg.AWS.KinesisStream
	streamName := streamARN
	for i := len(streamARN) - 1; i >= 0; i-- {
		if streamARN[i] == '/' || streamARN[i] == ':' {
			streamName = streamARN[i+1:]
			break
		}
	}

	// 測試 Kinesis 連接
	_, err = client.DescribeStream(context.Background(), &kinesis.DescribeStreamInput{
		StreamName: aws.String(streamName),
	})
	require.NoError(t, err, "Should connect to Kinesis without error")

	// 清理函數 - 這裡沒有特別需要清理的資源
	cleanup := func() {}

	return client, streamName, cfg, cleanup
}

func TestMerchantRedisToWorker(t *testing.T) {
	t.Logf("Starting merchant Redis-to-Worker-to-KDS test")

	// 設置 Worker 組件
	workerComponents, cfg, _, cleanupWorker := setupWorkerComponents(t)
	defer cleanupWorker()

	// 設置 Redis 客戶端
	redisClient, _, cleanupRedis := setupRedisClient(t)
	defer cleanupRedis()

	// 設置 Kinesis 客戶端
	kinesisClient, streamName, _, cleanupKinesis := setupKinesisClient(t)
	defer cleanupKinesis()

	// 建立商戶唯一標識
	testID := time.Now().Format("20060102150405")
	globalMerchantID := "TEST-MERCHANT-" + testID
	merchantName := "Test Merchant " + testID
	displayName := "Test Display " + testID

	// 建立商戶同步事件
	merchantEvent := event.MerchantSyncEvent{
		GlobalMerchantID: globalMerchantID,
		Merchant: event.MerchantData{
			ID:               100 + int(time.Now().Unix()%100),
			Name:             merchantName,
			DisplayName:      displayName,
			GlobalMerchantID: globalMerchantID,
		},
	}

	cloudEvent := event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            cfg.Events.MerchantSync,
		Source:          "/test/merchant",
		Subject:         "merchant_sync",
		ID:              uuid.New().String(),
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     "00-" + uuid.New().String() + "-" + uuid.New().String()[0:16] + "-01",
		Data:            merchantEvent,
	}

	// 序列化事件
	eventBytes, err := json.Marshal(cloudEvent)
	require.NoError(t, err, "Should marshal merchant event without error")

	// 建立任務
	task := asynq.NewTask(queue.TypeMerchantSync, eventBytes)

	// 加入任務到 Redis
	info, err := redisClient.Enqueue(task)
	require.NoError(t, err, "Should enqueue merchant sync task without error")
	t.Logf("Enqueued merchant sync task: %s", info.ID)

	// 建立 ServeMux
	mux := asynq.NewServeMux()
	workerComponents.Handler.RegisterHandlers(mux)

	// 不使用 worker server 的 ProcessTask，而是直接使用 mux
	err = processTaskDirectly(t, workerComponents.Handler, queue.TypeMerchantSync, eventBytes)
	assert.NoError(t, err, "Should process merchant sync task without error")
	t.Logf("Processed merchant sync task directly")

	// 驗證資料庫中有商戶記錄
	db, err := database.NewDatabase(cfg)
	require.NoError(t, err, "Should connect to database without error")
	var merchant models.Merchant
	result := db.DB.Where("global_merchant_id = ?", globalMerchantID).First(&merchant)
	assert.NoError(t, result.Error, "Should find merchant in database")
	assert.Equal(t, globalMerchantID, merchant.GlobalMerchantID, "Global merchant ID should match")
	assert.Equal(t, merchantName, merchant.Name, "Merchant name should match")
	assert.Equal(t, displayName, merchant.DisplayName, "Display name should match")
	t.Logf("Verified merchant record in database: %d", merchant.ID)

	// 等待消息同步到 KDS
	// 這部分較難驗證，因為我們無法知道確切的分片和序列號
	// 我們將嘗試列出所有分片，然後檢查最近的事件
	shardsOutput, err := kinesisClient.ListShards(context.Background(), &kinesis.ListShardsInput{
		StreamName: aws.String(streamName),
	})
	require.NoError(t, err, "Should list shards without error")

	// 檢查所有分片中是否有我們的事件
	var foundEvent bool
	for _, shard := range shardsOutput.Shards {
		if shard.ShardId != nil {
			shardIteratorOutput, err := kinesisClient.GetShardIterator(
				context.Background(),
				&kinesis.GetShardIteratorInput{
					StreamName:        aws.String(streamName),
					ShardId:           shard.ShardId,
					ShardIteratorType: types.ShardIteratorTypeAtTimestamp,
					Timestamp:         aws.Time(time.Now().Add(-40 * time.Second)), // 往前一些保險
				},
			)

			if err != nil {
				t.Logf("Error getting shard iterator: %v", err)
				continue
			}

			// 給 KDS 一些時間處理
			time.Sleep(2 * time.Second)

			// 嘗試讀取記錄
			getRecordsOutput, err := kinesisClient.GetRecords(
				context.Background(),
				&kinesis.GetRecordsInput{
					ShardIterator: shardIteratorOutput.ShardIterator,
					Limit:         aws.Int32(100),
				},
			)
			if err != nil {
				t.Logf("Error getting records: %v", err)
				continue
			}

			// 檢查記錄
			for _, record := range getRecordsOutput.Records {
				var recordEvent map[string]interface{}
				if err := json.Unmarshal(record.Data, &recordEvent); err != nil {
					continue
				}

				// 檢查是否是我們的商戶同步事件
				if typeStr, ok := recordEvent["type"].(string); ok &&
					typeStr == cfg.Events.IdentityMerchantSync {
					if dataObj, ok := recordEvent["data"].(map[string]interface{}); ok {
						if gmid, ok := dataObj["global_merchant_id"].(string); ok &&
							gmid == globalMerchantID {
							foundEvent = true
							t.Logf(
								"Found merchant sync event in KDS for global_merchant_id: %s",
								globalMerchantID,
							)
							break
						}
					}
				}
			}

			if foundEvent {
				break
			}
		}
	}

	// 清理 - 刪除測試商戶
	db.DB.Delete(&merchant)
	t.Logf("Deleted test merchant from database: %d", merchant.ID)

	// 最終驗證
	assert.True(t, foundEvent, "Should find merchant sync event in KDS")
	t.Logf("Merchant Redis-to-Worker-to-KDS test completed successfully")
}

func TestPlayerRedisToWorker(t *testing.T) {
	t.Logf("Starting player Redis-to-Worker-to-KDS test")

	// 設置 Worker 組件
	workerComponents, cfg, _, cleanupWorker := setupWorkerComponents(t)
	defer cleanupWorker()

	// 設置 Redis 客戶端
	redisClient, _, cleanupRedis := setupRedisClient(t)
	defer cleanupRedis()

	// 設置 Kinesis 客戶端
	kinesisClient, streamName, _, cleanupKinesis := setupKinesisClient(t)
	defer cleanupKinesis()

	// 建立資料庫連接
	db, err := database.NewDatabase(cfg)
	require.NoError(t, err, "Should connect to database without error")

	// 首先需要建立一個商戶作為外鍵關聯
	testID := time.Now().Format("20060102150405")
	globalMerchantID := "TEST-MERCHANT-PLAYER-" + testID
	merchantName := "Test Merchant " + testID
	merchant := &models.Merchant{
		GlobalMerchantID: globalMerchantID,
		Name:             merchantName,
		DisplayName:      "Test Display",
		APIKey:           "test-api-key-" + testID,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	result := db.DB.Create(merchant)
	require.NoError(t, result.Error, "Should create merchant without error")
	t.Logf("Created test merchant in database: %d", merchant.ID)

	// 建立玩家同步事件
	globalPlayerID := "TEST-PLAYER-" + testID
	playerAccount := "testplayer" + testID
	playerEmail := "test" + testID + "@example.com"

	playerEvent := event.PlayerSyncEvent{
		GlobalMerchantID: globalMerchantID,
		Player: event.PlayerData{
			GlobalPlayerID: globalPlayerID,
			Account:        playerAccount,
			Email:          playerEmail,
			Status:         "active",
		},
	}

	cloudEvent := event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            cfg.Events.PlayerSync,
		Source:          "/test/player",
		Subject:         "player_sync",
		ID:              uuid.New().String(),
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     "00-" + uuid.New().String() + "-" + uuid.New().String()[0:16] + "-01",
		Data:            playerEvent,
	}

	// 序列化事件
	eventBytes, err := json.Marshal(cloudEvent)
	require.NoError(t, err, "Should marshal player event without error")

	// 建立任務
	task := asynq.NewTask(queue.TypePlayerSync, eventBytes)

	// 加入任務到 Redis
	info, err := redisClient.Enqueue(task)
	require.NoError(t, err, "Should enqueue player sync task without error")
	t.Logf("Enqueued player sync task: %s", info.ID)

	// 建立 ServeMux
	mux := asynq.NewServeMux()
	workerComponents.Handler.RegisterHandlers(mux)

	// 直接處理任務
	err = processTaskDirectly(t, workerComponents.Handler, queue.TypePlayerSync, eventBytes)
	assert.NoError(t, err, "Should process player sync task without error")
	t.Logf("Processed player sync task directly")

	// 驗證資料庫中有玩家記錄
	var player models.Player
	result = db.DB.Where("global_player_id = ?", globalPlayerID).First(&player)
	assert.NoError(t, result.Error, "Should find player in database")
	assert.Equal(t, globalPlayerID, player.GlobalPlayerID, "Global player ID should match")
	assert.Equal(t, playerAccount, player.Account, "Player account should match")
	assert.Equal(t, playerEmail, *player.Email, "Player email should match")
	assert.Equal(t, merchant.ID, player.MerchantID, "Merchant ID should match")
	t.Logf("Verified player record in database: %d", player.ID)

	// 等待消息同步到 KDS

	shardsOutput, err := kinesisClient.ListShards(context.Background(), &kinesis.ListShardsInput{
		StreamName: aws.String(streamName),
	})
	require.NoError(t, err, "Should list shards without error")

	// 檢查所有分片中是否有我們的事件
	var foundEvent bool
	for _, shard := range shardsOutput.Shards {
		if shard.ShardId != nil {
			shardIteratorOutput, err := kinesisClient.GetShardIterator(
				context.Background(),
				&kinesis.GetShardIteratorInput{
					StreamName:        aws.String(streamName),
					ShardId:           shard.ShardId,
					ShardIteratorType: types.ShardIteratorTypeAtTimestamp,
					Timestamp:         aws.Time(time.Now().Add(-40 * time.Second)), // 往前一些保險
				},
			)

			if err != nil {
				t.Logf("Error getting shard iterator: %v", err)
				continue
			}

			// 給 KDS 一些時間處理
			time.Sleep(2 * time.Second)

			// 嘗試讀取記錄
			getRecordsOutput, err := kinesisClient.GetRecords(
				context.Background(),
				&kinesis.GetRecordsInput{
					ShardIterator: shardIteratorOutput.ShardIterator,
					Limit:         aws.Int32(100),
				},
			)
			if err != nil {
				t.Logf("Error getting records: %v", err)
				continue
			}

			// 檢查記錄
			for _, record := range getRecordsOutput.Records {
				var recordEvent map[string]interface{}
				if err := json.Unmarshal(record.Data, &recordEvent); err != nil {
					continue
				}

				// 檢查是否是我們的玩家同步事件
				if typeStr, ok := recordEvent["type"].(string); ok &&
					typeStr == cfg.Events.IdentityPlayerSync {
					if dataObj, ok := recordEvent["data"].(map[string]interface{}); ok {
						if gpid, ok := dataObj["global_player_id"].(string); ok &&
							gpid == globalPlayerID {
							foundEvent = true
							t.Logf(
								"Found player sync event in KDS for global_player_id: %s",
								globalPlayerID,
							)
							break
						}
					}
				}
			}

			if foundEvent {
				break
			}
		}
	}

	// 清理 - 刪除測試玩家和商戶
	db.DB.Delete(&player)
	t.Logf("Deleted test player from database: %d", player.ID)
	db.DB.Delete(&merchant)
	t.Logf("Deleted test merchant from database: %d", merchant.ID)

	// 最終驗證
	assert.True(t, foundEvent, "Should find player sync event in KDS")
	t.Logf("Player Redis-to-Worker-to-KDS test completed successfully")
}

func TestManagerRedisToWorker(t *testing.T) {
	t.Logf("Starting manager Redis-to-Worker-to-KDS test")

	// 設置 Worker 組件
	workerComponents, cfg, _, cleanupWorker := setupWorkerComponents(t)
	defer cleanupWorker()

	// 設置 Redis 客戶端
	redisClient, _, cleanupRedis := setupRedisClient(t)
	defer cleanupRedis()

	// 設置 Kinesis 客戶端
	kinesisClient, streamName, _, cleanupKinesis := setupKinesisClient(t)
	defer cleanupKinesis()

	// 建立資料庫連接
	db, err := database.NewDatabase(cfg)
	require.NoError(t, err, "Should connect to database without error")

	// 首先需要建立一個商戶作為外鍵關聯
	testID := time.Now().Format("20060102150405")
	globalMerchantID := "TEST-MERCHANT-MANAGER-" + testID
	merchantName := "Test Merchant " + testID
	merchant := &models.Merchant{
		GlobalMerchantID: globalMerchantID,
		Name:             merchantName,
		DisplayName:      "Test Display",
		APIKey:           "test-api-key-" + testID,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	result := db.DB.Create(merchant)
	require.NoError(t, result.Error, "Should create merchant without error")
	t.Logf("Created test merchant in database: %d", merchant.ID)

	// 建立管理員同步事件
	globalManagerID := "TEST-MANAGER-" + testID
	managerAccount := "testmanager" + testID
	managerEmail := "test" + testID + "@example.com"

	managerEvent := event.ManagerSyncEvent{
		GlobalMerchantID: globalMerchantID,
		Manager: event.ManagerData{
			ID:              231,
			GlobalManagerID: globalManagerID,
			Account:         managerAccount,
			Email:           managerEmail,
		},
	}

	cloudEvent := event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            cfg.Events.ManagerSync,
		Source:          "/test/manager",
		Subject:         "manager_sync",
		ID:              uuid.New().String(),
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     "00-" + uuid.New().String() + "-" + uuid.New().String()[0:16] + "-01",
		Data:            managerEvent,
	}

	// 序列化事件
	eventBytes, err := json.Marshal(cloudEvent)
	require.NoError(t, err, "Should marshal manager event without error")

	// 建立任務
	task := asynq.NewTask(queue.TypeManagerSync, eventBytes)

	// 加入任務到 Redis
	info, err := redisClient.Enqueue(task)
	require.NoError(t, err, "Should enqueue manager sync task without error")
	t.Logf("Enqueued manager sync task: %s", info.ID)

	// 建立 ServeMux
	mux := asynq.NewServeMux()
	workerComponents.Handler.RegisterHandlers(mux)

	// 直接處理任務
	err = processTaskDirectly(t, workerComponents.Handler, queue.TypeManagerSync, eventBytes)
	assert.NoError(t, err, "Should process manager sync task without error")
	t.Logf("Processed manager sync task directly")

	// 驗證資料庫中有管理員記錄
	var manager models.Manager
	result = db.DB.Where("global_manager_id = ?", globalManagerID).First(&manager)
	assert.NoError(t, result.Error, "Should find manager in database")
	assert.Equal(t, globalManagerID, manager.GlobalManagerID, "Global manager ID should match")
	assert.Equal(t, managerAccount, manager.Account, "Manager account should match")
	assert.Equal(t, managerEmail, *manager.Email, "Manager email should match")
	assert.Equal(t, merchant.ID, manager.MerchantID, "Merchant ID should match")
	t.Logf("Verified manager record in database: %d", manager.ID)

	// 等待消息同步到 KDS
	// 這部分較難驗證，因為我們無法知道確切的分片和序列號
	// 我們將嘗試列出所有分片，然後檢查最近的事件
	shardsOutput, err := kinesisClient.ListShards(context.Background(), &kinesis.ListShardsInput{
		StreamName: aws.String(streamName),
	})
	require.NoError(t, err, "Should list shards without error")

	// 檢查所有分片中是否有我們的事件
	var foundEvent bool
	for _, shard := range shardsOutput.Shards {
		if shard.ShardId != nil {
			shardIteratorOutput, err := kinesisClient.GetShardIterator(
				context.Background(),
				&kinesis.GetShardIteratorInput{
					StreamName:        aws.String(streamName),
					ShardId:           shard.ShardId,
					ShardIteratorType: types.ShardIteratorTypeAtTimestamp,
					Timestamp:         aws.Time(time.Now().Add(-40 * time.Second)), // 往前一些保險
				},
			)

			if err != nil {
				t.Logf("Error getting shard iterator: %v", err)
				continue
			}

			// 給 KDS 一些時間處理
			time.Sleep(2 * time.Second)

			// 嘗試讀取記錄
			getRecordsOutput, err := kinesisClient.GetRecords(
				context.Background(),
				&kinesis.GetRecordsInput{
					ShardIterator: shardIteratorOutput.ShardIterator,
					Limit:         aws.Int32(100),
				},
			)
			if err != nil {
				t.Logf("Error getting records: %v", err)
				continue
			}

			// 檢查記錄
			for _, record := range getRecordsOutput.Records {
				var recordEvent map[string]interface{}
				if err := json.Unmarshal(record.Data, &recordEvent); err != nil {
					continue
				}

				// 檢查是否是我們的管理員同步事件
				if typeStr, ok := recordEvent["type"].(string); ok &&
					typeStr == cfg.Events.IdentityManagerSync {
					if dataObj, ok := recordEvent["data"].(map[string]interface{}); ok {
						if gmid, ok := dataObj["global_manager_id"].(string); ok &&
							gmid == globalManagerID {
							foundEvent = true
							t.Logf(
								"Found manager sync event in KDS for global_manager_id: %s",
								globalManagerID,
							)
							break
						}
					}
				}
			}

			if foundEvent {
				break
			}
		}
	}

	// 清理 - 刪除測試管理員和商戶
	db.DB.Delete(&manager)
	t.Logf("Deleted test manager from database: %d", manager.ID)
	db.DB.Delete(&merchant)
	t.Logf("Deleted test merchant from database: %d", merchant.ID)

	// 最終驗證
	assert.True(t, foundEvent, "Should find manager sync event in KDS")
	t.Logf("Manager Redis-to-Worker-to-KDS test completed successfully")
}
