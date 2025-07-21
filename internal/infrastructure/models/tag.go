package models

import (
	"time"

	"gorm.io/gorm"
)

// Tag 玩家標籤模型
type Tag struct {
	ID          uint64         `gorm:"primaryKey;autoIncrement"`
	MerchantID  uint64         `gorm:"index;not null"`
	Name        string         `gorm:"size:255;not null"`
	GlobalTagID string         `gorm:"uniqueIndex;size:100;not null"`
	CreatedAt   time.Time      `gorm:"type:datetime;default:CURRENT_TIMESTAMP"                             json:"created_at"`
	UpdatedAt   time.Time      `gorm:"type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (*Tag) TableName() string {
	return "tags"
}
