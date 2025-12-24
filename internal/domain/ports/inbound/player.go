package inbound

import (
	"context"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
)

type PlayerUseCase interface {
	// 批次處理器生命週期管理
	StartBatchProcessor(ctx context.Context) error
	StopBatchProcessor(ctx context.Context) error

	// 業務方法
	SyncPlayer(
		ctx context.Context,
		data *event.PlayerSyncEvent,
	) error
	GetPlayerByID(ctx context.Context, id uint64) (*entity.Player, error)
	GetPlayerByGlobalID(ctx context.Context, globalID string) (*entity.Player, error)
	UpdatePlayerLastActive(ctx context.Context, id uint64) error
	PlayerLogin(ctx context.Context, account, playerGlobalID string) (string, error)
}
