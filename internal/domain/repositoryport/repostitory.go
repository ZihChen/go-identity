package repositoryport

import (
	"context"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
)

// MerchantRepository 商戶資料庫接口
type MerchantRepository interface {
	FindByID(ctx context.Context, id uint64) (*entity.Merchant, error)
	FindByGlobalID(ctx context.Context, globalID string) (*entity.Merchant, error)
	Create(ctx context.Context, merchant *entity.Merchant) error
	Update(ctx context.Context, merchant *entity.Merchant) error
	Delete(ctx context.Context, id uint64) error
}

// PlayerRepository 玩家資料庫接口
type PlayerRepository interface {
	FindByID(ctx context.Context, id uint64) (*entity.Player, error)
	FindByGlobalID(ctx context.Context, globalID string) (*entity.Player, error)
	Create(ctx context.Context, player *entity.Player) error
	Update(ctx context.Context, player *entity.Player) error
	Delete(ctx context.Context, id uint64) error
}

// ManagerRepository 管理員資料庫接口
type ManagerRepository interface {
	FindByID(ctx context.Context, id uint64) (*entity.Manager, error)
	FindByGlobalID(ctx context.Context, globalID string) (*entity.Manager, error)
	Create(ctx context.Context, manager *entity.Manager) error
	Update(ctx context.Context, manager *entity.Manager) error
	Delete(ctx context.Context, id uint64) error
}

type TagRepository interface {
	BatchUpsert(ctx context.Context, tags []*entity.Tag) error
}

type LevelRepository interface {
	Upsert(ctx context.Context, level *entity.Level) (uint64, error)
}
