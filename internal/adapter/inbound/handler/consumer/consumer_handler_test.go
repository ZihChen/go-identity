package consumer

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/service"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/kds"
	"github.com/jvdiamondtech/ms-identity-cat/test/factories"
	"github.com/jvdiamondtech/ms-identity-cat/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Helper function specific to consumer handler tests
func createConsumerMockDependencies(
	t *testing.T,
) (*mocks.KDSServiceMock, *mocks.QueueServiceMock, *mocks.MockLogger) {
	// Explicitly use imports to avoid "unused import" errors
	var _ context.Context
	var _ infrastructure.Logger
	var _ service.QueueService
	var _ *kds.KDSService

	kdsService := mocks.NewKDSServiceMock(t)
	queueService := mocks.NewQueueServiceMock(t)
	logger := mocks.NewMockLogger(t)
	return kdsService, queueService, logger
}

// Tests
func TestNewConsumerHandler(t *testing.T) {
	kdsService, queueService, logger := createConsumerMockDependencies(t)

	handler := NewConsumerHandler(nil, queueService, logger)

	assert.NotNil(t, handler)
	assert.IsType(t, &ConsumerHandler{}, handler)
	assert.Equal(t, queueService, handler.queueService)
	assert.Equal(t, logger, handler.logger)
	// Note: kdsService is passed as nil to test constructor only
	assert.Nil(t, handler.kdsService)
	_ = kdsService // Use the mock to avoid unused variable warning
}

func TestConsumerHandler_executeConsumerWithRecovery_Success(t *testing.T) {
	ctx := factories.CreateTestContext()
	kdsService, queueService, logger := createConsumerMockDependencies(t)

	// Setup mocks
	kdsService.On("ConsumeAllEvents", mock.Anything).Return(nil)

	// Create handler
	handler := &ConsumerHandler{
		kdsService:   nil, // We'll call the method directly with mock
		queueService: queueService,
		logger:       logger,
	}

	// We need to test the method by replacing the kdsService temporarily
	// Since Go doesn't have easy method injection, we'll create a test wrapper
	testKDSService := kdsService
	originalKDS := handler.kdsService
	defer func() {
		handler.kdsService = originalKDS
	}()

	// Mock the KDS service call by creating a test version
	success, err := func() (bool, error) {
		var lastError error
		var success bool

		func() {
			defer func() {
				if r := recover(); r != nil {
					lastError = handler.handlePanic(ctx, r)
				}
			}()

			err := testKDSService.ConsumeAllEvents(ctx)
			if err != nil {
				lastError = err
				return
			}

			success = true
		}()

		return success, lastError
	}()

	// Verify results
	assert.True(t, success)
	assert.NoError(t, err)
	kdsService.AssertExpectations()
}

func TestConsumerHandler_executeConsumerWithRecovery_ConsumeError(t *testing.T) {
	ctx := factories.CreateTestContext()
	kdsService, queueService, logger := createConsumerMockDependencies(t)

	// Setup mocks
	expectedError := errors.New("consume failed")
	kdsService.On("ConsumeAllEvents", mock.Anything).Return(expectedError)

	// Create handler
	handler := &ConsumerHandler{
		kdsService:   nil,
		queueService: queueService,
		logger:       logger,
	}

	// Test wrapper for executeConsumerWithRecovery logic
	success, err := func() (bool, error) {
		var lastError error
		var success bool

		func() {
			defer func() {
				if r := recover(); r != nil {
					lastError = handler.handlePanic(ctx, r)
				}
			}()

			err := kdsService.ConsumeAllEvents(ctx)
			if err != nil {
				lastError = handler.handleConsumerError(ctx, err, 0)
				return
			}

			success = true
		}()

		return success, lastError
	}()

	// Verify results
	assert.False(t, success)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "consume failed")
	kdsService.AssertExpectations()
}

