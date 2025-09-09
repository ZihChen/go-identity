package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/test/helper"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations are now in test/helper/repository_mock.go

// Helper functions
func createMockDependencies(
	t *testing.T,
) (*helper.MockPlayerRepository, *helper.MockMerchantRepository, *helper.MockLevelRepository, *helper.MockEventProducer, *helper.MockLogger, *redis.Client) {
	playerRepo := new(helper.MockPlayerRepository)
	merchantRepo := new(helper.MockMerchantRepository)
	levelRepo := new(helper.MockLevelRepository)
	eventProducer := new(helper.MockEventProducer)
	logger := helper.SetupLoggerMock(t)
	redisClient, _ := redismock.NewClientMock()
	return playerRepo, merchantRepo, levelRepo, eventProducer, logger, redisClient
}

func createPlayerSyncEvent() *event.CloudEvent {
	PlayerEvent := event.PlayerSyncEvent{
		GlobalMerchantID: "FATCAT-MERCHANT-1",
		Player: event.PlayerData{
			GlobalPlayerID: "FATCAT-PLAYER-1",
			Account:        "TestPlayer",
			Email:          "test@example.com",
			Status:         "active",
		},
	}

	cloudEvent := &event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "tw.jvd.fatidentitycat.player.sync.v1",
		Source:          "/fatidentitycat/FATCAT",
		Subject:         "player_sync",
		ID:              uuid.New().String(),
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		Data:            PlayerEvent,
	}

	return cloudEvent
}

// Tests
func TestNewPlayerUseCase(t *testing.T) {
	playerRepo, merchantRepo, levelRepo, eventProducer, logger, redisClient := createMockDependencies(
		t,
	)

	useCase := NewPlayerUseCase(
		playerRepo,
		merchantRepo,
		levelRepo,
		eventProducer,
		logger,
		redisClient,
	)

	assert.NotNil(t, useCase)
	assert.IsType(t, &PlayerUseCase{}, useCase)
}

func TestPlayerUseCase_SyncPlayer_Upsert(t *testing.T) {
	ctx := helper.CreateTestContext()
	playerRepo, merchantRepo, levelRepo, eventProducer, logger, redisClient := createMockDependencies(
		t,
	)

	// Setup mocks
	merchant := helper.CreateTestMerchant()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)

	// Player doesn't exist yet
	playerRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*entity.Player")).
		Return(nil)

	// Expect PublishPlayerSync to be called
	eventProducer.On("PublishPlayerSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).
		Return(nil)

	// Create the use case
	useCase := NewPlayerUseCase(
		playerRepo,
		merchantRepo,
		levelRepo,
		eventProducer,
		logger,
		redisClient,
	)

	// Create test event data
	eventData := createPlayerSyncEvent()
	dataBytes, err := jsoniter.Marshal(eventData.Data)
	assert.NoError(t, err)
	var playerEvent event.PlayerSyncEvent
	err = jsoniter.Unmarshal(dataBytes, &playerEvent)
	assert.NoError(t, err)

	// Execute the function
	err = useCase.SyncPlayer(
		ctx,
		&playerEvent,
	)

	// Verify results
	assert.NoError(t, err)
	playerRepo.AssertExpectations(t)
	merchantRepo.AssertExpectations(t)
	eventProducer.AssertExpectations(t)
}

func TestPlayerUseCase_SyncPlayer_UpsertError(t *testing.T) {
	ctx := helper.CreateTestContext()
	playerRepo, merchantRepo, levelRepo, eventProducer, logger, redisClient := createMockDependencies(
		t,
	)

	// Setup mocks
	merchant := helper.CreateTestMerchant()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)

	// Player doesn't exist yet
	playerRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*entity.Player")).
		Return(errors.New("timestamp-based upsert failed"))

	// Create the use case
	useCase := NewPlayerUseCase(
		playerRepo,
		merchantRepo,
		levelRepo,
		eventProducer,
		logger,
		redisClient,
	)

	// Create test event data
	eventData := createPlayerSyncEvent()
	dataBytes, err := jsoniter.Marshal(eventData.Data)
	assert.NoError(t, err)
	var playerEvent event.PlayerSyncEvent
	err = jsoniter.Unmarshal(dataBytes, &playerEvent)
	assert.NoError(t, err)

	// Execute the function
	err = useCase.SyncPlayer(
		ctx,
		&playerEvent,
	)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "upsert player")
	playerRepo.AssertExpectations(t)
	merchantRepo.AssertExpectations(t)
}

func TestPlayerUseCase_GetPlayerByID(t *testing.T) {
	ctx := helper.CreateTestContext()
	playerRepo, merchantRepo, levelRepo, eventProducer, logger, redisClient := createMockDependencies(
		t,
	)

	// Setup mocks
	player := helper.CreateTestPlayer()
	playerRepo.On("FindByID", mock.Anything, uint64(1)).Return(player, nil)

	// Create the use case
	useCase := NewPlayerUseCase(
		playerRepo,
		merchantRepo,
		levelRepo,
		eventProducer,
		logger,
		redisClient,
	)

	// Execute the function
	result, err := useCase.GetPlayerByID(ctx, 1)

	// Verify results
	assert.NoError(t, err)
	assert.Equal(t, player, result)
	playerRepo.AssertExpectations(t)
}

