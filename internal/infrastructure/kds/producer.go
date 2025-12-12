package kds

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/aws/aws-sdk-go-v2/service/kinesis/types"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// PublishMerchantSync 發布商戶同步事件
func (k *KDSService) PublishMerchantSync(ctx context.Context, merchant *entity.Merchant) error {
	ctx, span := k.tracing.StartSpan(ctx, "KDSService.PublishMerchantSync")
	defer k.tracing.SpanEnd(span)

	// 構建事件數據
	syncEvent := event.IdentityMerchantSyncEvent{
		GlobalMerchantID: merchant.GetGlobalMerchantID(),
		ID:               merchant.GetID(),
		Name:             merchant.GetName(),
		DisplayName:      merchant.GetDisplayName(),
		APIKey:           merchant.GetAPIKey(),
		CreatedAt:        merchant.GetCreatedAt().Format(time.RFC3339),
		UpdatedAt:        merchant.GetUpdatedAt().Format(time.RFC3339),
	}

	if merchant.GetDeletedAt() != nil {
		syncEvent.DeletedAt = merchant.GetDeletedAt().Format(time.RFC3339)
	}

	// 構建CloudEvent
	eventID := uuid.New().String()
	cloudEvent := event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            k.config.Events.IdentityMerchantSync,
		Source:          "/fatidentitycat/FATCAT",
		Subject:         "merchant_sync",
		ID:              eventID,
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     k.tracing.GetTraceparent(ctx),
		Data:            syncEvent,
	}

	k.tracing.RecordSpanAttributes(span,
		attribute.String("outgoing.event.id", eventID),
		attribute.String("outgoing.event.type", cloudEvent.Type),
		attribute.String("merchant.global_id", merchant.GetGlobalMerchantID()),
	)

	// 發布事件
	if err := k.publishEvent(ctx, &cloudEvent); err != nil {
		k.tracing.RecordSpanError(span, err)
		return fmt.Errorf("publish agent sync: %w", err)
	}

	// 記錄事件發布成功
	k.tracing.TraceEvent(span, "Merchant sync event published successfully")

	k.logger.InfoWithContext(ctx, "Merchant sync event published",
		k.logger.String("global_merchant_id", merchant.GetGlobalMerchantID()),
		k.logger.String("event_id", cloudEvent.ID))
	return nil
}

// PublishPlayerSync 發布玩家同步事件
func (k *KDSService) PublishPlayerSync(
	ctx context.Context,
	player *entity.Player,
	globalMerchantID string,
) error {
	ctx, span := k.tracing.StartSpan(ctx, "KDSService.PublishPlayerSync")
	defer k.tracing.SpanEnd(span)

	// 構建事件數據
	syncEvent := event.IdentityPlayerSyncEvent{
		GlobalMerchantID: globalMerchantID,
		GlobalPlayerID:   player.GetGlobalPlayerID(),
		ID:               player.GetID(),
		MerchantID:       player.GetMerchantID(),
		APIKey:           player.GetAPIKey(),
		Account:          player.GetAccount(),
		Email:            player.GetEmail(),
		CreatedAt:        player.GetCreatedAt().Format(time.RFC3339),
		UpdatedAt:        player.GetUpdatedAt().Format(time.RFC3339),
		PlayerLevel: event.PlayerLevel{
			GlobalPlayerLevelID: player.GetPlayerLevel().GlobalPlayerLevelID,
			Name:                player.GetPlayerLevel().Name,
		},
	}

	if player.GetLastActiveAt() != nil && !player.GetLastActiveAt().IsZero() {
		syncEvent.LastActiveAt = player.GetLastActiveAt().Format(time.RFC3339)
	}

	if player.GetDeletedAt() != nil {
		syncEvent.DeletedAt = player.GetDeletedAt().Format(time.RFC3339)
	}

	// 構建CloudEvent
	eventID := uuid.New().String()
	cloudEvent := event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            k.config.Events.IdentityPlayerSync,
		Source:          "/fatidentitycat/FATCAT",
		Subject:         "player_sync",
		ID:              eventID,
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     k.tracing.GetTraceparent(ctx),
		Data:            syncEvent,
	}

	k.tracing.RecordSpanAttributes(span,
		attribute.String("outgoing.event.id", eventID),
		attribute.String("outgoing.event.type", cloudEvent.Type),
		attribute.String("player.global_id", player.GetGlobalPlayerID()),
	)

	// 發布事件
	if err := k.publishEvent(ctx, &cloudEvent); err != nil {
		k.tracing.RecordSpanError(span, err)
		return fmt.Errorf("publish agent sync: %w", err)
	}

	// 記錄事件發布成功
	k.tracing.TraceEvent(span, "Player sync event published successfully")

	k.logger.InfoWithContext(ctx, "Player sync event published",
		k.logger.String("global_player_id", player.GetGlobalPlayerID()),
		k.logger.String("event_id", cloudEvent.ID))
	return nil
}

