package worker

import (
	"context"
	"errors"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/cache/redis"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/database/mysql"
	sLog "github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/logger"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jvdiamondtech/ms-identity-cat/cmd"
	"github.com/jvdiamondtech/ms-identity-cat/internal/di"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
	"github.com/spf13/cobra"
)

// Command 創建並返回worker
func Command() *cobra.Command {
	workerCmd := &cobra.Command{
		Use:   "worker",
		Short: "Start the worker service",
		Long:  `Start the worker service to process tasks from the queue`,
		Run:   runWorker,
	}

	return workerCmd
}

func init() {
	cmd.AddCommand(Command())
}

// runWorker 啟動Worker
func runWorker(cobraCmd *cobra.Command, args []string) {
	// 獲取配置和日誌
	cfg := cmd.GetConfig()
	logger := cmd.GetLogger()
	// 獲取ServiceLogger實例
	sLogger := sLog.NewServiceLogger(cfg)
	// 主程序的Context
	rootCtx := context.Background()
	// 初始化追踪器
	tracer, err := tracing.NewTracer(cfg)
	if err != nil {
		sLogger.FatalWithContext(rootCtx, "[Fatal][Worker][runWorker] Failed to initialize tracer", sLogger.Error("error", err))
	}
	defer tracer.Shutdown(context.Background())
	sLogger.InfoWithContext(rootCtx, "[Info][Worker][runWorker] Successfully initialized tracer!")

	// 初始化DB連線
	db, err := mysql.NewDatabase(cfg)
	if err != nil {
		sLogger.FatalWithContext(rootCtx, "[Fatal][Worker][runWebServer] Failed to initialize database", sLogger.Error("err", err))
	}
	defer func() {
		err = db.Close() // 主程序結束後關閉DB連線
		if err != nil {
			sLogger.ErrorLog("Failed to close database connection", sLogger.Error("err", err))
		}
		sLogger.InfoWithContext(rootCtx, "[Info][Worker][runWebServer] Database connection closed successfully")
	}()

	// 初始化Redis連線
	redisManager := redis.NewRedisManager(cfg)
	defer func() {
		err = redisManager.Close() // 主程序結束後關閉Redis連線
		if err != nil {
			sLogger.ErrorLog("Failed to close Redis connection", sLogger.Error("err", err))
		}
		sLogger.InfoWithContext(rootCtx, "[Info][Worker][runWorker] Redis connection closed successfully")
	}()
	if err = redisManager.Connect(rootCtx); err != nil {
		sLogger.FatalLog("Failed to connect to Redis after retry", sLogger.Error("err", err))
	}

	// 使用Wire初始化Worker組件
	components, err := di.InitializeWorkerComponents(cfg, logger, sLogger, redisManager, db.GetDBConnection())
	if err != nil {
		sLogger.FatalWithContext(rootCtx, "[Fatal][Worker][runWorker] Failed to initialize worker components", sLogger.Error("error", err))
		return
	}
	sLogger.InfoWithContext(rootCtx, "[Info][Worker][runWorker] Successfully initialized worker components!")

	mux := asynq.NewServeMux()

	// 記錄Worker啟動
	sLogger.InfoWithContext(rootCtx, "[Info][Worker][runWorker] Starting worker...",
		sLogger.String("redisDomain", cfg.Redis.Domain),
		sLogger.Int("redisPort", cfg.Redis.Port))

	components.Handler.RegisterHandlers(mux)
	sLogger.InfoWithContext(rootCtx, "[Info][Worker][runWorker] Task handlers registered")

	go func() {
		if err := components.Server.Start(mux); err != nil {
			if !errors.Is(err, asynq.ErrServerClosed) {
				sLogger.FatalWithContext(rootCtx, "[Fatal][Worker][runWorker] Failed to start worker server", sLogger.Error("error", err))
			}
		}
	}()

	// 等待中斷信號優雅地關閉服務器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	sLogger.InfoWithContext(rootCtx, "[Info][Worker][runWorker] Shutting down worker...")

	shutdownCtx, cancel := context.WithTimeout(rootCtx, 10*time.Second)
	defer cancel()

	// 關閉Worker
	done := make(chan struct{})
	go func() {
		components.Server.Shutdown()
		close(done)
	}()

	select {
	case <-done:
		sLogger.InfoWithContext(rootCtx, "[Info][Worker][runWorker] Worker service exited gracefully")
	case <-shutdownCtx.Done():
		sLogger.WarnWithContext(rootCtx, "[Warn][Worker][runWorker] Worker service forced to shutdown after 10 seconds",
			sLogger.Error("error", shutdownCtx.Err()))
	}

	// 記錄成功關閉
	sLogger.InfoWithContext(rootCtx, "[Info][Worker][runWorker] Worker exited")
}
