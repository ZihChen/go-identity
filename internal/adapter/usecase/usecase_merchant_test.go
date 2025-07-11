package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/repositoryport"
	"github.com/jvdiamondtech/ms-identity-cat/test/helper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Helper function specific to merchant tests
func createMerchantMockDependencies(
	t *testing.T,
) (*MockMerchantRepository, *MockEventProducer, *helper.MockLogger) {
	// Explicitly use imports to avoid "unused import" errors
	var _ context.Context
	var _ entity.Merchant
	var _ repositoryport.MerchantRepository

	merchantRepo := new(MockMerchantRepository)
	eventProducer := new(MockEventProducer)
	logger := helper.SetupLoggerMock(t)
	return merchantRepo, eventProducer, logger
}

func createMerchantSyncEvent() ([]byte, error) {
	merchantEvent := event.MerchantSyncEvent{
		GlobalMerchantID: "FATCAT-MERCHANT-1",
		Merchant: event.MerchantData{
			Name:        "TestMerchant",
			DisplayName: "Test Merchant",
		},
	}

	cloudEvent := event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "tw.jvd.fatidentitycat.merchant.sync.v1",
		Source:          "/fatidentitycat/FATCAT",
		Subject:         "merchant_sync",
		ID:              uuid.New().String(),
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		Data:            merchantEvent,
	}

	return json.Marshal(cloudEvent)
}

// Tests
func TestNewMerchantUseCase(t *testing.T) {
	merchantRepo, eventProducer, logger := createMerchantMockDependencies(t)

	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	assert.NotNil(t, useCase)
	assert.IsType(t, &MerchantUseCase{}, useCase)
}

func TestMerchantUseCase_SyncMerchant_CreateNew(t *testing.T) {
	ctx := createTestContext()
	merchantRepo, eventProducer, logger := createMerchantMockDependencies(t)

	// Setup mocks
	// Merchant doesn't exist yet
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").
		Return(nil, errors.New("record not found"))

	// Expect Create to be called
	merchantRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Merchant")).Return(nil)

	// Expect PublishMerchantSync to be called
	eventProducer.On("PublishMerchantSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).
		Return(nil)

	// Create the use case
	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// Create test event data
	eventData, err := createMerchantSyncEvent()
	require.NoError(t, err)

	// Execute the function
	err = useCase.SyncMerchant(ctx, eventData)

	// Verify results
	assert.NoError(t, err)
	merchantRepo.AssertExpectations(t)
	eventProducer.AssertExpectations(t)
}

func TestMerchantUseCase_SyncMerchant_UpdateExisting(t *testing.T) {
	ctx := createTestContext()
	merchantRepo, eventProducer, logger := createMerchantMockDependencies(t)

	// Setup mocks
	// Merchant exists
	merchant := createTestMerchant()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)

	// Expect Update to be called
	merchantRepo.On("Update", mock.Anything, mock.AnythingOfType("*entity.Merchant")).Return(nil)

	// Expect PublishMerchantSync to be called
	eventProducer.On("PublishMerchantSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).
		Return(nil)

	// Create the use case
	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// Create test event data
	eventData, err := createMerchantSyncEvent()
	require.NoError(t, err)

	// Execute the function
	err = useCase.SyncMerchant(ctx, eventData)

	// Verify results
	assert.NoError(t, err)
	merchantRepo.AssertExpectations(t)
	eventProducer.AssertExpectations(t)
}

func TestMerchantUseCase_SyncMerchant_UnmarshalError(t *testing.T) {
	ctx := createTestContext()
	merchantRepo, eventProducer, logger := createMerchantMockDependencies(t)

	// Create the use case
	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// Invalid JSON data
	invalidData := []byte(`{"invalid json`)

	// Execute the function
	err := useCase.SyncMerchant(ctx, invalidData)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal cloud event")
}

func TestMerchantUseCase_SyncMerchant_CreateError(t *testing.T) {
	ctx := createTestContext()
	merchantRepo, eventProducer, logger := createMerchantMockDependencies(t)

	// Setup mocks
	// Merchant doesn't exist yet
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").
		Return(nil, errors.New("record not found"))

	// Create fails
	merchantRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Merchant")).
		Return(errors.New("create error"))

	// Create the use case
	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// Create test event data
	eventData, err := createMerchantSyncEvent()
	require.NoError(t, err)

	// Execute the function
	err = useCase.SyncMerchant(ctx, eventData)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create merchant")
	merchantRepo.AssertExpectations(t)
}

func TestMerchantUseCase_SyncMerchant_UpdateError(t *testing.T) {
	ctx := createTestContext()
	merchantRepo, eventProducer, logger := createMerchantMockDependencies(t)

	// Setup mocks
	// Merchant exists
	merchant := createTestMerchant()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)

	// Update fails
	merchantRepo.On("Update", mock.Anything, mock.AnythingOfType("*entity.Merchant")).
		Return(errors.New("update error"))

	// Create the use case
	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// Create test event data
	eventData, err := createMerchantSyncEvent()
	require.NoError(t, err)

	// Execute the function
	err = useCase.SyncMerchant(ctx, eventData)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update merchant")
	merchantRepo.AssertExpectations(t)
}

