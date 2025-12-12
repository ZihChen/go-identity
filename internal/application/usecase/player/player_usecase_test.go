package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/test/factories"
	"github.com/jvdiamondtech/ms-identity-cat/test/mocks"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations are now in test/helper/repository_mock.go

// Helper functions
func createMockDependencies(
	t *testing.T,
) (*mocks.PlayerRepositoryMock, *mocks.MerchantRepositoryMock, *mocks.LevelRepositoryMock, *mocks.EventProducerMock, *mocks.MockLogger, *redis.Client) {
	playerRepo := mocks.NewPlayerRepositoryMock(t)
	merchantRepo := mocks.NewMerchantRepositoryMock(t)
	levelRepo := mocks.NewLevelRepositoryMock(t)
	eventProducer := mocks.NewEventProducerMock(t)
	logger := mocks.NewMockLogger(t)
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
		mocks.NewNilTracingService(),
	)

	assert.NotNil(t, useCase)
	assert.IsType(t, &PlayerUseCase{}, useCase)
}

func TestPlayerUseCase_SyncPlayer_Upsert(t *testing.T) {
	ctx := factories.CreateTestContext()
	playerRepo, merchantRepo, levelRepo, eventProducer, logger, redisClient := createMockDependencies(
		t,
	)

	// Setup mocks for batch processing
	merchant := factories.CreateTestMerchant()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)

	// Use BatchUpsert instead of Upsert since we're using batch processor
	playerRepo.On("BatchUpsert", mock.Anything, mock.AnythingOfType("[]*entity.Player")).
		Return(nil)

	// Expect BatchPublishPlayerSync to be called
	eventProducer.On("BatchPublishPlayerSync", mock.Anything, mock.AnythingOfType("[]*entity.Player"), mock.AnythingOfType("[]string")).
		Return(nil)

	// Setup logger
	logger.On("InfoLog", mock.AnythingOfType("string"), mock.Anything).Return()
	logger.On("InfoWithContext", mock.Anything, mock.AnythingOfType("string"), mock.Anything).
		Return()
	logger.On("ErrorLog", mock.AnythingOfType("string"), mock.Anything).Return()

	// Create the use case
	useCase := NewPlayerUseCase(
		playerRepo,
		merchantRepo,
		levelRepo,
		eventProducer,
		logger,
		redisClient,
		mocks.NewNilTracingService(),
	)

	// 啟動批次處理器
	err := useCase.StartBatchProcessor(ctx)
	assert.NoError(t, err)
	defer func() {
		stopErr := useCase.StopBatchProcessor(ctx)
		assert.NoError(t, stopErr)
	}()

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
	playerRepo.AssertExpectations()
	merchantRepo.AssertExpectations()
	eventProducer.AssertExpectations()
}

func TestPlayerUseCase_SyncPlayer_UpsertError(t *testing.T) {
	ctx := factories.CreateTestContext()
	playerRepo, merchantRepo, levelRepo, eventProducer, logger, redisClient := createMockDependencies(
		t,
	)

	// Setup mocks for batch processing with error
	merchant := factories.CreateTestMerchant()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)

	// BatchUpsert fails
	playerRepo.On("BatchUpsert", mock.Anything, mock.AnythingOfType("[]*entity.Player")).
		Return(errors.New("timestamp-based upsert failed"))

	// Setup logger
	logger.On("InfoLog", mock.AnythingOfType("string"), mock.Anything).Return()
	logger.On("ErrorLog", mock.AnythingOfType("string"), mock.Anything).Return()

	// Create the use case
	useCase := NewPlayerUseCase(
		playerRepo,
		merchantRepo,
		levelRepo,
		eventProducer,
		logger,
		redisClient,
		mocks.NewNilTracingService(),
	)

	// 啟動批次處理器
	err := useCase.StartBatchProcessor(ctx)
	assert.NoError(t, err)
	defer func() {
		stopErr := useCase.StopBatchProcessor(ctx)
		assert.NoError(t, stopErr)
	}()

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
	assert.Contains(t, err.Error(), "batch process player")
	playerRepo.AssertExpectations()
	merchantRepo.AssertExpectations()
}

