package kds

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// PublishMerchantSync 發布商戶同步事件
func (k *KDSService) PublishMerchantSync(ctx context.Context, event *event.CloudEvent) error {
	ctx, span := tracing.TraceWorkerToKDS(ctx, event.Type, event.ID)
	defer tracing.SpanEnd(span)

	k.logger.InfoWithContext(ctx, "Publishing merchant sync event",
		k.logger.String("event_id", event.ID),
		k.logger.String("global_merchant_id", extractGlobalMerchantID(event)))
	return k.publishEvent(ctx, event)
}

// PublishPlayerSync 發布玩家同步事件
func (k *KDSService) PublishPlayerSync(ctx context.Context, event *event.CloudEvent) error {
	ctx, span := tracing.TraceWorkerToKDS(ctx, event.Type, event.ID)
	defer tracing.SpanEnd(span)

	k.logger.InfoWithContext(ctx, "Publishing player sync event",
		k.logger.String("event_id", event.ID),
		k.logger.String("global_merchant_id", extractGlobalMerchantID(event)))
	return k.publishEvent(ctx, event)
}

// PublishManagerSync 發布管理員同步事件
func (k *KDSService) PublishManagerSync(ctx context.Context, event *event.CloudEvent) error {
	ctx, span := tracing.TraceWorkerToKDS(ctx, event.Type, event.ID)
	defer tracing.SpanEnd(span)

	k.logger.InfoWithContext(ctx, "Publishing manager sync event",
		k.logger.String("event_id", event.ID),
		k.logger.String("global_merchant_id", extractGlobalMerchantID(event)))
	return k.publishEvent(ctx, event)
}

// PublishPlayerLevelSync 發布玩家等級同步事件
func (k *KDSService) PublishPlayerLevelSync(ctx context.Context, event *event.CloudEvent) error {
	ctx, span := tracing.TraceWorkerToKDS(ctx, event.Type, event.ID)
	defer tracing.SpanEnd(span)

	k.logger.InfoWithContext(ctx, "Publishing player level sync event",
		k.logger.String("event_id", event.ID),
		k.logger.String("global_merchant_id", extractGlobalMerchantID(event)))
	return k.publishEvent(ctx, event)
}

// 內部方法：發布事件到KDS
func (k *KDSService) publishEvent(ctx context.Context, event *event.CloudEvent) error {
	if event == nil {
		return fmt.Errorf("event cannot be nil")
	}

	event.TraceParent = tracing.GetTraceparent(ctx)

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
	ctx, span := tracing.StartSpan(ctx, "KDS.Send")
	defer tracing.SpanEnd(span)

	// 添加屬性到 span
	tracing.RecordSpanAttributes(span,
		attribute.String("messaging.system", "kds"),
		attribute.String("messaging.operation", "send"),
		attribute.String("messaging.event_type", eventType),
		attribute.Int("messaging.payload_size_bytes", len(data)),
	)

	// 嘗試在 JSON 載荷中添加 traceparent
	var jsonData map[string]interface{}
	if err := jsoniter.Unmarshal(data, &jsonData); err == nil {
		traceparent := tracing.GetTraceparent(ctx)
		if traceparent != "" {
			jsonData["traceparent"] = traceparent
			if newData, err := jsoniter.Marshal(jsonData); err == nil {
				data = newData
				tracing.RecordSpanAttributes(
					span,
					attribute.Bool("messaging.trace_propagated", true),
				)
			}
		}
	}

	// 生成隨機分區鍵
	partitionKey := uuid.New().String()

	// 記錄事件到 span
	tracing.TraceEvent(span, "Sending message to KDS",
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
		tracing.RecordSpanError(span, err)
		tracing.RecordSpanStatus(span, codes.Error, err.Error())
		return fmt.Errorf("put record to kinesis: %w", err)
	}

	// 記錄成功事件
	tracing.TraceEvent(span, "Message sent to KDS successfully")

	k.logger.InfoLog("Published event to KDS",
		k.logger.String("event_type", eventType),
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