// BatchPublishPlayerSync 批次發布玩家同步事件
func (k *KDSService) BatchPublishPlayerSync(
	ctx context.Context,
	players []*entity.Player,
	globalMerchantIDs []string,
) error {
	ctx, span := k.tracing.StartSpan(ctx, "KDSService.BatchPublishPlayerSync")
	defer k.tracing.SpanEnd(span)

	if len(players) == 0 {
		return nil
	}

	if len(players) != len(globalMerchantIDs) {
		return fmt.Errorf("players count (%d) does not match globalMerchantIDs count (%d)",
			len(players), len(globalMerchantIDs))
	}

	k.tracing.RecordSpanAttributes(span,
		attribute.Int("batch.size", len(players)),
		attribute.String("batch.operation", "player_sync"))

	k.logger.InfoWithContext(ctx, "Starting batch publish player sync events",
		k.logger.Int("batch_size", len(players)))

	// 構建批次事件
	events := make([]*event.CloudEvent, len(players))
	for i, player := range players {
		// 構建事件數據
		syncEvent := event.IdentityPlayerSyncEvent{
			GlobalMerchantID: globalMerchantIDs[i],
			GlobalPlayerID:   player.GetGlobalPlayerID(),
			ID:               player.GetID(),
			MerchantID:       player.GetMerchantID(),
			APIKey:           player.GetAPIKey(),
			Account:          player.GetAccount(),
			Email:            player.GetEmail(),
			CreatedAt:        player.GetCreatedAt().Format(time.RFC3339),
			UpdatedAt:        player.GetUpdatedAt().Format(time.RFC3339),
			PlayerLevel: event.PlayerLevel{
				GlobalPlayerLevelID: player.GetPlayerLevel().GlobalPlayerLevelID,
				Name:                player.GetPlayerLevel().Name,
			},
		}

		if player.GetLastActiveAt() != nil && !player.GetLastActiveAt().IsZero() {
			syncEvent.LastActiveAt = player.GetLastActiveAt().Format(time.RFC3339)
		}

		if player.GetDeletedAt() != nil {
			syncEvent.DeletedAt = player.GetDeletedAt().Format(time.RFC3339)
		}

		// 構建CloudEvent
		eventID := uuid.New().String()
		events[i] = &event.CloudEvent{
			SpecVersion:     "1.0",
			Type:            k.config.Events.IdentityPlayerSync,
			Source:          "/fatidentitycat/FATCAT",
			Subject:         "player_sync",
			ID:              eventID,
			Time:            time.Now(),
			DataContentType: "application/json",
			TraceParent:     k.tracing.GetTraceparent(ctx),
			Data:            syncEvent,
		}
	}

	// 批次發布事件
	if err := k.batchPublishEvents(ctx, events); err != nil {
		k.tracing.RecordSpanError(span, err)
		return fmt.Errorf("batch publish events: %w", err)
	}

	// 記錄批次發布成功
	k.tracing.TraceEvent(span, "Batch player sync events published successfully")
	k.logger.InfoWithContext(ctx, "Batch player sync events published successfully",
		k.logger.Int("batch_size", len(players)))

	return nil
}

