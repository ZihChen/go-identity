package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/jvdiamondtech/ms-identity-cat/internal/adapter/model"
	domainModel "github.com/jvdiamondtech/ms-identity-cat/internal/domain/model"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/repository"
)

// MerchantRepository GORM 實現的商戶資料庫
type MerchantRepository struct {
	db *gorm.DB
}

// NewMerchantRepository 創建商戶資料庫
func NewMerchantRepository(db *gorm.DB) repository.MerchantRepository {
	return &MerchantRepository{db: db}
}

// FindByID 通過ID查找商戶
func (r *MerchantRepository) FindByID(ctx context.Context, id uint64) (*domainModel.Merchant, error) {
	var merchant model.Merchant
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
func (r *MerchantRepository) FindByGlobalID(ctx context.Context, globalID string) (*domainModel.Merchant, error) {
	var merchant model.Merchant
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
func (r *MerchantRepository) Create(ctx context.Context, merchant *domainModel.Merchant) error {
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
func (r *MerchantRepository) Update(ctx context.Context, merchant *domainModel.Merchant) error {
	merchantModel := mapToDBMerchant(merchant)
	result := r.db.WithContext(ctx).Save(merchantModel)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// Delete 刪除商戶
func (r *MerchantRepository) Delete(ctx context.Context, id uint64) error {
	result := r.db.WithContext(ctx).Delete(&model.Merchant{}, id)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("record not found")
	}

	return nil
}

// 將DB模型映射到領域模型
func mapToDomainMerchant(merchant *model.Merchant) *domainModel.Merchant {
	var deletedAt *time.Time
	if merchant.DeletedAt.Valid {
		deletedTime := merchant.DeletedAt.Time
		deletedAt = &deletedTime
	}

	return &domainModel.Merchant{
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
func mapToDBMerchant(merchant *domainModel.Merchant) *model.Merchant {
	dbMerchant := &model.Merchant{
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
