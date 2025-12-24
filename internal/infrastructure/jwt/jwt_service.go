package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// JWTService JWT服務介面
type JWTService interface {
	GenerateToken(account, playerGlobalID string) (string, error)
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
