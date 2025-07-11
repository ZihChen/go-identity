package worker

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jvdiamondtech/ms-identity-cat/cmd"
	"github.com/jvdiamondtech/ms-identity-cat/internal/di"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/cache/redis"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/database/mysql"
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
	cfg := cmd.GetConfig()
	logger := cmd.GetLogger()
	// 主程序的Context
	rootCtx := context.Background()
	// 初始化追踪器
	tracer, err := tracing.NewTracer(cfg)
	if err != nil {
		logger.FatalWithContext(
			rootCtx,
			"Failed to initialize tracer",
			logger.Error("err", err),
		)
	}
	defer func() {
		err = tracer.Shutdown(context.Background())
		if err != nil {
			logger.ErrorWithContext(
				rootCtx,
				"Failed to shutdown tracer",
				logger.Error("err", err),
			)
		}
	}()
	logger.InfoWithContext(rootCtx, "Successfully initialized tracer!")

	// 初始化DB連線
	db, err := mysql.NewDatabase(cfg, logger)
	if err != nil {
		logger.FatalWithContext(
			rootCtx,
			"Failed to initialize database",
			logger.Error("err", err),
		)
	}
	defer func() {
		err = db.Close() // 主程序結束後關閉DB連線
		if err != nil {
			logger.ErrorLog("Failed to close database connection", logger.Error("err", err))
		}
		logger.InfoWithContext(rootCtx, "Database connection closed successfully")
	}()

	// 初始化Redis連線
	redisManager := redis.NewRedisManager(cfg)
	defer func() {
		err = redisManager.Close() // 主程序結束後關閉Redis連線
		if err != nil {
			logger.ErrorLog("Failed to close Redis connection", logger.Error("err", err))
		}
		logger.InfoWithContext(rootCtx, "Redis connection closed successfully")
	}()
	if err = redisManager.Connect(rootCtx); err != nil {
		logger.FatalLog("Failed to connect to Redis after retry", logger.Error("err", err))
	}

	// 使用Wire初始化Worker組件
	components, err := di.InitializeWorkerComponents(
		cfg,
		logger,
		redisManager,
		db.GetDBConnection(),
	)
	if err != nil {
		logger.FatalWithContext(
			rootCtx,
			"Failed to initialize worker components",
			logger.Error("err", err),
		)
		return
	}
	logger.InfoWithContext(rootCtx, "Successfully initialized worker components!")

	mux := asynq.NewServeMux()

	// 記錄Worker啟動
	logger.InfoWithContext(rootCtx, "[Starting worker...",
		logger.String("redis_domain", cfg.Redis.Domain),
		logger.Int("redis_port", cfg.Redis.Port))

	components.Handler.RegisterHandlers(mux)
	logger.InfoWithContext(rootCtx, "Task handlers registered")

	go func() {
		if err := components.Server.Start(mux); err != nil {
			if !errors.Is(err, asynq.ErrServerClosed) {
				logger.FatalWithContext(
					rootCtx,
					"Failed to start worker server",
					logger.Error("err", err),
				)
			}
		}
	}()

	// 等待中斷信號優雅地關閉服務器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.InfoWithContext(rootCtx, "Shutting down worker...")

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
		logger.InfoWithContext(rootCtx, "Worker service exited gracefully")
	case <-shutdownCtx.Done():
		logger.WarnWithContext(rootCtx, "Worker service forced to shutdown after 10 seconds",
			logger.Error("err", shutdownCtx.Err()))
	}

	// 記錄成功關閉
	logger.InfoWithContext(rootCtx, "Worker exited")
}
