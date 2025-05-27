package repository

import (
	"context"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/model"
)

// MerchantRepository 商戶資料庫接口
type MerchantRepository interface {
	FindByID(ctx context.Context, id uint64) (*model.Merchant, error)
	FindByGlobalID(ctx context.Context, globalID string) (*model.Merchant, error)
	Create(ctx context.Context, merchant *model.Merchant) error
	Update(ctx context.Context, merchant *model.Merchant) error
	Delete(ctx context.Context, id uint64) error
}

// PlayerRepository 玩家資料庫接口
type PlayerRepository interface {
	FindByID(ctx context.Context, id uint64) (*model.Player, error)
	FindByGlobalID(ctx context.Context, globalID string) (*model.Player, error)
	Create(ctx context.Context, player *model.Player) error
	Update(ctx context.Context, player *model.Player) error
	Delete(ctx context.Context, id uint64) error
}

// ManagerRepository 管理員資料庫接口
type ManagerRepository interface {
	FindByID(ctx context.Context, id uint64) (*model.Manager, error)
	FindByGlobalID(ctx context.Context, globalID string) (*model.Manager, error)
	Create(ctx context.Context, manager *model.Manager) error
	Update(ctx context.Context, manager *model.Manager) error
	Delete(ctx context.Context, id uint64) error
}
