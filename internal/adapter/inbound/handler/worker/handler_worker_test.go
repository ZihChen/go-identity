package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/hibiken/asynq"
	jsoniter "github.com/json-iterator/go"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/queue"
	"github.com/jvdiamondtech/ms-identity-cat/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Setup function for tests
func setupWorkerTest(
	t *testing.T,
) (*mocks.MerchantUseCaseMock, *mocks.PlayerUseCaseMock, *mocks.ManagerUseCaseMock, *mocks.TagUseCaseMock, *WorkerHandler) {
	merchantUseCase := mocks.NewMerchantUseCaseMock(t)
	playerUseCase := mocks.NewPlayerUseCaseMock(t)
	managerUseCase := mocks.NewManagerUseCaseMock(t)
	tagUseCase := mocks.NewTagUseCaseMock(t)
	mockLevelUseCase := mocks.NewPlayerLevelUseCaseMock(t)
	logger := mocks.NewMockLogger(t)
	mockTracer := mocks.NewTracingServiceMock(t)

	handler := NewWorkerHandler(
		merchantUseCase,
		playerUseCase,
		managerUseCase,
		tagUseCase,
		mockLevelUseCase,
		logger,
		mockTracer,
	)

	return merchantUseCase, playerUseCase, managerUseCase, tagUseCase, handler
}

// Tests for HandleMerchantSync
func TestWorkerHandler_HandleMerchantSync_Success(t *testing.T) {
	// Setup
	merchantUseCase, _, _, _, handler := setupWorkerTest(t)

	// Create a valid CloudEvent with MerchantSyncEvent data
	cloudEvent := event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "merchant.sync",
		Source:          "test",
		Subject:         "test",
		ID:              "test-id",
		Time:            time.Now(),
		DataContentType: "application/json",
		Data: map[string]interface{}{
			"global_merchant_id": "test-merchant-id",
			"merchant": map[string]interface{}{
				"id":                 1,
				"name":               "Test Merchant",
				"display_name":       "Test Merchant Display",
				"global_merchant_id": "test-merchant-id",
				"updated_at":         time.Now(),
			},
		},
	}

	// Marshal the CloudEvent to JSON
	payload, err := jsoniter.Marshal(cloudEvent)
	assert.NoError(t, err)

	task := asynq.NewTask(queue.TypeMerchantSync, payload)

	// Setup expectations - the handler will unmarshal the CloudEvent and pass a MerchantSyncEvent to SyncMerchant
	merchantUseCase.On("SyncMerchant", mock.Anything, mock.MatchedBy(func(event *event.MerchantSyncEvent) bool {
		return event.GlobalMerchantID == "test-merchant-id"
	})).
		Return(nil)

	// Setup context
	ctx := context.Background()

	// Execute
	err = handler.HandleMerchantSync(ctx, task)

	// Assert
	assert.NoError(t, err)
	merchantUseCase.AssertExpectations()
}

func TestWorkerHandler_HandleMerchantSync_Error(t *testing.T) {
	// Setup
	merchantUseCase, _, _, _, handler := setupWorkerTest(t)

	// Create task
	payload := []byte(`{"test":"data"}`)
	task := asynq.NewTask(queue.TypeMerchantSync, payload)

	// Setup expectations
	expectedErr := errors.New("sync error")
	merchantUseCase.On("SyncMerchant", mock.Anything, mock.AnythingOfType("*event.MerchantSyncEvent")).
		Return(expectedErr)

	// Setup context
	ctx := context.Background()

	// Execute
	err := handler.HandleMerchantSync(ctx, task)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to sync merchant")
	merchantUseCase.AssertExpectations()
}

