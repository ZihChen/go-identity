package repository

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/repositoryport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/models"
	"gorm.io/gorm"
)

type PlayerTagRepository struct {
	db *gorm.DB
}

func NewPlayerTagRepository(db *gorm.DB) repositoryport.PlayerTagRepository {
	return &PlayerTagRepository{db: db}
}

func (r *PlayerTagRepository) BatchUpdate(
	ctx context.Context,
	playerID uint64,
	tagIDs []uint64,
) error {
	if len(tagIDs) == 0 {
		return fmt.Errorf("tagIDs is empty")
	}

	// 排序以確保一致的鎖定順序
	sort.Slice(tagIDs, func(i, j int) bool {
		return tagIDs[i] < tagIDs[j]
	})

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("player_id = ?", playerID).Delete(&models.PlayerTag{}).Error; err != nil {
			return fmt.Errorf("delete player tags failed: %w", err)
		}
		insertItems := make([]*models.PlayerTag, len(tagIDs))
		for k, tagID := range tagIDs {
			insertItems[k] = &models.PlayerTag{
				PlayerID:  playerID,
				TagID:     tagID,
				CreatedAt: time.Now(),
			}
		}
		if err := tx.Create(&insertItems).Error; err != nil {
			return fmt.Errorf("insert player tags failed: %w", err)
		}
		return nil
	})
}

func (r *PlayerTagRepository) DeleteByPlayerID(ctx context.Context, playerID uint64) error {
	result := r.db.WithContext(ctx).
		Where("player_id = ?", playerID).
		Delete(&models.PlayerTag{})
	if result.Error != nil {
		return fmt.Errorf("delete player tags failed: %w", result.Error)
	}
	return nil
}
