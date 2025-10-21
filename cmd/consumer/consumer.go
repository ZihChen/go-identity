package consumer

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/cmd"
	"github.com/jvdiamondtech/ms-identity-cat/internal/adapter/inbound/handler/consumer"
	"github.com/jvdiamondtech/ms-identity-cat/internal/di"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/cache/redis"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
	"github.com/spf13/cobra"
)

// Command 創建
func Command() *cobra.Command {
	consumerCmd := &cobra.Command{
		Use:   "consumer",
		Short: "Start the KDS consumer service",
		Long:  `Start the KDS consumer service to consume events from KDS and enqueue them for processing`,
		Run:   runConsumer,
	}

	return consumerCmd
}

func init() {
	cmd.AddCommand(Command())
}

const (
	gracefulShutdownTime = 10 * time.Second
)

type services struct {
	tracer          *tracing.TracingService
	redisManager    *redis.Manager
	consumerHandler *consumer.ConsumerHandler
}

// runConsumer 啟動消費者服務
func runConsumer(cobraCmd *cobra.Command, args []string) {
	cfg := cmd.GetConfig()
	logger := cmd.GetLogger()

	// 主程序的Context
	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

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

	// 等待中斷信號
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		s.consumerHandler.RunConsumerLoop(rootCtx)
	}()

	// 等待中斷信號
	<-quit
	logger.InfoWithContext(rootCtx, "Shutting down consumer...")
	// 收到中斷訊號後就要調用rootCancel()將次context全部取消
	rootCancel()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// 等待優雅關閉或超時
	select {
	case <-done:
		logger.InfoWithContext(
			rootCtx,
			"All consumers exited gracefully",
		)
	case <-time.After(gracefulShutdownTime):
		logger.WarnWithContext(
			rootCtx,
			"Force Shutdown - some consumers may still be running",
		)
	}
	logger.InfoWithContext(rootCtx, "All consumer exited")
}

func initializeServices(
	ctx context.Context,
	cfg *config.Config,
	logger infrastructure.Logger,
) (*services, error) {
	// 初始化追踪器
	tracer, err := tracing.NewTracingService(cfg)
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

	// 初始化Consumer Handler
	consumerHandler, err := di.InitializeConsumerHandler(cfg, logger, redisManager)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Consumer Handler: %w", err)
	}
	logger.InfoWithContext(ctx, "Successfully initialized Consumer Handler!")

	return &services{
		tracer:          tracer,
		redisManager:    redisManager,
		consumerHandler: consumerHandler,
	}, nil
}

func (s *services) cleanup(ctx context.Context, logger infrastructure.Logger) {
	// 關閉Consumer Handler (包含 queue service 資源釋放)
	if err := s.consumerHandler.Close(); err != nil {
		logger.ErrorWithContext(ctx, "Failed to close Consumer Handler", logger.Error("error", err))
	}

	// 關閉追蹤器
	if err := s.tracer.Shutdown(ctx); err != nil {
		logger.ErrorWithContext(ctx, "Failed to shutdown tracer", logger.Error("err", err))
	}

	// 關閉Redis連線
	if err := s.redisManager.Close(); err != nil {
		logger.ErrorWithContext(ctx, "Failed to close Redis connection", logger.Error("error", err))
	}
}
