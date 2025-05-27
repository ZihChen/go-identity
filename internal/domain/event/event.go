package event

import (
	"time"
)

// CloudEvent 通用事件模型
type CloudEvent struct {
	SpecVersion     string      `json:"specversion"`
	Type            string      `json:"type"`
	Source          string      `json:"source"`
	Subject         string      `json:"subject"`
	ID              string      `json:"id"`
	Time            time.Time   `json:"time"`
	DataContentType string      `json:"datacontenttype"`
	TraceParent     string      `json:"traceparent"`
	Data            interface{} `json:"data"`
}

// MerchantSyncEvent 商戶同步事件數據
type MerchantSyncEvent struct {
	GlobalMerchantID string       `json:"global_merchant_id"`
	Merchant         MerchantData `json:"merchant"`
}

// MerchantData 從 KDS 接收的商戶數據
type MerchantData struct {
	ID               int    `json:"id"`
	Name             string `json:"name"`
	DisplayName      string `json:"display_name"`
	GlobalMerchantID string `json:"global_merchant_id"`
	// 其他字段省略，因為不需要處理
}

// PlayerSyncEvent 玩家同步事件數據
type PlayerSyncEvent struct {
	GlobalMerchantID string     `json:"global_merchant_id"`
	Player           PlayerData `json:"player"`
	PlayerLevel      LevelData  `json:"player_level,omitempty"`
	PlayerTags       []TagData  `json:"player_tags,omitempty"`
}

// PlayerData 從 KDS 接收的玩家數據
type PlayerData struct {
	GlobalPlayerID string `json:"global_player_id"`
	Account        string `json:"account"`
	Email          string `json:"email,omitempty"`
	Status         string `json:"status"`
}

// LevelData 玩家等級數據
type LevelData struct {
	GlobalPlayerLevelID string `json:"global_player_level_id"`
	Name                string `json:"name"`
}

// TagData 玩家標籤數據
type TagData struct {
	Tag struct {
		GlobalTagID string `json:"global_tag_id"`
		Name        string `json:"name"`
	} `json:"tag"`
}

// ManagerSyncEvent 管理員同步事件數據
type ManagerSyncEvent struct {
	GlobalMerchantID string      `json:"global_merchant_id"`
	Manager          ManagerData `json:"manager"`
}

// ManagerData 從 KDS 接收的管理員數據
type ManagerData struct {
	ID              int    `json:"id"`
	GlobalManagerID string `json:"global_manager_id"`
	Account         string `json:"account"`
	Email           string `json:"email"`
}

// IdentityMerchantSyncEvent 發送到 KDS 的商戶同步事件
type IdentityMerchantSyncEvent struct {
	GlobalMerchantID string `json:"global_merchant_id"`
	ID               uint64 `json:"id"`
	Name             string `json:"name"`
	DisplayName      string `json:"display_name"`
	APIKey           string `json:"api_key"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
	DeletedAt        string `json:"deleted_at,omitempty"`
}

// IdentityPlayerSyncEvent 發送到 KDS 的玩家同步事件
type IdentityPlayerSyncEvent struct {
	GlobalMerchantID string  `json:"global_merchant_id"`
	GlobalPlayerID   string  `json:"global_player_id"`
	ID               uint64  `json:"id"`
	MerchantID       uint64  `json:"merchant_id"`
	APIKey           string  `json:"api_key"`
	Account          string  `json:"account"`
	Email            *string `json:"email,omitempty"`
	LastActiveAt     string  `json:"last_active_at,omitempty"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
	DeletedAt        string  `json:"deleted_at,omitempty"`
}

// IdentityManagerSyncEvent 發送到 KDS 的管理員同步事件
type IdentityManagerSyncEvent struct {
	GlobalMerchantID string  `json:"global_merchant_id"`
	GlobalManagerID  string  `json:"global_manager_id"`
	ID               uint64  `json:"id"`
	MerchantID       uint64  `json:"merchant_id"`
	Account          string  `json:"account"`
	Email            *string `json:"email,omitempty"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
	DeletedAt        string  `json:"deleted_at,omitempty"`
}
