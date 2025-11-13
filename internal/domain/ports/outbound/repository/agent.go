package repository

import (
	"context"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
)

// AgentRepository 代理資料庫接口
type AgentRepository interface {
	FindByID(ctx context.Context, id uint64) (*entity.Agent, error)
	FindByGlobalID(ctx context.Context, globalID string) (*entity.Agent, error)
	FindByMerchantID(ctx context.Context, merchantID uint64) ([]*entity.Agent, error)
	Upsert(ctx context.Context, agent *entity.Agent) error
}
