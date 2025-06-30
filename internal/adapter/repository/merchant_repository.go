package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/repositoryport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/models"
	"gorm.io/gorm"
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
			return nil, fmt.Errorf("record not found")
		}
		return nil, result.Error
	}

	return mapToDomainMerchant(&merchant), nil
}

// FindByGlobalID 通過全局ID查找商戶
func (r *merchantRepository) FindByGlobalID(ctx context.Context, globalID string) (*entity.Merchant, error) {
	var merchant models.Merchant
	result := r.db.WithContext(ctx).Where("global_merchant_id = ?", globalID).First(&merchant)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("record not found")
		}
		return nil, result.Error
	}

	return mapToDomainMerchant(&merchant), nil
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
		return fmt.Errorf("record not found")
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
