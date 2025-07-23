package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/queue"
	"github.com/jvdiamondtech/ms-identity-cat/test/helper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockTagUseCase is a mock implementation of the TagUseCase interface
type MockTagUseCase struct {
	mock.Mock
}

func (m *MockTagUseCase) SyncTag(
	ctx context.Context,
	data []event.TagData,
	globalMerchantID, traceParent string,
) error {
	args := m.Called(ctx, data, globalMerchantID, traceParent)
	return args.Error(0)
}

func (m *MockTagUseCase) SyncPlayerTag(
	ctx context.Context,
	globalPlayerID string,
	data []event.TagData,
) error {
	args := m.Called(ctx, globalPlayerID, data)
	return args.Error(0)
}

type MockLevelUseCase struct {
	mock.Mock
}

func (m *MockLevelUseCase) SyncPlayerLevel(
	ctx context.Context,
	data *event.LevelData,
	globalMerchantID, traceParent string,
) (uint64, error) {
	args := m.Called(ctx, data, globalMerchantID, traceParent)
	return args.Get(0).(uint64), args.Error(1)
}

// Setup function for tests
func setupWorkerTest(
	t *testing.T,
) (*MockMerchantUseCase, *MockPlayerUseCase, *MockManagerUseCase, *MockTagUseCase, *WorkerHandler) {
	merchantUseCase := new(MockMerchantUseCase)
	playerUseCase := new(MockPlayerUseCase)
	managerUseCase := new(MockManagerUseCase)
	tagUseCase := new(MockTagUseCase)
	mockLevelUseCase := new(MockLevelUseCase)
	logger := helper.SetupLoggerMock(t)

	handler := NewWorkerHandler(
		merchantUseCase,
		playerUseCase,
		managerUseCase,
		tagUseCase,
		mockLevelUseCase,
		logger,
	)

	return merchantUseCase, playerUseCase, managerUseCase, tagUseCase, handler
}

