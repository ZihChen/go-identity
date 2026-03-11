package usecase

import (
	"context"
	"fmt"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/repository"
)

// FailedTaskEventUseCase 失敗任務事件用例
type FailedTaskEventUseCase struct {
	failedTaskEventRepo repository.FailedTaskEventRepository
	logger              infrastructure.Logger
	tracing             infrastructure.TracingService
}

// NewFailedTaskEventUseCase 創建失敗任務事件用例
func NewFailedTaskEventUseCase(
	failedTaskEventRepo repository.FailedTaskEventRepository,
	logger infrastructure.Logger,
	tracing infrastructure.TracingService,
) inbound.FailedTaskEventUseCase {
	return &FailedTaskEventUseCase{
		failedTaskEventRepo: failedTaskEventRepo,
		logger:              logger,
		tracing:             tracing,
	}
}

// CreateFailedTaskEventWithRedisInfo 記錄帶Redis信息的失敗任務事件
func (u *FailedTaskEventUseCase) CreateFailedTaskEventWithRedisInfo(
	ctx context.Context,
	taskID, taskType, queueName, payload, errorMessage, redisKey, redisState string,
	retryCount int,
) error {
	ctx, span := u.tracing.StartSpan(
		ctx,
		"FailedTaskEventUseCase.CreateFailedTaskEventWithRedisInfo",
	)
	defer u.tracing.SpanEnd(span)

	u.tracing.RecordSpanAttributes(span,
		entity.StringAttr("task.id", taskID),
		entity.StringAttr("task.type", taskType),
		entity.StringAttr("task.queue", queueName),
		entity.StringAttr("redis.key", redisKey),
		entity.StringAttr("redis.state", redisState),
		entity.IntAttr("task.retry_count", retryCount))

	// 創建失敗任務事件實體
	failedEvent := entity.NewFailedTaskEvent(
		taskID,
		taskType,
		queueName,
		payload,
		errorMessage,
		retryCount,
	)
	failedEvent.SetRedisKey(&redisKey)
	failedEvent.SetRedisState(&redisState)

	// 驗證實體
	if !failedEvent.IsValid() {
		u.tracing.RecordSpanError(span, fmt.Errorf("invalid failed task event"))
		u.logger.ErrorWithContext(ctx, "Invalid failed task event with Redis info",
			u.logger.String("task_id", taskID),
			u.logger.String("task_type", taskType),
			u.logger.String("redis_key", redisKey))
		return fmt.Errorf("invalid failed task event")
	}

	// 保存到資料庫
	if err := u.failedTaskEventRepo.Create(ctx, failedEvent); err != nil {
		u.tracing.RecordSpanError(span, err)
		u.logger.ErrorWithContext(ctx, "Failed to create failed task event with Redis info",
			u.logger.Error("err", err),
			u.logger.String("task_id", taskID),
			u.logger.String("task_type", taskType),
			u.logger.String("redis_key", redisKey))
		return fmt.Errorf("create failed task event with Redis info: %w", err)
	}

	u.logger.InfoWithContext(ctx, "Failed task event with Redis info recorded successfully",
		u.logger.String("task_id", taskID),
		u.logger.String("task_type", taskType),
		u.logger.String("queue_name", queueName),
		u.logger.String("redis_key", redisKey),
		u.logger.String("redis_state", redisState),
		u.logger.Int("retry_count", retryCount))

	u.tracing.TraceEvent(span, "Failed task event with Redis info created successfully")
	return nil
}
