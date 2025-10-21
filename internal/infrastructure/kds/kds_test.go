package kds

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/service"
	redisCache "github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/cache/redis"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-identity-cat/test/mocks"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestKDSServiceImplementsInterfaces tests that KDSService implements the required interfaces
func TestKDSServiceImplementsInterfaces(t *testing.T) {
	// This test will fail to compile if KDSService doesn't implement service.EventProducer
	var _ service.EventProducer = (*KDSService)(nil)
}

// MockQueueService is a mock implementation of service.QueueService
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

func (m *MockQueueService) EnqueueLevelSync(ctx context.Context, data []byte) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *MockQueueService) EnqueueTagSync(ctx context.Context, data []byte) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *MockQueueService) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockRedisClient is a mock implementation of redis.Client
type MockRedisClient struct {
	mock.Mock
}

// Implement the necessary methods for the test
func (m *MockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	args := m.Called(ctx, key)
	return args.Get(0).(*redis.StringCmd)
}

func (m *MockRedisClient) Set(
	ctx context.Context,
	key string,
	value interface{},
	expiration time.Duration,
) *redis.StatusCmd {
	args := m.Called(ctx, key, value, expiration)
	return args.Get(0).(*redis.StatusCmd)
}

// MockKinesisClient is a mock implementation of Kinesis client
type MockKinesisClient struct {
	mock.Mock
}

func (m *MockKinesisClient) PutRecord(
	ctx context.Context,
	params *kinesis.PutRecordInput,
	optFns ...func(*kinesis.Options),
) (*kinesis.PutRecordOutput, error) {
	args := m.Called(ctx, params, optFns)
	return args.Get(0).(*kinesis.PutRecordOutput), args.Error(1)
}

func (m *MockKinesisClient) DescribeStream(
	ctx context.Context,
	params *kinesis.DescribeStreamInput,
	optFns ...func(*kinesis.Options),
) (*kinesis.DescribeStreamOutput, error) {
	args := m.Called(ctx, params, optFns)
	return args.Get(0).(*kinesis.DescribeStreamOutput), args.Error(1)
}

func (m *MockKinesisClient) GetShardIterator(
	ctx context.Context,
	params *kinesis.GetShardIteratorInput,
	optFns ...func(*kinesis.Options),
) (*kinesis.GetShardIteratorOutput, error) {
	args := m.Called(ctx, params, optFns)
	return args.Get(0).(*kinesis.GetShardIteratorOutput), args.Error(1)
}

func (m *MockKinesisClient) GetRecords(
	ctx context.Context,
	params *kinesis.GetRecordsInput,
	optFns ...func(*kinesis.Options),
) (*kinesis.GetRecordsOutput, error) {
	args := m.Called(ctx, params, optFns)
	return args.Get(0).(*kinesis.GetRecordsOutput), args.Error(1)
}

// MockDynamoDBClient is a mock implementation of DynamoDB client
type MockDynamoDBClient struct {
	mock.Mock
}

func (m *MockDynamoDBClient) GetItem(
	ctx context.Context,
	params *dynamodb.GetItemInput,
	optFns ...func(*dynamodb.Options),
) (*dynamodb.GetItemOutput, error) {
	args := m.Called(ctx, params, optFns)
	return args.Get(0).(*dynamodb.GetItemOutput), args.Error(1)
}

func (m *MockDynamoDBClient) PutItem(
	ctx context.Context,
	params *dynamodb.PutItemInput,
	optFns ...func(*dynamodb.Options),
) (*dynamodb.PutItemOutput, error) {
	args := m.Called(ctx, params, optFns)
	return args.Get(0).(*dynamodb.PutItemOutput), args.Error(1)
}

// TestNewKDSService tests the NewKDSService function
func TestNewKDSService(t *testing.T) {
	// Create a test config
	cfg := &config.Config{
		AWS: config.AWSConfig{
			Region:        "us-west-2",
			KinesisStream: "test-stream",
			DynamoDBTable: "test-table",
		},
	}

	// Skip the actual AWS config loading by mocking the LoadAWSConfig method
	// This requires modifying the config package or using a different approach
	// For now, we'll skip this test
	t.Skip("Skipping test that requires AWS config loading")

	// Create mocks
	mockQueueService := new(MockQueueService)
	redisManager := redisCache.NewRedisManager(cfg)
	mockLogger := mocks.NewMockLogger(t)

	// Setup mock expectations
	mockLogger.On("InfoWithContext", mock.Anything, mock.Anything, mock.Anything).Return()
	mockLogger.On("ErrorWithContext", mock.Anything, mock.Anything, mock.Anything).Return()

	// Call the function being tested
	service, err := NewKDSService(cfg, mockQueueService, redisManager, mockLogger)

	// Check the results
	assert.NoError(t, err)
	assert.NotNil(t, service)
	assert.Equal(t, "test-stream", service.streamName)
	assert.Equal(t, "test-table", service.tableName)
}

