package redis

import (
	"context"
	"fmt"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/redis/go-redis/v9"
)

// TaskCleanupService 任務清理服務
type TaskCleanupService struct {
	manager *Manager
	logger  infrastructure.Logger
	tracing infrastructure.TracingService
}

// NewTaskCleanupService 創建任務清理服務
func NewTaskCleanupService(
	manager *Manager,
	logger infrastructure.Logger,
	tracing infrastructure.TracingService,
) *TaskCleanupService {
	return &TaskCleanupService{
		manager: manager,
		logger:  logger,
		tracing: tracing,
	}
}

// CleanupTaskData 清理單個失敗任務的Redis數據
func (c *TaskCleanupService) CleanupTaskData(ctx context.Context, taskID string) error {
	ctx, span := c.tracing.StartSpan(ctx, "TaskCleanupService.CleanupTaskData")
	defer c.tracing.SpanEnd(span)

	c.tracing.RecordSpanAttributes(span, entity.StringAttr("task.id", taskID))

	client, err := c.manager.GetClient()
	if err != nil {
		c.tracing.RecordSpanError(span, err)
		return fmt.Errorf("get redis client: %w", err)
	}

	// 需要清理的Redis key patterns（基於taskID）
	keysToDelete := []string{
		fmt.Sprintf("asynq:default:t:%s", taskID),        // 任務hash
		fmt.Sprintf("asynq:default:failed:%s", taskID),   // 失敗任務
		fmt.Sprintf("asynq:default:dead:%s", taskID),     // 死信任務
		fmt.Sprintf("asynq:default:archived:%s", taskID), // 歸檔任務
	}

	deletedCount := 0

	// 使用pipeline批量刪除
	pipeline := client.Pipeline()
	for _, key := range keysToDelete {
		pipeline.Del(ctx, key)
	}

	cmds, err := pipeline.Exec(ctx)
	if err != nil && err != redis.Nil {
		c.tracing.RecordSpanError(span, err)
		c.logger.ErrorWithContext(ctx, "Failed to delete Redis keys",
			c.logger.Error("err", err),
			c.logger.String("task_id", taskID))
		return fmt.Errorf("pipeline delete exec: %w", err)
	}

	// 統計實際刪除的key數量
	for _, cmd := range cmds {
		if delCmd, ok := cmd.(*redis.IntCmd); ok {
			if delResult, err := delCmd.Result(); err == nil && delResult > 0 {
				deletedCount++
			}
		}
	}

	c.logger.InfoWithContext(ctx, "Task Redis data cleaned up",
		c.logger.String("task_id", taskID),
		c.logger.Int("deleted_keys", deletedCount))

	c.tracing.TraceEvent(span, "Task cleanup completed successfully",
		entity.IntAttr("deleted_keys", deletedCount))

	return nil
}