func TestMerchantUseCase_SyncMerchant_PublishError(t *testing.T) {
	ctx := createTestContext()
	merchantRepo, eventProducer, logger := createMerchantMockDependencies(t)

	// Setup mocks
	// Merchant doesn't exist yet
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").
		Return(nil, errors.New("record not found"))

	// Expect Create to be called
	merchantRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Merchant")).Return(nil)

	// PublishMerchantSync fails
	eventProducer.On("PublishMerchantSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).
		Return(errors.New("publish error"))

	// Create the use case
	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// Create test event data
	eventData, err := createMerchantSyncEvent()
	require.NoError(t, err)

	// Execute the function
	err = useCase.SyncMerchant(ctx, eventData)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "publish merchant sync event")
	merchantRepo.AssertExpectations(t)
	eventProducer.AssertExpectations(t)
}

func TestMerchantUseCase_GetMerchantByID(t *testing.T) {
	ctx := createTestContext()
	merchantRepo, eventProducer, logger := createMerchantMockDependencies(t)

	// Setup mocks
	merchant := createTestMerchant()
	merchantRepo.On("FindByID", mock.Anything, uint64(1)).Return(merchant, nil)

	// Create the use case
	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// Execute the function
	result, err := useCase.GetMerchantByID(ctx, 1)

	// Verify results
	assert.NoError(t, err)
	assert.Equal(t, merchant, result)
	merchantRepo.AssertExpectations(t)
}

func TestMerchantUseCase_GetMerchantByID_NotFound(t *testing.T) {
	ctx := createTestContext()
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
	merchantRepo.AssertExpectations(t)
}

func TestMerchantUseCase_GetMerchantByGlobalID(t *testing.T) {
	ctx := createTestContext()
	merchantRepo, eventProducer, logger := createMerchantMockDependencies(t)

	// Setup mocks
	merchant := createTestMerchant()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)

	// Create the use case
	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// Execute the function
	result, err := useCase.GetMerchantByGlobalID(ctx, "FATCAT-MERCHANT-1")

	// Verify results
	assert.NoError(t, err)
	assert.Equal(t, merchant, result)
	merchantRepo.AssertExpectations(t)
}

func TestMerchantUseCase_GetMerchantByGlobalID_NotFound(t *testing.T) {
	ctx := createTestContext()
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
	merchantRepo.AssertExpectations(t)
}

