package router

import (
	"github.com/gin-gonic/gin"
	"github.com/jvdiamondtech/ms-identity-cat/internal/adapter/inbound/handler/api"
)

// HealthRouter 處理健康檢查路由註冊
type HealthRouter struct {
	handler *api.HTTPHandler
}

// NewHealthRouter 創建健康檢查路由器
func NewHealthRouter(handler *api.HTTPHandler) *HealthRouter {
	return &HealthRouter{
		handler: handler,
	}
}

// RegisterRoutes 註冊健康檢查路由
func (r *HealthRouter) RegisterRoutes(router *gin.Engine) {
	// 健康檢查路由
	router.GET("/health", r.handler.HealthCheck)
}