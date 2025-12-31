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
	GlobalMerchantID string      `json:"global_merchant_id"`
	GlobalPlayerID   string      `json:"global_player_id"`
	ID               uint64      `json:"id"`
	MerchantID       uint64      `json:"merchant_id"`
	APIKey           string      `json:"api_key"`
	Account          string      `json:"account"`
	Email            *string     `json:"email,omitempty"`
	LastActiveAt     string      `json:"last_active_at,omitempty"`
	CreatedAt        string      `json:"created_at"`
	UpdatedAt        string      `json:"updated_at"`
	DeletedAt        string      `json:"deleted_at,omitempty"`
	PlayerLevel      PlayerLevel `json:"player_level,omitempty"`
	// 新增：玩家標籤資料，統一在玩家同步事件中處理
	Tags []*IdentityTagDataSyncEvent `json:"tags,omitempty"`
}

type PlayerLevel struct {
	GlobalPlayerLevelID string `json:"global_player_level_id"`
	Name                string `json:"name"`
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
	GlobalMerchantID string                      `json:"global_merchant_id"`
	GlobalPlayerID   string                      `json:"global_player_id"`
	Tags             []*IdentityTagDataSyncEvent `json:"tags"`
}

type IdentityTagSyncEvent struct {
	GlobalMerchantID string                    `json:"global_merchant_id"`
	Tag              *IdentityTagDataSyncEvent `json:"tag"`
}

type IdentityTagDataSyncEvent struct {
	GlobalTagID string `json:"global_tag_id"`
	Name        string `json:"name"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
	DeletedAt   string `json:"deleted_at,omitempty"`
}

// IdentityAgentSyncEvent 發送到 KDS 的代理同步事件
type IdentityAgentSyncEvent struct {
	GlobalMerchantID string  `json:"global_merchant_id"`
	GlobalAgentID    string  `json:"global_agent_id"`
	ID               uint64  `json:"id"`
	MerchantID       uint64  `json:"merchant_id"`
	Account          string  `json:"account"`
	Ancestry         string  `json:"ancestry"`
	CurrentSignInAt  *string `json:"current_sign_in_at,omitempty"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
	DeletedAt        string  `json:"deleted_at,omitempty"`
}
