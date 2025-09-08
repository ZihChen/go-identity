package inbound

import (
	"context"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
)

type ManagerUseCase interface {
	SyncManager(ctx context.Context, data *event.ManagerSyncEvent) error
	GetManagerByID(ctx context.Context, id uint64) (*entity.Manager, error)
	GetManagerByGlobalID(ctx context.Context, globalID string) (*entity.Manager, error)
}
