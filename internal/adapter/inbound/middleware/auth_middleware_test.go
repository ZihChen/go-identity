package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware_Success(t *testing.T) {
	// 設置gin為測試模式
	gin.SetMode(gin.TestMode)

	// 創建測試路由
	router := gin.New()
	router.Use(AuthMiddleware())
	router.GET("/test", func(c *gin.Context) {
		// 驗證context中的API Key
		apiKey, exists := c.Get("api_key")
		assert.True(t, exists)
		assert.Equal(t, "test-api-key", apiKey)

		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	// 創建測試請求
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("API-Key", "test-api-key")
	w := httptest.NewRecorder()

	// 執行請求
	router.ServeHTTP(w, req)

	// 驗證結果
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAuthMiddleware_MissingAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(AuthMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Missing API key")
}

func TestAuthMiddleware_ValidAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(AuthMiddleware())
	router.GET("/test", func(c *gin.Context) {
		// 驗證API Key已正確設置到context
		apiKey, exists := c.Get("api_key")
		assert.True(t, exists)
		assert.Equal(t, "valid-key", apiKey)
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("API-Key", "valid-key")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "success")
}

func TestAuthMiddleware_EmptyAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(AuthMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("API-Key", "   ") // 空白字符
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid API key format")
}
