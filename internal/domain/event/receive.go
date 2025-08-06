package event

import "time"

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
	ID               int       `json:"id"`
	Name             string    `json:"name"`
	DisplayName      string    `json:"display_name"`
	GlobalMerchantID string    `json:"global_merchant_id"`
	UpdatedAt        time.Time `json:"updated_at"`
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
	GlobalPlayerID      string    `json:"global_player_id"`
	Account             string    `json:"account"`
	Email               string    `json:"email,omitempty"`
	Status              string    `json:"status"`
	LevelID             uint64    `json:"level_id"`
	GlobalPlayerLevelID string    `json:"global_player_level_id"`
	DeletedAt           string    `json:"deleted_at,omitempty"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// LevelData 玩家等級數據
type LevelData struct {
	GlobalPlayerLevelID string `json:"global_player_level_id"`
	Name                string `json:"name"`
}

// TagData 玩家標籤數據
type TagData struct {
	Tag struct {
		GlobalTagID string    `json:"global_tag_id"`
		Name        string    `json:"name"`
		UpdatedAt   time.Time `json:"updated_at"`
	} `json:"tag"`
}

// ManagerSyncEvent 管理員同步事件數據
type ManagerSyncEvent struct {
	GlobalMerchantID string      `json:"global_merchant_id"`
	Manager          ManagerData `json:"manager"`
}

// ManagerData 從 KDS 接收的管理員數據
type ManagerData struct {
	ID              int       `json:"id"`
	GlobalManagerID string    `json:"global_manager_id"`
	Account         string    `json:"account"`
	Email           string    `json:"email"`
	UpdatedAt       time.Time `json:"updated_at"`
	DeletedAt       string    `json:"deleted_at,omitempty"`
}

type TagSyncEvent struct {
	GlobalMerchantID string `json:"global_merchant_id"`
	Tag              struct {
		Description string    `json:"description"`
		GlobalTagID string    `json:"global_tag_id"`
		IsOpen      bool      `json:"is_open"`
		Name        string    `json:"name"`
		TagUserType string    `json:"tag_user_type"`
		UpdatedAt   time.Time `json:"updated_at"`
		CreatedAt   time.Time `json:"created_at"`
	} `json:"tag"`
}

type LevelSyncEvent struct {
	GlobalMerchantID string `json:"global_merchant_id"`
	PlayerLevel      struct {
		Name                string    `json:"name"`
		GlobalPlayerLevelID string    `json:"global_player_level_id"`
		CreatedAt           time.Time `json:"created_at"`
		UpdatedAt           time.Time `json:"updated_at"`
	} `json:"player_level"`
}
