package repository

import (
	"context"
	"fmt"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/repositoryport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LevelRepository struct {
	db *gorm.DB
}

func NewLevelRepository(db *gorm.DB) repositoryport.LevelRepository {
	return &LevelRepository{db: db}
}

func (r *LevelRepository) Upsert(ctx context.Context, level *entity.Level) (uint64, error) {
	dbLevel := mapToDBLevel(level)
	if result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "global_player_level_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "updated_at"}),
	}).Create(dbLevel); result.Error != nil {
		return 0, fmt.Errorf("upsert level failed: %w", result.Error)
	}

	if err := r.db.WithContext(ctx).
		Where("global_player_level_id = ?", level.GlobalPlayerLevelID).
		First(dbLevel).Error; err != nil {
		return 0, fmt.Errorf("fetch level after upsert failed: %w", err)
	}
	return dbLevel.ID, nil
}

func mapToDBLevel(level *entity.Level) *models.Level {
	dbLevel := &models.Level{
		ID:                  level.ID,
		MerchantID:          level.MerchantID,
		GlobalPlayerLevelID: level.GlobalPlayerLevelID,
		Name:                level.Name,
		CreatedAt:           level.CreatedAt,
		UpdatedAt:           level.UpdatedAt,
	}
	if level.DeletedAt != nil {
		dbLevel.DeletedAt = gorm.DeletedAt{
			Time:  *level.DeletedAt,
			Valid: true,
		}
	}
	return dbLevel
}
