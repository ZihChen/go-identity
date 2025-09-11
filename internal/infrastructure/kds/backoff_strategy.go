package kds

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

// BackoffStrategy 退避策略介面
type BackoffStrategy interface {
	// NextBackoff 計算下一次退避時間
	NextBackoff() time.Duration
	// Reset 重置退避狀態
	Reset()
	// RecordSuccess 記錄成功操作
	RecordSuccess()
	// RecordError 記錄錯誤操作
	RecordError(err error)
}

// AdaptiveBackoffStrategy 自適應退避策略
type AdaptiveBackoffStrategy struct {
	minBackoff    time.Duration
	maxBackoff    time.Duration
	multiplier    float64
	jitterFactor  float64
	currentDelay  time.Duration
	successCount  int64
	errorCount    int64
	lastErrorTime time.Time
	classifier    *ErrorClassifier
	mu            sync.RWMutex
}

// NewAdaptiveBackoffStrategy 創建自適應退避策略
func NewAdaptiveBackoffStrategy(minBackoff, maxBackoff time.Duration, multiplier float64) *AdaptiveBackoffStrategy {
	return &AdaptiveBackoffStrategy{
		minBackoff:   minBackoff,
		maxBackoff:   maxBackoff,
		multiplier:   multiplier,
		jitterFactor: 0.1, // 10% jitter
		currentDelay: minBackoff,
		classifier:   NewErrorClassifier(),
	}
}

// NextBackoff 計算下一次退避時間
func (s *AdaptiveBackoffStrategy) NextBackoff() time.Duration {
	s.mu.RLock()
	delay := s.currentDelay
	s.mu.RUnlock()

	// 添加 jitter 以避免雷群效應
	jitter := time.Duration(rand.Float64() * float64(delay) * s.jitterFactor)
	return delay + jitter
}

// Reset 重置退避狀態
func (s *AdaptiveBackoffStrategy) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.currentDelay = s.minBackoff
	s.successCount = 0
	s.errorCount = 0
	s.lastErrorTime = time.Time{}
}

// RecordSuccess 記錄成功操作
func (s *AdaptiveBackoffStrategy) RecordSuccess() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.successCount++

	// 連續成功後逐漸減少退避時間
	if s.successCount >= 3 {
		s.currentDelay = time.Duration(float64(s.currentDelay) / s.multiplier)
		if s.currentDelay < s.minBackoff {
			s.currentDelay = s.minBackoff
		}
		s.successCount = 0
	}
}

// RecordError 記錄錯誤操作
func (s *AdaptiveBackoffStrategy) RecordError(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.errorCount++
	s.lastErrorTime = time.Now()
	s.successCount = 0 // 重置成功計數

	// 根據錯誤類型調整退避策略
	category := s.classifier.GetErrorCategory(err)
	multiplier := s.getMultiplierForError(category)

	s.currentDelay = time.Duration(float64(s.currentDelay) * multiplier)
	if s.currentDelay > s.maxBackoff {
		s.currentDelay = s.maxBackoff
	}
}

// getMultiplierForError 根據錯誤類型獲取倍數
func (s *AdaptiveBackoffStrategy) getMultiplierForError(category string) float64 {
	switch category {
	case "throttling":
		return s.multiplier * 2.0 // 限流錯誤需要更長的退避
	case "permanent":
		return 1.0 // 永久錯誤不增加退避時間
	case "temporary":
		return s.multiplier * 1.5 // 臨時錯誤中等退避
	case "retryable":
		return s.multiplier // 標準退避
	default:
		return s.multiplier
	}
}

// GetCurrentDelay 獲取當前退避時間
func (s *AdaptiveBackoffStrategy) GetCurrentDelay() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentDelay
}

// GetStats 獲取統計信息
func (s *AdaptiveBackoffStrategy) GetStats() (successCount, errorCount int64, lastErrorTime time.Time) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.successCount, s.errorCount, s.lastErrorTime
}

// ExponentialBackoffStrategy 指數退避策略
type ExponentialBackoffStrategy struct {
	minBackoff   time.Duration
	maxBackoff   time.Duration
	multiplier   float64
	currentDelay time.Duration
	attempt      int
	mu           sync.RWMutex
}

