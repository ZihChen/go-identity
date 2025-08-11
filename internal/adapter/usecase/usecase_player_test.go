package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	jsoniter "github.com/json-iterator/go"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/test/helper"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock implementations
type MockPlayerRepository struct {
	mock.Mock
}

func (m *MockPlayerRepository) FindByID(ctx context.Context, id uint64) (*entity.Player, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Player), args.Error(1)
}

func (m *MockPlayerRepository) FindByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Player, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Player), args.Error(1)
}

func (m *MockPlayerRepository) FirstOrCreate(ctx context.Context, player *entity.Player) error {
	args := m.Called(ctx, player)
	return args.Error(0)
}

func (m *MockPlayerRepository) Create(ctx context.Context, player *entity.Player) error {
	args := m.Called(ctx, player)
	// Simulate ID assignment like a real database would
	if player.ID == 0 {
		player.ID = 1
	}
	return args.Error(0)
}

func (m *MockPlayerRepository) Update(ctx context.Context, player *entity.Player) error {
	args := m.Called(ctx, player)
	return args.Error(0)
}

func (m *MockPlayerRepository) Delete(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPlayerRepository) Upsert(ctx context.Context, player *entity.Player) error {
	args := m.Called(ctx, player)
	return args.Error(0)
}

type MockLevelRepository struct {
	mock.Mock
}

func (m *MockLevelRepository) Upsert(ctx context.Context, level *entity.Level) error {
	args := m.Called(ctx, level)
	return args.Error(0)
}

func (m *MockLevelRepository) FindByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Level, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Level), args.Error(1)
}

type MockMerchantRepository struct {
	mock.Mock
}

func (m *MockMerchantRepository) FindByID(
	ctx context.Context,
	id uint64,
) (*entity.Merchant, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Merchant), args.Error(1)
}

func (m *MockMerchantRepository) FindByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Merchant, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Merchant), args.Error(1)
}

func (m *MockMerchantRepository) FirstOrCreate(
	ctx context.Context,
	merchant *entity.Merchant,
) error {
	args := m.Called(ctx, merchant)
	return args.Error(0)
}

func (m *MockMerchantRepository) Create(ctx context.Context, merchant *entity.Merchant) error {
	args := m.Called(ctx, merchant)
	return args.Error(0)
}

func (m *MockMerchantRepository) Update(ctx context.Context, merchant *entity.Merchant) error {
	args := m.Called(ctx, merchant)
	return args.Error(0)
}

func (m *MockMerchantRepository) Delete(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockMerchantRepository) Upsert(ctx context.Context, merchant *entity.Merchant) error {
	args := m.Called(ctx, merchant)
	return args.Error(0)
}

type MockEventProducer struct {
	mock.Mock
}

func (m *MockEventProducer) PublishMerchantSync(
	ctx context.Context,
	event *event.CloudEvent,
) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockEventProducer) PublishPlayerSync(ctx context.Context, event *event.CloudEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockEventProducer) PublishManagerSync(ctx context.Context, event *event.CloudEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockEventProducer) PublishPlayerLevelSync(
	ctx context.Context,
	event *event.CloudEvent,
) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockEventProducer) PublishPlayerTagsSync(
	ctx context.Context,
	event *event.CloudEvent,
) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockEventProducer) PublishTagSync(ctx context.Context, event *event.CloudEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

// Helper functions
func createTestContext() context.Context {
	return context.Background()
}

func createMockDependencies(
	t *testing.T,
) (*MockPlayerRepository, *MockMerchantRepository, *MockLevelRepository, *MockEventProducer, *helper.MockLogger, *redis.Client) {
	playerRepo := new(MockPlayerRepository)
	merchantRepo := new(MockMerchantRepository)
	levelRepo := new(MockLevelRepository)
	eventProducer := new(MockEventProducer)
	logger := helper.SetupLoggerMock(t)
	redisClient, _ := redismock.NewClientMock()
	return playerRepo, merchantRepo, levelRepo, eventProducer, logger, redisClient
}

func createTestPlayer() *entity.Player {
	email := "test@example.com"
	now := time.Now()
	lastActive := now.Add(-1 * time.Hour)
	return &entity.Player{
		ID:             1,
		MerchantID:     2,
		GlobalPlayerID: "FATCAT-PLAYER-1",
		APIKey:         "player-api-key",
		Account:        "TestPlayer",
		Email:          &email,
		LastActiveAt:   &lastActive,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func createTestMerchant() *entity.Merchant {
	now := time.Now()
	return &entity.Merchant{
		ID:               2,
		GlobalMerchantID: "FATCAT-MERCHANT-1",
		Name:             "TestMerchant",
		DisplayName:      "Test Merchant",
		APIKey:           "merchant-api-key",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
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
	ctx := createTestContext()
	playerRepo, merchantRepo, levelRepo, eventProducer, logger, redisClient := createMockDependencies(
		t,
	)

	// Setup mocks
	merchant := createTestMerchant()
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
	ctx := createTestContext()
	playerRepo, merchantRepo, levelRepo, eventProducer, logger, redisClient := createMockDependencies(
		t,
	)

	// Setup mocks
	merchant := createTestMerchant()
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
	ctx := createTestContext()
	playerRepo, merchantRepo, levelRepo, eventProducer, logger, redisClient := createMockDependencies(
		t,
	)

	// Setup mocks
	player := createTestPlayer()
	playerRepo.On("FindByID", mock.Anything, uint64(1)).Return(player, nil)

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

	// Execute the function
	result, err := useCase.GetPlayerByID(ctx, 1)

	// Verify results
	assert.NoError(t, err)
	assert.Equal(t, player, result)
	playerRepo.AssertExpectations(t)
	eventProducer.AssertExpectations(t)
}

func TestPlayerUseCase_GetPlayerByID_NotFound(t *testing.T) {
	ctx := createTestContext()
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

func TestPlayerUseCase_GetPlayerByID_PublishError(t *testing.T) {
	ctx := createTestContext()
	playerRepo, merchantRepo, levelRepo, eventProducer, logger, redisClient := createMockDependencies(
		t,
	)

	// Setup mocks
	player := createTestPlayer()
	playerRepo.On("FindByID", mock.Anything, uint64(1)).Return(player, nil)

	// PublishPlayerSync fails
	eventProducer.On("PublishPlayerSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).
		Return(errors.New("publish error"))

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
	assert.Error(t, err)
	assert.Nil(t, result)
	playerRepo.AssertExpectations(t)
	eventProducer.AssertExpectations(t)
}

func TestPlayerUseCase_GetPlayerByGlobalID(t *testing.T) {
	ctx := createTestContext()
	playerRepo, merchantRepo, levelRepo, eventProducer, logger, redisClient := createMockDependencies(
		t,
	)

	// Setup mocks
	player := createTestPlayer()
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
	ctx := createTestContext()
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
	ctx := createTestContext()
	playerRepo, merchantRepo, levelRepo, eventProducer, logger, redisClient := createMockDependencies(
		t,
	)

	// Setup mocks
	player := createTestPlayer()
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
	ctx := createTestContext()
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
	ctx := createTestContext()
	playerRepo, merchantRepo, levelRepo, eventProducer, logger, redisClient := createMockDependencies(
		t,
	)

	// Setup mocks
	player := createTestPlayer()
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
