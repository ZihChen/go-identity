package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-identity-cat/test/helper"
	"github.com/jvdiamondtech/ms-identity-cat/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Helper function specific to merchant tests
func createMerchantMockDependencies(
	t *testing.T,
) (*mocks.MerchantRepositoryMock, *mocks.EventProducerMock, *helper.MockLogger) {
	// Explicitly use imports to avoid "unused import" errors
	var _ context.Context
	var _ entity.Merchant
	var _ repository.MerchantRepository

	merchantRepo := mocks.NewMerchantRepositoryMock(t)
	eventProducer := mocks.NewEventProducerMock(t)
	logger := helper.NewMockLogger()
	return merchantRepo, eventProducer, logger
}

func createMerchantSyncEvent() *event.MerchantSyncEvent {
	return &event.MerchantSyncEvent{
		GlobalMerchantID: "FATCAT-MERCHANT-1",
		Merchant: event.MerchantData{
			Name:        "TestMerchant",
			DisplayName: "Test Merchant",
		},
	}
}

// Tests
func TestNewMerchantUseCase(t *testing.T) {
	merchantRepo, eventProducer, logger := createMerchantMockDependencies(t)

	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	assert.NotNil(t, useCase)
	assert.IsType(t, &MerchantUseCase{}, useCase)
}

func TestMerchantUseCase_SyncMerchant_CreateNew(t *testing.T) {
	ctx := mocks.CreateTestContext()
	merchantRepo, eventProducer, logger := createMerchantMockDependencies(t)

	// Setup mocks
	// Expect Upsert to be called
	merchantRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*entity.Merchant")).Return(nil)

	// Expect PublishMerchantSync to be called
	eventProducer.On("PublishMerchantSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).
		Return(nil)

	// Create the use case
	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// Create test event data
	eventData := createMerchantSyncEvent()

	// Execute the function
	err := useCase.SyncMerchant(ctx, eventData)

	// Verify results
	assert.NoError(t, err)
	merchantRepo.AssertExpectations()
	eventProducer.AssertExpectations()
}

func TestMerchantUseCase_SyncMerchant_UpdateExisting(t *testing.T) {
	ctx := mocks.CreateTestContext()
	merchantRepo, eventProducer, logger := createMerchantMockDependencies(t)

	// Setup mocks
	// Expect Upsert to be called
	merchantRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*entity.Merchant")).Return(nil)

	// Expect PublishMerchantSync to be called
	eventProducer.On("PublishMerchantSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).
		Return(nil)

	// Create the use case
	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// Create test event data
	eventData := createMerchantSyncEvent()

	// Execute the function
	err := useCase.SyncMerchant(ctx, eventData)

	// Verify results
	assert.NoError(t, err)
	merchantRepo.AssertExpectations()
	eventProducer.AssertExpectations()
}

func TestMerchantUseCase_SyncMerchant_UpsertError(t *testing.T) {
	ctx := mocks.CreateTestContext()
	// Setup mocks
	merchantRepo, eventProducer, logger := createMerchantMockDependencies(t)

	// Create fails
	merchantRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*entity.Merchant")).
		Return(errors.New("timestamp-based upsert failed"))

	// Create the use case
	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// Create test event data
	eventData := createMerchantSyncEvent()

	// Execute the function
	err := useCase.SyncMerchant(ctx, eventData)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "upsert merchant")
	merchantRepo.AssertExpectations()
}

func TestMerchantUseCase_SyncMerchant_PublishError(t *testing.T) {
	ctx := mocks.CreateTestContext()
	// Setup mocks
	merchantRepo, eventProducer, logger := createMerchantMockDependencies(t)

	// Expect Create to be called
	merchantRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*entity.Merchant")).
		Return(nil)

	// PublishMerchantSync fails
	eventProducer.On("PublishMerchantSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).
		Return(errors.New("publish error"))

	// Create the use case
	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// Create test event data
	eventData := createMerchantSyncEvent()

	// Execute the function
	err := useCase.SyncMerchant(ctx, eventData)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "publish merchant sync event")
	merchantRepo.AssertExpectations()
	eventProducer.AssertExpectations()
}

func TestMerchantUseCase_GetMerchantByID(t *testing.T) {
	ctx := mocks.CreateTestContext()
	merchantRepo, eventProducer, logger := createMerchantMockDependencies(t)

	// Setup mocks
	merchant := mocks.CreateTestMerchant()
	merchantRepo.On("FindByID", mock.Anything, uint64(1)).Return(merchant, nil)

	// Create the use case
	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// Execute the function
	result, err := useCase.GetMerchantByID(ctx, 1)

	// Verify results
	assert.NoError(t, err)
	assert.Equal(t, merchant, result)
	merchantRepo.AssertExpectations()
}

func TestMerchantUseCase_GetMerchantByID_NotFound(t *testing.T) {
	ctx := mocks.CreateTestContext()
	merchantRepo, eventProducer, logger := createMerchantMockDependencies(t)

	// Setup mocks - merchant not found
	merchantRepo.On("FindByID", mock.Anything, uint64(999)).
		Return(nil, errors.New("merchant not found"))

	// Create the use case
	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// Execute the function
	result, err := useCase.GetMerchantByID(ctx, 999)

	// Verify results
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "find merchant")
	merchantRepo.AssertExpectations()
}

