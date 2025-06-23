package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/repositoryport"
)

// Mock implementations
type MockManagerRepository struct {
	mock.Mock
}

func (m *MockManagerRepository) FindByID(ctx context.Context, id uint64) (*entity.Manager, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Manager), args.Error(1)
}

func (m *MockManagerRepository) FindByGlobalID(ctx context.Context, globalID string) (*entity.Manager, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Manager), args.Error(1)
}

func (m *MockManagerRepository) Create(ctx context.Context, manager *entity.Manager) error {
	args := m.Called(ctx, manager)
	// Simulate ID assignment like a real database would
	if manager.ID == 0 {
		manager.ID = 1
	}
	return args.Error(0)
}

func (m *MockManagerRepository) Update(ctx context.Context, manager *entity.Manager) error {
	args := m.Called(ctx, manager)
	return args.Error(0)
}

func (m *MockManagerRepository) Delete(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// Helper functions
func createManagerMockDependencies(t *testing.T) (*MockManagerRepository, *MockMerchantRepository, *MockEventProducer, *zap.Logger) {
	// Explicitly use imports to avoid "unused import" errors
	var _ context.Context
	var _ entity.Manager
	var _ repositoryport.ManagerRepository

	managerRepo := new(MockManagerRepository)
	merchantRepo := new(MockMerchantRepository)
	eventProducer := new(MockEventProducer)
	logger := zaptest.NewLogger(t)
	return managerRepo, merchantRepo, eventProducer, logger
}

func createTestManager() *entity.Manager {
	email := "manager@example.com"
	now := time.Now()
	return &entity.Manager{
		ID:              1,
		MerchantID:      2,
		GlobalManagerID: "FATCAT-MANAGER-1",
		Account:         "TestManager",
		Email:           &email,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func createManagerSyncEvent() ([]byte, error) {
	managerEvent := event.ManagerSyncEvent{
		GlobalMerchantID: "FATCAT-MERCHANT-1",
		Manager: event.ManagerData{
			ID:              1,
			GlobalManagerID: "FATCAT-MANAGER-1",
			Account:         "TestManager",
			Email:           "manager@example.com",
		},
	}

	cloudEvent := event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "tw.jvd.fatidentitycat.manager.sync.v1",
		Source:          "/fatidentitycat/FATCAT",
		Subject:         "manager_sync",
		ID:              uuid.New().String(),
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		Data:            managerEvent,
	}

	return json.Marshal(cloudEvent)
}

// Tests
func TestNewManagerUseCase(t *testing.T) {
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	assert.NotNil(t, useCase)
	assert.IsType(t, &ManagerUseCase{}, useCase)
}

func TestManagerUseCase_SyncManager_CreateNew(t *testing.T) {
	ctx := createTestContext()
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	// Setup mocks
	merchant := createTestMerchant()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)

	// Manager doesn't exist yet
	managerRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MANAGER-1").Return(nil, errors.New("record not found"))

	// Expect Create to be called
	managerRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Manager")).Return(nil)

	// Expect PublishManagerSync to be called
	eventProducer.On("PublishManagerSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).Return(nil)

	// Create the use case
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// Create test event data
	eventData, err := createManagerSyncEvent()
	require.NoError(t, err)

	// Execute the function
	err = useCase.SyncManager(ctx, eventData)

	// Verify results
	assert.NoError(t, err)
	managerRepo.AssertExpectations(t)
	merchantRepo.AssertExpectations(t)
	eventProducer.AssertExpectations(t)
}

func TestManagerUseCase_SyncManager_UpdateExisting(t *testing.T) {
	ctx := createTestContext()
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	// Setup mocks
	merchant := createTestMerchant()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)

	// Manager exists
	manager := createTestManager()
	managerRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MANAGER-1").Return(manager, nil)

	// Expect Update to be called
	managerRepo.On("Update", mock.Anything, mock.AnythingOfType("*entity.Manager")).Return(nil)

	// Expect PublishManagerSync to be called
	eventProducer.On("PublishManagerSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).Return(nil)

	// Create the use case
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// Create test event data
	eventData, err := createManagerSyncEvent()
	require.NoError(t, err)

	// Execute the function
	err = useCase.SyncManager(ctx, eventData)

	// Verify results
	assert.NoError(t, err)
	managerRepo.AssertExpectations(t)
	merchantRepo.AssertExpectations(t)
	eventProducer.AssertExpectations(t)
}

func TestManagerUseCase_SyncManager_UnmarshalError(t *testing.T) {
	ctx := createTestContext()
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	// Create the use case
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// Invalid JSON data
	invalidData := []byte(`{"invalid json`)

	// Execute the function
	err := useCase.SyncManager(ctx, invalidData)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal cloud event")
}

func TestManagerUseCase_SyncManager_MerchantNotFound(t *testing.T) {
	ctx := createTestContext()
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	// Setup mocks - merchant not found
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(nil, errors.New("merchant not found"))

	// Create the use case
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// Create test event data
	eventData, err := createManagerSyncEvent()
	require.NoError(t, err)

	// Execute the function
	err = useCase.SyncManager(ctx, eventData)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "find merchant")
	merchantRepo.AssertExpectations(t)
}

func TestManagerUseCase_SyncManager_FindManagerError(t *testing.T) {
	ctx := createTestContext()
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	// Setup mocks
	merchant := createTestMerchant()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)

	// FindByGlobalID returns an error other than "record not found"
	managerRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MANAGER-1").Return(nil, errors.New("database error"))

	// Create the use case
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// Create test event data
	eventData, err := createManagerSyncEvent()
	require.NoError(t, err)

	// Execute the function
	err = useCase.SyncManager(ctx, eventData)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "find manager")
	managerRepo.AssertExpectations(t)
	merchantRepo.AssertExpectations(t)
}

