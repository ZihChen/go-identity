package tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/google/uuid"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/queue"
	"github.com/jvdiamondtech/ms-identity-cat/test/helper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockLogger is now in test/mocks/service_mocks.go

func TestKDSToRedisManagerSync(t *testing.T) {
	// 加載配置
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "Should load config without error")

	// 創建日誌
	logger := helper.NewMockLogger()

	// 獲取AWS配置
	awsConfig, err := cfg.LoadAWSConfig(context.Background())
	require.NoError(t, err, "Should load AWS config without error")

	identityOutput, err := sts.NewFromConfig(awsConfig).
		GetCallerIdentity(context.Background(), &sts.GetCallerIdentityInput{})
	require.NoError(t, err, "Should get caller identity")
	t.Logf("Caller identity ARN: %s", *identityOutput.Arn)

	// 創建Kinesis客戶端
	kinesisClient := kinesis.NewFromConfig(awsConfig)

	// 創建Redis隊列客戶端
	queueService, err := queue.NewQueueService(cfg, logger)
	require.NoError(t, err, "Should create queue serviceport without error")

	// 從ARN中提取stream名稱
	streamARN := cfg.AWS.KinesisStream
	streamName := streamARN
	for i := len(streamARN) - 1; i >= 0; i-- {
		if streamARN[i] == '/' || streamARN[i] == ':' {
			streamName = streamARN[i+1:]
			break
		}
	}

	// 先發送管理員同步事件到KDS
	managerEvent := createManagerSyncEvent()
	eventBytes, err := json.Marshal(managerEvent)
	require.NoError(t, err, "Should marshal manager event without error")

	// 發送事件到KDS
	partitionKey := uuid.New().String()
	putOutput, err := kinesisClient.PutRecord(context.Background(), &kinesis.PutRecordInput{
		Data:         eventBytes,
		StreamName:   aws.String(streamName),
		PartitionKey: aws.String(partitionKey),
	})
	require.NoError(t, err, "Should put record to KDS without error")

	t.Logf("Sent manager sync event to KDS, event ID: %s, shard ID: %s",
		managerEvent.ID, *putOutput.ShardId)

	// 等待消息傳播
	time.Sleep(1 * time.Second)

	// 從寫入分片讀取消息
	shardIteratorOutput, err := kinesisClient.GetShardIterator(
		context.Background(),
		&kinesis.GetShardIteratorInput{
			StreamName:             aws.String(streamName),
			ShardId:                putOutput.ShardId,
			ShardIteratorType:      types.ShardIteratorTypeAtSequenceNumber,
			StartingSequenceNumber: putOutput.SequenceNumber,
		},
	)
	require.NoError(t, err, "Should get shard iterator without error")

	// 讀取記錄
	getRecordsOutput, err := kinesisClient.GetRecords(
		context.Background(),
		&kinesis.GetRecordsInput{
			ShardIterator: shardIteratorOutput.ShardIterator,
			Limit:         aws.Int32(10),
		},
	)
	require.NoError(t, err, "Should get records without error")

	// 驗證是否能找到我們發送的消息並轉發到Redis
	foundEvent := false
	for _, record := range getRecordsOutput.Records {
		// 檢查是否是我們發送的事件
		var cloudEvent map[string]interface{}
		err = json.Unmarshal(record.Data, &cloudEvent)
		if err != nil {
			continue
		}

		if id, ok := cloudEvent["id"].(string); ok && id == managerEvent.ID {
			foundEvent = true

			// 轉發到Redis
			err = queueService.EnqueueManagerSync(context.Background(), record.Data)
			require.NoError(t, err, "Should enqueue manager sync event to Redis without error")

			t.Logf(
				"Successfully forwarded manager sync event to Redis, event ID: %s",
				managerEvent.ID,
			)
			break
		}
	}

	assert.True(t, foundEvent, "Should find the sent manager sync event in KDS")
}