func TestMerchantUseCase_GetMerchantByGlobalID(t *testing.T) {
	ctx := mocks.CreateTestContext()
	merchantRepo, eventProducer, logger := createMerchantMockDependencies(t)

	// Setup mocks
	merchant := mocks.CreateTestMerchant()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)

	// Create the use case
	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// Execute the function
	result, err := useCase.GetMerchantByGlobalID(ctx, "FATCAT-MERCHANT-1")

	// Verify results
	assert.NoError(t, err)
	assert.Equal(t, merchant, result)
	merchantRepo.AssertExpectations()
}

func TestMerchantUseCase_GetMerchantByGlobalID_NotFound(t *testing.T) {
	ctx := mocks.CreateTestContext()
	merchantRepo, eventProducer, logger := createMerchantMockDependencies(t)

	// Setup mocks - merchant not found
	merchantRepo.On("FindByGlobalID", mock.Anything, "NONEXISTENT").
		Return(nil, errors.New("merchant not found"))

	// Create the use case
	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// Execute the function
	result, err := useCase.GetMerchantByGlobalID(ctx, "NONEXISTENT")

	// Verify results
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "find merchant")
	merchantRepo.AssertExpectations()
}

func TestMerchantUseCase_publishMerchantSyncEvent(t *testing.T) {
	ctx := mocks.CreateTestContext()
	merchantRepo, eventProducer, logger := createMerchantMockDependencies(t)

	// Setup mocks
	merchant := mocks.CreateTestMerchant()

	// Expect PublishMerchantSync to be called
	eventProducer.On("PublishMerchantSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).
		Return(nil)

	// Create the use case
	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// Execute the private function through a test-only wrapper
	err := useCase.(*MerchantUseCase).publishMerchantSyncEvent(
		ctx,
		merchant,
	)

	// Verify results
	assert.NoError(t, err)
	eventProducer.AssertExpectations()
}

func TestMerchantUseCase_publishMerchantSyncEvent_Error(t *testing.T) {
	ctx := mocks.CreateTestContext()
	merchantRepo, eventProducer, logger := createMerchantMockDependencies(t)

	// Setup mocks
	merchant := mocks.CreateTestMerchant()

	// PublishMerchantSync fails
	eventProducer.On("PublishMerchantSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).
		Return(errors.New("publish error"))

	// Create the use case
	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// Execute the private function through a test-only wrapper
	err := useCase.(*MerchantUseCase).publishMerchantSyncEvent(
		ctx,
		merchant,
	)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "publish merchant sync")
	eventProducer.AssertExpectations()
}
