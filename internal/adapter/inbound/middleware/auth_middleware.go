package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware 認證中間件，驗證API Key並設置商戶上下文
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 從header取得API Key
		apiKey := c.GetHeader("API-Key")
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Missing API key",
				"message": "The request is missing the API key in header",
			})
			c.Abort()
			return
		}

		// 清理API Key
		apiKey = strings.TrimSpace(apiKey)
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Invalid API key format",
				"message": "The API key cannot be empty",
			})
			c.Abort()
			return
		}

		// 將商戶資訊設置到上下文中
		c.Set("api_key", apiKey)

		c.Next()
	}
}
