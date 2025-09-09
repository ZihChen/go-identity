package router

import (
	"github.com/gin-gonic/gin"
	"github.com/jvdiamondtech/ms-identity-cat/internal/adapter/inbound/handler/api"
)

// APIRouter 處理API路由註冊
type APIRouter struct {
	handler *api.HTTPHandler
}

// NewAPIRouter 創建API路由器
func NewAPIRouter(handler *api.HTTPHandler) *APIRouter {
	return &APIRouter{
		handler: handler,
	}
}

// RegisterRoutes 註冊API路由
func (r *APIRouter) RegisterRoutes(router *gin.Engine) {
	// API 路由群組
	api := router.Group("/api/v1")

	// 商戶相關路由
	merchants := api.Group("/merchants")
	{
		merchants.GET("/:id", r.handler.GetMerchantByID)
		merchants.GET("/global/:global_id", r.handler.GetMerchantByGlobalID)
	}

	// 玩家相關路由
	players := api.Group("/players")
	{
		players.GET("/:id", r.handler.GetPlayerByID)
		players.GET("/global/:global_id", r.handler.GetPlayerByGlobalID)
		players.PUT("/:id/active", r.handler.UpdatePlayerLastActive)
	}

	// 管理員相關路由
	managers := api.Group("/managers")
	{
		managers.GET("/:id", r.handler.GetManagerByID)
		managers.GET("/global/:global_id", r.handler.GetManagerByGlobalID)
	}
}