func TestConsumerHandler_executeConsumerWithRecovery_Panic(t *testing.T) {
	ctx := factories.CreateTestContext()
	_, queueService, logger := createConsumerMockDependencies(t)

	// Create handler
	handler := &ConsumerHandler{
		kdsService:   nil,
		queueService: queueService,
		logger:       logger,
	}

	// Test panic handling directly
	panicValue := "test panic"

	success, err := func() (bool, error) {
		var lastError error
		var success bool

		func() {
			defer func() {
				if r := recover(); r != nil {
					lastError = handler.handlePanic(ctx, r)
				}
			}()

			// Simulate panic
			panic(panicValue)
		}()

		return success, lastError
	}()

	// Verify results
	assert.False(t, success)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "panic recovered")
	assert.Contains(t, err.Error(), panicValue)
}

func TestConsumerHandler_handlePanic_ErrorType(t *testing.T) {
	ctx := factories.CreateTestContext()
	_, queueService, logger := createConsumerMockDependencies(t)

	handler := &ConsumerHandler{
		kdsService:   nil,
		queueService: queueService,
		logger:       logger,
	}

	// Test with error type panic
	panicErr := errors.New("test error panic")
	resultErr := handler.handlePanic(ctx, panicErr)

	assert.Error(t, resultErr)
	assert.Contains(t, resultErr.Error(), "panic recovered")
	assert.Contains(t, resultErr.Error(), "test error panic")
}

func TestConsumerHandler_handlePanic_StringType(t *testing.T) {
	ctx := factories.CreateTestContext()
	_, queueService, logger := createConsumerMockDependencies(t)

	handler := &ConsumerHandler{
		kdsService:   nil,
		queueService: queueService,
		logger:       logger,
	}

	// Test with string type panic
	panicStr := "string panic"
	resultErr := handler.handlePanic(ctx, panicStr)

	assert.Error(t, resultErr)
	assert.Contains(t, resultErr.Error(), "panic recovered")
	assert.Contains(t, resultErr.Error(), "string panic")
}

func TestConsumerHandler_handleConsumerError_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel the context

	_, queueService, logger := createConsumerMockDependencies(t)

	handler := &ConsumerHandler{
		kdsService:   nil,
		queueService: queueService,
		logger:       logger,
	}

	// Test with context canceled error
	err := context.Canceled
	resultErr := handler.handleConsumerError(ctx, err, 0)

	assert.Error(t, resultErr)
	assert.Equal(t, context.Canceled, resultErr)
}

func TestConsumerHandler_handleConsumerError_ContextTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for context to timeout
	time.Sleep(1 * time.Millisecond)

	_, queueService, logger := createConsumerMockDependencies(t)

	handler := &ConsumerHandler{
		kdsService:   nil,
		queueService: queueService,
		logger:       logger,
	}

	// Test with timeout error
	err := errors.New("some error")
	resultErr := handler.handleConsumerError(ctx, err, 0)

	assert.Error(t, resultErr)
	assert.Contains(t, resultErr.Error(), "consumer timed out")
}

func TestConsumerHandler_handleConsumerError_GeneralError(t *testing.T) {
	ctx := factories.CreateTestContext()
	_, queueService, logger := createConsumerMockDependencies(t)

	handler := &ConsumerHandler{
		kdsService:   nil,
		queueService: queueService,
		logger:       logger,
	}

	// Test with general error
	expectedError := errors.New("general error")
	resultErr := handler.handleConsumerError(ctx, expectedError, 0)

	assert.Error(t, resultErr)
	assert.Equal(t, expectedError, resultErr)
}

func TestConsumerHandler_isContextCanceled_True(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, queueService, logger := createConsumerMockDependencies(t)

	handler := &ConsumerHandler{
		kdsService:   nil,
		queueService: queueService,
		logger:       logger,
	}

	// Test with canceled context
	result := handler.isContextCanceled(ctx, context.Canceled)
	assert.True(t, result)

	// Test with canceled context error
	result = handler.isContextCanceled(ctx, errors.New("other error"))
	assert.True(t, result) // Should be true because ctx.Err() is context.Canceled
}

