package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-identity-cat/test/factories"
	"github.com/jvdiamondtech/ms-identity-cat/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Helper function specific to level tests
func createLevelMockDependencies(
	t *testing.T,
) (*mocks.LevelRepositoryMock, *mocks.MerchantRepositoryMock, *mocks.EventProducerMock, *mocks.MockLogger) {
	// Explicitly use imports to avoid "unused import" errors
	var _ context.Context
	var _ entity.Level
	var _ repository.LevelRepository

	levelRepo := mocks.NewLevelRepositoryMock(t)
	merchantRepo := mocks.NewMerchantRepositoryMock(t)
	eventProducer := mocks.NewEventProducerMock(t)
	logger := mocks.NewMockLogger(t)
	return levelRepo, merchantRepo, eventProducer, logger
}

func createLevelSyncEvent() *event.LevelSyncEvent {
	return &event.LevelSyncEvent{
		GlobalMerchantID: "FATCAT-MERCHANT-1",
		PlayerLevel: struct {
			Name                string    `json:"name"`
			GlobalPlayerLevelID string    `json:"global_player_level_id"`
			CreatedAt           time.Time `json:"created_at"`
			UpdatedAt           time.Time `json:"updated_at"`
		}{
			GlobalPlayerLevelID: "FATCAT-LEVEL-1",
			Name:                "Bronze",
			UpdatedAt:           time.Now(),
		},
	}
}

// Tests
func TestNewLevelUseCase(t *testing.T) {
	levelRepo, merchantRepo, eventProducer, logger := createLevelMockDependencies(t)

	useCase := NewLevelUseCase(levelRepo, merchantRepo, eventProducer, logger, mocks.NewNilTracingService())

	assert.NotNil(t, useCase)
	assert.IsType(t, &LevelUseCase{}, useCase)
}

func TestLevelUseCase_SyncLevel_CreateNew(t *testing.T) {
	ctx := factories.CreateTestContext()
	levelRepo, merchantRepo, eventProducer, logger := createLevelMockDependencies(t)

	// Setup mocks
	merchant := factories.CreateTestMerchant()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)

	// Expect Upsert to be called
	levelRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*entity.Level")).Return(nil)

	// Expect PublishPlayerLevelSync to be called
	eventProducer.On("PublishPlayerLevelSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).
		Return(nil)

	// Create the use case
	useCase := NewLevelUseCase(levelRepo, merchantRepo, eventProducer, logger, mocks.NewNilTracingService())

	// Create test event data
	eventData := createLevelSyncEvent()

	// Execute the function
	err := useCase.SyncLevel(ctx, eventData)

	// Verify results
	assert.NoError(t, err)
	levelRepo.AssertExpectations()
	merchantRepo.AssertExpectations()
	eventProducer.AssertExpectations()
}

func TestLevelUseCase_SyncLevel_UpdateExisting(t *testing.T) {
	ctx := factories.CreateTestContext()
	levelRepo, merchantRepo, eventProducer, logger := createLevelMockDependencies(t)

	// Setup mocks
	merchant := factories.CreateTestMerchant()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)

	// Expect Upsert to be called
	levelRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*entity.Level")).Return(nil)

	// Expect PublishPlayerLevelSync to be called
	eventProducer.On("PublishPlayerLevelSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).
		Return(nil)

	// Create the use case
	useCase := NewLevelUseCase(levelRepo, merchantRepo, eventProducer, logger, mocks.NewNilTracingService())

	// Create test event data
	eventData := createLevelSyncEvent()

	// Execute the function
	err := useCase.SyncLevel(ctx, eventData)

	// Verify results
	assert.NoError(t, err)
	levelRepo.AssertExpectations()
	merchantRepo.AssertExpectations()
	eventProducer.AssertExpectations()
}

