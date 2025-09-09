package router

import (
	"github.com/gin-gonic/gin"
	"github.com/jvdiamondtech/ms-identity-cat/internal/adapter/inbound/handler/api"
)

// Manager 路由管理器，協調所有路由器
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

// RegisterRoutes 註冊所有路由
func (rm *Manager) RegisterRoutes(router *gin.Engine) {
	// 註冊 API 路由
	rm.apiRouter.RegisterRoutes(router)

	// 註冊 Swagger 路由
	rm.swaggerRouter.RegisterRoutes(router)

	// 註冊健康檢查路由
	rm.healthRouter.RegisterRoutes(router)
}