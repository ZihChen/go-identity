package queue

import (
	"context"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/serviceport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

// TestQueueServiceImplementsInterface tests that QueueService implements the serviceport.QueueService interface
func TestQueueServiceImplementsInterface(t *testing.T) {
	// This test will fail to compile if QueueService doesn't implement serviceport.QueueService
	var _ serviceport.QueueService = (*QueueService)(nil)
}

// TestEnqueueMethods tests all enqueue methods
func TestEnqueueMethods(t *testing.T) {
	// Skip this test in CI environment since it requires Redis
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Create a test config with Redis connection details
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Domain:   "localhost", // Use your Redis host
			Port:     6379,        // Use your Redis port
			Password: "",          // Use your Redis password if any
			DB:       0,           // Use your Redis DB
		},
	}

	// Create a test logger
	logger := zaptest.NewLogger(t)

	// Create a QueueService
	queueService, err := NewQueueService(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create QueueService: %v", err)
	}
	// Type assertion to access Close method
	concreteService, ok := queueService.(*QueueService)
	if !ok {
		t.Fatalf("Failed to cast to *QueueService")
	}
	defer func() {
		_ = concreteService.Close()
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

// TestWrapHandlerWithTracing tests the WrapHandlerWithTracing function
func TestWrapHandlerWithTracing(t *testing.T) {
	// Create a simple handler that returns nil
	handler := asynq.HandlerFunc(func(ctx context.Context, task *asynq.Task) error {
		return nil
	})

	// Create a wrapped handler
	wrappedHandler := WrapHandlerWithTracing(handler)

	// Create a task
	task := asynq.NewTask(TypeMerchantSync, []byte(`{"id":"test-id","data":"test-data"}`))

	// Call the wrapped handler
	err := wrappedHandler.ProcessTask(context.Background(), task)

	// Check the results
	assert.NoError(t, err)
}

// TestNewWorkerServer tests the NewWorkerServer function
func TestNewWorkerServer(t *testing.T) {
	// Skip this test in CI environment since it requires Redis
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Create a test config with Redis connection details
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Domain:   "localhost", // Use your Redis host
			Port:     6379,        // Use your Redis port
			Password: "",          // Use your Redis password if any
			DB:       0,           // Use your Redis DB
		},
	}

	// Create a test logger
	logger := zaptest.NewLogger(t)

	// Call the function being tested
	server, err := NewWorkerServer(cfg, logger)

	// Check the results
	assert.NoError(t, err)
	assert.NotNil(t, server)
}

// TestNewQueueService tests the NewQueueService function
func TestNewQueueService(t *testing.T) {
	// Skip this test in CI environment since it requires Redis
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Create a test config with Redis connection details
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Domain:   "localhost", // Use your Redis host
			Port:     6379,        // Use your Redis port
			Password: "",          // Use your Redis password if any
			DB:       0,           // Use your Redis DB
		},
	}

	// Create a test logger
	logger := zaptest.NewLogger(t)

	// Call the function being tested
	service, err := NewQueueService(cfg, logger)

	// Check the results
	assert.NoError(t, err)
	assert.NotNil(t, service)

	// Type assertion to access internal fields and methods
	concreteService, ok := service.(*QueueService)
	if !ok {
		t.Fatalf("Failed to cast to *QueueService")
	}

	assert.NotNil(t, concreteService.client)
	assert.Equal(t, logger, concreteService.logger)

	// Clean up
	err = concreteService.Close()
	assert.NoError(t, err)
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

	// Call the function being tested
	service, err := NewQueueService(cfg, logger)

	// Check the results - should fail to connect
	assert.Error(t, err)
	assert.Nil(t, service)
}

// TestClose tests the Close method
func TestClose(t *testing.T) {
	// Skip this test in CI environment since it requires Redis
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	// Create a test config with Redis connection details
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Domain:   "localhost", // Use your Redis host
			Port:     6379,        // Use your Redis port
			Password: "",          // Use your Redis password if any
			DB:       0,           // Use your Redis DB
		},
	}

	// Create a test logger
	logger := zaptest.NewLogger(t)

	// Create a QueueService
	service, err := NewQueueService(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create QueueService: %v", err)
	}

	// Type assertion to access Close method
	concreteService, ok := service.(*QueueService)
	if !ok {
		t.Fatalf("Failed to cast to *QueueService")
	}

	// Call the method being tested
	err = concreteService.Close()

	// Check the results
	assert.NoError(t, err)
}
