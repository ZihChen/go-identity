package worker

import (
	"context"
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
	"go.opentelemetry.io/otel/attribute"
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

	// 初始化追踪器
	tracer, err := tracing.NewTracer(cfg)
	if err != nil {
		sLogger.FatalLog("[RunWorker]Failed to initialize worker tracer", sLogger.Error("err", err))
	}
	defer tracer.Shutdown(context.Background())
	sLogger.InfoLog("[RunWorker]Successfully initialized worker tracer!")

	ctx, rootSpan := tracing.StartSpan(context.Background(), "WorkerService")
	defer rootSpan.End()

	rootSpan.SetAttributes(
		attribute.String("service.name", cfg.App.Name),
		attribute.String("service.type", "worker"),
		attribute.String("service.environment", cfg.App.Env),
	)

	// 使用Wire初始化Worker組件
	components, err := di.InitializeWorkerComponents(cfg, logger, sLogger)
	if err != nil {
		sLogger.FatalLog("[RunWorker]Failed to initialize worker components", sLogger.Error("err", err))
		rootSpan.RecordError(err)
		return
	}
	sLogger.InfoLog("[RunWorker]Successfully initialized worker components!")

	mux := asynq.NewServeMux()

	// 記錄Worker啟動
	sLogger.InfoLog("[RunWorker]Starting worker...", sLogger.String("redis_domain", cfg.Redis.Domain), sLogger.Int("redis_port", cfg.Redis.Port))
	tracing.TraceEvent(rootSpan, "Starting worker service")

	components.Handler.RegisterHandlers(mux)
	sLogger.InfoLog("[RunWorker]Task handlers registered")

	go func() {
		if err := components.Server.Start(mux); err != nil {
			if err != asynq.ErrServerClosed {
				sLogger.FatalLog("[RunWorker]Failed to start worker server", sLogger.Error("err", err))
				rootSpan.RecordError(err)
			}
		}
	}()

	// 等待中斷信號優雅地關閉服務器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	sLogger.InfoLog("[RunWorker]Shutting down worker...")
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
		sLogger.InfoLog("[RunWorker]Worker service exited gracefully")
	case <-shutdownCtx.Done():
		sLogger.WarnLog("[RunWorker]Worker service forced to shutdown after 10 seconds")
	}

	// 記錄成功關閉
	tracing.TraceEvent(rootSpan, "Worker service exited gracefully")
	sLogger.InfoLog("[RunWorker]Worker exited!")
}
