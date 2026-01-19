package dto

// PlayerLoginRequest 玩家登入請求模型
type PlayerLoginRequest struct {
	GlobalMerchantID string                 `json:"global_merchant_id" binding:"required" example:"FATCAT-MERCHANT-001"`
	Metadata         map[string]interface{} `json:"metadata" binding:"required" example:"{\"player_id\":\"883\",\"username\":\"winston\"}"`
}

// PlayerLoginResponse 玩家登入響應模型
type PlayerLoginResponse struct {
	Success  bool   `json:"success"   example:"true"`
	JwtToken string `json:"jwt_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}
