package models

import (
	"gorm.io/gorm"
	"time"
)

// Player 玩家數據模型
type Player struct {
	ID             uint64         `gorm:"primaryKey;autoIncrement"`
	MerchantID     uint64         `gorm:"index;not null"`
	GlobalPlayerID string         `gorm:"uniqueIndex;size:100;not null"`
	APIKey         string         `gorm:"uniqueIndex;size:255;not null"`
	Account        string         `gorm:"size:255;not null;index:idx_merchant_account,priority:2"`
	Email          *string        `gorm:"size:255"`
	LastActiveAt   *time.Time     `gorm:"type:datetime" json:"last_active_at"`
	CreatedAt      time.Time      `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

// TableName 指定表名
func (*Player) TableName() string {
	return "players"
}
