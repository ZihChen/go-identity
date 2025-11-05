package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AgentRepository GORM 實現的代理資料庫
type agentRepository struct {
	db *gorm.DB
}

// NewAgentRepository 創建代理資料庫
func NewAgentRepository(db *gorm.DB) repository.AgentRepository {
	return &agentRepository{db: db}
}

// FindByID 通過ID查找代理
func (r *agentRepository) FindByID(ctx context.Context, id uint64) (*entity.Agent, error) {
	var agent models.Agent
	result := r.db.WithContext(ctx).First(&agent, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errmsg.ErrRepoAgentNotFound
		}
		return nil, result.Error
	}

	return mapToDomainAgent(&agent), nil
}

// FindByGlobalID 通過全局ID查找代理
func (r *agentRepository) FindByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Agent, error) {
	var agent models.Agent
	result := r.db.WithContext(ctx).Where("global_agent_id = ?", globalID).First(&agent)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errmsg.ErrRepoAgentNotFound
		}
		return nil, result.Error
	}

	return mapToDomainAgent(&agent), nil
}

// FindByMerchantID 通過商戶ID查找代理列表
func (r *agentRepository) FindByMerchantID(
	ctx context.Context,
	merchantID uint64,
) ([]*entity.Agent, error) {
	var agentModels []*models.Agent
	result := r.db.WithContext(ctx).Where("merchant_id = ?", merchantID).Find(&agentModels)
	if result.Error != nil {
		return nil, result.Error
	}

	agents := make([]*entity.Agent, len(agentModels))
	for i, model := range agentModels {
		agents[i] = mapToDomainAgent(model)
	}

	return agents, nil
}

// Create 創建代理
func (r *agentRepository) Create(ctx context.Context, agent *entity.Agent) error {
	agentModel := mapToDBAgent(agent)
	result := r.db.WithContext(ctx).Create(agentModel)
	if result.Error != nil {
		return result.Error
	}

	// 更新ID
	agent.SetID(agentModel.ID)

	return nil
}

// Update 更新代理
func (r *agentRepository) Update(ctx context.Context, agent *entity.Agent) error {
	agentModel := mapToDBAgent(agent)
	result := r.db.WithContext(ctx).Save(agentModel)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// Delete 刪除代理
func (r *agentRepository) Delete(ctx context.Context, id uint64) error {
	result := r.db.WithContext(ctx).Delete(&models.Agent{}, id)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errmsg.ErrRepoDeleteAgentNotFound
	}

	return nil
}

// Upsert 資料冪等性設計：只有當新資料的UpdatedAt要大於當前資料，並且內容要不同時才更新
func (r *agentRepository) Upsert(ctx context.Context, agent *entity.Agent) error {
	agentModel := mapToDBAgent(agent)

	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "global_agent_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"account": gorm.Expr(
				"CASE WHEN ? > updated_at AND account != ? THEN ? ELSE account END",
				agentModel.UpdatedAt, agentModel.Account, agentModel.Account),
			"ancestry": gorm.Expr(
				"CASE WHEN ? > updated_at AND ancestry != ? THEN ? ELSE ancestry END",
				agentModel.UpdatedAt, agentModel.Ancestry, agentModel.Ancestry),
			"current_sign_in_at": gorm.Expr(
				"CASE WHEN ? > updated_at THEN ? ELSE current_sign_in_at END",
				agentModel.UpdatedAt, agentModel.CurrentSignInAt),
			"updated_at": gorm.Expr(
				"CASE WHEN ? > updated_at THEN ? ELSE updated_at END",
				agentModel.UpdatedAt, agentModel.UpdatedAt),
			"deleted_at": gorm.Expr(
				"CASE WHEN ? > updated_at AND deleted_at IS NULL THEN ? ELSE deleted_at END",
				agentModel.UpdatedAt, agentModel.DeletedAt),
		}),
	}).Create(agentModel)

	if result.Error != nil {
		return fmt.Errorf("timestamp-based upsert failed: %w", result.Error)
	}

	return nil
}

// mapToDomainAgent - 將資料庫模型映射到領域實體
func mapToDomainAgent(agent *models.Agent) *entity.Agent {
	var deletedAt *time.Time
	if agent.DeletedAt.Valid {
		deletedTime := agent.DeletedAt.Time
		deletedAt = &deletedTime
	}

	agentEntity := entity.NewAgentWithTimes(
		agent.MerchantID,
		agent.GlobalAgentID,
		agent.Account,
		agent.Ancestry,
		agent.CurrentSignInAt,
		agent.CreatedAt,
		agent.UpdatedAt,
	)
	agentEntity.SetID(agent.ID)
	if deletedAt != nil {
		agentEntity.SetDeletedAt(deletedAt)
	}
	return agentEntity
}

// mapToDBAgent - 將領域實體映射到資料庫模型
func mapToDBAgent(agent *entity.Agent) *models.Agent {
	dbAgent := &models.Agent{
		ID:              agent.GetID(),
		MerchantID:      agent.GetMerchantID(),
		GlobalAgentID:   agent.GetGlobalAgentID(),
		Account:         agent.GetAccount(),
		Ancestry:        agent.GetAncestry(),
		CurrentSignInAt: agent.GetCurrentSignInAt(),
		CreatedAt:       agent.GetCreatedAt(),
		UpdatedAt:       agent.GetUpdatedAt(),
	}

	if agent.GetDeletedAt() != nil {
		dbAgent.DeletedAt = gorm.DeletedAt{
			Time:  *agent.GetDeletedAt(),
			Valid: true,
		}
	}

	return dbAgent
}
