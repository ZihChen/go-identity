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
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKDSPlayerSyncEvent(t *testing.T) {
	// 加載配置
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "Should load config without error")

	// 獲取AWS配置
	awsConfig, err := cfg.LoadAWSConfig(context.Background())
	require.NoError(t, err, "Should load AWS config without error")

	identityOutput, err := sts.NewFromConfig(awsConfig).GetCallerIdentity(context.Background(), &sts.GetCallerIdentityInput{})
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

	// 創建玩家同步事件
	playerEvent := createPlayerSyncEvent()

	// 序列化事件
	eventBytes, err := json.Marshal(playerEvent)
	require.NoError(t, err, "Should marshal player event without error")

	// 添加日誌
	t.Logf("Sending player sync event to KDS: %s", playerEvent.ID)
	t.Logf("Global merchant ID: %s",
		playerEvent.Data.(map[string]interface{})["global_merchant_id"])
	t.Logf("Global player ID: %s",
		playerEvent.Data.(map[string]interface{})["player"].(map[string]interface{})["global_player_id"])

	// 發送事件到KDS
	partitionKey := uuid.New().String()
	putOutput, err := client.PutRecord(context.Background(), &kinesis.PutRecordInput{
		Data:         eventBytes,
		StreamName:   aws.String(streamName),
		PartitionKey: aws.String(partitionKey),
	})

	require.NoError(t, err, "Should put player sync event to KDS without error")
	assert.NotEmpty(t, putOutput.SequenceNumber, "Put record should return a sequence number")
	assert.NotEmpty(t, putOutput.ShardId, "Put record should return a shard ID")

	t.Logf("Successfully sent player sync event to KDS, shard ID: %s, sequence number: %s",
		*putOutput.ShardId, *putOutput.SequenceNumber)
}

// createPlayerSyncEvent 創建玩家同步事件
func createPlayerSyncEvent() *event.CloudEvent {
	// 創建玩家數據
	playerData := map[string]interface{}{
		"global_merchant_id": "FATCAT-MERCHANT-1",
		"player": map[string]interface{}{
			"global_player_id": "FATCAT-PLAYER-7241",
			"account":          "player1",
			"email":            "player@example.com",
			"status":           "active",
		},
	}

	// 創建 CloudEvent
	return &event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "tw.jvd.fatcat.player.sync.v1",
		Source:          "/fatcat/FATCAT",
		Subject:         "player_sync",
		ID:              uuid.New().String(),
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     "00-" + uuid.New().String() + "-" + uuid.New().String()[0:16] + "-01",
		Data:            playerData,
	}
}
