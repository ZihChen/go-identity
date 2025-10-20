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

// Mock implementations are now in test/mocks/repository_mocks.go

// Helper functions
func createManagerMockDependencies(
	t *testing.T,
) (*mocks.ManagerRepositoryMock, *mocks.MerchantRepositoryMock, *mocks.EventProducerMock, *helper.MockLogger) {
	// Explicitly use imports to avoid "unused import" errors
	var _ context.Context
	var _ entity.Manager
	var _ repository.ManagerRepository

	managerRepo := mocks.NewManagerRepositoryMock(t)
	merchantRepo := mocks.NewMerchantRepositoryMock(t)
	eventProducer := mocks.NewEventProducerMock(t)
	logger := helper.NewMockLogger()
	return managerRepo, merchantRepo, eventProducer, logger
}

func createManagerSyncEvent() *event.ManagerSyncEvent {
	return &event.ManagerSyncEvent{
		GlobalMerchantID: "FATCAT-MERCHANT-1",
		Manager: event.ManagerData{
			ID:              1,
			GlobalManagerID: "FATCAT-MANAGER-1",
			Account:         "TestManager",
			Email:           "manager@example.com",
		},
	}
}

// Tests
func TestNewManagerUseCase(t *testing.T) {
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	assert.NotNil(t, useCase)
	assert.IsType(t, &ManagerUseCase{}, useCase)
}

func TestManagerUseCase_SyncManager_CreateNew(t *testing.T) {
	ctx := mocks.CreateTestContext()
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	// Setup mocks
	merchant := mocks.CreateTestMerchant()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)

	// Expect Upsert to be called
	managerRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*entity.Manager")).Return(nil)

	// Expect PublishManagerSync to be called
	eventProducer.On("PublishManagerSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).
		Return(nil)

	// Create the use case
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// Create test event data
	eventData := createManagerSyncEvent()
	// Execute the function
	err := useCase.SyncManager(ctx, eventData)

	// Verify results
	assert.NoError(t, err)
	managerRepo.AssertExpectations()
	merchantRepo.AssertExpectations()
	eventProducer.AssertExpectations()
}

func TestManagerUseCase_SyncManager_UpdateExisting(t *testing.T) {
	ctx := mocks.CreateTestContext()
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	// Setup mocks
	merchant := mocks.CreateTestMerchant()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)

	// Expect Upsert to be called
	managerRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*entity.Manager")).Return(nil)

	// Expect PublishManagerSync to be called
	eventProducer.On("PublishManagerSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).
		Return(nil)

	// Create the use case
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// Create test event data
	eventData := createManagerSyncEvent()

	// Execute the function
	err := useCase.SyncManager(ctx, eventData)

	// Verify results
	assert.NoError(t, err)
	managerRepo.AssertExpectations()
	merchantRepo.AssertExpectations()
	eventProducer.AssertExpectations()
}

func TestManagerUseCase_SyncManager_MerchantNotFound(t *testing.T) {
	ctx := mocks.CreateTestContext()
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	// Setup mocks - merchant not found
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").
		Return(nil, errors.New("merchant not found"))

	// Create the use case
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// Create test event data
	eventData := createManagerSyncEvent()

	// Execute the function
	err := useCase.SyncManager(ctx, eventData)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "find merchant")
	merchantRepo.AssertExpectations()
}

func TestManagerUseCase_SyncManager_FindManagerError(t *testing.T) {
	ctx := mocks.CreateTestContext()
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	// Setup mocks
	merchant := mocks.CreateTestMerchant()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)

	// Upsert fails with a database error
	managerRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*entity.Manager")).
		Return(errors.New("database error"))

	// Create the use case
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// Create test event data
	eventData := createManagerSyncEvent()

	// Execute the function
	err := useCase.SyncManager(ctx, eventData)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "upsert manager")
	managerRepo.AssertExpectations()
	merchantRepo.AssertExpectations()
}

func TestManagerUseCase_SyncManager_CreateError(t *testing.T) {
	ctx := mocks.CreateTestContext()
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	// Setup mocks
	merchant := mocks.CreateTestMerchant()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)

	// Upsert fails
	managerRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*entity.Manager")).
		Return(errors.New("upsert error"))

	// Create the use case
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// Create test event data
	eventData := createManagerSyncEvent()

	// Execute the function
	err := useCase.SyncManager(ctx, eventData)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "upsert manager")
	managerRepo.AssertExpectations()
	merchantRepo.AssertExpectations()
}

func TestManagerUseCase_SyncManager_UpdateError(t *testing.T) {
	ctx := mocks.CreateTestContext()
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	// Setup mocks
	merchant := mocks.CreateTestMerchant()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)

	// Upsert fails
	managerRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*entity.Manager")).
		Return(errors.New("upsert error"))

	// Create the use case
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// Create test event data
	eventData := createManagerSyncEvent()

	// Execute the function
	err := useCase.SyncManager(ctx, eventData)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "upsert manager")
	managerRepo.AssertExpectations()
	merchantRepo.AssertExpectations()
}

