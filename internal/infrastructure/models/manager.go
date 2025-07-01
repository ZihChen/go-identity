package models

import (
	"time"

	"gorm.io/gorm"
)

// Manager 管理員數據模型
type Manager struct {
	ID              uint64         `gorm:"primaryKey;autoIncrement"`
	MerchantID      uint64         `gorm:"index;not null"`
	GlobalManagerID string         `gorm:"uniqueIndex;size:100;not null"`
	Account         string         `gorm:"size:255;not null;index:idx_merchant_account,priority:2"`
	Email           *string        `gorm:"size:255"`
	CreatedAt       time.Time      `gorm:"type:datetime;default:CURRENT_TIMESTAMP"                             json:"created_at"`
	UpdatedAt       time.Time      `gorm:"type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

func (*Manager) TableName() string {
	return "managers"
}
