package usecaseport

import (
	"context"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
)

type MerchantUseCase interface {
	SyncMerchant(ctx context.Context, data *event.MerchantSyncEvent) error
	GetMerchantByID(ctx context.Context, id uint64) (*entity.Merchant, error)
	GetMerchantByGlobalID(ctx context.Context, globalID string) (*entity.Merchant, error)
}

type ManagerUseCase interface {
	SyncManager(ctx context.Context, data *event.ManagerSyncEvent) error
	GetManagerByID(ctx context.Context, id uint64) (*entity.Manager, error)
	GetManagerByGlobalID(ctx context.Context, globalID string) (*entity.Manager, error)
}

type PlayerUseCase interface {
	SyncPlayer(
		ctx context.Context,
		data *event.PlayerSyncEvent,
	) error
	GetPlayerByID(ctx context.Context, id uint64) (*entity.Player, error)
	GetPlayerByGlobalID(ctx context.Context, globalID string) (*entity.Player, error)
	UpdatePlayerLastActive(ctx context.Context, id uint64) error
}

type TagUseCase interface {
	SyncPlayerTag(
		ctx context.Context,
		data []event.TagData,
		globalMerchantID, globalPlayerID string,
	) error
	SyncTag(ctx context.Context, data *event.TagSyncEvent) error
}

type PlayerLevelUseCase interface {
	SyncLevel(ctx context.Context, data *event.LevelSyncEvent) error
}
