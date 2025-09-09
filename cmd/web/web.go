package web

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jvdiamondtech/ms-identity-cat/cmd"
	"github.com/jvdiamondtech/ms-identity-cat/internal/adapter/handler/api"
	"github.com/jvdiamondtech/ms-identity-cat/internal/adapter/middleware"
	"github.com/jvdiamondtech/ms-identity-cat/internal/di"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/cache/redis"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/database/mysql"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
	"github.com/spf13/cobra"
)

const (
	defaultPort     = 8080
	shutdownTimeout = 5 * time.Second
)

var (
	port int
)

// services 包含所有需要清理的服務
type services struct {
	tracer       *tracing.Tracer
	db           *mysql.Database
	redisManager *redis.Manager
	httpHandler  *api.HTTPHandler
}

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
func runWebServer(_ *cobra.Command, _ []string) {
	cfg := cmd.GetConfig()
	logger := cmd.GetLogger()

	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	// 初始化所有服務
	svc, err := initializeServices(rootCtx, cfg, logger)
	if err != nil {
		logger.FatalWithContext(
			rootCtx,
			"Failed to initialize services",
			logger.Error("err", err),
		)
	}
	defer svc.cleanup(rootCtx, logger)

	// 創建追蹤 span
	ctx, rootSpan := tracing.StartSpan(rootCtx, "WebService")
	defer tracing.SpanEnd(rootSpan)

	// 決定服務端口
	serverPort := determinePort(port, cfg)

	// 設置 Gin 模式
	if !cfg.App.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	// 創建 Gin 路由
	router := gin.Default()

	// 加入Middleware
	router.Use(middleware.TracingMiddleware())

	// 註冊路由
	svc.httpHandler.RegisterRoutes(router)

	// 創建HTTP服務器
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", serverPort),
		Handler:      router,
		ReadTimeout:  30 * time.Second,  // 讀取請求的超時時間
		WriteTimeout: 30 * time.Second,  // 寫入響應的超時時間
		IdleTimeout:  120 * time.Second, // 空閒連接的超時時間
	}
	server.SetKeepAlivesEnabled(true)

	// 在後台運行服務器
	go func() {
		logger.InfoWithContext(
			ctx,
			"Starting web server",
			logger.Int("port", serverPort),
		)
		if serverErr := server.ListenAndServe(); serverErr != nil &&
			!errors.Is(serverErr, http.ErrServerClosed) {
			logger.FatalWithContext(
				ctx,
				"Failed to start server",
				logger.Error("err", serverErr),
			)
		}
	}()

	// 等待中斷信號優雅地關閉服務器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.InfoWithContext(ctx, "Shutting down server...")
	tracing.TraceEvent(rootSpan, "Shutting down web server")

	// 創建帶超時的上下文用於優雅關閉
	shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	// 關閉HTTP服務器，停止接受新請求，並等待現有請求完成
	if err = server.Shutdown(shutdownCtx); err != nil {
		logger.ErrorWithContext(
			ctx,
			"Failed to gracefully shutdown server",
			logger.Error("err", err),
		)
	} else {
		logger.InfoWithContext(ctx, "HTTP server shutdown gracefully")
	}

	// 額外檢查是否還有活動連接
	select {
	case <-shutdownCtx.Done():
		if errors.Is(shutdownCtx.Err(), context.DeadlineExceeded) {
			logger.WarnWithContext(ctx, "Server shutdown timeout exceeded, forcing exit")
		}
	default:
		logger.InfoWithContext(ctx, "All connections closed gracefully")
	}

	tracing.TraceEvent(rootSpan, "Web server exited gracefully")
	logger.InfoWithContext(ctx, "Server exited")
}

// initializeServices 初始化所有必要的服務
func initializeServices(
	ctx context.Context,
	cfg *config.Config,
	logger infrastructure.Logger,
) (*services, error) {
	// 初始化追蹤器
	tracer, err := tracing.NewTracer(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracer: %w", err)
	}
	logger.InfoWithContext(ctx, "Successfully initialized web tracer!")

	// 初始化DB連線
	db, err := mysql.NewDatabase(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	logger.InfoWithContext(ctx, "Successfully initialized database connection!")

	// 初始化Redis連線
	redisManager := redis.NewRedisManager(cfg)
	if err = redisManager.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	logger.InfoWithContext(ctx, "Successfully initialized Redis connection!")

	// 使用Wire初始化HTTP處理器
	httpHandler, err := di.InitializeWebServer(
		cfg,
		logger,
		redisManager,
		db.GetDBConnection(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize web server: %w", err)
	}
	logger.InfoWithContext(ctx, "Successfully initialized web server!")

	return &services{
		tracer:       tracer,
		db:           db,
		redisManager: redisManager,
		httpHandler:  httpHandler,
	}, nil
}

// cleanup 清理所有服務資源
func (s *services) cleanup(ctx context.Context, logger infrastructure.Logger) {
	// 關閉追蹤器
	if err := s.tracer.Shutdown(ctx); err != nil {
		logger.ErrorWithContext(
			ctx,
			"Failed to shutdown tracer",
			logger.Error("err", err),
		)
	}

	// 關閉資料庫連線
	if err := s.db.Close(); err != nil {
		logger.ErrorWithContext(
			ctx,
			"Failed to close database connection",
			logger.Error("err", err),
		)
	} else {
		logger.InfoWithContext(ctx, "Database connection closed successfully")
	}

	// 關閉Redis連線
	if err := s.redisManager.Close(); err != nil {
		logger.ErrorWithContext(
			ctx,
			"Failed to close Redis connection",
			logger.Error("err", err),
		)
	} else {
		logger.InfoWithContext(ctx, "Redis connection closed successfully")
	}
}

// determinePort 決定使用的端口
func determinePort(cmdPort int, cfg *config.Config) int {
	// 優先使用命令行參數
	if cmdPort != 0 {
		return cmdPort
	}
	// 其次使用配置文件
	if cfg.App.Port != 0 {
		return cfg.App.Port
	}
	// 最後使用默認值
	return defaultPort
}
