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

func (r *LevelRepository) Upsert(ctx context.Context, level *entity.Level) error {
	if result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "global_player_level_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"name", "updated_at"}),
	}).Create(mapToDBLevel(level)); result.Error != nil {
		return fmt.Errorf("upsert level failed: %w", result.Error)
	}
	return nil
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
