package models

import (
	"time"
)

// FailedTaskEvent 失敗任務事件數據模型
type FailedTaskEvent struct {
	ID           uint      `gorm:"primaryKey;autoIncrement"                                            json:"id"`
	TaskID       string    `gorm:"size:255;not null;index"                                             json:"task_id"`
	TaskType     string    `gorm:"size:100;not null;index"                                             json:"task_type"`
	QueueName    string    `gorm:"size:100;not null;default:default;index"                             json:"queue_name"`
	Payload      string    `gorm:"type:longtext;not null"                                              json:"payload"`
	ErrorMessage string    `gorm:"type:text;not null"                                                  json:"error_message"`
	RetryCount   int       `gorm:"not null;default:0;index"                                            json:"retry_count"`
	FailedAt     time.Time `gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP;index"              json:"failed_at"`
	RedisKey     *string   `gorm:"size:500"                                                            json:"redis_key"`
	RedisState   *string   `gorm:"size:50;index"                                                       json:"redis_state"`
	CreatedAt    time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP;index"                       json:"created_at"`
	UpdatedAt    time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
}

func (*FailedTaskEvent) TableName() string {
	return "failed_task_events"
}
