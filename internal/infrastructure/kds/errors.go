package kds

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"time"
)

// Consumer 相關錯誤定義
var (
	// 批次處理錯誤
	ErrBatchProcessingFailed = errors.New("batch processing failed")
	ErrBatchSizeTooLarge     = errors.New("batch size too large")
	ErrBatchTimeout          = errors.New("batch processing timeout")

	// Worker Pool 錯誤
	ErrWorkerPoolClosed    = errors.New("worker pool is closed")
	ErrWorkerPanic         = errors.New("worker panic occurred")
	ErrWorkerTimeout       = errors.New("worker processing timeout")
	ErrAllWorkersExhausted = errors.New("all workers are exhausted")

	// 分片處理錯誤
	ErrShardLockFailed     = errors.New("failed to acquire shard lock")
	ErrShardLockTimeout    = errors.New("shard lock timeout")
	ErrShardIteratorFailed = errors.New("shard iterator failed")
	ErrCheckpointFailed    = errors.New("checkpoint update failed")

	// KDS 相關錯誤
	ErrKDSConnectionFailed = errors.New("KDS connection failed")
	ErrKDSThrottling       = errors.New("KDS throttling occurred")
	ErrKDSRecordInvalid    = errors.New("KDS record is invalid")

	// 恢復機制錯誤
	ErrRecoveryFailed        = errors.New("recovery attempt failed")
	ErrMaxRecoveryExceeded   = errors.New("max recovery attempts exceeded")
	ErrRecoveryContextCancel = errors.New("recovery context cancelled")
)

// ConsumerError 提供更詳細的錯誤信息
type ConsumerError struct {
	Op        string            // 操作名稱
	Code      string            // 錯誤代碼
	Message   string            // 錯誤消息
	Err       error             // 原始錯誤
	Context   map[string]string // 上下文信息
	Timestamp time.Time         // 錯誤發生時間
	Stack     []byte            // 錯誤堆疊信息
}

// Error 實現 error 介面
func (e *ConsumerError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%s): %v", e.Op, e.Message, e.Code, e.Err)
	}
	return fmt.Sprintf("%s: %s (%s)", e.Op, e.Message, e.Code)
}

// Unwrap 支持錯誤鏈
func (e *ConsumerError) Unwrap() error {
	return e.Err
}

// NewConsumerError 創建新的 Consumer 錯誤
func NewConsumerError(op, code, message string, err error) *ConsumerError {
	stack := make([]byte, 4096)
	stackSize := runtime.Stack(stack, false)

	return &ConsumerError{
		Op:        op,
		Code:      code,
		Message:   message,
		Err:       err,
		Context:   make(map[string]string),
		Timestamp: time.Now(),
		Stack:     stack[:stackSize],
	}
}

// WithContext 添加上下文信息
func (e *ConsumerError) WithContext(key, value string) *ConsumerError {
	if e.Context == nil {
		e.Context = make(map[string]string)
	}
	e.Context[key] = value
	return e
}

// WithShardID 添加分片ID信息
func (e *ConsumerError) WithShardID(shardID string) *ConsumerError {
	return e.WithContext("shard_id", shardID)
}

// WithSequenceNumber 添加序列號信息
func (e *ConsumerError) WithSequenceNumber(seqNum string) *ConsumerError {
	return e.WithContext("sequence_number", seqNum)
}

// WithWorkerID 添加 Worker ID 信息
func (e *ConsumerError) WithWorkerID(workerID string) *ConsumerError {
	return e.WithContext("worker_id", workerID)
}

// PanicRecovery Panic 恢復機制
type PanicRecovery struct {
	maxAttempts    int
	retryInterval  time.Duration
	onRecovery     func(panicValue interface{}, stack []byte)
	onMaxExceeded  func(attempts int, lastPanic interface{})
	attemptCount   int
	lastPanicValue interface{}
	lastPanicTime  time.Time
}

// NewPanicRecovery 創建 Panic 恢復機制
func NewPanicRecovery(maxAttempts int, retryInterval time.Duration) *PanicRecovery {
	return &PanicRecovery{
		maxAttempts:   maxAttempts,
		retryInterval: retryInterval,
	}
}

// OnRecovery 設置恢復回調
func (r *PanicRecovery) OnRecovery(callback func(panicValue interface{}, stack []byte)) *PanicRecovery {
	r.onRecovery = callback
	return r
}