func TestPlayerUseCase_GetPlayerByID(t *testing.T) {
	ctx := factories.CreateTestContext()
	playerRepo, merchantRepo, levelRepo, eventProducer, logger, redisClient := createMockDependencies(
		t,
	)

	// Setup mocks
	player := factories.CreateTestPlayer()
	playerRepo.On("FindByID", mock.Anything, uint64(1)).Return(player, nil)

	// Create the use case
	useCase := NewPlayerUseCase(
		playerRepo,
		merchantRepo,
		levelRepo,
		eventProducer,
		logger,
		redisClient,
		mocks.NewNilTracingService(),
	)

	// Execute the function
	result, err := useCase.GetPlayerByID(ctx, 1)

	// Verify results
	assert.NoError(t, err)
	assert.Equal(t, player, result)
	playerRepo.AssertExpectations()
}

func TestPlayerUseCase_GetPlayerByID_NotFound(t *testing.T) {
	ctx := factories.CreateTestContext()
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
		mocks.NewNilTracingService(),
	)

	// Execute the function
	result, err := useCase.GetPlayerByID(ctx, 999)

	// Verify results
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "find player")
	playerRepo.AssertExpectations()
}

func TestPlayerUseCase_GetPlayerByGlobalID(t *testing.T) {
	ctx := factories.CreateTestContext()
	playerRepo, merchantRepo, levelRepo, eventProducer, logger, redisClient := createMockDependencies(
		t,
	)

	// Setup mocks
	player := factories.CreateTestPlayer()
	playerRepo.On("FindByGlobalID", mock.Anything, "FATCAT-PLAYER-1").Return(player, nil)

	// Create the use case
	useCase := NewPlayerUseCase(
		playerRepo,
		merchantRepo,
		levelRepo,
		eventProducer,
		logger,
		redisClient,
		mocks.NewNilTracingService(),
	)

	// Execute the function
	result, err := useCase.GetPlayerByGlobalID(ctx, "FATCAT-PLAYER-1")

	// Verify results
	assert.NoError(t, err)
	assert.Equal(t, player, result)
	playerRepo.AssertExpectations()
}

func TestPlayerUseCase_GetPlayerByGlobalID_NotFound(t *testing.T) {
	ctx := factories.CreateTestContext()
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
		mocks.NewNilTracingService(),
	)

	// Execute the function
	result, err := useCase.GetPlayerByGlobalID(ctx, "NONEXISTENT")

	// Verify results
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "find player")
	playerRepo.AssertExpectations()
}

func TestPlayerUseCase_UpdatePlayerLastActive(t *testing.T) {
	ctx := factories.CreateTestContext()
	playerRepo, merchantRepo, levelRepo, eventProducer, logger, redisClient := createMockDependencies(
		t,
	)

	// Setup mocks
	player := factories.CreateTestPlayer()
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
		mocks.NewNilTracingService(),
	)

	// Execute the function
	err := useCase.UpdatePlayerLastActive(ctx, 1)

	// Verify results
	assert.NoError(t, err)
	playerRepo.AssertExpectations()
}

func TestPlayerUseCase_UpdatePlayerLastActive_NotFound(t *testing.T) {
	ctx := factories.CreateTestContext()
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
		mocks.NewNilTracingService(),
	)

	// Execute the function
	err := useCase.UpdatePlayerLastActive(ctx, 999)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "find player")
	playerRepo.AssertExpectations()
}

func TestPlayerUseCase_UpdatePlayerLastActive_UpdateError(t *testing.T) {
	ctx := factories.CreateTestContext()
	playerRepo, merchantRepo, levelRepo, eventProducer, logger, redisClient := createMockDependencies(
		t,
	)

	// Setup mocks
	player := factories.CreateTestPlayer()
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
		mocks.NewNilTracingService(),
	)

	// Execute the function
	err := useCase.UpdatePlayerLastActive(ctx, 1)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update player")
	playerRepo.AssertExpectations()
}
