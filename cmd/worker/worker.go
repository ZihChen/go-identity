package worker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/jvdiamondtech/ms-identity-cat/cmd"
	"github.com/jvdiamondtech/ms-identity-cat/internal/di"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/infraport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/cache/redis"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
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

type services struct {
	tracer       *tracing.Tracer
	db           *mysql.Database
	redisManager *redis.Manager
	components   *di.WorkerComponents
}

// runWorker 啟動Worker
func runWorker(cobraCmd *cobra.Command, args []string) {
	cfg := cmd.GetConfig()
	logger := cmd.GetLogger()

	// 主程序的Context
	rootCtx := context.Background()

	// 初始化所有服務
	s, err := initializeServices(rootCtx, cfg, logger)
	if err != nil {
		logger.FatalWithContext(
			rootCtx,
			"Failed to initialize services",
			logger.Error("error", err),
		)
	}
	defer s.cleanup(rootCtx, logger)

	mux := asynq.NewServeMux()

	// 記錄Worker啟動
	logger.InfoWithContext(rootCtx, "[Starting worker...",
		logger.String("redis_domain", cfg.Redis.Domain),
		logger.Int("redis_port", cfg.Redis.Port))

	s.components.Handler.RegisterHandlers(mux)
	logger.InfoWithContext(rootCtx, "Task handlers registered")

	go func() {
		if err := s.components.Server.Start(mux); err != nil {
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
		s.components.Server.Shutdown()
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

func initializeServices(
	ctx context.Context,
	cfg *config.Config,
	logger infraport.Logger,
) (*services, error) {
	// 初始化追踪器
	tracer, err := tracing.NewTracer(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tracer: %w", err)
	}
	logger.InfoWithContext(ctx, "Successfully initialized tracer!")

	// 初始化Redis連線
	redisManager := redis.NewRedisManager(cfg)
	if err = redisManager.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	logger.InfoWithContext(ctx, "Successfully initialized Redis connection!")

	// 初始化DB連線
	db, err := mysql.NewDatabase(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}
	logger.InfoWithContext(ctx, "Successfully initialized DB connection!")

	// 初始化Worker組件
	components, err := di.InitializeWorkerComponents(
		cfg,
		logger,
		redisManager,
		db.GetDBConnection(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize worker components: %w", err)
	}
	return &services{
		tracer:       tracer,
		db:           db,
		redisManager: redisManager,
		components:   components,
	}, nil
}

func (s *services) cleanup(ctx context.Context, logger infraport.Logger) {
	// 關閉Tracer
	if err := s.tracer.Shutdown(ctx); err != nil {
		logger.ErrorWithContext(ctx, "Failed to shutdown tracer", logger.Error("err", err))
	}
	// 關閉Redis連線
	if err := s.db.Close(); err != nil {
		logger.ErrorWithContext(
			ctx,
			"Failed to close database connection",
			logger.Error("err", err),
		)
	}
	// 關閉DB連線
	if err := s.redisManager.Close(); err != nil {
		logger.ErrorWithContext(ctx, "Failed to close Redis connection", logger.Error("err", err))
	}
	logger.InfoWithContext(ctx, "All connections closed successfully")
}
