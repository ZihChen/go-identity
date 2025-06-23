package entity

// 這個文件提供 Swagger 文檔所需的示例和註解

// 商戶示例
// @name MerchantExample
type SwaggerMerchantExample struct {
	ID               uint64  `json:"id" example:"1"`
	GlobalMerchantID string  `json:"global_merchant_id" example:"FATCAT-MERCHANT-1"`
	Name             string  `json:"name" example:"JV Diamond"`
	DisplayName      string  `json:"display_name" example:"JV Diamond"`
	APIKey           string  `json:"api_key" example:"api-key-example"`
	CreatedAt        string  `json:"created_at" example:"2025-01-01T00:00:00Z"`
	UpdatedAt        string  `json:"updated_at" example:"2025-01-01T00:00:00Z"`
	DeletedAt        *string `json:"deleted_at,omitempty" example:"2025-01-01T00:00:00Z"`
}

// 玩家示例
// @name PlayerExample
type SwaggerPlayerExample struct {
	ID             uint64  `json:"id" example:"1"`
	MerchantID     uint64  `json:"merchant_id" example:"1"`
	GlobalPlayerID string  `json:"global_player_id" example:"FATCAT-PLAYER-7241"`
	APIKey         string  `json:"api_key" example:"api-key-example"`
	Account        string  `json:"account" example:"player1"`
	Email          *string `json:"email,omitempty" example:"player@example.com"`
	LastActiveAt   *string `json:"last_active_at,omitempty" example:"2025-01-01T00:00:00Z"`
	CreatedAt      string  `json:"created_at" example:"2025-01-01T00:00:00Z"`
	UpdatedAt      string  `json:"updated_at" example:"2025-01-01T00:00:00Z"`
	DeletedAt      *string `json:"deleted_at,omitempty" example:"2025-01-01T00:00:00Z"`
}

// 管理員示例
// @name ManagerExample
type SwaggerManagerExample struct {
	ID              uint64  `json:"id" example:"1"`
	MerchantID      uint64  `json:"merchant_id" example:"1"`
	GlobalManagerID string  `json:"global_manager_id" example:"FATCAT-MANAGER-231"`
	Account         string  `json:"account" example:"manager1"`
	Email           *string `json:"email,omitempty" example:"manager@example.com"`
	CreatedAt       string  `json:"created_at" example:"2025-01-01T00:00:00Z"`
	UpdatedAt       string  `json:"updated_at" example:"2025-01-01T00:00:00Z"`
	DeletedAt       *string `json:"deleted_at,omitempty" example:"2025-01-01T00:00:00Z"`
}
