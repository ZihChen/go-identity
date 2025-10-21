package mocks

import (
	"context"
	"testing"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
)

// PlayerUseCaseMock 統一的 Player UseCase Mock
type PlayerUseCaseMock struct {
	*BaseMock
}

// NewPlayerUseCaseMock 創建新的 Player UseCase Mock
func NewPlayerUseCaseMock(t *testing.T) *PlayerUseCaseMock {
	return &PlayerUseCaseMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *PlayerUseCaseMock) GetPlayerByID(ctx context.Context, id uint64) (*entity.Player, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Player), args.Error(1)
}

func (m *PlayerUseCaseMock) GetPlayerByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Player, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Player), args.Error(1)
}

func (m *PlayerUseCaseMock) UpsertPlayer(ctx context.Context, player *entity.Player) error {
	args := m.Called(ctx, player)
	return args.Error(0)
}

func (m *PlayerUseCaseMock) UpdatePlayerLastActive(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *PlayerUseCaseMock) SyncPlayer(ctx context.Context, data *event.PlayerSyncEvent) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

// MerchantUseCaseMock 統一的 Merchant UseCase Mock
type MerchantUseCaseMock struct {
	*BaseMock
}

// NewMerchantUseCaseMock 創建新的 Merchant UseCase Mock
func NewMerchantUseCaseMock(t *testing.T) *MerchantUseCaseMock {
	return &MerchantUseCaseMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *MerchantUseCaseMock) GetMerchantByID(
	ctx context.Context,
	id uint64,
) (*entity.Merchant, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Merchant), args.Error(1)
}

func (m *MerchantUseCaseMock) GetMerchantByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Merchant, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Merchant), args.Error(1)
}

func (m *MerchantUseCaseMock) UpsertMerchant(ctx context.Context, merchant *entity.Merchant) error {
	args := m.Called(ctx, merchant)
	return args.Error(0)
}

func (m *MerchantUseCaseMock) SyncMerchant(
	ctx context.Context,
	data *event.MerchantSyncEvent,
) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

// ManagerUseCaseMock 統一的 Manager UseCase Mock
type ManagerUseCaseMock struct {
	*BaseMock
}

// NewManagerUseCaseMock 創建新的 Manager UseCase Mock
func NewManagerUseCaseMock(t *testing.T) *ManagerUseCaseMock {
	return &ManagerUseCaseMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *ManagerUseCaseMock) GetManagerByID(
	ctx context.Context,
	id uint64,
) (*entity.Manager, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Manager), args.Error(1)
}

func (m *ManagerUseCaseMock) GetManagerByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Manager, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Manager), args.Error(1)
}

func (m *ManagerUseCaseMock) UpsertManager(ctx context.Context, manager *entity.Manager) error {
	args := m.Called(ctx, manager)
	return args.Error(0)
}

func (m *ManagerUseCaseMock) SyncManager(ctx context.Context, data *event.ManagerSyncEvent) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

// TagUseCaseMock 統一的 Tag UseCase Mock
type TagUseCaseMock struct {
	*BaseMock
}

// NewTagUseCaseMock 創建新的 Tag UseCase Mock
func NewTagUseCaseMock(t *testing.T) *TagUseCaseMock {
	return &TagUseCaseMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *TagUseCaseMock) UpsertTag(ctx context.Context, tag *entity.Tag) error {
	args := m.Called(ctx, tag)
	return args.Error(0)
}

func (m *TagUseCaseMock) GetTagByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Tag, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Tag), args.Error(1)
}

func (m *TagUseCaseMock) GetTagsByGlobalIDs(
	ctx context.Context,
	globalIDs []string,
) ([]*entity.Tag, error) {
	args := m.Called(ctx, globalIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Tag), args.Error(1)
}

func (m *TagUseCaseMock) UpdatePlayerTags(
	ctx context.Context,
	playerID uint64,
	tagGlobalIDs []string,
) error {
	args := m.Called(ctx, playerID, tagGlobalIDs)
	return args.Error(0)
}

func (m *TagUseCaseMock) SyncPlayerTag(
	ctx context.Context,
	data []event.TagData,
	globalMerchantID, globalPlayerID string,
) error {
	args := m.Called(ctx, data, globalMerchantID, globalPlayerID)
	return args.Error(0)
}

func (m *TagUseCaseMock) SyncTag(ctx context.Context, data *event.TagSyncEvent) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}

// PlayerLevelUseCaseMock 統一的 PlayerLevel UseCase Mock
type PlayerLevelUseCaseMock struct {
	*BaseMock
}

// NewPlayerLevelUseCaseMock 創建新的 PlayerLevel UseCase Mock
func NewPlayerLevelUseCaseMock(t *testing.T) *PlayerLevelUseCaseMock {
	return &PlayerLevelUseCaseMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *PlayerLevelUseCaseMock) UpsertPlayerLevel(
	ctx context.Context,
	playerID uint64,
	levelGlobalID string,
) error {
	args := m.Called(ctx, playerID, levelGlobalID)
	return args.Error(0)
}

func (m *PlayerLevelUseCaseMock) SyncLevel(ctx context.Context, data *event.LevelSyncEvent) error {
	args := m.Called(ctx, data)
	return args.Error(0)
}
