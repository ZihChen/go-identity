package mocks

import (
	"context"
	"testing"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
)

// PlayerRepositoryMock 統一的 Player Repository Mock
type PlayerRepositoryMock struct {
	*BaseMock
}

// NewPlayerRepositoryMock 創建新的 Player Repository Mock
func NewPlayerRepositoryMock(t *testing.T) *PlayerRepositoryMock {
	return &PlayerRepositoryMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *PlayerRepositoryMock) FindByID(ctx context.Context, id uint64) (*entity.Player, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Player), args.Error(1)
}

func (m *PlayerRepositoryMock) FindByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Player, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Player), args.Error(1)
}

func (m *PlayerRepositoryMock) FirstOrCreate(ctx context.Context, player *entity.Player) error {
	args := m.Called(ctx, player)
	return args.Error(0)
}

func (m *PlayerRepositoryMock) Create(ctx context.Context, player *entity.Player) error {
	args := m.Called(ctx, player)
	if player.GetID() == 0 {
		player.SetID(1)
	}
	return args.Error(0)
}

func (m *PlayerRepositoryMock) Update(ctx context.Context, player *entity.Player) error {
	args := m.Called(ctx, player)
	return args.Error(0)
}

func (m *PlayerRepositoryMock) Delete(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *PlayerRepositoryMock) Upsert(ctx context.Context, player *entity.Player) error {
	args := m.Called(ctx, player)
	return args.Error(0)
}

func (m *PlayerRepositoryMock) UpdateLastActiveAt(
	ctx context.Context,
	id uint64,
	lastActiveAt time.Time,
) error {
	args := m.Called(ctx, id, lastActiveAt)
	return args.Error(0)
}

// MerchantRepositoryMock 統一的 Merchant Repository Mock
type MerchantRepositoryMock struct {
	*BaseMock
}

// NewMerchantRepositoryMock 創建新的 Merchant Repository Mock
func NewMerchantRepositoryMock(t *testing.T) *MerchantRepositoryMock {
	return &MerchantRepositoryMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *MerchantRepositoryMock) FindByID(
	ctx context.Context,
	id uint64,
) (*entity.Merchant, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Merchant), args.Error(1)
}

func (m *MerchantRepositoryMock) FindByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Merchant, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Merchant), args.Error(1)
}

func (m *MerchantRepositoryMock) FirstOrCreate(
	ctx context.Context,
	merchant *entity.Merchant,
) error {
	args := m.Called(ctx, merchant)
	return args.Error(0)
}

func (m *MerchantRepositoryMock) Create(ctx context.Context, merchant *entity.Merchant) error {
	args := m.Called(ctx, merchant)
	return args.Error(0)
}

func (m *MerchantRepositoryMock) Update(ctx context.Context, merchant *entity.Merchant) error {
	args := m.Called(ctx, merchant)
	return args.Error(0)
}

func (m *MerchantRepositoryMock) Delete(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MerchantRepositoryMock) Upsert(ctx context.Context, merchant *entity.Merchant) error {
	args := m.Called(ctx, merchant)
	return args.Error(0)
}

// ManagerRepositoryMock 統一的 Manager Repository Mock
type ManagerRepositoryMock struct {
	*BaseMock
}

// NewManagerRepositoryMock 創建新的 Manager Repository Mock
func NewManagerRepositoryMock(t *testing.T) *ManagerRepositoryMock {
	return &ManagerRepositoryMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *ManagerRepositoryMock) FindByID(ctx context.Context, id uint64) (*entity.Manager, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Manager), args.Error(1)
}

func (m *ManagerRepositoryMock) FindByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Manager, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Manager), args.Error(1)
}

func (m *ManagerRepositoryMock) FirstOrCreate(ctx context.Context, manager *entity.Manager) error {
	args := m.Called(ctx, manager)
	return args.Error(0)
}

func (m *ManagerRepositoryMock) Create(ctx context.Context, manager *entity.Manager) error {
	args := m.Called(ctx, manager)
	return args.Error(0)
}

func (m *ManagerRepositoryMock) Update(ctx context.Context, manager *entity.Manager) error {
	args := m.Called(ctx, manager)
	return args.Error(0)
}

func (m *ManagerRepositoryMock) Delete(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *ManagerRepositoryMock) Upsert(ctx context.Context, manager *entity.Manager) error {
	args := m.Called(ctx, manager)
	return args.Error(0)
}

// LevelRepositoryMock 統一的 Level Repository Mock
type LevelRepositoryMock struct {
	*BaseMock
}

// NewLevelRepositoryMock 創建新的 Level Repository Mock
func NewLevelRepositoryMock(t *testing.T) *LevelRepositoryMock {
	return &LevelRepositoryMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *LevelRepositoryMock) Upsert(ctx context.Context, level *entity.Level) error {
	args := m.Called(ctx, level)
	return args.Error(0)
}

func (m *LevelRepositoryMock) FindByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Level, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Level), args.Error(1)
}

// TagRepositoryMock 統一的 Tag Repository Mock
type TagRepositoryMock struct {
	*BaseMock
}

// NewTagRepositoryMock 創建新的 Tag Repository Mock
func NewTagRepositoryMock(t *testing.T) *TagRepositoryMock {
	return &TagRepositoryMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *TagRepositoryMock) Upsert(ctx context.Context, tag *entity.Tag) error {
	args := m.Called(ctx, tag)
	return args.Error(0)
}

func (m *TagRepositoryMock) FindByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Tag, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Tag), args.Error(1)
}

func (m *TagRepositoryMock) FindByGlobalIDs(
	ctx context.Context,
	globalIDs []string,
) ([]*entity.Tag, error) {
	args := m.Called(ctx, globalIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*entity.Tag), args.Error(1)
}

func (m *TagRepositoryMock) BatchUpsert(ctx context.Context, tags []*entity.Tag) error {
	args := m.Called(ctx, tags)
	return args.Error(0)
}

// PlayerTagRepositoryMock 統一的 PlayerTag Repository Mock
type PlayerTagRepositoryMock struct {
	*BaseMock
}

// NewPlayerTagRepositoryMock 創建新的 PlayerTag Repository Mock
func NewPlayerTagRepositoryMock(t *testing.T) *PlayerTagRepositoryMock {
	return &PlayerTagRepositoryMock{
		BaseMock: NewBaseMock(t),
	}
}

func (m *PlayerTagRepositoryMock) BatchUpdate(
	ctx context.Context,
	playerID uint64,
	tagIDs []uint64,
) error {
	args := m.Called(ctx, playerID, tagIDs)
	return args.Error(0)
}

func (m *PlayerTagRepositoryMock) DeleteByPlayerID(ctx context.Context, playerID uint64) error {
	args := m.Called(ctx, playerID)
	return args.Error(0)
}
