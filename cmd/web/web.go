package web

import (
	"context"
	"fmt"
	"github.com/spf13/viper"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/jvdiamondtech/ms-identity-cat/cmd"
	"github.com/jvdiamondtech/ms-identity-cat/internal/adapter/middleware"
	"github.com/jvdiamondtech/ms-identity-cat/internal/di"
	asynclogger "github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/logger"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
)

var (
	port int
)

// Command 創建並返回web子命令
func Command() *cobra.Command {
	webCmd := &cobra.Command{
		Use:   "web",
		Short: "Start the web server",
		Long:  `Start the web server to handle HTTP API requests`,
		Run:   runWebServer,
	}

	webCmd.Flags().IntVarP(&port, "port", "p", 0, "server port (default is from config)")

	return webCmd
}

func init() {
	cmd.AddCommand(Command())
}

// runWebServer 啟動Web服務
func runWebServer(cobraCmd *cobra.Command, args []string) {
	cfg := cmd.GetConfig()
	logger := cmd.GetLogger()

	// 初始化追踪器
	tracer, err := tracing.NewTracer(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize tracer", zap.Error(err))
	}
	defer tracer.Shutdown(context.Background())

	asyncLogger := asynclogger.NewAsyncLogger(cfg, 1)
	defer asyncLogger.Close()

	ctx, rootSpan := tracing.StartSpan(context.Background(), "WebService")
	defer rootSpan.End()

	// 使用Wire初始化HTTP處理器
	httpHandler, err := di.InitializeWebServer(cfg, logger, asyncLogger) // 使用di包中的函數
	if err != nil {
		logger.Fatal("Failed to initialize web server", zap.Error(err), zap.Any("DB_HOST", viper.GetString("DB_HOST")), zap.Any("DB_USER", viper.GetString("DB_USER")), zap.Any("DB_PASSWORD", viper.GetString("DB_PASSWORD")))
	}
	// 使用命令行指定的端口或配置中的端口
	if port == 0 {
		port = cfg.App.Port
	}
	if port == 0 {
		port = 8080 // 默認端口
	}

	// 設置 Gin 模式
	if !cfg.App.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	// 創建 Gin 路由
	router := gin.Default()

	// 添加追踪中間件
	router.Use(middleware.TracingMiddleware())

	// 註冊路由
	httpHandler.RegisterRoutes(router)

	// 創建HTTP服務器
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}
	// 在後台運行服務器
	go func() {
		logger.Info("Starting web server", zap.Int("port", port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()
	// 等待中斷信號優雅地關閉服務器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// 記錄關閉事件
	tracing.TraceEvent(rootSpan, "Shutting down web server")

	// 創建上下文用於通知服務器關閉
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	// 記錄成功關閉
	tracing.TraceEvent(rootSpan, "Web server exited gracefully")

	logger.Info("Server exited")

}
