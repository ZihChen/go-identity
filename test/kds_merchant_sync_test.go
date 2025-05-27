package tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
)

func TestKDSMerchantSyncEvent(t *testing.T) {
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

	// 創建商戶同步事件
	merchantEvent := createMerchantSyncEvent()

	// 序列化事件
	eventBytes, err := json.Marshal(merchantEvent)
	require.NoError(t, err, "Should marshal merchant event without error")

	// 添加日誌
	t.Logf("Sending merchant sync event to KDS: %s", merchantEvent.ID)
	t.Logf("Global merchant ID: %s",
		merchantEvent.Data.(map[string]interface{})["global_merchant_id"])

	// 發送事件到KDS
	partitionKey := uuid.New().String()
	putOutput, err := client.PutRecord(context.Background(), &kinesis.PutRecordInput{
		Data:         eventBytes,
		StreamName:   aws.String(streamName),
		PartitionKey: aws.String(partitionKey),
	})

	require.NoError(t, err, "Should put merchant sync event to KDS without error")
	assert.NotEmpty(t, putOutput.SequenceNumber, "Put record should return a sequence number")
	assert.NotEmpty(t, putOutput.ShardId, "Put record should return a shard ID")

	t.Logf("Successfully sent merchant sync event to KDS, shard ID: %s, sequence number: %s",
		*putOutput.ShardId, *putOutput.SequenceNumber)
}

// createMerchantSyncEvent 創建商戶同步事件
func createMerchantSyncEvent() *event.CloudEvent {
	// 創建商戶數據
	merchantData := map[string]interface{}{
		"global_merchant_id": "FATCAT-MERCHANT-1",
		"merchant": map[string]interface{}{
			"id":                 1,
			"name":               "JV Diamond",
			"display_name":       "JV Diamond",
			"global_merchant_id": "FATCAT-MERCHANT-1",
		},
	}

	// 創建 CloudEvent
	return &event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "tw.jvd.fatcat.merchant.sync.v1",
		Source:          "/fatcat/FATCAT",
		Subject:         "merchant_sync",
		ID:              uuid.New().String(),
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     "00-" + uuid.New().String() + "-" + uuid.New().String()[0:16] + "-01",
		Data:            merchantData,
	}
}
