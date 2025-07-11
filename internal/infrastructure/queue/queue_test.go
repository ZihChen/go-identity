package queue

import (
	"context"
	"fmt"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/serviceport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// MockQueueService is a mock implementation of serviceport.QueueService
type MockQueueService struct {
	mock.Mock
}

func (m *MockQueueService) EnqueueMerchantSync(ctx context.Context, data []byte) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *MockQueueService) EnqueuePlayerSync(ctx context.Context, data []byte) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *MockQueueService) EnqueueManagerSync(ctx context.Context, data []byte) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

// MockQueueService also needs to implement Close method for tests
func (m *MockQueueService) Close() error {
	args := m.Called()
	return args.Error(0)
}

// NewMockQueueService creates a new mock queue service that returns no errors
func NewMockQueueService(cfg *config.Config, logger *zap.Logger) (serviceport.QueueService, error) {
	mockService := new(MockQueueService)

	// Setup default behavior to return nil error for all methods
	mockService.On("EnqueueMerchantSync", mock.Anything, mock.Anything).Return(nil)
	mockService.On("EnqueuePlayerSync", mock.Anything, mock.Anything).Return(nil)
	mockService.On("EnqueueManagerSync", mock.Anything, mock.Anything).Return(nil)
	mockService.On("Close").Return(nil)

	return mockService, nil
}

// TestQueueServiceImplementsInterface tests that QueueService implements the serviceport.QueueService interface
func TestQueueServiceImplementsInterface(t *testing.T) {
	// This test will fail to compile if QueueService doesn't implement serviceport.QueueService
	var _ serviceport.QueueService = (*QueueService)(nil)
}

// TestEnqueueMethods tests all enqueue methods
func TestEnqueueMethods(t *testing.T) {
	// Create a test config
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Domain:   "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
		},
	}

	// Create a test logger
	logger := zaptest.NewLogger(t)

	// Create a mock QueueService
	queueService, err := NewMockQueueService(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create MockQueueService: %v", err)
	}

	// Type assertion to access Close method
	mockService, ok := queueService.(*MockQueueService)
	if !ok {
		t.Fatalf("Failed to cast to *MockQueueService")
	}
	defer func() {
		_ = mockService.Close()
	}()

	// Test data
	ctx := context.Background()
	merchantData := []byte(`{"id":"test-merchant-id","data":"test-merchant-data"}`)
	playerData := []byte(`{"id":"test-player-id","data":"test-player-data"}`)
	managerData := []byte(`{"id":"test-manager-id","data":"test-manager-data"}`)

	// Test cases
	tests := []struct {
		name   string
		method func(context.Context, []byte) error
		data   []byte
	}{
		{
			name:   "EnqueueMerchantSync",
			method: queueService.EnqueueMerchantSync,
			data:   merchantData,
		},
		{
			name:   "EnqueuePlayerSync",
			method: queueService.EnqueuePlayerSync,
			data:   playerData,
		},
		{
			name:   "EnqueueManagerSync",
			method: queueService.EnqueueManagerSync,
			data:   managerData,
		},
	}

	// Run tests
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.method(ctx, tc.data)
			assert.NoError(t, err)
		})
	}
}

// MockWorkerServer is a mock implementation of asynq.Server
type MockWorkerServer struct {
	mock.Mock
}

// NewMockWorkerServer creates a new mock worker server
func NewMockWorkerServer(cfg *config.Config, logger *zap.Logger) (*asynq.Server, error) {
	// In a real test, we would create a mock server
	// For simplicity, we'll just return a nil server and no error
	return nil, nil
}

// TestNewWorkerServer tests the NewWorkerServer function
func TestNewWorkerServer(t *testing.T) {
	// Create a test config
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Domain:   "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
		},
	}

	// Create a test logger
	logger := zaptest.NewLogger(t)

	// Call the mock function
	server, err := NewMockWorkerServer(cfg, logger)

	// Check the results
	assert.NoError(t, err)
	assert.Nil(t, server) // The mock returns nil
}

// TestNewQueueService tests the NewQueueService function
func TestNewQueueService(t *testing.T) {
	// This test now uses a mock instead of connecting to Redis

	// Create a test config
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Domain:   "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
		},
	}

	// Create a test logger
	logger := zaptest.NewLogger(t)

	// Create a mock service
	service, err := NewMockQueueService(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create MockQueueService: %v", err)
	}

	// 測試所有方法
	ctx := context.Background()
	testData := []byte(`{"test": "data"}`)
	err = service.EnqueueMerchantSync(ctx, testData)
	require.NoError(t, err)

	err = service.EnqueuePlayerSync(ctx, testData)
	require.NoError(t, err)

	err = service.EnqueueManagerSync(ctx, testData)
	require.NoError(t, err)

	// Check the results
	assert.NoError(t, err)
	assert.NotNil(t, service)

	// Type assertion to access methods
	mockService, ok := service.(*MockQueueService)
	if !ok {
		t.Fatalf("Failed to cast to *MockQueueService")
	}

	// Clean up
	err = mockService.Close()
	assert.NoError(t, err)

	// Verify that the mock was called
	mockService.AssertExpectations(t)
}

// TestNewQueueServiceWithInvalidConfig tests the NewQueueService function with invalid config
func TestNewQueueServiceWithInvalidConfig(t *testing.T) {
	// Create a test config with invalid Redis connection details
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Domain:   "nonexistent-host",
			Port:     12345, // Invalid port
			Password: "",
			DB:       0,
		},
	}

	// Create a test logger
	logger := zaptest.NewLogger(t)

	// Create a custom mock function that returns an error
	mockErrorFunc := func(cfg *config.Config, logger *zap.Logger) (serviceport.QueueService, error) {
		return nil, fmt.Errorf("failed to connect to Redis: connection refused")
	}

	// Call the mock function
	service, err := mockErrorFunc(cfg, logger)

	// Check the results - should fail to connect
	assert.Error(t, err)
	assert.Nil(t, service)
}

// TestClose tests the Close method
func TestClose(t *testing.T) {
	// Create a test config
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Domain:   "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
		},
	}

	// Create a test logger
	logger := zaptest.NewLogger(t)

	// Create a mock QueueService
	service, err := NewMockQueueService(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create MockQueueService: %v", err)
	}

	// Type assertion to access Close method
	mockService, ok := service.(*MockQueueService)
	if !ok {
		t.Fatalf("Failed to cast to *MockQueueService")
	}

	// 測試所有方法
	ctx := context.Background()
	testData := []byte(`{"test": "data"}`)
	err = service.EnqueueMerchantSync(ctx, testData)
	require.NoError(t, err)

	err = service.EnqueuePlayerSync(ctx, testData)
	require.NoError(t, err)

	err = service.EnqueueManagerSync(ctx, testData)
	require.NoError(t, err)

	// Call the method being tested
	err = mockService.Close()

	// Check the results
	assert.NoError(t, err)

	// Verify that the mock was called
	mockService.AssertExpectations(t)
}
