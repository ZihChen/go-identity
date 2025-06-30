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
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/cache/redis"
	sLog "github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/logger"
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

// runConsumer 啟動消費者服務
func runConsumer(cobraCmd *cobra.Command, args []string) {
	// 獲取配置和日誌
	cfg := cmd.GetConfig()
	logger := cmd.GetLogger()
	// 獲取ServiceLogger實例
	sLogger := sLog.NewServiceLogger(cfg)

	// 主程序的Context
	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	// 初始化追踪器
	tracer, err := tracing.NewTracer(cfg)
	if err != nil {
		sLogger.FatalWithContext(rootCtx, "[Fatal][Consumer][runConsumer] Failed to initialize tracer", sLogger.Error("error", err))
	}
	defer tracer.Shutdown(rootCtx)
	sLogger.InfoWithContext(rootCtx, "[Info][Consumer][runConsumer] Successfully initialized tracer!")

	// 初始化Redis連線
	redisManager := redis.NewRedisManager(cfg)
	defer func() {
		_ = redisManager.Close()
		sLogger.InfoWithContext(rootCtx, "[Info][Consumer][runConsumer] Redis connection closed successfully")
	}()
	if err = redisManager.Connect(rootCtx); err != nil {
		sLogger.FatalLog("Failed to connect to Redis after retry", sLogger.Error("err", err))
	}

	// 使用Wire初始化KDS服務
	kdsService, err := di.InitializeConsumer(cfg, logger, sLogger, redisManager)
	if err != nil {
		sLogger.FatalWithContext(rootCtx, "[Fatal][Consumer][runConsumer] Failed to initialize KDS service", sLogger.Error("error", err))
	}
	sLogger.InfoWithContext(rootCtx, "[Info][Consumer][runConsumer] Successfully initialized KDS service!")

	// 等待中斷信號
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		sLogger.InfoWithContext(rootCtx, "[Info][Consumer][runConsumer] Starting Consumer for all event listening")

		// 外層無限循環，確保Consumer持續運行
		for {
			// 使用rootCtx創建consumerCtx，確保上下文取消可以正確傳遞
			consumerCtx, consumerCancel := context.WithCancel(rootCtx)

			// 主程序Context取消，次Context也需一並取消
			if rootCtx.Err() != nil {
				sLogger.WarnWithContext(consumerCtx, "[Warn][Consumer][runConsumer] All events consumer stopping due to rootCtx cancellation")
				consumerCancel()
				return
			}

			// 添加重試邏輯
			maxRetries := 5
			retryDelay := 200 * time.Millisecond // 重試延遲時間
			var lastError error
			var success bool // 標記是否成功消費

			for attempt := 0; attempt < maxRetries; attempt++ {
				// Retry前先檢查主程序是否終止
				if rootCtx.Err() != nil {
					sLogger.WarnWithContext(consumerCtx, "[Warn][Consumer][runConsumer] All events consumer stopping due to rootCtx cancellation during retry")
					consumerCancel()
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
							sLogger.ErrorWithContext(consumerCtx, "[Error][Consumer][runConsumer] All events consumer panicked", sLogger.Error("error", lastError))
						}
					}()

					if attempt > 0 {
						sLogger.InfoWithContext(consumerCtx, "[Info][Consumer][runConsumer] Retrying",
							sLogger.Int("attempt", attempt+1),
							sLogger.Int("max_retries", maxRetries),
							sLogger.String("retry_delay", retryDelay.String()))

						// 退避策略，避免後續重試等待時間過長
						backoffDuration := retryDelay * time.Duration(1+attempt/2) // 每兩次重試才增加一次基本延遲
						// 加入些許隨機性，避免集中重試
						jitter := time.Duration(rand.Int63n(int64(200 * time.Millisecond)))
						time.Sleep(backoffDuration + jitter)
					}

					// 啟動Consumer，使用consumerCtx創建帶有超時的consumeCtx
					consumerCtx, consumeCancel := context.WithTimeout(consumerCtx, 30*time.Second)
					defer consumeCancel()

					// 執行消費操作
					err := kdsService.ConsumeAllEvents(consumerCtx)

					// 檢查Context是否被取消
					if errors.Is(err, context.Canceled) || errors.Is(consumerCtx.Err(), context.Canceled) {
						sLogger.WarnWithContext(consumerCtx, "[Warn][Consumer][runConsumer] All events consumer stopped due to context cancellation during consume")
						return
					}

					// 檢查Context是否超時
					if errors.Is(consumerCtx.Err(), context.DeadlineExceeded) {
						lastError = fmt.Errorf("consumer timed out: %w", consumerCtx.Err())
						sLogger.ErrorWithContext(consumerCtx, "[Error][Consumer][runConsumer] All events consumer timed out",
							sLogger.Error("error", lastError),
							sLogger.Int("attempt", attempt+1),
							sLogger.Int("max_retries", maxRetries))
						return
					}

					// 其他錯誤重試
					if err != nil {
						lastError = err
						sLogger.ErrorWithContext(consumerCtx, "[Error][Consumer][runConsumer] All events consumer failed",
							sLogger.Error("error", err),
							sLogger.Int("attempt", attempt+1),
							sLogger.Int("max_retries", maxRetries))
						return
					}

					sLogger.InfoWithContext(consumerCtx, "[Info][Consumer][runConsumer] All events consumer completed successfully")
					// 如果沒有錯誤，則成功
					success = true
				}()

				if success || rootCtx.Err() != nil {
					break // 成功或上下文取消，跳出內層重試循環
				}
			}

			// 重試次數達到上限
			if !success {
				sLogger.ErrorWithContext(consumerCtx, "[Error][Consumer][runConsumer] All events consumer failed after max retries",
					sLogger.Error("error", lastError),
					sLogger.Int("max_retries", maxRetries))
				consumerCancel()            // 確保在循環結束前取消consumerCtx
				time.Sleep(1 * time.Second) // 失敗後等待一段時間再嘗試下一次外層循環
			} else {
				consumerCancel()                   // 確保在循環結束前取消consumerCtx
				time.Sleep(500 * time.Millisecond) // 完整做完一個循環，防止頻繁消費
			}
		}
	}()

	// 等待中斷信號
	<-quit
	sLogger.InfoWithContext(rootCtx, "[Info][Consumer][runConsumer] Shutting down consumer...")
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
		sLogger.InfoWithContext(rootCtx, "[Info][Consumer][runConsumer] All consumers exited gracefully")
	case <-time.After(10 * time.Second):
		sLogger.WarnWithContext(rootCtx, "[Warn][Consumer][runConsumer] Force Shutdown - some consumers may still be running")
	}
	sLogger.InfoWithContext(rootCtx, "[Info][Consumer][runConsumer] All consumer exited")
}
