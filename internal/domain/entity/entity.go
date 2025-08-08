package entity

import (
	"time"
)

// Merchant 商戶模型
type Merchant struct {
	ID               uint64     `json:"id"`
	GlobalMerchantID string     `json:"global_merchant_id"`
	Name             string     `json:"name"`
	DisplayName      string     `json:"display_name"`
	APIKey           string     `json:"api_key"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
	DeletedAt        *time.Time `json:"deleted_at,omitempty"`
}

// Player 玩家模型
type Player struct {
	ID             uint64      `json:"id"`
	MerchantID     uint64      `json:"merchant_id"`
	GlobalPlayerID string      `json:"global_player_id"`
	LevelID        uint64      `json:"level_id"`
	APIKey         string      `json:"api_key"`
	Account        string      `json:"account"`
	Email          *string     `json:"email,omitempty"`
	LastActiveAt   *time.Time  `json:"last_active_at,omitempty"`
	CreatedAt      time.Time   `json:"created_at"`
	UpdatedAt      time.Time   `json:"updated_at"`
	DeletedAt      *time.Time  `json:"deleted_at,omitempty"`
	PlayerLevel    PlayerLevel `json:"player_level,omitempty"`
}

type PlayerLevel struct {
	GlobalPlayerLevelID string `json:"global_player_level_id"`
	Name                string `json:"name"`
}

// Manager 管理員模型
type Manager struct {
	ID              uint64     `json:"id"`
	MerchantID      uint64     `json:"merchant_id"`
	GlobalManagerID string     `json:"global_manager_id"`
	Account         string     `json:"account"`
	Email           *string    `json:"email,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
}

type Tag struct {
	ID          uint64     `json:"id"`
	MerchantID  uint64     `json:"merchant_id"`
	Name        string     `json:"name"`
	GlobalTagID string     `json:"global_tag_id"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

type Level struct {
	ID                  uint64     `json:"id"`
	MerchantID          uint64     `json:"merchant_id"`
	Name                string     `json:"name"`
	GlobalPlayerLevelID string     `json:"global_player_level_id"`
	GlobalMerchantID    string     `json:"global_merchant_id"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	DeletedAt           *time.Time `json:"deleted_at,omitempty"`
}

type PlayerTag struct {
	PlayerID  uint64    `json:"player_id"`
	TagID     uint64    `json:"tag_id"`
	CreatedAt time.Time `json:"created_at"`
}

type LoggerFiled struct {
	Key   string
	Value interface{}
}