// NewExponentialBackoffStrategy 創建指數退避策略
func NewExponentialBackoffStrategy(minBackoff, maxBackoff time.Duration, multiplier float64) *ExponentialBackoffStrategy {
	return &ExponentialBackoffStrategy{
		minBackoff:   minBackoff,
		maxBackoff:   maxBackoff,
		multiplier:   multiplier,
		currentDelay: minBackoff,
	}
}

// NextBackoff 計算下一次退避時間
func (s *ExponentialBackoffStrategy) NextBackoff() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.attempt++
	s.currentDelay = time.Duration(float64(s.minBackoff) * math.Pow(s.multiplier, float64(s.attempt-1)))

	if s.currentDelay > s.maxBackoff {
		s.currentDelay = s.maxBackoff
	}

	// 添加 jitter
	jitter := time.Duration(rand.Float64() * float64(s.currentDelay) * 0.1)
	return s.currentDelay + jitter
}

// Reset 重置退避狀態
func (s *ExponentialBackoffStrategy) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.attempt = 0
	s.currentDelay = s.minBackoff
}

// RecordSuccess 記錄成功操作
func (s *ExponentialBackoffStrategy) RecordSuccess() {
	s.Reset()
}

// RecordError 記錄錯誤操作
func (s *ExponentialBackoffStrategy) RecordError(err error) {
	// 指數退避不需要特殊處理錯誤類型
}

// RetryExecutor 重試執行器
type RetryExecutor struct {
	strategy    BackoffStrategy
	maxAttempts int
	classifier  *ErrorClassifier
}

// NewRetryExecutor 創建重試執行器
func NewRetryExecutor(strategy BackoffStrategy, maxAttempts int) *RetryExecutor {
	return &RetryExecutor{
		strategy:    strategy,
		maxAttempts: maxAttempts,
		classifier:  NewErrorClassifier(),
	}
}

// Execute 執行重試邏輯
func (r *RetryExecutor) Execute(ctx context.Context, operation string, fn func() error) error {
	var lastErr error

	for attempt := 1; attempt <= r.maxAttempts; attempt++ {
		err := fn()
		if err == nil {
			r.strategy.RecordSuccess()
			return nil
		}

		lastErr = err
		r.strategy.RecordError(err)

		// 檢查是否為永久錯誤，如果是則不重試
		if r.classifier.IsPermanent(err) {
			return NewConsumerError(operation, "PERMANENT_ERROR",
				"permanent error, no retry needed", err)
		}

		// 最後一次嘗試，不需要退避
		if attempt >= r.maxAttempts {
			break
		}

		// 計算退避時間
		backoffDuration := r.strategy.NextBackoff()

		// 等待退避時間或取消
		select {
		case <-ctx.Done():
			return NewConsumerError(operation, "RETRY_CANCELLED",
				"retry cancelled due to context", ctx.Err())
		case <-time.After(backoffDuration):
			continue
		}
	}

	return NewConsumerError(operation, "MAX_RETRIES_EXCEEDED",
		fmt.Sprintf("max retries (%d) exceeded", r.maxAttempts), lastErr)
}

// ExecuteWithCustomRetry 自定義重試條件執行
func (r *RetryExecutor) ExecuteWithCustomRetry(ctx context.Context, operation string,
	fn func() error, shouldRetry func(error) bool) error {

	var lastErr error

	for attempt := 1; attempt <= r.maxAttempts; attempt++ {
		err := fn()
		if err == nil {
			r.strategy.RecordSuccess()
			return nil
		}

		lastErr = err
		r.strategy.RecordError(err)

		// 使用自定義重試判斷
		if !shouldRetry(err) {
			return NewConsumerError(operation, "CUSTOM_NO_RETRY",
				"custom condition indicates no retry needed", err)
		}

		// 最後一次嘗試，不需要退避
		if attempt >= r.maxAttempts {
			break
		}

		// 計算退避時間
		backoffDuration := r.strategy.NextBackoff()

		// 等待退避時間或取消
		select {
		case <-ctx.Done():
			return NewConsumerError(operation, "RETRY_CANCELLED",
				"retry cancelled due to context", ctx.Err())
		case <-time.After(backoffDuration):
			continue
		}
	}

	return NewConsumerError(operation, "MAX_RETRIES_EXCEEDED",
		fmt.Sprintf("max retries (%d) exceeded", r.maxAttempts), lastErr)
}
