package service

import (
	"context"
	"time"
)

// EventDeduplicationService 事件去重服務接口
type EventDeduplicationService interface {
	// IsEventProcessed 檢查事件是否已被處理
	IsEventProcessed(ctx context.Context, eventID string) (bool, error)

	// MarkEventProcessed 標記事件為已處理
	MarkEventProcessed(ctx context.Context, eventID string, ttl time.Duration) error
}
