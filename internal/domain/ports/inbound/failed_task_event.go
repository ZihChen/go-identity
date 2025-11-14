package inbound

import (
	"context"
)

type FailedTaskEventUseCase interface {
	// CreateFailedTaskEventWithRedisInfo 記錄帶Redis信息的失敗任務事件
	CreateFailedTaskEventWithRedisInfo(ctx context.Context, taskID, taskType, queueName, payload, errorMessage, redisKey, redisState string, retryCount int) error
}