// batchPublishEvents 批次發布事件到KDS
func (k *KDSService) batchPublishEvents(ctx context.Context, events []*event.CloudEvent) error {
	ctx, span := k.tracing.StartSpan(ctx, "KDSService.batchPublishEvents")
	defer k.tracing.SpanEnd(span)

	if len(events) == 0 {
		return nil
	}

	// 使用AWS Kinesis PutRecords API進行批次發送
	records := make([]types.PutRecordsRequestEntry, len(events))

	for i, cloudEvent := range events {
		// 序列化事件
		eventBytes, err := jsoniter.Marshal(cloudEvent)
		if err != nil {
			return fmt.Errorf("marshal event %d: %w", i, err)
		}

		// 生成分區鍵
		partitionKey := uuid.New().String()

		records[i] = types.PutRecordsRequestEntry{
			Data:         eventBytes,
			PartitionKey: aws.String(partitionKey),
		}
	}

	k.tracing.RecordSpanAttributes(span,
		attribute.Int("batch.records_count", len(records)),
		attribute.String("messaging.system", "kds"),
		attribute.String("messaging.operation", "batch_send"))

	// 執行批次發送
	response, err := k.client.PutRecords(ctx, &kinesis.PutRecordsInput{
		StreamName: aws.String(k.produceStream),
		Records:    records,
	})

	if err != nil {
		k.tracing.RecordSpanError(span, err)
		k.logger.ErrorWithContext(ctx, "Failed to put records to kinesis",
			k.logger.Int("records_count", len(records)),
			k.logger.Error("err", err))
		return fmt.Errorf("put records to kinesis: %w", err)
	}

	// 檢查失敗的記錄
	failedCount := 0
	if response.FailedRecordCount != nil {
		failedCount = int(*response.FailedRecordCount)
	}
	if failedCount > 0 {
		k.logger.ErrorWithContext(ctx, "Some records failed to publish",
			k.logger.Int("failed_count", failedCount),
			k.logger.Int("total_count", len(records)))

		// 記錄失敗的記錄詳情
		for i, record := range response.Records {
			if record.ErrorCode != nil {
				k.logger.ErrorWithContext(ctx, "Record failed",
					k.logger.Int("record_index", i),
					k.logger.String("error_code", *record.ErrorCode),
					k.logger.String("error_message", aws.ToString(record.ErrorMessage)))
			}
		}

		return fmt.Errorf("failed to publish %d out of %d records", failedCount, len(records))
	}

	k.tracing.TraceEvent(span, "All records published successfully")
	k.logger.InfoWithContext(ctx, "Batch events published to KDS successfully",
		k.logger.Int("records_count", len(records)),
		k.logger.String("stream_name", k.produceStream))

	return nil
}

// PublishManagerSync 發布管理員同步事件
func (k *KDSService) PublishManagerSync(ctx context.Context, event *event.CloudEvent) error {
	ctx, span := k.tracing.TraceWorkerToKDS(ctx, event.Type, event.ID)
	defer k.tracing.SpanEnd(span)

	k.logger.InfoWithContext(ctx, "Publishing manager sync event",
		k.logger.String("event_id", event.ID),
		k.logger.String("global_merchant_id", extractGlobalMerchantID(event)))
	return k.publishEvent(ctx, event)
}

// PublishPlayerLevelSync 發布玩家等級同步事件
func (k *KDSService) PublishPlayerLevelSync(ctx context.Context, event *event.CloudEvent) error {
	ctx, span := k.tracing.TraceWorkerToKDS(ctx, event.Type, event.ID)
	defer k.tracing.SpanEnd(span)

	k.logger.InfoWithContext(ctx, "Publishing player level sync event",
		k.logger.String("event_id", event.ID),
		k.logger.String("global_merchant_id", extractGlobalMerchantID(event)))
	return k.publishEvent(ctx, event)
}

// PublishPlayerTagsSync 發布玩家標籤同步事件
func (k *KDSService) PublishPlayerTagsSync(ctx context.Context, event *event.CloudEvent) error {
	ctx, span := k.tracing.TraceWorkerToKDS(ctx, event.Type, event.ID)
	defer k.tracing.SpanEnd(span)

	k.logger.InfoWithContext(ctx, "Publishing player tags sync event",
		k.logger.String("event_id", event.ID),
		k.logger.String("global_merchant_id", extractGlobalMerchantID(event)))
	return k.publishEvent(ctx, event)
}

// PublishTagSync 發布標籤同步事件
func (k *KDSService) PublishTagSync(ctx context.Context, event *event.CloudEvent) error {
	ctx, span := k.tracing.TraceWorkerToKDS(ctx, event.Type, event.ID)
	defer k.tracing.SpanEnd(span)

	k.logger.InfoWithContext(ctx, "Publishing tag sync event",
		k.logger.String("event_id", event.ID),
		k.logger.String("global_merchant_id", extractGlobalMerchantID(event)))
	return k.publishEvent(ctx, event)
}

