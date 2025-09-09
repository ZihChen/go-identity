package repository

import (
	"context"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
)

// PlayerRepository 玩家資料庫接口
type PlayerRepository interface {
	FindByID(ctx context.Context, id uint64) (*entity.Player, error)
	FindByGlobalID(ctx context.Context, globalID string) (*entity.Player, error)
	FirstOrCreate(ctx context.Context, player *entity.Player) error
	Create(ctx context.Context, player *entity.Player) error
	Update(ctx context.Context, player *entity.Player) error
	Delete(ctx context.Context, id uint64) error
	Upsert(ctx context.Context, player *entity.Player) error
}
