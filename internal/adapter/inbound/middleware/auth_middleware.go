package middleware

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/utils"
)

type AuthConfig struct {
	APIKeys        map[string]string
	HeaderKey      string
	EncryptionType string // "base64", "hex", "aes-gcm", "url", "plain" (已棄用: "md5", "bcrypt"用於雜湊而非解密)
	// AES-GCM加密所需的金鑰 (32字節用於AES-256)
	AESKey []byte
}

// AuthMiddleware 認證中間件，驗證API Key並設置商戶上下文
func AuthMiddleware(config AuthConfig) gin.HandlerFunc {
	// 設定預設header key
	headerKey := config.HeaderKey
	if headerKey == "" {
		headerKey = "API-Key"
	}

	return func(c *gin.Context) {
		// 從header取得API Key
		encryptedKey := c.GetHeader(headerKey)
		if encryptedKey == "" {
			utils.Unauthorized(c, "Missing API key", "The request is missing the API key").Abort()
			return
		}

		// 解密API Key
		decryptedKey, err := decryptAPIKey(encryptedKey, config.EncryptionType, config.AESKey)
		if err != nil {
			utils.Unauthorized(c, "Invalid API key format", err.Error()).Abort()
			return
		}

		// 使用常數時間比較驗證API Key並取得對應的merchant ID
		merchantID, isValid := utils.ValidateAPIKeyConstantTime(decryptedKey, config.APIKeys)
		if !isValid {
			utils.Unauthorized(c, "Invalid API key", "The provided API key is invalid").Abort()
			return
		}

		c.Set("api_key", decryptedKey)
		c.Set("global_merchant_id", merchantID)
		c.Next()
	}
}

func decryptAPIKey(encryptedKey string, encryptionType string, aesKey []byte) (string, error) {
	switch strings.ToLower(encryptionType) {
	case "base64", "base64std":
		// 標準Base64解碼
		decoded, err := base64.StdEncoding.DecodeString(encryptedKey)
		if err != nil {
			return "", errors.New("invalid base64 encoding")
		}
		return string(decoded), nil

	case "base64url":
		// URL安全的Base64解碼 (RFC 4648)
		decoded, err := base64.URLEncoding.DecodeString(encryptedKey)
		if err != nil {
			return "", errors.New("invalid base64url encoding")
		}
		return string(decoded), nil

	case "hex", "hexadecimal":
		// 十六進制解碼
		decoded, err := hex.DecodeString(encryptedKey)
		if err != nil {
			return "", errors.New("invalid hexadecimal encoding")
		}
		return string(decoded), nil

	case "url", "urlencoded":
		// URL解碼
		decoded, err := url.QueryUnescape(encryptedKey)
		if err != nil {
			return "", errors.New("invalid URL encoding")
		}
		return decoded, nil

	case "aes-gcm", "aes256-gcm":
		// AES-GCM對稱加密解密
		if len(aesKey) == 0 {
			return "", errors.New("AES key is required for AES-GCM decryption")
		}
		return decryptAESGCM(encryptedKey, aesKey)

	case "plain", "plaintext", "":
		// 明文，直接返回
		return encryptedKey, nil

	case "md5":
		// MD5已棄用，不支援解密
		return "", errors.New("MD5 is deprecated and unsuitable for encryption/decryption")

	case "bcrypt":
		// bcrypt是雜湊函數，不支援解密
		return "", errors.New("bcrypt is a hash function, not suitable for decryption")

	default:
		return "", errors.New("unsupported encryption type: " + encryptionType)
	}
}

// decryptAESGCM 使用AES-GCM解密API Key
// 預期格式: base64(nonce + ciphertext + tag)
func decryptAESGCM(encryptedData string, key []byte) (string, error) {
	// 驗證金鑰長度
	if len(key) != 32 {
		return "", errors.New("AES key must be 32 bytes for AES-256")
	}

	// Base64解碼加密數據
	data, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", errors.New("invalid base64 encoded AES data")
	}

	// 檢查最小數據長度 (nonce: 12字節 + tag: 16字節 + 至少1字節密文)
	if len(data) < 29 {
		return "", errors.New("invalid AES-GCM data length")
	}

	// 創建AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	// 創建GCM模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("invalid nonce size")
	}

	// 分離nonce和密文+認證標籤
	nonce := data[:nonceSize]
	ciphertext := data[nonceSize:]

	// 解密
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", errors.New("AES-GCM decryption failed: " + err.Error())
	}

	return string(plaintext), nil
}

// EncryptAESGCM 輔助函數：使用AES-GCM加密API Key (供測試和工具使用)
// 返回格式: base64(nonce + ciphertext + tag)
func EncryptAESGCM(plaintext string, key []byte) (string, error) {
	if len(key) != 32 {
		return "", errors.New("AES key must be 32 bytes for AES-256")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 生成安全的隨機nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", errors.New("failed to generate secure nonce: " + err.Error())
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}