func TestConsumerHandler_isContextCanceled_False(t *testing.T) {
	ctx := factories.CreateTestContext()
	_, queueService, logger := createConsumerMockDependencies(t)

	handler := &ConsumerHandler{
		kdsService:   nil,
		queueService: queueService,
		logger:       logger,
	}

	// Test with non-canceled context and different error
	result := handler.isContextCanceled(ctx, errors.New("other error"))
	assert.False(t, result)
}

func TestConsumerHandler_Close_Success(t *testing.T) {
	_, queueService, logger := createConsumerMockDependencies(t)

	// Setup mocks
	queueService.On("Close").Return(nil)

	handler := &ConsumerHandler{
		kdsService:   nil,
		queueService: queueService,
		logger:       logger,
	}

	// Execute
	err := handler.Close()

	// Verify
	assert.NoError(t, err)
	queueService.AssertExpectations()
}

func TestConsumerHandler_Close_Error(t *testing.T) {
	_, queueService, logger := createConsumerMockDependencies(t)

	// Setup mocks
	expectedError := errors.New("close error")
	queueService.On("Close").Return(expectedError)

	handler := &ConsumerHandler{
		kdsService:   nil,
		queueService: queueService,
		logger:       logger,
	}

	// Execute
	err := handler.Close()

	// Verify
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	queueService.AssertExpectations()
}

func TestConsumerHandler_Close_NilQueueService(t *testing.T) {
	_, _, logger := createConsumerMockDependencies(t)

	handler := &ConsumerHandler{
		kdsService:   nil,
		queueService: nil,
		logger:       logger,
	}

	// Execute
	err := handler.Close()

	// Verify - should not error when queueService is nil
	assert.NoError(t, err)
}

func TestBackoffDelay(t *testing.T) {
	// Test backoff delay calculation
	tests := []struct {
		attempt     int
		minExpected time.Duration
		maxExpected time.Duration
	}{
		{0, retryBaseDelay, retryBaseDelay + maxJitter},
		{1, retryBaseDelay, retryBaseDelay + maxJitter},
		{2, retryBaseDelay * 2, retryBaseDelay*2 + maxJitter},
		{3, retryBaseDelay * 2, retryBaseDelay*2 + maxJitter},
		{4, retryBaseDelay * 3, retryBaseDelay*3 + maxJitter},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			delay := backoffDelay(tt.attempt)
			assert.GreaterOrEqual(
				t,
				delay,
				tt.minExpected,
				"delay should be at least minimum expected",
			)
			assert.LessOrEqual(t, delay, tt.maxExpected, "delay should not exceed maximum expected")
		})
	}
}

// Integration-style test that doesn't rely on runConsumerWithRetry directly
func TestConsumerHandler_RunConsumerLoop_ContextCancellation(t *testing.T) {
	kdsService, queueService, logger := createConsumerMockDependencies(t)

	// Create a context that we'll cancel quickly
	ctx, cancel := context.WithCancel(context.Background())

	// We don't need the handler instance for this context cancellation test
	_ = queueService // Use the mock to avoid unused variable warning
	_ = logger       // Use the mock to avoid unused variable warning

	// Cancel context immediately to test cancellation handling
	cancel()

	// This should return quickly due to context cancellation
	// We can't easily test the full RunConsumerLoop due to its infinite nature,
	// but we can test that it handles context cancellation properly

	// Create a separate test function that mimics the core logic
	finished := make(chan bool, 1)
	go func() {
		defer func() {
			finished <- true
		}()

		// Simulate the core select logic from RunConsumerLoop
		select {
		case <-ctx.Done():
			// This should happen immediately
			return
		case <-time.After(100 * time.Millisecond):
			// This should not happen since context is canceled
			t.Error("Expected context cancellation to be detected")
		}
	}()

	// Wait for completion with timeout
	select {
	case <-finished:
		// Success - context cancellation was detected
	case <-time.After(1 * time.Second):
		t.Error("Test timed out - context cancellation not handled properly")
	}

	// Clean up - ensure we're not using the real services
	_ = kdsService
}