// PublishAgentSync 從 Agent 實體發布同步事件
func (k *KDSService) PublishAgentSync(
	ctx context.Context,
	agent *entity.Agent,
	globalMerchantID string,
) error {
	ctx, span := k.tracing.StartSpan(ctx, "KDSService.PublishAgentSync")
	defer k.tracing.SpanEnd(span)

	// 記錄發布事件開始
	k.tracing.TraceEvent(span, "Preparing agent sync event for KDS")

	// 構建事件數據
	syncEvent := event.IdentityAgentSyncEvent{
		GlobalMerchantID: globalMerchantID,
		GlobalAgentID:    agent.GetGlobalAgentID(),
		ID:               agent.GetID(),
		MerchantID:       agent.GetMerchantID(),
		Account:          agent.GetAccount(),
		Ancestry:         agent.GetAncestry(),
		CreatedAt:        agent.GetCreatedAt().Format(time.RFC3339),
		UpdatedAt:        agent.GetUpdatedAt().Format(time.RFC3339),
	}

	// 處理可選字段
	if agent.GetCurrentSignInAt() != nil {
		signInAt := agent.GetCurrentSignInAt().Format(time.RFC3339)
		syncEvent.CurrentSignInAt = &signInAt
	}

	if agent.GetDeletedAt() != nil {
		deletedAt := agent.GetDeletedAt().Format(time.RFC3339)
		syncEvent.DeletedAt = deletedAt
	}

	// 構建CloudEvent
	eventID := uuid.New().String()
	cloudEvent := event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "tw.jvd.fatidentitycat.agent.sync.v1",
		Source:          "/fatidentitycat/FATCAT",
		Subject:         "agent_sync",
		ID:              eventID,
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     k.tracing.GetTraceparent(ctx),
		Data:            syncEvent,
	}

	k.tracing.RecordSpanAttributes(span,
		attribute.String("outgoing.event.id", eventID),
		attribute.String("outgoing.event.type", cloudEvent.Type),
		attribute.String("agent.global_id", agent.GetGlobalAgentID()),
	)

	// 發布事件
	if err := k.publishEvent(ctx, &cloudEvent); err != nil {
		k.tracing.RecordSpanError(span, err)
		return fmt.Errorf("publish agent sync: %w", err)
	}

	// 記錄事件發布成功
	k.tracing.TraceEvent(span, "Agent sync event published successfully")

	k.logger.InfoWithContext(ctx, "Agent sync event published",
		k.logger.String("global_agent_id", agent.GetGlobalAgentID()),
		k.logger.String("event_id", cloudEvent.ID))

	return nil
}

// 內部方法：發布事件到KDS
func (k *KDSService) publishEvent(ctx context.Context, event *event.CloudEvent) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	event.TraceParent = k.tracing.GetTraceparent(ctx)

	eventBytes, err := jsoniter.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	if err = k.Send(ctx, eventBytes, event.Type); err != nil {
		return fmt.Errorf("send event to KDS: %w", err)
	}

	k.logger.InfoWithContext(ctx, "Publishing event to KDS successfully",
		k.logger.String("event_id", event.ID),
		k.logger.String("event_type", event.Type),
		k.logger.Any("event", event))
	return nil
}

