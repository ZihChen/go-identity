package event

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

// IdentityPlayerLevelSyncEvent 發送到 KDS 的玩家等級同步事件
type IdentityPlayerLevelSyncEvent struct {
	GlobalMerchantID    string `json:"global_merchant_id"`
	GlobalPlayerLevelID string `json:"global_player_level_id"`
	Name                string `json:"name"`
	CreatedAt           string `json:"created_at"`
	UpdatedAt           string `json:"updated_at"`
	DeletedAt           string `json:"deleted_at,omitempty"`
}

type IdentityPlayerTagSyncEvent struct {
	GlobalMerchantID string `json:"global_merchant_id"`
	GlobalTagID      string `json:"global_tag_id"`
	Name             string `json:"name"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
	DeletedAt        string `json:"deleted_at,omitempty"`
}
