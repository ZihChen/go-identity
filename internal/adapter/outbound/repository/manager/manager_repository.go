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

// ManagerRepository GORM 實現的管理員資料庫
type ManagerRepository struct {
	db *gorm.DB
}

// NewManagerRepository 創建管理員資料庫
func NewManagerRepository(db *gorm.DB) repository.ManagerRepository {
	return &ManagerRepository{db: db}
}

// FindByID 通過ID查找管理員
func (r *ManagerRepository) FindByID(ctx context.Context, id uint64) (*entity.Manager, error) {
	var manager models.Manager
	result := r.db.WithContext(ctx).First(&manager, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errmsg.ErrRepoManagerNotFound
		}
		return &entity.Manager{}, result.Error
	}

	return mapToDomainManager(&manager), nil
}

// FindByGlobalID 通過全局ID查找管理員
func (r *ManagerRepository) FindByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Manager, error) {
	var manager models.Manager
	result := r.db.WithContext(ctx).Where("global_manager_id = ?", globalID).First(&manager)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errmsg.ErrRepoManagerNotFound
		}
		return &entity.Manager{}, result.Error
	}

	return mapToDomainManager(&manager), nil
}

// FirstOrCreate 取得或創建，避免重複插入
func (r *ManagerRepository) FirstOrCreate(ctx context.Context, manager *entity.Manager) error {
	managerModel := mapToDBManager(manager)
	result := r.db.WithContext(ctx).Where("global_manager_id = ?", manager.GetGlobalManagerID()).
		FirstOrCreate(managerModel)
	if result.Error != nil {
		return result.Error
	}

	manager.SetID(managerModel.ID)
	return nil
}

// Create 創建管理員
func (r *ManagerRepository) Create(ctx context.Context, manager *entity.Manager) error {
	managerModel := mapToDBManager(manager)
	result := r.db.WithContext(ctx).Create(managerModel)
	if result.Error != nil {
		return result.Error
	}

	// 更新ID
	manager.SetID(managerModel.ID)

	return nil
}

// Update 更新管理員
func (r *ManagerRepository) Update(ctx context.Context, manager *entity.Manager) error {
	managerModel := mapToDBManager(manager)
	result := r.db.WithContext(ctx).Save(managerModel)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// Delete 刪除管理員
func (r *ManagerRepository) Delete(ctx context.Context, id uint64) error {
	result := r.db.WithContext(ctx).Delete(&models.Manager{}, id)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errmsg.ErrRepoDeleteManagerNotFound
	}

	return nil
}

// Upsert 資料冪等性設計：只有當新資料的UpdatedAt要大於當前資料，並且內容要不同時才更新
func (r *ManagerRepository) Upsert(ctx context.Context, manager *entity.Manager) error {
	managerModel := mapToDBManager(manager)

	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "global_manager_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"account": gorm.Expr(
				"CASE WHEN ? > updated_at AND account != ? THEN ? ELSE account END",
				managerModel.UpdatedAt, managerModel.Account, managerModel.Account),
			"email": gorm.Expr(
				"CASE WHEN ? > updated_at AND email != ? THEN ? ELSE email END",
				managerModel.UpdatedAt, managerModel.Email, managerModel.Email),
			"updated_at": gorm.Expr(
				"CASE WHEN ? > updated_at THEN ? ELSE updated_at END",
				managerModel.UpdatedAt, managerModel.UpdatedAt),
			"deleted_at": gorm.Expr(
				"CASE WHEN ? > updated_at AND deleted_at IS NULL THEN ? ELSE deleted_at END",
				managerModel.UpdatedAt, managerModel.DeletedAt),
		}),
	}).Create(managerModel)

	if result.Error != nil {
		return fmt.Errorf("manager upsert failed: %w", result.Error)
	}
	return nil
}

// 將DB模型映射到領域模型
func mapToDomainManager(manager *models.Manager) *entity.Manager {
	var deletedAt *time.Time
	if manager.DeletedAt.Valid {
		deletedTime := manager.DeletedAt.Time
		deletedAt = &deletedTime
	}

	managerEntity := entity.NewManagerWithTimes(
		manager.MerchantID,
		manager.GlobalManagerID,
		manager.Account,
		manager.Email,
		manager.CreatedAt,
		manager.UpdatedAt,
	)
	managerEntity.SetID(manager.ID)
	if deletedAt != nil {
		managerEntity.SetDeletedAt(deletedAt)
	}
	return managerEntity
}

// 將領域模型映射到DB模型
func mapToDBManager(manager *entity.Manager) *models.Manager {
	dbManager := &models.Manager{
		ID:              manager.GetID(),
		MerchantID:      manager.GetMerchantID(),
		GlobalManagerID: manager.GetGlobalManagerID(),
		Account:         manager.GetAccount(),
		Email:           manager.GetEmail(),
		CreatedAt:       manager.GetCreatedAt(),
		UpdatedAt:       manager.GetUpdatedAt(),
	}

	if manager.GetDeletedAt() != nil {
		dbManager.DeletedAt = gorm.DeletedAt{
			Time:  *manager.GetDeletedAt(),
			Valid: true,
		}
	}

	return dbManager
}
