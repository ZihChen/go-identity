package repository

import (
	"context"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
)

// MerchantRepository 商戶資料庫接口
type MerchantRepository interface {
	FindByID(ctx context.Context, id uint64) (*entity.Merchant, error)
	FindByGlobalID(ctx context.Context, globalID string) (*entity.Merchant, error)
	FirstOrCreate(ctx context.Context, merchant *entity.Merchant) error
	Create(ctx context.Context, merchant *entity.Merchant) error
	Update(ctx context.Context, merchant *entity.Merchant) error
	Delete(ctx context.Context, id uint64) error
	Upsert(ctx context.Context, merchant *entity.Merchant) error
}