func TestManagerUseCase_SyncManager_PublishError(t *testing.T) {
	ctx := mocks.CreateTestContext()
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	// Setup mocks
	merchant := mocks.CreateTestMerchant()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)

	// Expect Upsert to be called
	managerRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*entity.Manager")).Return(nil)

	// PublishManagerSync fails
	eventProducer.On("PublishManagerSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).
		Return(errors.New("publish error"))

	// Create the use case
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// Create test event data
	eventData := createManagerSyncEvent()

	// Execute the function
	err := useCase.SyncManager(ctx, eventData)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "publish manager sync event")
	managerRepo.AssertExpectations()
	merchantRepo.AssertExpectations()
	eventProducer.AssertExpectations()
}

func TestManagerUseCase_GetManagerByID(t *testing.T) {
	ctx := mocks.CreateTestContext()
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	// Setup mocks
	manager := mocks.CreateTestManager()
	managerRepo.On("FindByID", mock.Anything, uint64(1)).Return(manager, nil)

	// Create the use case
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// Execute the function
	result, err := useCase.GetManagerByID(ctx, 1)

	// Verify results
	assert.NoError(t, err)
	assert.Equal(t, manager, result)
	managerRepo.AssertExpectations()
}

func TestManagerUseCase_GetManagerByID_NotFound(t *testing.T) {
	ctx := mocks.CreateTestContext()
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	// Setup mocks - manager not found
	managerRepo.On("FindByID", mock.Anything, uint64(999)).
		Return(nil, errors.New("manager not found"))

	// Create the use case
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// Execute the function
	result, err := useCase.GetManagerByID(ctx, 999)

	// Verify results
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "find manager")
	managerRepo.AssertExpectations()
}

func TestManagerUseCase_GetManagerByGlobalID(t *testing.T) {
	ctx := mocks.CreateTestContext()
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	// Setup mocks
	manager := mocks.CreateTestManager()
	managerRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MANAGER-1").Return(manager, nil)

	// Create the use case
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// Execute the function
	result, err := useCase.GetManagerByGlobalID(ctx, "FATCAT-MANAGER-1")

	// Verify results
	assert.NoError(t, err)
	assert.Equal(t, manager, result)
	managerRepo.AssertExpectations()
}

func TestManagerUseCase_GetManagerByGlobalID_NotFound(t *testing.T) {
	ctx := mocks.CreateTestContext()
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	// Setup mocks - manager not found
	managerRepo.On("FindByGlobalID", mock.Anything, "NONEXISTENT").
		Return(nil, errors.New("manager not found"))

	// Create the use case
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// Execute the function
	result, err := useCase.GetManagerByGlobalID(ctx, "NONEXISTENT")

	// Verify results
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "find manager")
	managerRepo.AssertExpectations()
}

func TestManagerUseCase_publishManagerSyncEvent(t *testing.T) {
	ctx := mocks.CreateTestContext()
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	// Setup mocks
	manager := mocks.CreateTestManager()

	// Expect PublishManagerSync to be called
	eventProducer.On("PublishManagerSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).
		Return(nil)

	// Create the use case
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// Execute the private function through a test-only wrapper
	err := useCase.(*ManagerUseCase).publishManagerSyncEvent(
		ctx,
		manager,
		"FATCAT-MERCHANT-1",
	)

	// Verify results
	assert.NoError(t, err)
	eventProducer.AssertExpectations()
}

func TestManagerUseCase_publishManagerSyncEvent_Error(t *testing.T) {
	ctx := mocks.CreateTestContext()
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	// Setup mocks
	manager := mocks.CreateTestManager()

	// PublishManagerSync fails
	eventProducer.On("PublishManagerSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).
		Return(errors.New("publish error"))

	// Create the use case
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// Execute the private function through a test-only wrapper
	err := useCase.(*ManagerUseCase).publishManagerSyncEvent(
		ctx,
		manager,
		"FATCAT-MERCHANT-1",
	)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "publish manager sync")
	eventProducer.AssertExpectations()
}

func TestManagerUseCase_SyncManager_MarshalError(t *testing.T) {
	ctx := mocks.CreateTestContext()
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	// Create a CloudEvent with valid data
	managerEvent := event.ManagerSyncEvent{
		GlobalMerchantID: "FATCAT-MERCHANT-1",
		Manager: event.ManagerData{
			ID:              1,
			GlobalManagerID: "FATCAT-MANAGER-1",
			Account:         "TestManager",
			Email:           "manager@example.com",
		},
	}

	// Setup mocks
	merchant := mocks.CreateTestMerchant()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)

	// Expect Upsert to be called
	managerRepo.On("Upsert", mock.Anything, mock.AnythingOfType("*entity.Manager")).Return(nil)

	// Mock json.Marshal to return an error
	// We can't directly mock json.Marshal, but we can make the PublishManagerSync method
	// return an error that looks like it came from a marshal operation
	eventProducer.On("PublishManagerSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).
		Return(errors.New("json: unsupported type"))

	// Create the use case
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// Execute the function
	err := useCase.SyncManager(ctx, &managerEvent)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "publish manager sync event")
}