func TestPlayerUseCase_GetPlayerByID_NotFound(t *testing.T) {
	ctx := helper.CreateTestContext()
	playerRepo, merchantRepo, levelRepo, eventProducer, logger, redisClient := createMockDependencies(
		t,
	)

	// Setup mocks - player not found
	playerRepo.On("FindByID", mock.Anything, uint64(999)).
		Return(nil, errors.New("player not found"))

	// Create the use case
	useCase := NewPlayerUseCase(
		playerRepo,
		merchantRepo,
		levelRepo,
		eventProducer,
		logger,
		redisClient,
	)

	// Execute the function
	result, err := useCase.GetPlayerByID(ctx, 999)

	// Verify results
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "find player")
	playerRepo.AssertExpectations(t)
}

func TestPlayerUseCase_GetPlayerByGlobalID(t *testing.T) {
	ctx := helper.CreateTestContext()
	playerRepo, merchantRepo, levelRepo, eventProducer, logger, redisClient := createMockDependencies(
		t,
	)

	// Setup mocks
	player := helper.CreateTestPlayer()
	playerRepo.On("FindByGlobalID", mock.Anything, "FATCAT-PLAYER-1").Return(player, nil)

	// Create the use case
	useCase := NewPlayerUseCase(
		playerRepo,
		merchantRepo,
		levelRepo,
		eventProducer,
		logger,
		redisClient,
	)

	// Execute the function
	result, err := useCase.GetPlayerByGlobalID(ctx, "FATCAT-PLAYER-1")

	// Verify results
	assert.NoError(t, err)
	assert.Equal(t, player, result)
	playerRepo.AssertExpectations(t)
}

func TestPlayerUseCase_GetPlayerByGlobalID_NotFound(t *testing.T) {
	ctx := helper.CreateTestContext()
	playerRepo, merchantRepo, levelRepo, eventProducer, logger, redisClient := createMockDependencies(
		t,
	)

	// Setup mocks - player not found
	playerRepo.On("FindByGlobalID", mock.Anything, "NONEXISTENT").
		Return(nil, errors.New("player not found"))

	// Create the use case
	useCase := NewPlayerUseCase(
		playerRepo,
		merchantRepo,
		levelRepo,
		eventProducer,
		logger,
		redisClient,
	)

	// Execute the function
	result, err := useCase.GetPlayerByGlobalID(ctx, "NONEXISTENT")

	// Verify results
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "find player")
	playerRepo.AssertExpectations(t)
}

func TestPlayerUseCase_UpdatePlayerLastActive(t *testing.T) {
	ctx := helper.CreateTestContext()
	playerRepo, merchantRepo, levelRepo, eventProducer, logger, redisClient := createMockDependencies(
		t,
	)

	// Setup mocks
	player := helper.CreateTestPlayer()
	playerRepo.On("FindByID", mock.Anything, uint64(1)).Return(player, nil)
	playerRepo.On("Update", mock.Anything, mock.AnythingOfType("*entity.Player")).Return(nil)

	// Create the use case
	useCase := NewPlayerUseCase(
		playerRepo,
		merchantRepo,
		levelRepo,
		eventProducer,
		logger,
		redisClient,
	)

	// Execute the function
	err := useCase.UpdatePlayerLastActive(ctx, 1)

	// Verify results
	assert.NoError(t, err)
	playerRepo.AssertExpectations(t)
}

func TestPlayerUseCase_UpdatePlayerLastActive_NotFound(t *testing.T) {
	ctx := helper.CreateTestContext()
	playerRepo, merchantRepo, levelRepo, eventProducer, logger, redisClient := createMockDependencies(
		t,
	)

	// Setup mocks - player not found
	playerRepo.On("FindByID", mock.Anything, uint64(999)).
		Return(nil, errors.New("player not found"))

	// Create the use case
	useCase := NewPlayerUseCase(
		playerRepo,
		merchantRepo,
		levelRepo,
		eventProducer,
		logger,
		redisClient,
	)

	// Execute the function
	err := useCase.UpdatePlayerLastActive(ctx, 999)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "find player")
	playerRepo.AssertExpectations(t)
}

func TestPlayerUseCase_UpdatePlayerLastActive_UpdateError(t *testing.T) {
	ctx := helper.CreateTestContext()
	playerRepo, merchantRepo, levelRepo, eventProducer, logger, redisClient := createMockDependencies(
		t,
	)

	// Setup mocks
	player := helper.CreateTestPlayer()
	playerRepo.On("FindByID", mock.Anything, uint64(1)).Return(player, nil)

	// Update fails
	playerRepo.On("Update", mock.Anything, mock.AnythingOfType("*entity.Player")).
		Return(errors.New("update error"))

	// Create the use case
	useCase := NewPlayerUseCase(
		playerRepo,
		merchantRepo,
		levelRepo,
		eventProducer,
		logger,
		redisClient,
	)

	// Execute the function
	err := useCase.UpdatePlayerLastActive(ctx, 1)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update player")
	playerRepo.AssertExpectations(t)
}
