package kds

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// PublishMerchantSync 發布商戶同步事件
func (k *KDSService) PublishMerchantSync(ctx context.Context, event *event.CloudEvent) error {
	ctx, span := k.tracing.TraceWorkerToKDS(ctx, event.Type, event.ID)
	defer k.tracing.SpanEnd(span)

	k.logger.InfoWithContext(ctx, "Publishing merchant sync event",
		k.logger.String("event_id", event.ID),
		k.logger.String("global_merchant_id", extractGlobalMerchantID(event)))
	return k.publishEvent(ctx, event)
}

// PublishPlayerSync 發布玩家同步事件
func (k *KDSService) PublishPlayerSync(ctx context.Context, event *event.CloudEvent) error {
	ctx, span := k.tracing.TraceWorkerToKDS(ctx, event.Type, event.ID)
	defer k.tracing.SpanEnd(span)

	k.logger.InfoWithContext(ctx, "Publishing player sync event",
		k.logger.String("event_id", event.ID),
		k.logger.String("global_merchant_id", extractGlobalMerchantID(event)))
	return k.publishEvent(ctx, event)
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
func (k *KDSService) PublishAgentSync(ctx context.Context, agent *entity.Agent, globalMerchantID string) error {
	ctx, span := k.tracing.StartSpan(ctx, "KDSService.PublishAgentSyncFromEntity")
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