// Send 發送事件到KDS
func (k *KDSService) Send(ctx context.Context, data []byte, eventType string) error {
	ctx, span := k.tracing.StartSpan(ctx, "KDS.Send")
	defer k.tracing.SpanEnd(span)

	// 添加屬性到 span
	k.tracing.RecordSpanAttributes(span,
		attribute.String("messaging.system", "kds"),
		attribute.String("messaging.operation", "send"),
		attribute.String("messaging.event_type", eventType),
		attribute.Int("messaging.payload_size_bytes", len(data)),
	)

	// 嘗試在 JSON 載荷中添加 traceparent
	var jsonData map[string]interface{}
	if err := jsoniter.Unmarshal(data, &jsonData); err == nil {
		traceparent := k.tracing.GetTraceparent(ctx)
		if traceparent != "" {
			jsonData["traceparent"] = traceparent
			if newData, err := jsoniter.Marshal(jsonData); err == nil {
				data = newData
				k.tracing.RecordSpanAttributes(
					span,
					attribute.Bool("messaging.trace_propagated", true),
				)
			}
		}
	}

	// 生成隨機分區鍵
	partitionKey := uuid.New().String()

	// 記錄事件到 span
	k.tracing.TraceEvent(span, "Sending message to KDS",
		attribute.String("messaging.partition_key", partitionKey),
	)

	res, err := k.client.PutRecord(ctx, &kinesis.PutRecordInput{
		Data:         data,
		StreamName:   aws.String(k.produceStream),
		PartitionKey: aws.String(partitionKey),
	})
	if err != nil {
		k.logger.ErrorLog("Failed to put record to kinesis",
			k.logger.String("event_type", eventType),
			k.logger.Error("err", err))
		k.tracing.RecordSpanError(span, err)
		k.tracing.RecordSpanStatus(span, codes.Error, err.Error())
		return fmt.Errorf("put record to kinesis: %w", err)
	}

	// 記錄成功事件
	k.tracing.TraceEvent(span, "Message sent to KDS successfully")

	k.logger.InfoLog("Published event to KDS",
		k.logger.String("event_type", eventType),
		k.logger.String("partition_key", partitionKey),
		k.logger.Any("response", res))

	return nil
}

// SendToConsumeStream 發送事件到Consumer Stream
func (k *KDSService) SendToConsumeStream(ctx context.Context, data []byte, eventType string) error {
	ctx, span := k.tracing.StartSpan(ctx, "KDS.SendToConsumeStream")
	defer k.tracing.SpanEnd(span)

	// 添加屬性到 span
	k.tracing.RecordSpanAttributes(span,
		attribute.String("messaging.system", "kds"),
		attribute.String("messaging.operation", "send_to_consume_stream"),
		attribute.String("messaging.event_type", eventType),
		attribute.Int("messaging.payload_size_bytes", len(data)),
	)

	// 嘗試在 JSON 載荷中添加 traceparent
	var jsonData map[string]interface{}
	if err := jsoniter.Unmarshal(data, &jsonData); err == nil {
		traceparent := k.tracing.GetTraceparent(ctx)
		if traceparent != "" {
			jsonData["traceparent"] = traceparent
			if newData, err := jsoniter.Marshal(jsonData); err == nil {
				data = newData
				k.tracing.RecordSpanAttributes(
					span,
					attribute.Bool("messaging.trace_propagated", true),
				)
			}
		}
	}

	// 生成隨機分區鍵
	partitionKey := uuid.New().String()

	// 記錄事件到 span
	k.tracing.TraceEvent(span, "Sending message to Consumer Stream",
		attribute.String("messaging.partition_key", partitionKey),
		attribute.String("messaging.stream_name", k.consumeStream),
	)

	res, err := k.client.PutRecord(ctx, &kinesis.PutRecordInput{
		Data:         data,
		StreamName:   aws.String(k.consumeStream),
		PartitionKey: aws.String(partitionKey),
	})
	if err != nil {
		k.logger.ErrorLog("Failed to put record to consume stream",
			k.logger.String("event_type", eventType),
			k.logger.String("stream_name", k.consumeStream),
			k.logger.Error("err", err))
		k.tracing.RecordSpanError(span, err)
		k.tracing.RecordSpanStatus(span, codes.Error, err.Error())
		return fmt.Errorf("put record to consume stream: %w", err)
	}

	// 記錄成功事件
	k.tracing.TraceEvent(span, "Message sent to Consumer Stream successfully")

	k.logger.InfoLog("Published event to Consumer Stream",
		k.logger.String("event_type", eventType),
		k.logger.String("stream_name", k.consumeStream),
		k.logger.String("partition_key", partitionKey),
		k.logger.Any("response", res))

	return nil
}

func extractGlobalMerchantID(event *event.CloudEvent) string {
	if event == nil || event.Data == nil {
		return ""
	}

	// 將 data 轉換為 map
	dataBytes, _ := jsoniter.Marshal(event.Data)
	var dataMap map[string]interface{}
	if err := jsoniter.Unmarshal(dataBytes, &dataMap); err != nil {
		return ""
	}

	// 嘗試提取 global_merchant_id
	if globalID, ok := dataMap["global_merchant_id"].(string); ok {
		return globalID
	}

	return ""
}
