package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// JWTService JWT服務介面
type JWTService interface {
	GenerateToken(account, playerGlobalID string) (string, error)
	GenerateTokenWithMetadata(globalMerchantID string, metadata map[string]interface{}) (string, error)
}

// jwtService JWT服務實現
type jwtService struct {
	secret string
	issuer string
}

// NewJWTService 創建新的JWT服務
func NewJWTService(secret, issuer string) JWTService {
	return &jwtService{
		secret: secret,
		issuer: issuer,
	}
}

// CustomClaims JWT自定義claims
type CustomClaims struct {
	Account        string `json:"account"`
	PlayerGlobalID string `json:"player_global_id"`
	jwt.RegisteredClaims
}

// MetadataClaims JWT metadata claims
type MetadataClaims struct {
	GlobalMerchantID string                 `json:"global_merchant_id"`
	Metadata         map[string]interface{} `json:"metadata"`
	jwt.RegisteredClaims
}

// GenerateToken 生成JWT token
func (j *jwtService) GenerateToken(account, playerGlobalID string) (string, error) {
	if account == "" {
		return "", errors.New("account cannot be empty")
	}
	if playerGlobalID == "" {
		return "", errors.New("player_global_id cannot be empty")
	}

	now := time.Now()
	// 設置過期時間為1小時
	expirationTime := now.Add(1 * time.Hour)

	claims := CustomClaims{
		Account:        account,
		PlayerGlobalID: playerGlobalID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Subject:   "player",
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(j.secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// GenerateTokenWithMetadata 使用 metadata 生成JWT token
func (j *jwtService) GenerateTokenWithMetadata(globalMerchantID string, metadata map[string]interface{}) (string, error) {
	if globalMerchantID == "" {
		return "", errors.New("global_merchant_id cannot be empty")
	}
	if metadata == nil || len(metadata) == 0 {
		return "", errors.New("metadata cannot be empty")
	}

	now := time.Now()
	// 設置過期時間為1小時
	expirationTime := now.Add(1 * time.Hour)

	claims := MetadataClaims{
		GlobalMerchantID: globalMerchantID,
		Metadata:         metadata,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    j.issuer,
			Subject:   "player",
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(j.secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
