package repository

import (
	"context"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
)

type TagRepository interface {
	Upsert(ctx context.Context, tag *entity.Tag) error
	BatchUpsert(ctx context.Context, tags []*entity.Tag) error
	FindByGlobalIDs(ctx context.Context, globalIDs []string) ([]*entity.Tag, error)
}

type PlayerTagRepository interface {
	BatchUpdate(ctx context.Context, playerID uint64, tagIDs []uint64) error
	DeleteByPlayerID(ctx context.Context, playerID uint64) error
	FindTagsByPlayerID(ctx context.Context, playerID uint64) ([]*entity.Tag, error)
}