// OnMaxExceeded 設置最大嘗試次數超出回調
func (r *PanicRecovery) OnMaxExceeded(callback func(attempts int, lastPanic interface{})) *PanicRecovery {
	r.onMaxExceeded = callback
	return r
}

// Execute 執行函數並處理 Panic
func (r *PanicRecovery) Execute(ctx context.Context, operation string, fn func() error) error {
	defer func() {
		if panicValue := recover(); panicValue != nil {
			r.attemptCount++
			r.lastPanicValue = panicValue
			r.lastPanicTime = time.Now()

			// 獲取 panic stack
			stack := make([]byte, 8192)
			stackSize := runtime.Stack(stack, false)

			// 調用恢復回調
			if r.onRecovery != nil {
				r.onRecovery(panicValue, stack[:stackSize])
			}

			// 檢查是否超出最大嘗試次數
			if r.attemptCount >= r.maxAttempts {
				if r.onMaxExceeded != nil {
					r.onMaxExceeded(r.attemptCount, panicValue)
				}
				panic(NewConsumerError(operation, "PANIC_MAX_EXCEEDED",
					fmt.Sprintf("max recovery attempts (%d) exceeded", r.maxAttempts),
					ErrMaxRecoveryExceeded))
			}

			// 等待重試間隔
			select {
			case <-ctx.Done():
				panic(NewConsumerError(operation, "PANIC_RECOVERY_CANCELLED",
					"recovery cancelled due to context", ErrRecoveryContextCancel))
			case <-time.After(r.retryInterval):
				// 重試
			}
		}
	}()

	return fn()
}

// Reset 重置恢復狀態
func (r *PanicRecovery) Reset() {
	r.attemptCount = 0
	r.lastPanicValue = nil
	r.lastPanicTime = time.Time{}
}

// GetAttemptCount 獲取嘗試次數
func (r *PanicRecovery) GetAttemptCount() int {
	return r.attemptCount
}

// GetLastPanic 獲取最後一次 panic 信息
func (r *PanicRecovery) GetLastPanic() (interface{}, time.Time) {
	return r.lastPanicValue, r.lastPanicTime
}

// ErrorClassifier 錯誤分類器
type ErrorClassifier struct {
	retryableErrors  []error
	temporaryErrors  []error
	permanentErrors  []error
	throttlingErrors []error
}

// NewErrorClassifier 創建錯誤分類器
func NewErrorClassifier() *ErrorClassifier {
	return &ErrorClassifier{
		retryableErrors: []error{
			ErrKDSConnectionFailed,
			ErrShardLockTimeout,
			ErrWorkerTimeout,
		},
		temporaryErrors: []error{
			ErrKDSThrottling,
			ErrWorkerPoolClosed,
		},
		permanentErrors: []error{
			ErrKDSRecordInvalid,
			ErrBatchSizeTooLarge,
		},
		throttlingErrors: []error{
			ErrKDSThrottling,
		},
	}
}

// IsRetryable 判斷錯誤是否可重試
func (c *ErrorClassifier) IsRetryable(err error) bool {
	return c.containsError(c.retryableErrors, err)
}

// IsTemporary 判斷錯誤是否是臨時的
func (c *ErrorClassifier) IsTemporary(err error) bool {
	return c.containsError(c.temporaryErrors, err)
}

// IsPermanent 判斷錯誤是否是永久的
func (c *ErrorClassifier) IsPermanent(err error) bool {
	return c.containsError(c.permanentErrors, err)
}

// IsThrottling 判斷錯誤是否是限流相關
func (c *ErrorClassifier) IsThrottling(err error) bool {
	return c.containsError(c.throttlingErrors, err)
}

// containsError 檢查錯誤列表是否包含指定錯誤
func (c *ErrorClassifier) containsError(errorList []error, target error) bool {
	for _, err := range errorList {
		if errors.Is(target, err) {
			return true
		}
	}
	return false
}

// GetErrorCategory 獲取錯誤類別
func (c *ErrorClassifier) GetErrorCategory(err error) string {
	switch {
	case c.IsThrottling(err):
		return "throttling"
	case c.IsPermanent(err):
		return "permanent"
	case c.IsTemporary(err):
		return "temporary"
	case c.IsRetryable(err):
		return "retryable"
	default:
		return "unknown"
	}
}
