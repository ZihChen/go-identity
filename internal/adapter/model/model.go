package model

import (
	"time"

	"gorm.io/gorm"
)

// Merchant 商戶數據模型
type Merchant struct {
	ID               uint64         `gorm:"primaryKey;autoIncrement"`
	GlobalMerchantID string         `gorm:"uniqueIndex;size:100;not null"`
	Name             string         `gorm:"uniqueIndex;size:255;not null"`
	DisplayName      string         `gorm:"size:255;not null"`
	APIKey           string         `gorm:"uniqueIndex;size:255;not null"`
	CreatedAt        time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt        time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP;onUpdate:CURRENT_TIMESTAMP"`
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

// TableName 指定表名
func (Merchant) TableName() string {
	return "merchants"
}

// Player 玩家數據模型
type Player struct {
	ID             uint64         `gorm:"primaryKey;autoIncrement"`
	MerchantID     uint64         `gorm:"index;not null"`
	GlobalPlayerID string         `gorm:"uniqueIndex;size:100;not null"`
	APIKey         string         `gorm:"uniqueIndex;size:255;not null"`
	Account        string         `gorm:"size:255;not null;index:idx_merchant_account,priority:2"`
	Email          *string        `gorm:"size:255"`
	LastActiveAt   *time.Time     `gorm:""`
	CreatedAt      time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt      time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP;onUpdate:CURRENT_TIMESTAMP"`
	DeletedAt      gorm.DeletedAt `gorm:"index"`

	Merchant Merchant `gorm:"foreignKey:MerchantID"`
}

// TableName 指定表名
func (Player) TableName() string {
	return "players"
}

// Manager 管理員數據模型
type Manager struct {
	ID              uint64         `gorm:"primaryKey;autoIncrement"`
	MerchantID      uint64         `gorm:"index;not null"`
	GlobalManagerID string         `gorm:"uniqueIndex;size:100;not null"`
	Account         string         `gorm:"size:255;not null;index:idx_merchant_account,priority:2"`
	Email           *string        `gorm:"size:255"`
	CreatedAt       time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP"`
	UpdatedAt       time.Time      `gorm:"not null;default:CURRENT_TIMESTAMP;onUpdate:CURRENT_TIMESTAMP"`
	DeletedAt       gorm.DeletedAt `gorm:"index"`

	Merchant Merchant `gorm:"foreignKey:MerchantID"`
}

// TableName 指定表名
func (Manager) TableName() string {
	return "managers"
}