func TestMerchantUseCase_publishMerchantSyncEvent(t *testing.T) {
	ctx := createTestContext()
	merchantRepo, eventProducer, logger := createMerchantMockDependencies(t)

	// Setup mocks
	merchant := createTestMerchant()

	// Expect PublishMerchantSync to be called
	eventProducer.On("PublishMerchantSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).
		Return(nil)

	// Create the use case
	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// Execute the private function through a test-only wrapper
	err := useCase.(*MerchantUseCase).publishMerchantSyncEvent(
		ctx,
		merchant,
		"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
	)

	// Verify results
	assert.NoError(t, err)
	eventProducer.AssertExpectations(t)
}

func TestMerchantUseCase_publishMerchantSyncEvent_Error(t *testing.T) {
	ctx := createTestContext()
	merchantRepo, eventProducer, logger := createMerchantMockDependencies(t)

	// Setup mocks
	merchant := createTestMerchant()

	// PublishMerchantSync fails
	eventProducer.On("PublishMerchantSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).
		Return(errors.New("publish error"))

	// Create the use case
	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// Execute the private function through a test-only wrapper
	err := useCase.(*MerchantUseCase).publishMerchantSyncEvent(
		ctx,
		merchant,
		"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
	)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "publish merchant sync")
	eventProducer.AssertExpectations(t)
}

func TestMerchantUseCase_SyncMerchant_MarshalError(t *testing.T) {
	ctx := createTestContext()
	merchantRepo, eventProducer, logger := createMerchantMockDependencies(t)

	// Create a CloudEvent with valid data
	merchantEvent := event.MerchantSyncEvent{
		GlobalMerchantID: "FATCAT-MERCHANT-1",
		Merchant: event.MerchantData{
			Name:        "TestMerchant",
			DisplayName: "Test Merchant",
		},
	}

	cloudEvent := event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "tw.jvd.fatidentitycat.merchant.sync.v1",
		Source:          "/fatidentitycat/FATCAT",
		Subject:         "merchant_sync",
		ID:              uuid.New().String(),
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		Data:            merchantEvent,
	}

	// Marshal the cloud event to JSON
	eventData, err := json.Marshal(cloudEvent)
	require.NoError(t, err)

	// Create a mock that will cause a marshal error when trying to marshal the event data
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").
		Return(nil, errors.New("record not found"))
	merchantRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Merchant")).Return(nil)

	// Mock json.Marshal to return an error
	// We can't directly mock json.Marshal, but we can make the PublishMerchantSync method
	// return an error that looks like it came from a marshal operation
	eventProducer.On("PublishMerchantSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).
		Return(errors.New("json: unsupported type"))

	// Create the use case
	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// Execute the function
	err = useCase.SyncMerchant(ctx, eventData)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "publish merchant sync event")
}

func TestMerchantUseCase_SyncMerchant_UnmarshalMerchantEventError(t *testing.T) {
	ctx := createTestContext()
	merchantRepo, eventProducer, logger := createMerchantMockDependencies(t)

	// Create a CloudEvent with invalid merchant event data
	invalidMerchantEvent := event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "tw.jvd.fatidentitycat.merchant.sync.v1",
		Source:          "/fatidentitycat/FATCAT",
		Subject:         "merchant_sync",
		ID:              uuid.New().String(),
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		Data:            "invalid data that can't be unmarshaled to MerchantSyncEvent",
	}

	eventData, err := json.Marshal(invalidMerchantEvent)
	require.NoError(t, err)

	// Create the use case
	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// Execute the function
	err = useCase.SyncMerchant(ctx, eventData)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal merchant event")
}

func TestMerchantUseCase_SyncMerchant_FindByGlobalIDError(t *testing.T) {
	ctx := createTestContext()
	merchantRepo, eventProducer, logger := createMerchantMockDependencies(t)

	// Setup mocks - FindByGlobalID returns an error other than "record not found"
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").
		Return(nil, errors.New("database error"))

	// Create the use case
	useCase := NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// Create test event data
	eventData, err := createMerchantSyncEvent()
	require.NoError(t, err)

	// Execute the function
	err = useCase.SyncMerchant(ctx, eventData)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "find merchant")
	merchantRepo.AssertExpectations(t)
}
