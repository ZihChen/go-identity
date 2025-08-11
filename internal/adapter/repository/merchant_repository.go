package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/repositoryport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MerchantRepository GORM 實現的商戶資料庫
type merchantRepository struct {
	db *gorm.DB
}

// NewMerchantRepository 創建商戶資料庫
func NewMerchantRepository(db *gorm.DB) repositoryport.MerchantRepository {
	return &merchantRepository{db: db}
}

// FindByID 通過ID查找商戶
func (r *merchantRepository) FindByID(ctx context.Context, id uint64) (*entity.Merchant, error) {
	var merchant models.Merchant
	result := r.db.WithContext(ctx).First(&merchant, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errmsg.ErrRepoMerchantNotFound
		}
		return &entity.Merchant{}, result.Error
	}

	return mapToDomainMerchant(&merchant), nil
}

// FindByGlobalID 通過全局ID查找商戶
func (r *merchantRepository) FindByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Merchant, error) {
	var merchant models.Merchant
	result := r.db.WithContext(ctx).Where("global_merchant_id = ?", globalID).First(&merchant)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return &entity.Merchant{}, errmsg.ErrRepoMerchantNotFound
		}
		return &entity.Merchant{}, result.Error
	}

	return mapToDomainMerchant(&merchant), nil
}

func (r *merchantRepository) FirstOrCreate(ctx context.Context, merchant *entity.Merchant) error {
	merchantModel := mapToDBMerchant(merchant)
	result := r.db.WithContext(ctx).Where("global_merchant_id = ?", merchant.GlobalMerchantID).
		FirstOrCreate(merchantModel)
	if result.Error != nil {
		return result.Error
	}

	merchant.ID = merchantModel.ID
	return nil
}

// Create 創建商戶
func (r *merchantRepository) Create(ctx context.Context, merchant *entity.Merchant) error {
	merchantModel := mapToDBMerchant(merchant)
	result := r.db.WithContext(ctx).Create(merchantModel)
	if result.Error != nil {
		return result.Error
	}

	// 更新ID
	merchant.ID = merchantModel.ID

	return nil
}

// Update 更新商戶
func (r *merchantRepository) Update(ctx context.Context, merchant *entity.Merchant) error {
	merchantModel := mapToDBMerchant(merchant)
	result := r.db.WithContext(ctx).Save(merchantModel)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// Delete 刪除商戶
func (r *merchantRepository) Delete(ctx context.Context, id uint64) error {
	result := r.db.WithContext(ctx).Delete(&models.Merchant{}, id)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errmsg.ErrRepoDeleteMerchantNotFound
	}

	return nil
}

// Upsert 資料冪等性設計：只有當新資料的UpdatedAt要大於當前資料，並且內容要不同時才更新
func (r *merchantRepository) Upsert(ctx context.Context, merchant *entity.Merchant) error {
	merchantModel := mapToDBMerchant(merchant)

	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "global_merchant_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"name": gorm.Expr(
				"CASE WHEN ? > updated_at AND name != ? THEN ? ELSE name END",
				merchantModel.UpdatedAt, merchantModel.Name, merchantModel.Name),
			"display_name": gorm.Expr(
				"CASE WHEN ? > updated_at AND display_name != ? THEN ? ELSE display_name END",
				merchantModel.UpdatedAt, merchantModel.DisplayName, merchantModel.DisplayName),
			"updated_at": gorm.Expr(
				"CASE WHEN ? > updated_at THEN ? ELSE updated_at END",
				merchantModel.UpdatedAt, merchantModel.UpdatedAt),
			"deleted_at": gorm.Expr(
				"CASE WHEN ? > updated_at AND deleted_at IS NULL THEN ? ELSE deleted_at END",
				merchantModel.UpdatedAt, merchantModel.DeletedAt),
		}),
	}).Create(merchantModel)
	if result.Error != nil {
		return fmt.Errorf("timestamp-based upsert failed: %w", result.Error)
	}
	return nil
}

// 將DB模型映射到領域模型
func mapToDomainMerchant(merchant *models.Merchant) *entity.Merchant {
	var deletedAt *time.Time
	if merchant.DeletedAt.Valid {
		deletedTime := merchant.DeletedAt.Time
		deletedAt = &deletedTime
	}

	return &entity.Merchant{
		ID:               merchant.ID,
		GlobalMerchantID: merchant.GlobalMerchantID,
		Name:             merchant.Name,
		DisplayName:      merchant.DisplayName,
		APIKey:           merchant.APIKey,
		CreatedAt:        merchant.CreatedAt,
		UpdatedAt:        merchant.UpdatedAt,
		DeletedAt:        deletedAt,
	}
}

// 將領域模型映射到DB模型
func mapToDBMerchant(merchant *entity.Merchant) *models.Merchant {
	dbMerchant := &models.Merchant{
		ID:               merchant.ID,
		GlobalMerchantID: merchant.GlobalMerchantID,
		Name:             merchant.Name,
		DisplayName:      merchant.DisplayName,
		APIKey:           merchant.APIKey,
		CreatedAt:        merchant.CreatedAt,
		UpdatedAt:        merchant.UpdatedAt,
	}

	if merchant.DeletedAt != nil {
		dbMerchant.DeletedAt = gorm.DeletedAt{
			Time:  *merchant.DeletedAt,
			Valid: true,
		}
	}

	return dbMerchant
}
