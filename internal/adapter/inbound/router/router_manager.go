package router

import (
	"github.com/gin-gonic/gin"
	"github.com/jvdiamondtech/ms-identity-cat/internal/adapter/inbound/handler/api"
	"github.com/jvdiamondtech/ms-identity-cat/internal/adapter/inbound/middleware"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
)

// Manager 路由管理器，協調所有路由器
// 提供多種配置方式：
// 1. RegisterRoutes - 僅註冊路由，需手動配置中間件
// 2. SetupRoutersWithMiddleware - 自動配置所有中間件並註冊基本路由
type Manager struct {
	apiRouter     *APIRouter
	swaggerRouter *SwaggerRouter
	healthRouter  *HealthRouter
}

// NewRouterManager 創建路由管理器
func NewRouterManager(handler *api.HTTPHandler) *Manager {
	return &Manager{
		apiRouter:     NewAPIRouter(handler),
		swaggerRouter: NewSwaggerRouter(),
		healthRouter:  NewHealthRouter(handler),
	}
}

// SetupRoutersWithMiddleware 配置所有中間件並註冊路由
func (rm *Manager) SetupRoutersWithMiddleware(
	router *gin.Engine,
	cfg *config.Config,
	handler *api.HTTPHandler,
) {
	// 加入全局Middleware
	router.Use(
		middleware.NewCorsMiddleware(cfg),
		middleware.NewTracingMiddleware(handler.GetTracingService()),
	)

	var authMiddleware gin.HandlerFunc
	if cfg.Auth.Enabled {
		authMiddleware = middleware.AuthMiddleware(middleware.AuthConfig{
			APIKeys:        cfg.Auth.APIKeys,
			HeaderKey:      cfg.Auth.HeaderKey,
			EncryptionType: cfg.Auth.EncryptionType,
		})
	}

	// 註冊所有路由
	rm.registerRoutes(router, authMiddleware)
}

// registerRoutes 註冊所有路由，每個路由器使用各自的中間件
func (rm *Manager) registerRoutes(
	router *gin.Engine,
	middleware gin.HandlerFunc,
) {
	// 註冊 API 路由
	rm.apiRouter.RegisterRoutes(router, middleware)

	// 註冊 Swagger 路由
	rm.swaggerRouter.RegisterRoutes(router)

	// 註冊健康檢查路由
	rm.healthRouter.RegisterRoutes(router)
}