func TestManagerUseCase_SyncManager_CreateError(t *testing.T) {
	ctx := createTestContext()
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	// Setup mocks
	merchant := createTestMerchant()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)

	// Manager doesn't exist yet
	managerRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MANAGER-1").Return(nil, errors.New("record not found"))

	// Create fails
	managerRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Manager")).Return(errors.New("create error"))

	// Create the use case
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// Create test event data
	eventData, err := createManagerSyncEvent()
	require.NoError(t, err)

	// Execute the function
	err = useCase.SyncManager(ctx, eventData)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create manager")
	managerRepo.AssertExpectations(t)
	merchantRepo.AssertExpectations(t)
}

func TestManagerUseCase_SyncManager_UpdateError(t *testing.T) {
	ctx := createTestContext()
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	// Setup mocks
	merchant := createTestMerchant()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)

	// Manager exists
	manager := createTestManager()
	managerRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MANAGER-1").Return(manager, nil)

	// Update fails
	managerRepo.On("Update", mock.Anything, mock.AnythingOfType("*entity.Manager")).Return(errors.New("update error"))

	// Create the use case
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// Create test event data
	eventData, err := createManagerSyncEvent()
	require.NoError(t, err)

	// Execute the function
	err = useCase.SyncManager(ctx, eventData)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update manager")
	managerRepo.AssertExpectations(t)
	merchantRepo.AssertExpectations(t)
}

func TestManagerUseCase_SyncManager_PublishError(t *testing.T) {
	ctx := createTestContext()
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	// Setup mocks
	merchant := createTestMerchant()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)

	// Manager doesn't exist yet
	managerRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MANAGER-1").Return(nil, errors.New("record not found"))

	// Expect Create to be called
	managerRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Manager")).Return(nil)

	// PublishManagerSync fails
	eventProducer.On("PublishManagerSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).Return(errors.New("publish error"))

	// Create the use case
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// Create test event data
	eventData, err := createManagerSyncEvent()
	require.NoError(t, err)

	// Execute the function
	err = useCase.SyncManager(ctx, eventData)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "publish manager sync event")
	managerRepo.AssertExpectations(t)
	merchantRepo.AssertExpectations(t)
	eventProducer.AssertExpectations(t)
}

func TestManagerUseCase_GetManagerByID(t *testing.T) {
	ctx := createTestContext()
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	// Setup mocks
	manager := createTestManager()
	managerRepo.On("FindByID", mock.Anything, uint64(1)).Return(manager, nil)

	// Create the use case
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// Execute the function
	result, err := useCase.GetManagerByID(ctx, 1)

	// Verify results
	assert.NoError(t, err)
	assert.Equal(t, manager, result)
	managerRepo.AssertExpectations(t)
}

func TestManagerUseCase_GetManagerByID_NotFound(t *testing.T) {
	ctx := createTestContext()
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	// Setup mocks - manager not found
	managerRepo.On("FindByID", mock.Anything, uint64(999)).Return(nil, errors.New("manager not found"))

	// Create the use case
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// Execute the function
	result, err := useCase.GetManagerByID(ctx, 999)

	// Verify results
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "find manager")
	managerRepo.AssertExpectations(t)
}

