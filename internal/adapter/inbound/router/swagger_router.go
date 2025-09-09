package router

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SwaggerRouter 處理Swagger文檔路由註冊
type SwaggerRouter struct{}

// NewSwaggerRouter 創建Swagger路由器
func NewSwaggerRouter() *SwaggerRouter {
	return &SwaggerRouter{}
}

// RegisterRoutes 註冊Swagger路由
func (r *SwaggerRouter) RegisterRoutes(router *gin.Engine) {
	// Swagger 文檔路由
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}