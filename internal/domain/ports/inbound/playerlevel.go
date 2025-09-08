package inbound

import (
	"context"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
)

type PlayerLevelUseCase interface {
	SyncLevel(ctx context.Context, data *event.LevelSyncEvent) error
}