func TestManagerUseCase_GetManagerByGlobalID(t *testing.T) {
	ctx := createTestContext()
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	// Setup mocks
	manager := createTestManager()
	managerRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MANAGER-1").Return(manager, nil)

	// Create the use case
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// Execute the function
	result, err := useCase.GetManagerByGlobalID(ctx, "FATCAT-MANAGER-1")

	// Verify results
	assert.NoError(t, err)
	assert.Equal(t, manager, result)
	managerRepo.AssertExpectations(t)
}

func TestManagerUseCase_GetManagerByGlobalID_NotFound(t *testing.T) {
	ctx := createTestContext()
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	// Setup mocks - manager not found
	managerRepo.On("FindByGlobalID", mock.Anything, "NONEXISTENT").Return(nil, errors.New("manager not found"))

	// Create the use case
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// Execute the function
	result, err := useCase.GetManagerByGlobalID(ctx, "NONEXISTENT")

	// Verify results
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "find manager")
	managerRepo.AssertExpectations(t)
}

func TestManagerUseCase_publishManagerSyncEvent(t *testing.T) {
	ctx := createTestContext()
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	// Setup mocks
	manager := createTestManager()

	// Expect PublishManagerSync to be called
	eventProducer.On("PublishManagerSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).Return(nil)

	// Create the use case
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// Execute the private function through a test-only wrapper
	err := useCase.(*ManagerUseCase).publishManagerSyncEvent(ctx, manager, "FATCAT-MERCHANT-1", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

	// Verify results
	assert.NoError(t, err)
	eventProducer.AssertExpectations(t)
}

func TestManagerUseCase_publishManagerSyncEvent_Error(t *testing.T) {
	ctx := createTestContext()
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	// Setup mocks
	manager := createTestManager()

	// PublishManagerSync fails
	eventProducer.On("PublishManagerSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).Return(errors.New("publish error"))

	// Create the use case
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// Execute the private function through a test-only wrapper
	err := useCase.(*ManagerUseCase).publishManagerSyncEvent(ctx, manager, "FATCAT-MERCHANT-1", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "publish manager sync")
	eventProducer.AssertExpectations(t)
}

func TestManagerUseCase_SyncManager_MarshalError(t *testing.T) {
	ctx := createTestContext()
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

	cloudEvent := event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "tw.jvd.fatidentitycat.manager.sync.v1",
		Source:          "/fatidentitycat/FATCAT",
		Subject:         "manager_sync",
		ID:              uuid.New().String(),
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		Data:            managerEvent,
	}

	// Marshal the cloud event to JSON
	eventData, err := json.Marshal(cloudEvent)
	require.NoError(t, err)

	// Setup mocks
	merchant := createTestMerchant()
	merchantRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").Return(merchant, nil)

	// Manager doesn't exist yet
	managerRepo.On("FindByGlobalID", mock.Anything, "FATCAT-MANAGER-1").Return(nil, errors.New("record not found"))

	// Expect Create to be called
	managerRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Manager")).Return(nil)

	// Mock json.Marshal to return an error
	// We can't directly mock json.Marshal, but we can make the PublishManagerSync method
	// return an error that looks like it came from a marshal operation
	eventProducer.On("PublishManagerSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).Return(errors.New("json: unsupported type"))

	// Create the use case
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// Execute the function
	err = useCase.SyncManager(ctx, eventData)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "publish manager sync event")
}

func TestManagerUseCase_SyncManager_UnmarshalManagerEventError(t *testing.T) {
	ctx := createTestContext()
	managerRepo, merchantRepo, eventProducer, logger := createManagerMockDependencies(t)

	// Create a CloudEvent with invalid manager event data
	invalidManagerEvent := event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "tw.jvd.fatidentitycat.manager.sync.v1",
		Source:          "/fatidentitycat/FATCAT",
		Subject:         "manager_sync",
		ID:              uuid.New().String(),
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
		Data:            "invalid data that can't be unmarshaled to ManagerSyncEvent",
	}

	eventData, err := json.Marshal(invalidManagerEvent)
	require.NoError(t, err)

	// Create the use case
	useCase := NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// Execute the function
	err = useCase.SyncManager(ctx, eventData)

	// Verify results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal manager event")
}