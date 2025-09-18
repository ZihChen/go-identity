package consumer

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"runtime"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/service"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/kds"
)

const (
	maxRetries        = 3
	successCycleDelay = 5 * time.Second
	failureRetryDelay = 30 * time.Second
	retryBaseDelay    = 2 * time.Second
	maxJitter         = 1 * time.Second
)

type ConsumerHandler struct {
	kdsService   *kds.KDSService
	queueService service.QueueService
	logger       infrastructure.Logger
}

func NewConsumerHandler(kdsService *kds.KDSService, queueService service.QueueService, logger infrastructure.Logger) *ConsumerHandler {
	return &ConsumerHandler{
		kdsService:   kdsService,
		queueService: queueService,
		logger:       logger,
	}
}

func (h *ConsumerHandler) RunConsumerLoop(rootCtx context.Context) {
	h.logger.InfoWithContext(rootCtx, "Starting Consumer for all event listening")

	ticker := time.NewTicker(successCycleDelay)
	defer ticker.Stop()

	for {
		select {
		case <-rootCtx.Done():
			h.logger.WarnWithContext(rootCtx, "All events consumer stopping due to rootCtx cancellation")
			return
		case <-ticker.C:
			// Ticker觸發，執行consumer
			consumerCtx, consumerCancel := context.WithCancel(rootCtx)
			success := h.runConsumerWithRetry(consumerCtx)
			consumerCancel()

			// 動態調整ticker間隔
			if success {
				ticker.Reset(successCycleDelay) // 成功: 5秒後再執行
			} else {
				ticker.Reset(failureRetryDelay) // 失敗: 30秒後再執行
			}
		}
	}
}

func (h *ConsumerHandler) runConsumerWithRetry(consumerCtx context.Context) bool {
	var lastError error

	for attempt := 0; attempt < maxRetries; attempt++ {
		// 只在第一次嘗試後檢查context
		if attempt > 0 {
			select {
			case <-consumerCtx.Done():
				// 如果context已取消，立即退出， 避免無意義重試
				h.logger.WarnWithContext(consumerCtx, "All events consumer stopping due to context cancellation during retry")
				return false
			default:
			}

			h.logger.InfoWithContext(consumerCtx, "Retrying to restart all events consumer...",
				h.logger.Int("attempt", attempt+1),
				h.logger.Int("max_retries", maxRetries))

			// 使用select避免阻塞等待
			delay := backoffDelay(attempt)
			timer := time.NewTimer(delay)
			select {
			case <-consumerCtx.Done():
				timer.Stop() // 清理資源 + 退出
				return false
			case <-timer.C: // 延遲完成，繼續執行
			}
		}

		success, err := h.executeConsumerWithRecovery(consumerCtx, attempt)
		if success {
			return true
		}
		lastError = err
	}

	h.logger.ErrorWithContext(consumerCtx, "All events consumer failed after max retries",
		h.logger.Error("error", lastError),
		h.logger.Int("max_retries", maxRetries))
	return false
}

func (h *ConsumerHandler) executeConsumerWithRecovery(
	consumerCtx context.Context,
	attempt int,
) (bool, error) {
	var lastError error
	var success bool

	func() {
		defer func() {
			if r := recover(); r != nil {
				lastError = h.handlePanic(consumerCtx, r)
			}
		}()

		err := h.kdsService.ConsumeAllEvents(consumerCtx)
		if err != nil {
			lastError = h.handleConsumerError(consumerCtx, err, attempt)
			return
		}

		h.logger.InfoWithContext(consumerCtx, "All events consumer completed successfully")
		success = true
	}()

	return success, lastError
}

// handlePanic 處理panic恢復並優化stack trace收集
func (h *ConsumerHandler) handlePanic(ctx context.Context, r interface{}) error {
	// 預分配較大的buffer避免動態擴展
	buf := make([]byte, 64<<10) // 64KB
	n := runtime.Stack(buf, false)
	buf = buf[:n]

	var panicErr error
	if err, ok := r.(error); ok {
		panicErr = fmt.Errorf("panic recovered: %w\n%s", err, buf)
	} else {
		panicErr = fmt.Errorf("panic recovered: %v\n%s", r, buf)
	}

	h.logger.ErrorWithContext(ctx, "All events consumer panicked",
		h.logger.Error("error", panicErr))
	return panicErr
}

// handleConsumerError 統一處理consumer錯誤
func (h *ConsumerHandler) handleConsumerError(ctx context.Context, err error, attempt int) error {
	// 快速檢查context取消
	if h.isContextCanceled(ctx, err) {
		h.logger.WarnWithContext(ctx, "All events consumer stopped due to context cancellation during consume")
		return err
	}

	// 檢查超時
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		timeoutErr := fmt.Errorf("consumer timed out: %w", ctx.Err())
		h.logger.ErrorWithContext(ctx, "All events consumer timed out",
			h.logger.Error("error", timeoutErr),
			h.logger.Int("attempt", attempt+1),
			h.logger.Int("max_retries", maxRetries))
		return timeoutErr
	}

	// 一般錯誤
	h.logger.ErrorWithContext(ctx, "All events consumer failed",
		h.logger.Error("error", err),
		h.logger.Int("attempt", attempt+1),
		h.logger.Int("max_retries", maxRetries))
	return err
}

// isContextCanceled 快速檢查context是否已取消
func (h *ConsumerHandler) isContextCanceled(ctx context.Context, err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled)
}

func (h *ConsumerHandler) Close() error {
	if h.queueService != nil {
		if err := h.queueService.Close(); err != nil {
			h.logger.ErrorLog("Failed to close queue service", h.logger.Error("error", err))
			return err
		}
		h.logger.InfoLog("Queue service closed successfully")
	}
	return nil
}

func backoffDelay(attempt int) time.Duration {
	backoffDuration := retryBaseDelay * time.Duration(1+attempt/2)
	jitter := time.Duration(rand.Int63n(int64(maxJitter)))
	return backoffDuration + jitter
}
