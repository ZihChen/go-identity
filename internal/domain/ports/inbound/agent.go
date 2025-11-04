package inbound

import (
	"context"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
)

type AgentUseCase interface {
	SyncAgentData(ctx context.Context, event *event.AgentSyncEvent) error
	GetAgentByID(ctx context.Context, id uint64) (*entity.Agent, error)
	GetAgentByGlobalID(ctx context.Context, globalID string) (*entity.Agent, error)
	GetAgentsByMerchantID(ctx context.Context, merchantID uint64) ([]*entity.Agent, error)
}