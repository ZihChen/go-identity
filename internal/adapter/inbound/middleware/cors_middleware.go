package middleware

import (
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
)

// CorsConfig CORS 中間件配置
type CorsConfig struct {
	AllowAllOrigins  bool     `json:"allow_all_origins"`
	AllowOrigins     []string `json:"allow_origins"`
	AllowMethods     []string `json:"allow_methods"`
	AllowHeaders     []string `json:"allow_headers"`
	ExposeHeaders    []string `json:"expose_headers"`
	AllowCredentials bool     `json:"allow_credentials"`
	MaxAge           int      `json:"max_age"` // in hours
}

// defaultCorsConfig 返回適合開發環境的預設 CORS 配置
func defaultCorsConfig() CorsConfig {
	return CorsConfig{
		AllowAllOrigins: true,
		AllowMethods: []string{
			"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD", "PATCH",
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Length",
			"Content-Type",
			"Authorization",
			"API-Key",
			"X-Requested-With",
			"Accept",
			"Cache-Control",
			"X-File-Name",
		},
		ExposeHeaders: []string{
			"Content-Length",
			"Content-Type",
		},
		AllowCredentials: false, // 開發環境禁用以避免 CORS 衝突
		MaxAge:           12,    // 12小時
	}
}

// ProductionCorsConfig 返回適合生產環境的嚴格 CORS 配置
func ProductionCorsConfig(allowedOrigins []string) CorsConfig {
	return CorsConfig{
		AllowAllOrigins: false,
		AllowOrigins:    allowedOrigins,
		AllowMethods: []string{
			"GET", "POST", "PUT", "DELETE", "OPTIONS",
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Authorization",
			"API-Key",
		},
		ExposeHeaders: []string{
			"Content-Type",
		},
		AllowCredentials: true,
		MaxAge:           6, // 6小時
	}
}

// Setup 創建 CORS 中間件
func Setup(config CorsConfig) gin.HandlerFunc {
	corsConfig := cors.DefaultConfig()

	// 配置 origins
	if config.AllowAllOrigins {
		corsConfig.AllowAllOrigins = true
	} else if len(config.AllowOrigins) > 0 {
		corsConfig.AllowOrigins = config.AllowOrigins
	}

	// 配置允許的方法
	if len(config.AllowMethods) > 0 {
		corsConfig.AllowMethods = config.AllowMethods
	}

	// 配置允許的 headers
	if len(config.AllowHeaders) > 0 {
		corsConfig.AllowHeaders = config.AllowHeaders
	}

	// 配置暴露的 headers
	if len(config.ExposeHeaders) > 0 {
		corsConfig.ExposeHeaders = config.ExposeHeaders
	}

	// 配置 credentials
	corsConfig.AllowCredentials = config.AllowCredentials

	// 配置 MaxAge (轉換小時為秒)
	if config.MaxAge > 0 {
		corsConfig.MaxAge = time.Duration(config.MaxAge) * time.Hour
	}

	return cors.New(corsConfig)
}

// CorsMiddleware 返回使用預設配置的 CORS 中間件 (舊版相容性)
func CorsMiddleware(cfg *config.Config) gin.HandlerFunc {
	if cfg != nil {
		return NewCorsMiddleware(cfg)
	}
	// 向後兼容：沒有參數時使用預設配置
	return Setup(defaultCorsConfig())
}

// NewCorsMiddleware 基於配置系統創建 CORS 中間件 (推薦使用)
func NewCorsMiddleware(cfg *config.Config) gin.HandlerFunc {
	if !cfg.CORS.Enabled {
		// 如果 CORS 被禁用，返回一個空的中間件
		return func(c *gin.Context) {
			c.Next()
		}
	}

	corsConfig := cors.DefaultConfig()

	// 根據環境和配置設定 Origins
	if cfg.App.Env == "production" {
		// 生產環境：嚴格白名單模式
		if len(cfg.CORS.AllowedOrigins) > 0 {
			corsConfig.AllowOrigins = cfg.CORS.AllowedOrigins
			corsConfig.AllowAllOrigins = false
		} else {
			// 生產環境沒有白名單時，設定一個無效的來源來禁止所有跨域請求
			// 使用特殊的不可能匹配的URL來確保沒有真實的來源會匹配
			corsConfig.AllowOrigins = []string{"https://production-cors-disabled.invalid"}
			corsConfig.AllowAllOrigins = false
		}
	} else {
		// 開發環境：可以使用 AllowAllOrigins 或白名單
		if len(cfg.CORS.AllowedOrigins) > 0 {
			corsConfig.AllowOrigins = cfg.CORS.AllowedOrigins
			corsConfig.AllowAllOrigins = false
		} else {
			// 開發環境允許所有來源 (向後相容)
			corsConfig.AllowAllOrigins = true
		}
	}

	// 設定允許的方法
	if len(cfg.CORS.AllowedMethods) > 0 {
		corsConfig.AllowMethods = cfg.CORS.AllowedMethods
	}

	// 設定允許的標頭
	if len(cfg.CORS.AllowedHeaders) > 0 {
		corsConfig.AllowHeaders = cfg.CORS.AllowedHeaders
	}

	// 設定暴露的標頭
	if len(cfg.CORS.ExposedHeaders) > 0 {
		corsConfig.ExposeHeaders = cfg.CORS.ExposedHeaders
	}

	// 設定憑證支援
	corsConfig.AllowCredentials = cfg.CORS.AllowCredentials

	// 設定快取時間
	if cfg.CORS.MaxAge > 0 {
		corsConfig.MaxAge = time.Duration(cfg.CORS.MaxAge) * time.Hour
	}

	return cors.New(corsConfig)
}

// parseCorsOrigins 解析 CORS origins 字符串
// 支持逗號分隔的字符串格式: "http://localhost:3000,https://example.com"
func parseCorsOrigins(origins string) []string {
	if origins == "" {
		return []string{}
	}

	// 分割並清理空白
	parts := strings.Split(origins, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
