package repository

import (
	"context"
	"errors"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/errmsg"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/repositoryport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/models"
	"gorm.io/gorm"
)

// ManagerRepository GORM 實現的管理員資料庫
type ManagerRepository struct {
	db *gorm.DB
}

// NewManagerRepository 創建管理員資料庫
func NewManagerRepository(db *gorm.DB) repositoryport.ManagerRepository {
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
	result := r.db.WithContext(ctx).Where("global_manager_id = ?", manager.GlobalManagerID).
		FirstOrCreate(managerModel)
	if result.Error != nil {
		return result.Error
	}

	manager.ID = managerModel.ID
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
	manager.ID = managerModel.ID

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

// 將DB模型映射到領域模型
func mapToDomainManager(manager *models.Manager) *entity.Manager {
	var deletedAt *time.Time
	if manager.DeletedAt.Valid {
		deletedTime := manager.DeletedAt.Time
		deletedAt = &deletedTime
	}

	return &entity.Manager{
		ID:              manager.ID,
		MerchantID:      manager.MerchantID,
		GlobalManagerID: manager.GlobalManagerID,
		Account:         manager.Account,
		Email:           manager.Email,
		CreatedAt:       manager.CreatedAt,
		UpdatedAt:       manager.UpdatedAt,
		DeletedAt:       deletedAt,
	}
}

// 將領域模型映射到DB模型
func mapToDBManager(manager *entity.Manager) *models.Manager {
	dbManager := &models.Manager{
		ID:              manager.ID,
		MerchantID:      manager.MerchantID,
		GlobalManagerID: manager.GlobalManagerID,
		Account:         manager.Account,
		Email:           manager.Email,
		CreatedAt:       manager.CreatedAt,
		UpdatedAt:       manager.UpdatedAt,
	}

	if manager.DeletedAt != nil {
		dbManager.DeletedAt = gorm.DeletedAt{
			Time:  *manager.DeletedAt,
			Valid: true,
		}
	}

	return dbManager
}