// Tests for HandleMerchantSync
func TestWorkerHandler_HandleMerchantSync_Success(t *testing.T) {
	// Setup
	merchantUseCase, _, _, _, handler := setupWorkerTest(t)

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
	_, _, _, _, handler := setupWorkerTest(t)

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
	merchantUseCase, _, _, _, handler := setupWorkerTest(t)

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
	_, playerUseCase, _, _, handler := setupWorkerTest(t)

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
	_, _, _, _, handler := setupWorkerTest(t)

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
	_, playerUseCase, _, _, handler := setupWorkerTest(t)

	// Create task
	payload := []byte(`{"test":"data"}`)
	task := asynq.NewTask(queue.TypePlayerSync, payload)

	// Setup expectations
	expectedErr := errors.New("sync error")
	playerUseCase.On("SyncPlayer", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(expectedErr)

	// Setup context
	ctx := context.Background()

	// Execute
	err := handler.HandlePlayerSync(ctx, task)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to sync player")
	playerUseCase.AssertExpectations(t)
}

// Tests for HandlePlayerSync with tags and level data
func TestWorkerHandler_HandlePlayerSync_WithTagsAndLevel_Success(t *testing.T) {
	// Setup
	_, playerUseCase, _, tagUseCase, handler := setupWorkerTest(t)
	mockLevelUseCase := new(MockLevelUseCase)
	handler.levelUseCase = mockLevelUseCase

	// Create task with player, tags, and level data
	payload := []byte(`{
		"id": "test-id",
		"type": "player.sync",
		"source": "test-source",
		"data": {
			"player": {
				"global_player_id": "test-player-id"
			},
			"player_tags": [
				{
					"global_tag_id": "tag1",
					"name": "Tag 1"
				}
			],
			"player_level": {
				"global_player_level_id": "level1",
				"name": "Level 1"
			},
			"global_merchant_id": "merchant1"
		},
		"traceparent": "test-trace"
	}`)
	task := asynq.NewTask(queue.TypePlayerSync, payload)

	// Setup expectations
	mockLevelUseCase.On("SyncPlayerLevel", mock.Anything, mock.Anything, "merchant1", "test-trace").
		Return(uint64(1), nil)
	playerUseCase.On("SyncPlayer", mock.Anything, mock.Anything, "merchant1", "test-trace").
		Return(nil)
	tagUseCase.On("SyncTag", mock.Anything, mock.Anything, "merchant1", "test-trace").Return(nil)
	tagUseCase.On("SyncPlayerTag", mock.Anything, "test-player-id", mock.Anything).Return(nil)

	// Setup context
	ctx := context.Background()

	// Execute
	err := handler.HandlePlayerSync(ctx, task)

	// Assert
	assert.NoError(t, err)
	playerUseCase.AssertExpectations(t)
	tagUseCase.AssertExpectations(t)
	mockLevelUseCase.AssertExpectations(t)
}

// Tests for HandlePlayerSync with level data error
func TestWorkerHandler_HandlePlayerSync_LevelError(t *testing.T) {
	// Setup
	_, _, _, _, handler := setupWorkerTest(t)
	mockLevelUseCase := new(MockLevelUseCase)
	handler.levelUseCase = mockLevelUseCase

	// Create task with player and level data
	payload := []byte(`{
		"id": "test-id",
		"type": "player.sync",
		"source": "test-source",
		"data": {
			"player": {
				"global_player_id": "test-player-id"
			},
			"player_level": {
				"global_player_level_id": "level1",
				"name": "Level 1"
			},
			"global_merchant_id": "merchant1"
		},
		"traceparent": "test-trace"
	}`)
	task := asynq.NewTask(queue.TypePlayerSync, payload)

	// Setup expectations
	expectedErr := errors.New("level sync error")
	mockLevelUseCase.On("SyncPlayerLevel", mock.Anything, mock.Anything, "merchant1", "test-trace").
		Return(uint64(0), expectedErr)

	// Setup context
	ctx := context.Background()

	// Execute
	err := handler.HandlePlayerSync(ctx, task)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to sync player level")
	mockLevelUseCase.AssertExpectations(t)
}

// Tests for HandlePlayerSync with tag error
func TestWorkerHandler_HandlePlayerSync_TagError(t *testing.T) {
	// Setup
	_, playerUseCase, _, tagUseCase, handler := setupWorkerTest(t)

	// Create task with player and tags data
	payload := []byte(`{
		"id": "test-id",
		"type": "player.sync",
		"source": "test-source",
		"data": {
			"player": {
				"global_player_id": "test-player-id"
			},
			"player_tags": [
				{
					"global_tag_id": "tag1",
					"name": "Tag 1"
				}
			],
			"global_merchant_id": "merchant1"
		},
		"traceparent": "test-trace"
	}`)
	task := asynq.NewTask(queue.TypePlayerSync, payload)

	// Setup expectations
	playerUseCase.On("SyncPlayer", mock.Anything, mock.Anything, "merchant1", "test-trace").
		Return(nil)
	expectedErr := errors.New("tag sync error")
	tagUseCase.On("SyncTag", mock.Anything, mock.Anything, "merchant1", "test-trace").
		Return(expectedErr)

	// Setup context
	ctx := context.Background()

	// Execute
	err := handler.HandlePlayerSync(ctx, task)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to sync player tags")
	playerUseCase.AssertExpectations(t)
	tagUseCase.AssertExpectations(t)
}

// Tests for HandlePlayerSync with player tag relation error
func TestWorkerHandler_HandlePlayerSync_PlayerTagRelationError(t *testing.T) {
	// Setup
	_, playerUseCase, _, tagUseCase, handler := setupWorkerTest(t)

	// Create task with player and tags data
	payload := []byte(`{
		"id": "test-id",
		"type": "player.sync",
		"source": "test-source",
		"data": {
			"player": {
				"global_player_id": "test-player-id"
			},
			"player_tags": [
				{
					"global_tag_id": "tag1",
					"name": "Tag 1"
				}
			],
			"global_merchant_id": "merchant1"
		},
		"traceparent": "test-trace"
	}`)
	task := asynq.NewTask(queue.TypePlayerSync, payload)

	// Setup expectations
	playerUseCase.On("SyncPlayer", mock.Anything, mock.Anything, "merchant1", "test-trace").
		Return(nil)
	tagUseCase.On("SyncTag", mock.Anything, mock.Anything, "merchant1", "test-trace").Return(nil)
	expectedErr := errors.New("player tag relation error")
	tagUseCase.On("SyncPlayerTag", mock.Anything, "test-player-id", mock.Anything).
		Return(expectedErr)

	// Setup context
	ctx := context.Background()

	// Execute
	err := handler.HandlePlayerSync(ctx, task)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to sync player tags relation")
	playerUseCase.AssertExpectations(t)
	tagUseCase.AssertExpectations(t)
}

// Tests for HandleManagerSync
func TestWorkerHandler_HandleManagerSync_Success(t *testing.T) {
	// Setup
	_, _, managerUseCase, _, handler := setupWorkerTest(t)

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
	_, _, _, _, handler := setupWorkerTest(t)

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
	_, _, managerUseCase, _, handler := setupWorkerTest(t)

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
