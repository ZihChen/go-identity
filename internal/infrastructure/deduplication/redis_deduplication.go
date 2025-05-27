package deduplication

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/service"
)

const (
	// 事件處理記錄的默認過期時間
	defaultEventTTL = 24 * time.Hour
	// 已處理事件的鍵前綴
	processedEventKeyPrefix = "event:processed:"
)

// RedisDeduplicationService 基於Redis的事件去重服務
type RedisDeduplicationService struct {
	client *redis.Client
	logger *zap.Logger
}

// NewRedisDeduplicationService 創建新的Redis去重服務
func NewRedisDeduplicationService(client *redis.Client, logger *zap.Logger) service.EventDeduplicationService {
	return &RedisDeduplicationService{
		client: client,
		logger: logger,
	}
}

// IsEventProcessed 檢查事件是否已被處理
func (s *RedisDeduplicationService) IsEventProcessed(ctx context.Context, eventID string) (bool, error) {
	if eventID == "" {
		return false, nil // 無法檢查沒有ID的事件
	}

	key := processedEventKeyPrefix + eventID
	exists, err := s.client.Exists(ctx, key).Result()
	if err != nil {
		s.logger.Warn("Error checking if event is processed",
			zap.String("event_id", eventID),
			zap.Error(err))
		return false, fmt.Errorf("check event processed status: %w", err)
	}

	return exists > 0, nil
}

// MarkEventProcessed 標記事件為已處理
func (s *RedisDeduplicationService) MarkEventProcessed(ctx context.Context, eventID string, ttl time.Duration) error {
	if eventID == "" {
		return nil // 無法標記沒有ID的事件
	}

	if ttl <= 0 {
		ttl = defaultEventTTL
	}

	key := processedEventKeyPrefix + eventID
	_, err := s.client.Set(ctx, key, time.Now().Unix(), ttl).Result()
	if err != nil {
		s.logger.Warn("Error marking event as processed",
			zap.String("event_id", eventID),
			zap.Error(err))
		return fmt.Errorf("mark event as processed: %w", err)
	}

	s.logger.Debug("Marked event as processed",
		zap.String("event_id", eventID),
		zap.Duration("ttl", ttl))

	return nil
}
