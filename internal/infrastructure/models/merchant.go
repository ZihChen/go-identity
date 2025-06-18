package models

import (
	"gorm.io/gorm"
	"time"
)

// Merchant 商戶數據模型
type Merchant struct {
	ID               uint64         `gorm:"primaryKey;autoIncrement"`
	GlobalMerchantID string         `gorm:"uniqueIndex;size:100;not null"`
	Name             string         `gorm:"uniqueIndex;size:255;not null"`
	DisplayName      string         `gorm:"size:255;not null"`
	APIKey           string         `gorm:"uniqueIndex;size:255;not null"`
	CreatedAt        time.Time      `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

func (*Merchant) TableName() string {
	return "merchants"
}
