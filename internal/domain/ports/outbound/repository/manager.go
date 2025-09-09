package repository

import (
	"context"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
)

// ManagerRepository 管理員資料庫接口
type ManagerRepository interface {
	FindByID(ctx context.Context, id uint64) (*entity.Manager, error)
	FindByGlobalID(ctx context.Context, globalID string) (*entity.Manager, error)
	FirstOrCreate(ctx context.Context, manager *entity.Manager) error
	Create(ctx context.Context, manager *entity.Manager) error
	Update(ctx context.Context, manager *entity.Manager) error
	Delete(ctx context.Context, id uint64) error
	Upsert(ctx context.Context, manager *entity.Manager) error
}