// TestPublishMethods tests all publish methods
//func TestPublishMethods(t *testing.T) {
//	// Create a mock Kinesis client
//	mockKinesisClient := new(MockKinesisClient)
//
//	// Setup mock expectations for PutRecord
//	mockKinesisClient.On("PutRecord", mock.Anything, mock.Anything, mock.Anything).Return(&kinesis.PutRecordOutput{
//		SequenceNumber: aws.String("test-sequence-number"),
//		ShardId:        aws.String("test-shard-id"),
//	}, nil)
//
//	// Create a mock Redis client
//	mockRedisClient := new(MockRedisClient)
//
//	// Create a mock redis.StringCmd for the Get method
//	mockStringCmd := redis.NewStringCmd(context.Background())
//	mockStringCmd.SetVal("")
//	mockRedisClient.On("Get", mock.Anything, mock.Anything).Return(mockStringCmd)
//
//	// Create a mock redis.StatusCmd for the Set method
//	mockStatusCmd := redis.NewStatusCmd(context.Background())
//	mockRedisClient.On("Set", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(mockStatusCmd)
//
//	// Create a KDSService with mocked dependencies
//	service := &KDSService{
//		client:       mockKinesisClient,
//		dynamoClient: &dynamodb.Client{},
//		redisClient:  mockRedisClient,
//		streamName:   "test-stream",
//		tableName:    "test-table",
//		partitionKey: "id",
//		sortKey:      "sort",
//		config: &config.Config{
//			AWS: config.AWSConfig{
//				Region:        "us-west-2",
//				KinesisStream: "test-stream",
//				DynamoDBTable: "test-table",
//			},
//			Events: config.EventsConfig{
//				MerchantSync: "merchant.sync",
//				PlayerSync:   "player.sync",
//				ManagerSync:  "manager.sync",
//			},
//		},
//		queueService: new(MockQueueService),
//		logger:       zaptest.NewLogger(t),
//		sLogger:      new(MockLogger),
//	}
//
//	// Create test data
//	ctx := context.Background()
//	now := time.Now()
//	eventID := uuid.New().String()
//
//	// Create test events
//	merchantEvent := &event.CloudEvent{
//		SpecVersion:     "1.0",
//		Type:            "merchant.sync",
//		Source:          "test",
//		Subject:         "merchant",
//		ID:              eventID,
//		Time:            now,
//		DataContentType: "application/json",
//		Data: &event.MerchantSyncEvent{
//			GlobalMerchantID: "test-merchant",
//			Merchant: event.MerchantData{
//				ID:               1,
//				Name:             "Test Merchant",
//				DisplayName:      "Test Merchant Display",
//				GlobalMerchantID: "test-merchant",
//			},
//		},
//	}
//
//	playerEvent := &event.CloudEvent{
//		SpecVersion:     "1.0",
//		Type:            "player.sync",
//		Source:          "test",
//		Subject:         "player",
//		ID:              eventID,
//		Time:            now,
//		DataContentType: "application/json",
//		Data: &event.PlayerSyncEvent{
//			GlobalMerchantID: "test-merchant",
//			Player: event.PlayerData{
//				GlobalPlayerID: "test-player",
//				Account:        "test-account",
//				Email:          "test@example.com",
//				Status:         "active",
//			},
//		},
//	}
//
//	managerEvent := &event.CloudEvent{
//		SpecVersion:     "1.0",
//		Type:            "manager.sync",
//		Source:          "test",
//		Subject:         "manager",
//		ID:              eventID,
//		Time:            now,
//		DataContentType: "application/json",
//		Data: &event.ManagerSyncEvent{
//			GlobalMerchantID: "test-merchant",
//			Manager: event.ManagerData{
//				ID:              1,
//				GlobalManagerID: "test-manager",
//				Account:         "test-account",
//				Email:           "test@example.com",
//			},
//		},
//	}
//
//	// Test cases
//	tests := []struct {
//		name   string
//		method func(context.Context, *event.CloudEvent) error
//		event  *event.CloudEvent
//	}{
//		{
//			name:   "PublishMerchantSync",
//			method: service.PublishMerchantSync,
//			event:  merchantEvent,
//		},
//		{
//			name:   "PublishPlayerSync",
//			method: service.PublishPlayerSync,
//			event:  playerEvent,
//		},
//		{
//			name:   "PublishManagerSync",
//			method: service.PublishManagerSync,
//			event:  managerEvent,
//		},
//	}
//
//	// Run tests
//	for _, tc := range tests {
//		t.Run(tc.name, func(t *testing.T) {
//			err := tc.method(ctx, tc.event)
//			assert.NoError(t, err)
//		})
//	}
//
//	// Verify mock expectations
//	mockKinesisClient.AssertExpectations(t)
//	mockRedisClient.AssertExpectations(t)
//}

// TestConsumeAllEvents tests the ConsumeAllEvents method
func TestConsumeAllEvents(t *testing.T) {
	// Skip the actual AWS calls
	t.Skip("Skipping test that requires AWS calls")

	// Create a KDSService with mocked dependencies
	service := &KDSService{
		client:       &kinesis.Client{},
		dynamoClient: &dynamodb.Client{},
		redisManager: &redisCache.Manager{},
		streamName:   "test-stream",
		tableName:    "test-table",
		partitionKey: "id",
		sortKey:      "sort",
		config: &config.Config{
			AWS: config.AWSConfig{
				Region:        "us-west-2",
				KinesisStream: "test-stream",
				DynamoDBTable: "test-table",
			},
		},
		queueService: new(MockQueueService),
		logger:       mocks.NewMockLogger(t),
	}

	// Setup mock expectations for the logger
	mockLogger := service.logger.(*mocks.MockLogger)
	mockLogger.On("InfoWithContext", mock.Anything, mock.Anything, mock.Anything).Return()
	mockLogger.On("ErrorWithContext", mock.Anything, mock.Anything, mock.Anything).Return()
	mockLogger.On("DebugWithContext", mock.Anything, mock.Anything, mock.Anything).Return()

	// Call the method being tested
	err := service.ConsumeAllEvents(context.Background())

	// Check the results
	assert.NoError(t, err)

}

// TestHelperMethods tests the helper methods
func TestHelperMethods(t *testing.T) {
	// Test extractMerchantID
	merchantEvent := &event.CloudEvent{
		Data: &event.MerchantSyncEvent{
			GlobalMerchantID: "test-merchant",
		},
	}
	merchantID := extractMerchantID(merchantEvent)
	assert.Equal(t, "test-merchant", merchantID)
}
