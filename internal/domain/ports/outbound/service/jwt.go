package service

// JWTService JWT服務介面
type JWTService interface {
	GenerateToken(account, playerGlobalID string) (string, error)
	GenerateTokenWithMetadata(globalMerchantID string, metadata map[string]interface{}) (string, error)
}
