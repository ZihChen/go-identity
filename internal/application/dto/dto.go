package dto

// PlayerLoginRequest 玩家登入請求模型
type PlayerLoginRequest struct {
	PlayerGlobalID string `json:"player_global_id" binding:"required" example:"FATCAT-PLAYER-883"`
	Account        string `json:"account"          binding:"required" example:"winston883"`
}

// PlayerLoginResponse 玩家登入響應模型
type PlayerLoginResponse struct {
	Success  bool   `json:"success"   example:"true"`
	JwtToken string `json:"jwt_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}
