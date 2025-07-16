package consumer

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/cmd"
	"github.com/jvdiamondtech/ms-identity-cat/internal/di"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/infraport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/cache/redis"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/kds"
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
	maxRetries           = 5
	retryBaseDelay       = 200 * time.Millisecond
	maxJitter            = 200 * time.Millisecond
	failureRetryDelay    = 1 * time.Second
	successCycleDelay    = 500 * time.Millisecond
	gracefulShutdownTime = 10 * time.Second
)

type services struct {
	tracer       *tracing.Tracer
	redisManager *redis.Manager
	kdsService   *kds.KDSService
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
		runConsumerLoop(rootCtx, s.kdsService, logger)
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

	// 初始化KDS服務
	kdsService, err := di.InitializeConsumer(cfg, logger, redisManager)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize KDS service: %w", err)
	}
	logger.InfoWithContext(ctx, "Successfully initialized KDS service!")

	return &services{
		tracer:       tracer,
		redisManager: redisManager,
		kdsService:   kdsService,
	}, nil
}

func (s *services) cleanup(ctx context.Context, logger infraport.Logger) {
	// 關閉追蹤器
	if err := s.tracer.Shutdown(ctx); err != nil {
		logger.ErrorLog("Failed to shutdown tracer", logger.Error("err", err))
	}

	// 關閉Redis連線
	if err := s.redisManager.Close(); err != nil {
		logger.ErrorLog("Failed to close Redis connection", logger.Error("error", err))
	} else {
		logger.InfoWithContext(ctx, "Redis connection closed successfully")
	}
}

func runConsumerLoop(rootCtx context.Context, kdsService *kds.KDSService, logger infraport.Logger) {
	logger.InfoWithContext(rootCtx, "Starting Consumer for all event listening")

	for {
		if rootCtx.Err() != nil {
			logger.WarnWithContext(
				rootCtx,
				"All events consumer stopping due to rootCtx cancellation",
			)
			return
		}

		consumerCtx, consumerCancel := context.WithCancel(rootCtx)
		success := runConsumerWithRetry(consumerCtx, kdsService, logger)
		consumerCancel()

		if success {
			time.Sleep(successCycleDelay)
		} else {
			time.Sleep(failureRetryDelay)
		}
	}
}

func runConsumerWithRetry(
	consumerCtx context.Context,
	kdsService *kds.KDSService,
	logger infraport.Logger,
) bool {
	var lastError error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if consumerCtx.Err() != nil {
			logger.WarnWithContext(
				consumerCtx,
				"All events consumer stopping due to context cancellation during retry",
			)
			return false
		}

		if attempt > 0 {
			logger.InfoWithContext(consumerCtx, "Retrying to restart all events consumer...",
				logger.Int("attempt", attempt+1),
				logger.Int("max_retries", maxRetries))
			time.Sleep(backoffDelay(attempt))
		}

		success, err := executeConsumerWithRecovery(consumerCtx, kdsService, logger, attempt)
		if success {
			return true
		}
		lastError = err
	}

	logger.ErrorWithContext(consumerCtx, "All events consumer failed after max retries",
		logger.Error("error", lastError),
		logger.Int("max_retries", maxRetries))
	return false
}

func executeConsumerWithRecovery(
	consumerCtx context.Context,
	kdsService *kds.KDSService,
	logger infraport.Logger,
	attempt int,
) (bool, error) {
	var lastError error
	var success bool

	func() {
		defer func() {
			if r := recover(); r != nil {
				const size = 64 << 10
				buf := make([]byte, size)
				buf = buf[:runtime.Stack(buf, false)]

				if err, ok := r.(error); ok {
					lastError = fmt.Errorf("panic recovered: %w\n%s", err, buf)
				} else {
					lastError = fmt.Errorf("panic recovered: %v\n%s", r, buf)
				}

				logger.ErrorWithContext(consumerCtx, "All events consumer panicked",
					logger.Error("error", lastError))
			}
		}()

		err := kdsService.ConsumeAllEvents(consumerCtx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(consumerCtx.Err(), context.Canceled) {
				logger.WarnWithContext(
					consumerCtx,
					"All events consumer stopped due to context cancellation during consume",
				)
				return
			}

			if errors.Is(consumerCtx.Err(), context.DeadlineExceeded) {
				lastError = fmt.Errorf("consumer timed out: %w", consumerCtx.Err())
				logger.ErrorWithContext(consumerCtx, "All events consumer timed out",
					logger.Error("error", lastError),
					logger.Int("attempt", attempt+1),
					logger.Int("max_retries", maxRetries))
				return
			}

			lastError = err
			logger.ErrorWithContext(consumerCtx, "All events consumer failed",
				logger.Error("error", err),
				logger.Int("attempt", attempt+1),
				logger.Int("max_retries", maxRetries))
			return
		}

		logger.InfoWithContext(consumerCtx, "All events consumer completed successfully")
		success = true
	}()

	return success, lastError
}

func backoffDelay(attempt int) time.Duration {
	backoffDuration := retryBaseDelay * time.Duration(1+attempt/2)
	jitter := time.Duration(rand.Int63n(int64(maxJitter)))
	return backoffDuration + jitter
}
