package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
)

func TestCorsMiddleware_Enabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		App: config.AppConfig{
			Env: "development",
		},
		CORS: config.CORSConfig{
			Enabled:          true,
			AllowedOrigins:   []string{"http://localhost:3000"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Origin", "Content-Type", "Authorization", "API-Key"},
			ExposedHeaders:   []string{"Content-Type"},
			AllowCredentials: false,
			MaxAge:           12,
		},
	}

	router := gin.New()
	router.Use(NewCorsMiddleware(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	// Test preflight request
	req, _ := http.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "API-Key")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "http://localhost:3000", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
	allowedHeaders := w.Header().Get("Access-Control-Allow-Headers")
	// gin-contrib/cors normalizes header names, so API-Key becomes Api-Key
	assert.Contains(t, allowedHeaders, "Api-Key")
}

func TestCorsMiddleware_Disabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		CORS: config.CORSConfig{
			Enabled: false,
		},
	}

	router := gin.New()
	router.Use(NewCorsMiddleware(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// CORS headers should not be present when disabled
	assert.Empty(t, w.Header().Get("Access-Control-Allow-Origin"))
}

func TestCorsMiddleware_ProductionMode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		App: config.AppConfig{
			Env: "production",
		},
		CORS: config.CORSConfig{
			Enabled:        true,
			AllowedOrigins: []string{"https://example.com"},
			AllowedMethods: []string{"GET", "POST"},
			AllowedHeaders: []string{"Content-Type", "Authorization"},
		},
	}

	router := gin.New()
	router.Use(NewCorsMiddleware(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	// Test with allowed origin
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "https://example.com", w.Header().Get("Access-Control-Allow-Origin"))

	// Test with disallowed origin
	req2, _ := http.NewRequest("GET", "/test", nil)
	req2.Header.Set("Origin", "https://malicious.com")
	w2 := httptest.NewRecorder()

	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w.Code) // Request still succeeds
	// Origin should not be allowed
	assert.NotEqual(t, "https://malicious.com", w2.Header().Get("Access-Control-Allow-Origin"))
}

func TestCorsMiddleware_DevelopmentModeAllowAll(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		App: config.AppConfig{
			Env: "development",
		},
		CORS: config.CORSConfig{
			Enabled: true,
			// No specific origins configured - should allow all in development
		},
	}

	router := gin.New()
	router.Use(NewCorsMiddleware(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://any-origin.com")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// In development mode with no specific origins, should allow all
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestDefaultCorsConfig(t *testing.T) {
	cfg := defaultCorsConfig()

	assert.True(t, cfg.AllowAllOrigins)
	assert.Contains(t, cfg.AllowMethods, "GET")
	assert.Contains(t, cfg.AllowMethods, "POST")
	assert.Contains(t, cfg.AllowMethods, "PUT")
	assert.Contains(t, cfg.AllowMethods, "DELETE")
	assert.Contains(t, cfg.AllowMethods, "OPTIONS")
	assert.Contains(t, cfg.AllowHeaders, "API-Key")
	assert.Contains(t, cfg.AllowHeaders, "Authorization")
	assert.False(t, cfg.AllowCredentials)
	assert.Equal(t, 12, cfg.MaxAge)
}

func TestProductionCorsConfig(t *testing.T) {
	origins := []string{"https://example.com", "https://app.example.com"}
	cfg := ProductionCorsConfig(origins)

	assert.False(t, cfg.AllowAllOrigins)
	assert.Equal(t, origins, cfg.AllowOrigins)
	assert.Contains(t, cfg.AllowMethods, "GET")
	assert.Contains(t, cfg.AllowMethods, "POST")
	assert.NotContains(t, cfg.AllowMethods, "PATCH") // Production should be more restrictive
	assert.Contains(t, cfg.AllowHeaders, "API-Key")
	assert.True(t, cfg.AllowCredentials)
	assert.Equal(t, 6, cfg.MaxAge)
}