func TestLevelUseCase_SyncLevel_MerchantNotFound(t *testing.T) {
	ctx := factories.CreateTestContext()
	levelRepo, merchantRepo, eventProducer, logger := createLevelMockDependencies(t)

	// Setup mocks - merchant not found, but this should not cause error
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").
		Return(nil, errors.New("merchant not found"))

	// Create the use case
	useCase := NewLevelUseCase(levelRepo, merchantRepo, eventProducer, logger, mocks.NewNilTracingService())

	// Create test event data
	eventData := createLevelSyncEvent()

	// Execute the function
	err := useCase.SyncLevel(ctx, eventData)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "find merchant")
	merchantRepo.AssertExpectations()
}

func TestLevelUseCase_SyncLevel_UpsertError(t *testing.T) {
	ctx := factories.CreateTestContext()
	levelRepo, merchantRepo, eventProducer, logger := createLevelMockDependencies(t)

	// Setup mocks
	merchant := factories.CreateTestMerchant()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)

	// Upsert fails
	levelRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*entity.Level")).
		Return(errors.New("database upsert failed"))

	// Create the use case
	useCase := NewLevelUseCase(levelRepo, merchantRepo, eventProducer, logger, mocks.NewNilTracingService())

	// Create test event data
	eventData := createLevelSyncEvent()

	// Execute the function
	err := useCase.SyncLevel(ctx, eventData)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "upsert player level")
	levelRepo.AssertExpectations()
	merchantRepo.AssertExpectations()
}

func TestLevelUseCase_SyncLevel_PublishError(t *testing.T) {
	ctx := factories.CreateTestContext()
	levelRepo, merchantRepo, eventProducer, logger := createLevelMockDependencies(t)

	// Setup mocks
	merchant := factories.CreateTestMerchant()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)

	// Expect Upsert to be called
	levelRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*entity.Level")).Return(nil)

	// PublishPlayerLevelSync fails
	eventProducer.On("PublishPlayerLevelSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).
		Return(errors.New("publish error"))

	// Create the use case
	useCase := NewLevelUseCase(levelRepo, merchantRepo, eventProducer, logger, mocks.NewNilTracingService())

	// Create test event data
	eventData := createLevelSyncEvent()

	// Execute the function
	err := useCase.SyncLevel(ctx, eventData)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "publish level sync event")
	levelRepo.AssertExpectations()
	merchantRepo.AssertExpectations()
	eventProducer.AssertExpectations()
}

func TestLevelUseCase_publishPlayerLevelSyncEvent(t *testing.T) {
	ctx := factories.CreateTestContext()
	levelRepo, merchantRepo, eventProducer, logger := createLevelMockDependencies(t)

	// Setup mocks
	level := factories.CreateTestLevel()

	// Expect PublishPlayerLevelSync to be called
	eventProducer.On("PublishPlayerLevelSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).
		Return(nil)

	// Create the use case
	useCase := NewLevelUseCase(levelRepo, merchantRepo, eventProducer, logger, mocks.NewNilTracingService())

	// Execute the private function through a test-only wrapper
	err := useCase.(*LevelUseCase).publishPlayerLevelSyncEvent(
		ctx,
		level,
	)

	// Verify results
	assert.NoError(t, err)
	eventProducer.AssertExpectations()
}

func TestLevelUseCase_publishPlayerLevelSyncEvent_Error(t *testing.T) {
	ctx := factories.CreateTestContext()
	levelRepo, merchantRepo, eventProducer, logger := createLevelMockDependencies(t)

	// Setup mocks
	level := factories.CreateTestLevel()

	// PublishPlayerLevelSync fails
	eventProducer.On("PublishPlayerLevelSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).
		Return(errors.New("publish error"))

	// Create the use case
	useCase := NewLevelUseCase(levelRepo, merchantRepo, eventProducer, logger, mocks.NewNilTracingService())

	// Execute the private function through a test-only wrapper
	err := useCase.(*LevelUseCase).publishPlayerLevelSyncEvent(
		ctx,
		level,
	)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "publish player level sync")
	eventProducer.AssertExpectations()
}
