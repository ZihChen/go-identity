package repository

import (
	"context"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
)

type LevelRepository interface {
	Upsert(ctx context.Context, level *entity.Level) error
	FindByGlobalID(ctx context.Context, globalID string) (*entity.Level, error)
}