// Tests for HandlePlayerSync
func TestWorkerHandler_HandlePlayerSync_Success(t *testing.T) {
	// Setup
	_, playerUseCase, _, tagUseCase, handler := setupWorkerTest(t)

	// Create task
	payload := []byte(`{"test":"data"}`)
	task := asynq.NewTask(queue.TypePlayerSync, payload)

	// Setup expectations
	playerUseCase.On("SyncPlayer", mock.Anything, mock.Anything, mock.Anything).
		Return(nil)
	tagUseCase.On("SyncPlayerTag", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// Setup context
	ctx := context.Background()

	// Execute
	err := handler.HandlePlayerSync(ctx, task)

	// Assert
	assert.NoError(t, err)
	playerUseCase.AssertExpectations()
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
	playerUseCase.On("SyncPlayer", mock.Anything, mock.Anything, mock.Anything).
		Return(expectedErr)

	// Setup context
	ctx := context.Background()

	// Execute
	err := handler.HandlePlayerSync(ctx, task)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to sync player")
	playerUseCase.AssertExpectations()
}

// Tests for HandlePlayerSync with tags and level data
func TestWorkerHandler_HandlePlayerSync_WithTagsAndLevel_Success(t *testing.T) {
	// Setup
	_, playerUseCase, _, tagUseCase, handler := setupWorkerTest(t)
	mockLevelUseCase := mocks.NewPlayerLevelUseCaseMock(t)
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
	playerUseCase.On("SyncPlayer", mock.Anything, mock.AnythingOfType("*event.PlayerSyncEvent")).
		Return(nil)
	tagUseCase.On("SyncPlayerTag", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil)

	// Setup context
	ctx := context.Background()

	// Execute
	err := handler.HandlePlayerSync(ctx, task)

	// Assert
	assert.NoError(t, err)
	playerUseCase.AssertExpectations()
	tagUseCase.AssertExpectations()
	mockLevelUseCase.AssertExpectations()
}

// Tests for HandlePlayerSync with level data error
func TestWorkerHandler_HandlePlayerSync_LevelError(t *testing.T) {
	// Setup
	_, playerUseCase, _, _, handler := setupWorkerTest(t)
	mockLevelUseCase := mocks.NewPlayerLevelUseCaseMock(t)
	handler.levelUseCase = mockLevelUseCase

	// Setup expectations
	playerUseCase.On("SyncPlayer", mock.Anything, mock.AnythingOfType("*event.PlayerSyncEvent")).
		Return(nil)

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

	// Setup context
	ctx := context.Background()

	// Execute
	err := handler.HandlePlayerSync(ctx, task)

	// Assert
	assert.NoError(t, err)
	playerUseCase.AssertExpectations()
	mockLevelUseCase.AssertExpectations()
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
	playerUseCase.On("SyncPlayer", mock.Anything, mock.AnythingOfType("*event.PlayerSyncEvent")).
		Return(nil)
	expectedErr := errors.New("tag sync error")
	tagUseCase.On("SyncPlayerTag", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(expectedErr)

	// Setup context
	ctx := context.Background()

	// Execute
	err := handler.HandlePlayerSync(ctx, task)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to sync player tags")
	playerUseCase.AssertExpectations()
	tagUseCase.AssertExpectations()
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
	playerUseCase.On("SyncPlayer", mock.Anything, mock.AnythingOfType("*event.PlayerSyncEvent")).
		Return(nil)
	expectedErr := errors.New("player tag relation error")
	tagUseCase.On("SyncPlayerTag", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(expectedErr)

	// Setup context
	ctx := context.Background()

	// Execute
	err := handler.HandlePlayerSync(ctx, task)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to sync player tags relation")
	playerUseCase.AssertExpectations()
	tagUseCase.AssertExpectations()
}

// Tests for HandleManagerSync
func TestWorkerHandler_HandleManagerSync_Success(t *testing.T) {
	// Setup
	_, _, managerUseCase, _, handler := setupWorkerTest(t)

	// Create task
	payload := []byte(`{"test":"data"}`)
	task := asynq.NewTask(queue.TypeManagerSync, payload)

	// Setup expectations
	managerUseCase.On("SyncManager", mock.Anything, mock.AnythingOfType("*event.ManagerSyncEvent")).
		Return(nil)

	// Setup context
	ctx := context.Background()

	// Execute
	err := handler.HandleManagerSync(ctx, task)

	// Assert
	assert.NoError(t, err)
	managerUseCase.AssertExpectations()
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
	managerUseCase.On("SyncManager", mock.Anything, mock.AnythingOfType("*event.ManagerSyncEvent")).
		Return(expectedErr)

	// Setup context
	ctx := context.Background()

	// Execute
	err := handler.HandleManagerSync(ctx, task)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to sync manager")
	managerUseCase.AssertExpectations()
}
