package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LevelRepository struct {
	db *gorm.DB
}

func NewLevelRepository(db *gorm.DB) repository.LevelRepository {
	return &LevelRepository{db: db}
}

func (r *LevelRepository) Upsert(ctx context.Context, level *entity.Level) error {
	dbLevel := mapToDBLevel(level)
	if result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "global_player_level_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"name": gorm.Expr(
				"CASE WHEN VALUES(updated_at) > updated_at AND name != VALUES(name) THEN VALUES(name) ELSE name END"),
			"updated_at": gorm.Expr(
				"CASE WHEN VALUES(updated_at) > updated_at THEN VALUES(updated_at) ELSE updated_at END")}),
	}).Create(dbLevel); result.Error != nil {
		return fmt.Errorf("upsert level failed: %w", result.Error)
	}
	return nil
}

func (r *LevelRepository) FindByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Level, error) {
	var dbLevel models.Level
	err := r.db.WithContext(ctx).
		Where("global_player_level_id = ?", globalID).
		First(&dbLevel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &entity.Level{}, errmsg.ErrRepoLevelNotFound
		}
		return &entity.Level{}, err
	}
	return mapToDomainLevel(&dbLevel), nil
}

func mapToDomainLevel(level *models.Level) *entity.Level {
	levelEntity := entity.NewLevelWithTimes(
		level.MerchantID,
		level.Name,
		level.GlobalPlayerLevelID,
		level.CreatedAt,
		level.UpdatedAt,
	)
	levelEntity.SetID(level.ID)
	if level.DeletedAt.Valid {
		deletedTime := level.DeletedAt.Time
		levelEntity.SetDeletedAt(&deletedTime)
	}
	return levelEntity
}

func mapToDBLevel(level *entity.Level) *models.Level {
	dbLevel := &models.Level{
		ID:                  level.GetID(),
		MerchantID:          level.GetMerchantID(),
		GlobalPlayerLevelID: level.GetGlobalPlayerLevelID(),
		Name:                level.GetName(),
		CreatedAt:           level.GetCreatedAt(),
		UpdatedAt:           level.GetUpdatedAt(),
	}
	if level.GetDeletedAt() != nil {
		dbLevel.DeletedAt = gorm.DeletedAt{
			Time:  *level.GetDeletedAt(),
			Valid: true,
		}
	}
	return dbLevel
}