func TestKDSToRedisPlayerSync(t *testing.T) {
	// 加載配置
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "Should load config without error")

	// 創建日誌
	logger := helper.NewMockLogger()

	// 獲取AWS配置
	awsConfig, err := cfg.LoadAWSConfig(context.Background())
	require.NoError(t, err, "Should load AWS config without error")

	// 創建Kinesis客戶端
	kinesisClient := kinesis.NewFromConfig(awsConfig)

	// 創建Redis隊列客戶端
	queueService, err := queue.NewQueueService(cfg, logger)
	require.NoError(t, err, "Should create queue serviceport without error")

	// 從ARN中提取stream名稱
	streamARN := cfg.AWS.KinesisStream
	streamName := streamARN
	for i := len(streamARN) - 1; i >= 0; i-- {
		if streamARN[i] == '/' || streamARN[i] == ':' {
			streamName = streamARN[i+1:]
			break
		}
	}

	// 先發送玩家同步事件到KDS
	playerEvent := createPlayerSyncEvent()
	eventBytes, err := json.Marshal(playerEvent)
	require.NoError(t, err, "Should marshal player event without error")

	// 發送事件到KDS
	partitionKey := uuid.New().String()
	putOutput, err := kinesisClient.PutRecord(context.Background(), &kinesis.PutRecordInput{
		Data:         eventBytes,
		StreamName:   aws.String(streamName),
		PartitionKey: aws.String(partitionKey),
	})
	require.NoError(t, err, "Should put record to KDS without error")

	t.Logf("Sent player sync event to KDS, event ID: %s, shard ID: %s",
		playerEvent.ID, *putOutput.ShardId)

	// 等待消息傳播
	time.Sleep(1 * time.Second)

	// 從寫入分片讀取消息
	shardIteratorOutput, err := kinesisClient.GetShardIterator(
		context.Background(),
		&kinesis.GetShardIteratorInput{
			StreamName:             aws.String(streamName),
			ShardId:                putOutput.ShardId,
			ShardIteratorType:      types.ShardIteratorTypeAtSequenceNumber,
			StartingSequenceNumber: putOutput.SequenceNumber,
		},
	)
	require.NoError(t, err, "Should get shard iterator without error")

	// 讀取記錄
	getRecordsOutput, err := kinesisClient.GetRecords(
		context.Background(),
		&kinesis.GetRecordsInput{
			ShardIterator: shardIteratorOutput.ShardIterator,
			Limit:         aws.Int32(10),
		},
	)
	require.NoError(t, err, "Should get records without error")

	// 驗證是否能找到我們發送的消息並轉發到Redis
	foundEvent := false
	for _, record := range getRecordsOutput.Records {
		// 檢查是否是我們發送的事件
		var cloudEvent map[string]interface{}
		err = json.Unmarshal(record.Data, &cloudEvent)
		if err != nil {
			continue
		}

		if id, ok := cloudEvent["id"].(string); ok && id == playerEvent.ID {
			foundEvent = true

			// 轉發到Redis
			err = queueService.EnqueuePlayerSync(context.Background(), record.Data)
			require.NoError(t, err, "Should enqueue player sync event to Redis without error")

			t.Logf(
				"Successfully forwarded player sync event to Redis, event ID: %s",
				playerEvent.ID,
			)
			break
		}
	}

	assert.True(t, foundEvent, "Should find the sent player sync event in KDS")
}

func TestKDSToRedisMerchantSync(t *testing.T) {
	// 加載配置
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "Should load config without error")

	// 創建日誌
	logger := helper.NewMockLogger()

	// 獲取AWS配置
	awsConfig, err := cfg.LoadAWSConfig(context.Background())
	require.NoError(t, err, "Should load AWS config without error")

	// 創建Kinesis客戶端
	kinesisClient := kinesis.NewFromConfig(awsConfig)

	// 創建Redis隊列客戶端
	queueService, err := queue.NewQueueService(cfg, logger)
	require.NoError(t, err, "Should create queue serviceport without error")

	// 從ARN中提取stream名稱
	streamARN := cfg.AWS.KinesisStream
	streamName := streamARN
	for i := len(streamARN) - 1; i >= 0; i-- {
		if streamARN[i] == '/' || streamARN[i] == ':' {
			streamName = streamARN[i+1:]
			break
		}
	}

	// 先發送商戶同步事件到KDS
	merchantEvent := createMerchantSyncEvent()
	eventBytes, err := json.Marshal(merchantEvent)
	require.NoError(t, err, "Should marshal merchant event without error")

	// 發送事件到KDS
	partitionKey := uuid.New().String()
	putOutput, err := kinesisClient.PutRecord(context.Background(), &kinesis.PutRecordInput{
		Data:         eventBytes,
		StreamName:   aws.String(streamName),
		PartitionKey: aws.String(partitionKey),
	})
	require.NoError(t, err, "Should put record to KDS without error")

	t.Logf("Sent merchant sync event to KDS, event ID: %s, shard ID: %s",
		merchantEvent.ID, *putOutput.ShardId)

	// 等待消息傳播
	time.Sleep(1 * time.Second)

	// 從寫入分片讀取消息
	shardIteratorOutput, err := kinesisClient.GetShardIterator(
		context.Background(),
		&kinesis.GetShardIteratorInput{
			StreamName:             aws.String(streamName),
			ShardId:                putOutput.ShardId,
			ShardIteratorType:      types.ShardIteratorTypeAtSequenceNumber,
			StartingSequenceNumber: putOutput.SequenceNumber,
		},
	)
	require.NoError(t, err, "Should get shard iterator without error")

	// 讀取記錄
	getRecordsOutput, err := kinesisClient.GetRecords(
		context.Background(),
		&kinesis.GetRecordsInput{
			ShardIterator: shardIteratorOutput.ShardIterator,
			Limit:         aws.Int32(10),
		},
	)
	require.NoError(t, err, "Should get records without error")

	// 驗證是否能找到我們發送的消息並轉發到Redis
	foundEvent := false
	for _, record := range getRecordsOutput.Records {
		// 檢查是否是我們發送的事件
		var cloudEvent map[string]interface{}
		err = json.Unmarshal(record.Data, &cloudEvent)
		if err != nil {
			continue
		}

		if id, ok := cloudEvent["id"].(string); ok && id == merchantEvent.ID {
			foundEvent = true

			// 轉發到Redis
			err = queueService.EnqueueMerchantSync(context.Background(), record.Data)
			require.NoError(t, err, "Should enqueue merchant sync event to Redis without error")

			t.Logf(
				"Successfully forwarded merchant sync event to Redis, event ID: %s",
				merchantEvent.ID,
			)
			break
		}
	}

	assert.True(t, foundEvent, "Should find the sent merchant sync event in KDS")
}
