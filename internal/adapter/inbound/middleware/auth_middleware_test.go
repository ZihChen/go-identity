package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware_PlaintextSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := AuthConfig{
		APIKeys: map[string]string{
			"test-api-key": "merchant-123",
		},
		HeaderKey:      "API-Key",
		EncryptionType: "plain",
	}

	router := gin.New()
	router.Use(AuthMiddleware(config))
	router.GET("/test", func(c *gin.Context) {
		apiKey, exists := c.Get("api_key")
		assert.True(t, exists)
		assert.Equal(t, "test-api-key", apiKey)

		merchantID, exists := c.Get("global_merchant_id")
		assert.True(t, exists)
		assert.Equal(t, "merchant-123", merchantID)

		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("API-Key", "test-api-key")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_MissingAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := AuthConfig{
		APIKeys: map[string]string{
			"test-api-key": "merchant-123",
		},
		EncryptionType: "plain",
	}

	router := gin.New()
	router.Use(AuthMiddleware(config))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Missing API key")
}

func TestAuthMiddleware_InvalidAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := AuthConfig{
		APIKeys: map[string]string{
			"valid-key": "merchant-123",
		},
		EncryptionType: "plain",
	}

	router := gin.New()
	router.Use(AuthMiddleware(config))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("API-Key", "invalid-key")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid API key")
}

func TestAuthMiddleware_Base64Encoding(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalKey := "test-api-key"
	encodedKey := base64.StdEncoding.EncodeToString([]byte(originalKey))

	config := AuthConfig{
		APIKeys: map[string]string{
			originalKey: "merchant-123",
		},
		EncryptionType: "base64",
	}

	router := gin.New()
	router.Use(AuthMiddleware(config))
	router.GET("/test", func(c *gin.Context) {
		apiKey, exists := c.Get("api_key")
		assert.True(t, exists)
		assert.Equal(t, originalKey, apiKey)

		merchantID, exists := c.Get("global_merchant_id")
		assert.True(t, exists)
		assert.Equal(t, "merchant-123", merchantID)

		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("API-Key", encodedKey)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_HexEncoding(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalKey := "test-api-key"
	encodedKey := hex.EncodeToString([]byte(originalKey))

	config := AuthConfig{
		APIKeys: map[string]string{
			originalKey: "merchant-456",
		},
		EncryptionType: "hex",
	}

	router := gin.New()
	router.Use(AuthMiddleware(config))
	router.GET("/test", func(c *gin.Context) {
		apiKey, exists := c.Get("api_key")
		assert.True(t, exists)
		assert.Equal(t, originalKey, apiKey)

		merchantID, exists := c.Get("global_merchant_id")
		assert.True(t, exists)
		assert.Equal(t, "merchant-456", merchantID)

		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("API-Key", encodedKey)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_URLEncoding(t *testing.T) {
	gin.SetMode(gin.TestMode)

	originalKey := "test api key with spaces"
	encodedKey := url.QueryEscape(originalKey)

	config := AuthConfig{
		APIKeys: map[string]string{
			originalKey: "merchant-789",
		},
		EncryptionType: "url",
	}

	router := gin.New()
	router.Use(AuthMiddleware(config))
	router.GET("/test", func(c *gin.Context) {
		apiKey, exists := c.Get("api_key")
		assert.True(t, exists)
		assert.Equal(t, originalKey, apiKey)

		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("API-Key", encodedKey)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_AESGCMEncryption(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 生成32字節AES密鑰
	aesKey := make([]byte, 32)
	_, err := rand.Read(aesKey)
	require.NoError(t, err)

	originalKey := "test-api-key"

	// 使用EncryptAESGCM加密
	encryptedKey, err := EncryptAESGCM(originalKey, aesKey)
	require.NoError(t, err)

	config := AuthConfig{
		APIKeys: map[string]string{
			originalKey: "merchant-aes",
		},
		EncryptionType: "aes-gcm",
		AESKey:         aesKey,
	}

	router := gin.New()
	router.Use(AuthMiddleware(config))
	router.GET("/test", func(c *gin.Context) {
		apiKey, exists := c.Get("api_key")
		assert.True(t, exists)
		assert.Equal(t, originalKey, apiKey)

		merchantID, exists := c.Get("global_merchant_id")
		assert.True(t, exists)
		assert.Equal(t, "merchant-aes", merchantID)

		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("API-Key", encryptedKey)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_CustomHeaderKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := AuthConfig{
		APIKeys: map[string]string{
			"custom-key": "merchant-custom",
		},
		HeaderKey:      "X-API-Token",
		EncryptionType: "plain",
	}

	router := gin.New()
	router.Use(AuthMiddleware(config))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-API-Token", "custom-key")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_InvalidBase64(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := AuthConfig{
		APIKeys: map[string]string{
			"test-key": "merchant-123",
		},
		EncryptionType: "base64",
	}

	router := gin.New()
	router.Use(AuthMiddleware(config))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("API-Key", "invalid-base64!!")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid API key format")
}

func TestAuthMiddleware_UnsupportedEncryption(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := AuthConfig{
		APIKeys: map[string]string{
			"test-key": "merchant-123",
		},
		EncryptionType: "unsupported",
	}

	router := gin.New()
	router.Use(AuthMiddleware(config))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("API-Key", "test-key")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "unsupported encryption type")
}

func TestEncryptDecryptAESGCM(t *testing.T) {
	// 生成32字節AES密鑰
	aesKey := make([]byte, 32)
	_, err := rand.Read(aesKey)
	require.NoError(t, err)

	originalText := "test-plaintext-key"

	// 加密
	encrypted, err := EncryptAESGCM(originalText, aesKey)
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)

	// 解密
	decrypted, err := decryptAESGCM(encrypted, aesKey)
	require.NoError(t, err)
	assert.Equal(t, originalText, decrypted)
}

func TestDecryptAESGCM_InvalidKey(t *testing.T) {
	invalidKey := make([]byte, 16) // 錯誤長度
	_, err := decryptAESGCM("test", invalidKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "AES key must be 32 bytes")
}

func TestDecryptAESGCM_InvalidData(t *testing.T) {
	aesKey := make([]byte, 32)
	_, err := rand.Read(aesKey)
	require.NoError(t, err)

	_, err = decryptAESGCM("invalid-base64", aesKey)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid base64")
}
