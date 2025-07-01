package tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/google/uuid"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKDSConnection(t *testing.T) {

	// 加載配置
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "Should load config without error")

	// 獲取AWS配置
	awsConfig, err := cfg.LoadAWSConfig(context.Background())
	require.NoError(t, err, "Should load AWS config without error")

	identityOutput, err := sts.NewFromConfig(awsConfig).
		GetCallerIdentity(context.Background(), &sts.GetCallerIdentityInput{})
	require.NoError(t, err, "Should get caller identity")
	t.Logf("Caller identity ARN: %s", *identityOutput.Arn)
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

	t.Logf("Testing KDS connection with stream name: %s", streamName)

	// 測試能否描述stream
	describeOutput, err := client.DescribeStream(context.Background(), &kinesis.DescribeStreamInput{
		StreamName: aws.String(streamName),
	})

	require.NoError(t, err, "Should describe KDS stream without error")
	assert.Equal(
		t,
		streamName,
		*describeOutput.StreamDescription.StreamName,
		"Stream name should match",
	)
	assert.NotEmpty(t, describeOutput.StreamDescription.Shards, "Stream should have shards")

	t.Logf("Successfully connected to KDS stream: %s", streamName)
	t.Logf("Stream status: %s", describeOutput.StreamDescription.StreamStatus)
	t.Logf("Number of shards: %d", len(describeOutput.StreamDescription.Shards))

	// 測試寫入和讀取消息
	testMessage := map[string]interface{}{
		"test_id":   uuid.New().String(),
		"message":   "KDS connection test",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	// 序列化消息
	messageBytes, err := json.Marshal(testMessage)
	require.NoError(t, err, "Should marshal test message without error")

	// 發送消息到KDS
	partitionKey := uuid.New().String()
	putOutput, err := client.PutRecord(context.Background(), &kinesis.PutRecordInput{
		Data:         messageBytes,
		StreamName:   aws.String(streamName),
		PartitionKey: aws.String(partitionKey),
	})

	require.NoError(t, err, "Should put record to KDS without error")
	assert.NotEmpty(t, putOutput.SequenceNumber, "Put record should return a sequence number")
	assert.NotEmpty(t, putOutput.ShardId, "Put record should return a shard ID")

	t.Logf("Successfully wrote test message to KDS, shard ID: %s, sequence number: %s",
		*putOutput.ShardId, *putOutput.SequenceNumber)

	// 等待消息傳播
	time.Sleep(1 * time.Second)

	// 從寫入分片讀取消息
	shardIteratorOutput, err := client.GetShardIterator(
		context.Background(),
		&kinesis.GetShardIteratorInput{
			StreamName:             aws.String(streamName),
			ShardId:                putOutput.ShardId,
			ShardIteratorType:      "AT_SEQUENCE_NUMBER",
			StartingSequenceNumber: putOutput.SequenceNumber,
		},
	)

	require.NoError(t, err, "Should get shard iterator without error")
	assert.NotEmpty(t, shardIteratorOutput.ShardIterator, "Should receive a valid shard iterator")

	// 讀取記錄
	getRecordsOutput, err := client.GetRecords(context.Background(), &kinesis.GetRecordsInput{
		ShardIterator: shardIteratorOutput.ShardIterator,
		Limit:         aws.Int32(10),
	})

	require.NoError(t, err, "Should get records without error")

	// 驗證是否能找到我們發送的消息
	var foundMessage bool
	var recordData map[string]interface{}

	for _, record := range getRecordsOutput.Records {
		err = json.Unmarshal(record.Data, &recordData)
		if err != nil {
			continue
		}

		if testID, ok := recordData["test_id"].(string); ok {
			if testID == testMessage["test_id"] {
				foundMessage = true
				break
			}
		}
	}

	assert.True(t, foundMessage, "Should find the sent message in the KDS stream")
	t.Log("KDS connection test completed successfully")
}
