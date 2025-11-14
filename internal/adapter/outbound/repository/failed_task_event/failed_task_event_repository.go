package repository

import (
	"context"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/models"
	"gorm.io/gorm"
)

// failedTaskEventRepository GORM 實現的失敗任務事件資料庫
type failedTaskEventRepository struct {
	db *gorm.DB
}

// NewFailedTaskEventRepository 創建失敗任務事件資料庫
func NewFailedTaskEventRepository(db *gorm.DB) repository.FailedTaskEventRepository {
	return &failedTaskEventRepository{db: db}
}

// Create 創建失敗任務事件記錄
func (r *failedTaskEventRepository) Create(ctx context.Context, failedEvent *entity.FailedTaskEvent) error {
	if !failedEvent.IsValid() {
		return errmsg.ErrInvalidEntity
	}

	eventModel := mapToDBFailedTaskEvent(failedEvent)
	result := r.db.WithContext(ctx).Create(eventModel)
	if result.Error != nil {
		return result.Error
	}

	failedEvent.SetID(eventModel.ID)
	return nil
}

// mapToDBFailedTaskEvent - 將領域實體映射到資料庫模型
func mapToDBFailedTaskEvent(event *entity.FailedTaskEvent) *models.FailedTaskEvent {
	return &models.FailedTaskEvent{
		ID:           event.GetID(),
		TaskID:       event.GetTaskID(),
		TaskType:     event.GetTaskType(),
		QueueName:    event.GetQueueName(),
		Payload:      event.GetPayload(),
		ErrorMessage: event.GetErrorMessage(),
		RetryCount:   event.GetRetryCount(),
		FailedAt:     event.GetFailedAt(),
		RedisKey:     event.GetRedisKey(),
		RedisState:   event.GetRedisState(),
		CreatedAt:    event.GetCreatedAt(),
		UpdatedAt:    event.GetUpdatedAt(),
	}
}