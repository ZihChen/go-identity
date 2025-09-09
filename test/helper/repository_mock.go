package helper

import (
	"context"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/stretchr/testify/mock"
)

// MockPlayerRepository provides a mock implementation of PlayerRepository
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

// MockMerchantRepository provides a mock implementation of MerchantRepository
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

// MockManagerRepository provides a mock implementation of ManagerRepository
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

func (m *MockManagerRepository) FindByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Manager, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Manager), args.Error(1)
}

func (m *MockManagerRepository) FirstOrCreate(ctx context.Context, manager *entity.Manager) error {
	args := m.Called(ctx, manager)
	return args.Error(0)
}

func (m *MockManagerRepository) Create(ctx context.Context, manager *entity.Manager) error {
	args := m.Called(ctx, manager)
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

func (m *MockManagerRepository) Upsert(ctx context.Context, manager *entity.Manager) error {
	args := m.Called(ctx, manager)
	return args.Error(0)
}

// MockLevelRepository provides a mock implementation of LevelRepository
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

// MockEventProducer provides a mock implementation of EventProducer
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

// Test helper functions
func CreateTestContext() context.Context {
	return context.Background()
}

func CreateTestPlayer() *entity.Player {
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

func CreateTestMerchant() *entity.Merchant {
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

func CreateTestManager() *entity.Manager {
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
