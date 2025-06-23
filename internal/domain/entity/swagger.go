package entity

// 這個文件包含 Swagger 文檔使用的模型定義和註解

// ErrorResponse 錯誤響應
// @Description 錯誤響應模型
type ErrorResponse struct {
	// 錯誤信息
	Error string `json:"error" example:"Invalid request parameters"`
}

// SuccessResponse 成功響應
// @Description 成功響應模型
type SuccessResponse struct {
	// 操作狀態
	Status string `json:"status" example:"success"`
}

// HealthResponse 健康檢查響應
// @Description 健康檢查響應模型
type HealthResponse struct {
	// 服務狀態
	Status string `json:"status" example:"ok"`
}

// 以下是 Swagger 文檔示例值
// @Description 商戶示例
func merchantExample() {
	// 用於 Swagger 文檔顯示的商戶示例
	_ = Merchant{
		ID:               1,
		GlobalMerchantID: "FATCAT-MERCHANT-1",
		Name:             "JV Diamond",
		DisplayName:      "JV Diamond",
		APIKey:           "api-key-example",
		// 其他字段...
	}
}

// @Description 玩家示例
func playerExample() {
	// 用於 Swagger 文檔顯示的玩家示例
	email := "player@example.com"
	_ = Player{
		ID:             1,
		MerchantID:     1,
		GlobalPlayerID: "FATCAT-PLAYER-7241",
		APIKey:         "api-key-example",
		Account:        "player1",
		Email:          &email,
		// 其他字段...
	}
}

// @Description 管理員示例
func managerExample() {
	// 用於 Swagger 文檔顯示的管理員示例
	email := "manager@example.com"
	_ = Manager{
		ID:              1,
		MerchantID:      1,
		GlobalManagerID: "FATCAT-MANAGER-231",
		Account:         "manager1",
		Email:           &email,
		// 其他字段...
	}
}
