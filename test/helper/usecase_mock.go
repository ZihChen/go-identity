package helper

import (
	"context"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/stretchr/testify/mock"
)

type MockMerchantUseCase struct {
	mock.Mock
}

func (m *MockMerchantUseCase) GetMerchantByID(
	ctx context.Context,
	id uint64,
) (*entity.Merchant, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Merchant), args.Error(1)
}

func (m *MockMerchantUseCase) GetMerchantByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Merchant, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Merchant), args.Error(1)
}

func (m *MockMerchantUseCase) SyncMerchant(
	ctx context.Context,
	eventData *event.MerchantSyncEvent,
) error {
	args := m.Called(ctx, eventData)
	return args.Error(0)
}

type MockPlayerUseCase struct {
	mock.Mock
}

func (m *MockPlayerUseCase) GetPlayerByID(ctx context.Context, id uint64) (*entity.Player, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Player), args.Error(1)
}

func (m *MockPlayerUseCase) GetPlayerByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Player, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Player), args.Error(1)
}

func (m *MockPlayerUseCase) UpdatePlayerLastActive(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPlayerUseCase) SyncPlayer(
	ctx context.Context,
	data *event.PlayerSyncEvent,
) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

type MockManagerUseCase struct {
	mock.Mock
}

func (m *MockManagerUseCase) GetManagerByID(
	ctx context.Context,
	id uint64,
) (*entity.Manager, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Manager), args.Error(1)
}

func (m *MockManagerUseCase) GetManagerByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Manager, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Manager), args.Error(1)
}

func (m *MockManagerUseCase) SyncManager(ctx context.Context, data *event.ManagerSyncEvent) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

type MockTagUseCase struct {
	mock.Mock
}

func (m *MockTagUseCase) SyncTag(ctx context.Context, data *event.TagSyncEvent) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

func (m *MockTagUseCase) SyncPlayerTag(
	ctx context.Context,
	data []event.TagData,
	globalMerchantID, globalPlayerID string,
) error {
	args := m.Called(ctx, data, globalMerchantID, globalPlayerID)
	return args.Error(0)
}

type MockLevelUseCase struct {
	mock.Mock
}

func (m *MockLevelUseCase) SyncLevel(ctx context.Context, data *event.LevelSyncEvent) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}
