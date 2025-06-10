package consumer

import (
	"context"
	"fmt"
	sLog "github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/logger"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/jvdiamondtech/ms-identity-cat/cmd"
	"github.com/jvdiamondtech/ms-identity-cat/internal/di"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
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

// 不再需要單獨的Consumer結構，因為我們現在使用單一的Consumer來處理所有事件類型

// runConsumer 啟動Consumer
func runConsumer(cobraCmd *cobra.Command, args []string) {
	// 獲取配置和日誌
	cfg := cmd.GetConfig()
	logger := cmd.GetLogger()
	// 獲取ServiceLogger實例
	sLogger := sLog.NewServiceLogger(cfg)

	rootCtx := context.Background()

	// 初始化追踪器
	tracer, err := tracing.NewTracer(cfg)
	if err != nil {
		sLogger.FatalLog("Failed to initialize consumer tracer", sLogger.Error("err", err))
	}
	defer tracer.Shutdown(context.Background())
	sLogger.InfoLog("Successfully initialized consumer tracer!")

	rootCtx, rootSpan := tracing.StartSpan(rootCtx, "ConsumerService.Start")
	rootSpan.SetAttributes(
		attribute.String("service.name", cfg.App.Name),
		attribute.String("service.type", "consumer"),
		attribute.String("service.environment", cfg.App.Env),
	)
	defer rootSpan.End()

	// 使用Wire初始化KDS服務
	kdsService, err := di.InitializeConsumer(cfg, logger, sLogger)
	if err != nil {
		sLogger.FatalLog("Failed to initialize KDS service", sLogger.Error("err", err))
	}
	sLogger.InfoLog("Successfully initialized KDS service!")

	// 創建上下文
	ctx, cancel := context.WithCancel(rootCtx)
	defer cancel()

	// 等待中斷信號
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup

	// 記錄Consumer啟動
	tracing.TraceEvent(rootSpan, "Starting KDS consumers")

	sLogger.InfoLog("Starting KDS consumer for all event types")
	wg.Add(1)
	go func() {
		defer wg.Done()
		// 創建Consumer span
		consumerCtx, consumerSpan := tracing.StartSpan(rootCtx, "AllEventsConsumer")
		defer consumerSpan.End()

		consumerSpan.SetAttributes(
			attribute.String("consumer.type", "all_events"),
		)
		sLogger.InfoWithContext(consumerCtx, "Starting all events consumer")
		tracing.TraceEvent(consumerSpan, "All events consumer started")

		for {
			// 外層無限循環，確保Consumer持續運行
			// 檢查上下文是否取消 (每次Consume前)
			if ctx.Err() != nil {
				sLogger.InfoWithContext(consumerCtx, "All events consumer stopping due to context cancellation")
				tracing.TraceEvent(consumerSpan, "All events consumer stopped due to cancellation")
				return
			}

			// 添加重試邏輯
			maxRetries := 5
			retryDelay := 2 * time.Second
			var lastError error
			var success bool // 標記是否成功消費

			for attempt := 0; attempt < maxRetries; attempt++ {
				// 檢查上下文是否取消 (在每次重試嘗試前)
				if ctx.Err() != nil {
					sLogger.InfoWithContext(consumerCtx, "All events consumer stopping due to context cancellation during retry")
					tracing.TraceEvent(consumerSpan, "All events consumer stopped due to cancellation during retry")
					return
				}

				// 添加 recover 機制
				func() {
					defer func() {
						if r := recover(); r != nil {
							const size = 64 << 10
							buf := make([]byte, size)
							buf = buf[:runtime.Stack(buf, false)]
							err, ok := r.(error)
							if ok {
								lastError = fmt.Errorf("panic recovered: %w\n%s", err, buf)
							} else {
								lastError = fmt.Errorf("panic recovered: %v\n%s", r, buf)
							}
							sLogger.ErrorWithContext(consumerCtx, "All events consumer panicked", sLogger.Error("error", lastError))
							consumerSpan.RecordError(lastError)
							consumerSpan.SetStatus(codes.Error, lastError.Error())
						}
					}()

					if attempt > 0 {
						sLogger.InfoWithContext(consumerCtx, "[x]Retrying all events consumer",
							sLogger.Int("attempt", attempt+1),
							sLogger.Int("max_retries", maxRetries),
							sLogger.String("retry_delay", retryDelay.String()))
						tracing.TraceEvent(consumerSpan, "Retrying consumer",
							attribute.Int("attempt", attempt+1),
							attribute.Int("max_retries", maxRetries))
						backoffDuration := retryDelay * time.Duration(attempt)
						// 加入些許隨機性，避免集中重試
						jitter := time.Duration(rand.Int63n(int64(500 * time.Millisecond)))
						time.Sleep(backoffDuration + jitter)
					}
					// 嘗試啟動Consumer
					consumeCtx, cancel := context.WithTimeout(consumerCtx, 30*time.Second)
					err := kdsService.ConsumeAllEvents(consumeCtx)
					cancel()
					// 上下文被取消，
					if err == context.Canceled || consumeCtx.Err() == context.Canceled {
						sLogger.InfoWithContext(consumeCtx, "All events consumer stopped due to context cancellation during consume")
						tracing.TraceEvent(consumerSpan, "All events consumer stopped gracefully during consume")
						return
					}
					// 超時錯誤，記錄並重試
					if consumeCtx.Err() == context.DeadlineExceeded {
						lastError = fmt.Errorf("consumer timed out: %w", consumeCtx.Err())
						sLogger.ErrorWithContext(consumeCtx, "All events consumer timed out, will retry",
							sLogger.Error("error", lastError),
							sLogger.Int("attempt", attempt+1),
							sLogger.Int("max_retries", maxRetries))
						consumerSpan.RecordError(lastError)
						return
					}
					// 其他錯誤重試
					if err != nil {
						lastError = err
						sLogger.ErrorWithContext(consumeCtx, "All events consumer failed, will retry",
							sLogger.Error("error", err),
							sLogger.Int("attempt", attempt+1),
							sLogger.Int("max_retries", maxRetries))
						consumerSpan.RecordError(err)
						return
					}

					// 如果沒有錯誤，則成功
					sLogger.InfoWithContext(consumeCtx, "All events consumer completed successfully")
					tracing.TraceEvent(consumerSpan, "All events consumer completed")
					success = true
				}() // 立即執行 recover 的匿名函數

				if success || ctx.Err() != nil {
					break // 成功或上下文取消，跳出內層重試循環
				}
			}

			// 重試次數達到上限
			if !success {
				sLogger.ErrorWithContext(consumerCtx, "All events consumer failed after max retries",
					sLogger.Error("error", lastError),
					sLogger.Int("max_retries", maxRetries))
				tracing.TraceEvent(consumerSpan, "All events consumer failed after max retries",
					attribute.String("error", lastError.Error()))
				consumerSpan.SetStatus(codes.Error, fmt.Sprintf("Consumer failed after %d retries: %v", maxRetries, lastError))
				time.Sleep(10 * time.Second) // 失敗後等待一段時間再嘗試下一次外層循環
			} else {
				time.Sleep(5 * time.Second) // 防止頻繁消費
			}
		}
	}()

	// 等待中斷信號
	<-quit
	sLogger.InfoWithContext(rootCtx, "Shutting down consumer...")

	// 記錄關閉事件
	tracing.TraceEvent(rootSpan, "Shutting down consumer service")

	cancel()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// 等待優雅關閉或超時
	select {
	case <-done:
		sLogger.InfoWithContext(rootCtx, "All consumers exited gracefully")
	case <-time.After(10 * time.Second):
		sLogger.WarnWithContext(rootCtx, "Shutdown timeout - some consumers may still be running")
	}

	// 記錄成功退出
	tracing.TraceEvent(rootSpan, "Consumer service exited gracefully")
	sLogger.InfoWithContext(rootCtx, "Consumer exited")
}
