package worker

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
	"go.uber.org/zap"

	"github.com/jvdiamondtech/ms-identity-cat/cmd"
	"github.com/jvdiamondtech/ms-identity-cat/internal/di"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
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

	// 初始化追踪器
	tracer, err := tracing.NewTracer(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize tracer", zap.Error(err))
	}
	defer tracer.Shutdown(context.Background())

	ctx, rootSpan := tracing.StartSpan(context.Background(), "WorkerService")
	defer rootSpan.End()

	rootSpan.SetAttributes(
		attribute.String("service.name", cfg.App.Name),
		attribute.String("service.type", "worker"),
		attribute.String("service.environment", cfg.App.Env),
	)

	// 使用Wire初始化Worker組件
	components, err := di.InitializeWorkerComponents(cfg, logger)
	if err != nil {
		logger.Fatal("Failed to initialize worker components", zap.Error(err))
		rootSpan.RecordError(err)
		return
	}

	mux := asynq.NewServeMux()

	// 記錄Worker啟動
	logger.Info("Starting worker service",
		zap.String("redis", cfg.Redis.Domain),
		zap.Int("redis_port", cfg.Redis.Port))
	tracing.TraceEvent(rootSpan, "Starting worker service")

	components.Handler.RegisterHandlers(mux)
	logger.Info("Task handlers registered")

	go func() {
		if err := components.Server.Start(mux); err != nil {
			if err != asynq.ErrServerClosed {
				logger.Fatal("Failed to start worker server", zap.Error(err))
				rootSpan.RecordError(err)
			}
		}
	}()

	// 等待中斷信號優雅地關閉服務器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down worker...")
	// 記錄關閉事件
	tracing.TraceEvent(rootSpan, "Shutting down worker service")

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// 關閉Worker
	done := make(chan struct{})
	go func() {
		components.Server.Shutdown()
		close(done)
	}()

	select {
	case <-done:
		logger.Info("Worker service shutdown completed gracefully")
	case <-shutdownCtx.Done():
		logger.Warn("Worker service shutdown timeout - forcing exit")
	}

	// 記錄成功關閉
	tracing.TraceEvent(rootSpan, "Worker service exited gracefully")

	logger.Info("Worker exited")
}
