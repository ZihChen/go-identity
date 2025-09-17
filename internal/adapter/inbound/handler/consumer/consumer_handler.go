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

	for {
		if rootCtx.Err() != nil {
			h.logger.WarnWithContext(
				rootCtx,
				"All events consumer stopping due to rootCtx cancellation",
			)
			return
		}

		consumerCtx, consumerCancel := context.WithCancel(rootCtx)
		success := h.runConsumerWithRetry(consumerCtx)
		consumerCancel()

		if success {
			time.Sleep(successCycleDelay)
		} else {
			time.Sleep(failureRetryDelay)
		}
	}
}

func (h *ConsumerHandler) runConsumerWithRetry(consumerCtx context.Context) bool {
	var lastError error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if consumerCtx.Err() != nil {
			h.logger.WarnWithContext(
				consumerCtx,
				"All events consumer stopping due to context cancellation during retry",
			)
			return false
		}

		if attempt > 0 {
			h.logger.InfoWithContext(consumerCtx, "Retrying to restart all events consumer...",
				h.logger.Int("attempt", attempt+1),
				h.logger.Int("max_retries", maxRetries))
			time.Sleep(backoffDelay(attempt))
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
				// 動態分配 stack buffer，初始 4KB，最大 1MB
				initialSize := 4 << 10 // 4KB
				maxSize := 1 << 20     // 1MB

				buf := make([]byte, initialSize)
				n := runtime.Stack(buf, false)

				// 如果 buffer 不夠大，動態擴展
				for n >= len(buf) && len(buf) < maxSize {
					buf = make([]byte, len(buf)*2)
					n = runtime.Stack(buf, false)
				}
				buf = buf[:n]

				if err, ok := r.(error); ok {
					lastError = fmt.Errorf("panic recovered: %w\n%s", err, buf)
				} else {
					lastError = fmt.Errorf("panic recovered: %v\n%s", r, buf)
				}

				h.logger.ErrorWithContext(consumerCtx, "All events consumer panicked",
					h.logger.Error("error", lastError))
			}
		}()

		err := h.kdsService.ConsumeAllEvents(consumerCtx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(consumerCtx.Err(), context.Canceled) {
				h.logger.WarnWithContext(
					consumerCtx,
					"All events consumer stopped due to context cancellation during consume",
				)
				return
			}

			if errors.Is(consumerCtx.Err(), context.DeadlineExceeded) {
				lastError = fmt.Errorf("consumer timed out: %w", consumerCtx.Err())
				h.logger.ErrorWithContext(consumerCtx, "All events consumer timed out",
					h.logger.Error("error", lastError),
					h.logger.Int("attempt", attempt+1),
					h.logger.Int("max_retries", maxRetries))
				return
			}

			lastError = err
			h.logger.ErrorWithContext(consumerCtx, "All events consumer failed",
				h.logger.Error("error", err),
				h.logger.Int("attempt", attempt+1),
				h.logger.Int("max_retries", maxRetries))
			return
		}

		h.logger.InfoWithContext(consumerCtx, "All events consumer completed successfully")
		success = true
	}()

	return success, lastError
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
