package mocks

import (
	"context"
	"testing"

	"github.com/go-redsync/redsync/v4"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
)

// EventProducerMock 統一的 Event Producer Mock
type EventProducerMock struct {
	*BaseMock
}

// NewEventProducerMock 創建新的 Event Producer Mock
func NewEventProducerMock(t *testing.T) *EventProducerMock {
	return &EventProducerMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *EventProducerMock) PublishMerchantSync(
	ctx context.Context,
	merchant *entity.Merchant,
) error {
	args := m.Called(ctx, merchant)
	return args.Error(0)
}

func (m *EventProducerMock) PublishPlayerSync(
	ctx context.Context,
	player *entity.Player,
	globalMerchantID string,
) error {
	args := m.Called(ctx, player, globalMerchantID)
	return args.Error(0)
}

func (m *EventProducerMock) BatchPublishPlayerSync(
	ctx context.Context,
	players []*entity.Player,
	globalMerchantIDs []string,
) error {
	args := m.Called(ctx, players, globalMerchantIDs)
	return args.Error(0)
}

func (m *EventProducerMock) PublishManagerSync(ctx context.Context, event *event.CloudEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *EventProducerMock) PublishPlayerLevelSync(
	ctx context.Context,
	event *event.CloudEvent,
) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *EventProducerMock) PublishPlayerTagsSync(
	ctx context.Context,
	event *event.CloudEvent,
) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *EventProducerMock) PublishTagSync(ctx context.Context, event *event.CloudEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *EventProducerMock) PublishAgentSync(
	ctx context.Context,
	agent *entity.Agent,
	globalMerchantID string,
) error {
	args := m.Called(ctx, agent, globalMerchantID)
	return args.Error(0)
}

// JWTServiceMock 統一的 JWT Service Mock
type JWTServiceMock struct {
	*BaseMock
}

// NewJWTServiceMock 創建新的 JWT Service Mock
func NewJWTServiceMock(t *testing.T) *JWTServiceMock {
	return &JWTServiceMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *JWTServiceMock) GenerateToken(account, playerGlobalID string) (string, error) {
	args := m.Called(account, playerGlobalID)
	return args.String(0), args.Error(1)
}

func (m *JWTServiceMock) GenerateTokenWithMetadata(
	globalMerchantID string,
	metadata map[string]interface{},
) (string, error) {
	args := m.Called(globalMerchantID, metadata)
	return args.String(0), args.Error(1)
}

// RedisManagerMock 統一的 Redis Manager Mock
type RedisManagerMock struct {
	*BaseMock
}

// NewRedisManagerMock 創建新的 Redis Manager Mock
func NewRedisManagerMock(t *testing.T) *RedisManagerMock {
	return &RedisManagerMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *RedisManagerMock) GetMutexWithOption(
	key string,
	options ...redsync.Option,
) (*redsync.Mutex, error) {
	args := m.Called(key, options)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*redsync.Mutex), args.Error(1)
}

// MockMutex 模擬 redsync.Mutex 的行為
type MockMutex struct {
	*BaseMock
	locked bool
}

// NewMockMutex 創建新的 Mock Mutex
func NewMockMutex(t *testing.T) *MockMutex {
	return &MockMutex{
		BaseMock: NewBaseMock(t),
		locked:   false,
	}
}

func (m *MockMutex) Lock() error {
	args := m.Called()
	if args.Error(0) == nil {
		m.locked = true
	}
	return args.Error(0)
}

func (m *MockMutex) Unlock() (bool, error) {
	args := m.Called()
	if args.Error(1) == nil && args.Bool(0) {
		m.locked = false
	}
	return args.Bool(0), args.Error(1)
}

// QueueServiceMock 統一的 Queue Service Mock
type QueueServiceMock struct {
	*BaseMock
}

// NewQueueServiceMock 創建新的 Queue Service Mock
func NewQueueServiceMock(t *testing.T) *QueueServiceMock {
	return &QueueServiceMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *QueueServiceMock) EnqueueMerchantSync(ctx context.Context, data []byte) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *QueueServiceMock) EnqueuePlayerSync(ctx context.Context, data []byte) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *QueueServiceMock) EnqueueManagerSync(ctx context.Context, data []byte) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *QueueServiceMock) EnqueueLevelSync(ctx context.Context, data []byte) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *QueueServiceMock) EnqueueTagSync(ctx context.Context, data []byte) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *QueueServiceMock) EnqueueAgentSync(ctx context.Context, data []byte) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *QueueServiceMock) Close() error {
	args := m.Called()
	return args.Error(0)
}

// KDSServiceMock 統一的 KDS Service Mock
type KDSServiceMock struct {
	*BaseMock
}

// NewKDSServiceMock 創建新的 KDS Service Mock
func NewKDSServiceMock(t *testing.T) *KDSServiceMock {
	return &KDSServiceMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *KDSServiceMock) ConsumeAllEvents(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}
