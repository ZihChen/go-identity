package usecaseport

import (
	"context"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/model"
)

type MerchantUseCase interface {
	SyncMerchant(ctx context.Context, eventData []byte) error
	GetMerchantByID(ctx context.Context, id uint64) (*model.Merchant, error)
	GetMerchantByGlobalID(ctx context.Context, globalID string) (*model.Merchant, error)
}

type ManagerUseCase interface {
	SyncManager(ctx context.Context, eventData []byte) error
	GetManagerByID(ctx context.Context, id uint64) (*model.Manager, error)
	GetManagerByGlobalID(ctx context.Context, globalID string) (*model.Manager, error)
}

type PlayerUseCase interface {
	SyncPlayer(ctx context.Context, eventData []byte) error
	GetPlayerByID(ctx context.Context, id uint64) (*model.Player, error)
	GetPlayerByGlobalID(ctx context.Context, globalID string) (*model.Player, error)
	UpdatePlayerLastActive(ctx context.Context, id uint64) error
}
