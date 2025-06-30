package tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/google/uuid"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKDSManagerSyncEvent(t *testing.T) {
	// 加載配置
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "Should load config without error")

	// 獲取AWS配置
	awsConfig, err := cfg.LoadAWSConfig(context.Background())
	require.NoError(t, err, "Should load AWS config without error")

	// 創建Kinesis客戶端
	client := kinesis.NewFromConfig(awsConfig)

	// 從ARN中提取stream名稱
	streamARN := cfg.AWS.KinesisStream
	streamName := streamARN
	for i := len(streamARN) - 1; i >= 0; i-- {
		if streamARN[i] == '/' || streamARN[i] == ':' {
			streamName = streamARN[i+1:]
			break
		}
	}

	// 創建管理員同步事件
	managerEvent := createManagerSyncEvent()

	// 序列化事件
	eventBytes, err := json.Marshal(managerEvent)
	require.NoError(t, err, "Should marshal manager event without error")

	// 添加日誌
	t.Logf("Sending manager sync event to KDS: %s", managerEvent.ID)
	t.Logf("Global merchant ID: %s",
		managerEvent.Data.(map[string]interface{})["global_merchant_id"])
	t.Logf("Global manager ID: %s",
		managerEvent.Data.(map[string]interface{})["manager"].(map[string]interface{})["global_manager_id"])

	// 發送事件到KDS
	partitionKey := uuid.New().String()
	putOutput, err := client.PutRecord(context.Background(), &kinesis.PutRecordInput{
		Data:         eventBytes,
		StreamName:   aws.String(streamName),
		PartitionKey: aws.String(partitionKey),
	})

	require.NoError(t, err, "Should put manager sync event to KDS without error")
	assert.NotEmpty(t, putOutput.SequenceNumber, "Put record should return a sequence number")
	assert.NotEmpty(t, putOutput.ShardId, "Put record should return a shard ID")

	t.Logf("Successfully sent manager sync event to KDS, shard ID: %s, sequence number: %s",
		*putOutput.ShardId, *putOutput.SequenceNumber)
}

// createManagerSyncEvent 創建管理員同步事件
func createManagerSyncEvent() *event.CloudEvent {
	// 創建管理員數據
	managerData := map[string]interface{}{
		"global_merchant_id": "FATCAT-MERCHANT-1",
		"manager": map[string]interface{}{
			"id":                231,
			"account":           "alan",
			"company_id":        2,
			"active":            true,
			"created_at":        "2024-10-23T07:28:05.000Z",
			"updated_at":        "2025-03-19T01:55:01.000Z",
			"email":             "alan.c@jvd.tw",
			"auth_token":        "b6ba9054d9bd10fe2ba748a5b8939e5331e9b045",
			"deleted_at":        nil,
			"global_manager_id": "FATCAT-MANAGER-231",
		},
	}

	// 創建 CloudEvent
	return &event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "tw.jvd.fatcat.manager.sync.v1",
		Source:          "/fatcat/FATCAT",
		Subject:         "manager_sync",
		ID:              uuid.New().String(),
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     "00-" + uuid.New().String() + "-" + uuid.New().String()[0:16] + "-01",
		Data:            managerData,
	}
}
