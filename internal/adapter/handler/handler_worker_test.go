package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/queue"
	"github.com/jvdiamondtech/ms-identity-cat/test/helper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Setup function for tests
func setupWorkerTest(
	t *testing.T,
) (*MockMerchantUseCase, *MockPlayerUseCase, *MockManagerUseCase, *WorkerHandler) {
	merchantUseCase := new(MockMerchantUseCase)
	playerUseCase := new(MockPlayerUseCase)
	managerUseCase := new(MockManagerUseCase)
	logger := helper.SetupLoggerMock(t)

	handler := NewWorkerHandler(
		merchantUseCase,
		playerUseCase,
		managerUseCase,
		logger,
	)

	return merchantUseCase, playerUseCase, managerUseCase, handler
}

// Tests for HandleMerchantSync
func TestWorkerHandler_HandleMerchantSync_Success(t *testing.T) {
	// Setup
	merchantUseCase, _, _, handler := setupWorkerTest(t)

	// Create task
	payload := []byte(`{"test":"data"}`)
	task := asynq.NewTask(queue.TypeMerchantSync, payload)

	// Setup expectations
	merchantUseCase.On("SyncMerchant", mock.Anything, payload).Return(nil)

	// Setup context
	ctx := context.Background()

	// Execute
	err := handler.HandleMerchantSync(ctx, task)

	// Assert
	assert.NoError(t, err)
	merchantUseCase.AssertExpectations(t)
}

func TestWorkerHandler_HandleMerchantSync_NilTask(t *testing.T) {
	// Setup
	_, _, _, handler := setupWorkerTest(t)

	// Setup context
	ctx := context.Background()

	// Execute
	err := handler.HandleMerchantSync(ctx, nil)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task is empty")
}

func TestWorkerHandler_HandleMerchantSync_Error(t *testing.T) {
	// Setup
	merchantUseCase, _, _, handler := setupWorkerTest(t)

	// Create task
	payload := []byte(`{"test":"data"}`)
	task := asynq.NewTask(queue.TypeMerchantSync, payload)

	// Setup expectations
	expectedErr := errors.New("sync error")
	merchantUseCase.On("SyncMerchant", mock.Anything, payload).Return(expectedErr)

	// Setup context
	ctx := context.Background()

	// Execute
	err := handler.HandleMerchantSync(ctx, task)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to sync merchant")
	merchantUseCase.AssertExpectations(t)
}

// Tests for HandlePlayerSync
func TestWorkerHandler_HandlePlayerSync_Success(t *testing.T) {
	// Setup
	_, playerUseCase, _, handler := setupWorkerTest(t)

	// Create task
	payload := []byte(`{"test":"data"}`)
	task := asynq.NewTask(queue.TypePlayerSync, payload)

	// Setup expectations
	playerUseCase.On("SyncPlayer", mock.Anything, payload).Return(nil)

	// Setup context
	ctx := context.Background()

	// Execute
	err := handler.HandlePlayerSync(ctx, task)

	// Assert
	assert.NoError(t, err)
	playerUseCase.AssertExpectations(t)
}

func TestWorkerHandler_HandlePlayerSync_NilTask(t *testing.T) {
	// Setup
	_, _, _, handler := setupWorkerTest(t)

	// Setup context
	ctx := context.Background()

	// Execute
	err := handler.HandlePlayerSync(ctx, nil)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task is empty")
}

func TestWorkerHandler_HandlePlayerSync_Error(t *testing.T) {
	// Setup
	_, playerUseCase, _, handler := setupWorkerTest(t)

	// Create task
	payload := []byte(`{"test":"data"}`)
	task := asynq.NewTask(queue.TypePlayerSync, payload)

	// Setup expectations
	expectedErr := errors.New("sync error")
	playerUseCase.On("SyncPlayer", mock.Anything, payload).Return(expectedErr)

	// Setup context
	ctx := context.Background()

	// Execute
	err := handler.HandlePlayerSync(ctx, task)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to sync player")
	playerUseCase.AssertExpectations(t)
}

// Tests for HandleManagerSync
func TestWorkerHandler_HandleManagerSync_Success(t *testing.T) {
	// Setup
	_, _, managerUseCase, handler := setupWorkerTest(t)

	// Create task
	payload := []byte(`{"test":"data"}`)
	task := asynq.NewTask(queue.TypeManagerSync, payload)

	// Setup expectations
	managerUseCase.On("SyncManager", mock.Anything, payload).Return(nil)

	// Setup context
	ctx := context.Background()

	// Execute
	err := handler.HandleManagerSync(ctx, task)

	// Assert
	assert.NoError(t, err)
	managerUseCase.AssertExpectations(t)
}

func TestWorkerHandler_HandleManagerSync_NilTask(t *testing.T) {
	// Setup
	_, _, _, handler := setupWorkerTest(t)

	// Setup context
	ctx := context.Background()

	// Execute
	err := handler.HandleManagerSync(ctx, nil)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task is empty")
}

func TestWorkerHandler_HandleManagerSync_Error(t *testing.T) {
	// Setup
	_, _, managerUseCase, handler := setupWorkerTest(t)

	// Create task
	payload := []byte(`{"test":"data"}`)
	task := asynq.NewTask(queue.TypeManagerSync, payload)

	// Setup expectations
	expectedErr := errors.New("sync error")
	managerUseCase.On("SyncManager", mock.Anything, payload).Return(expectedErr)

	// Setup context
	ctx := context.Background()

	// Execute
	err := handler.HandleManagerSync(ctx, task)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to sync manager")
	managerUseCase.AssertExpectations(t)
}
