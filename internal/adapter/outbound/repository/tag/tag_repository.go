package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TagRepository struct {
	db *gorm.DB
}

func NewTagRepository(db *gorm.DB) repository.TagRepository {
	return &TagRepository{db: db}
}

func (r *TagRepository) Upsert(ctx context.Context, tag *entity.Tag) error {
	tagModel := mapToDBTag(tag)
	result := r.db.WithContext(ctx).Debug().Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "global_tag_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"name": gorm.Expr(
				"CASE WHEN VALUES(updated_at) > updated_at AND name != VALUES(name) THEN VALUES(name) ELSE name END",
			),
			"updated_at": gorm.Expr(
				"CASE WHEN VALUES(updated_at) > updated_at THEN VALUES(updated_at) ELSE updated_at END",
			),
			"deleted_at": gorm.Expr(
				"CASE WHEN VALUES(updated_at) > updated_at AND deleted_at IS NULL THEN VALUES(deleted_at) ELSE deleted_at END",
			),
		}),
	}).Create(&tagModel)

	if result.Error != nil {
		return fmt.Errorf("upsert tag failed: %w", result.Error)
	}
	return nil
}

func (r *TagRepository) BatchUpsert(ctx context.Context, tags []*entity.Tag) error {
	if len(tags) == 0 {
		return nil
	}
	tagModels := make([]*models.Tag, len(tags))
	for k, tag := range tags {
		dbTag := mapToDBTag(tag)
		tagModels[k] = dbTag
	}

	result := r.db.WithContext(ctx).Debug().Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "global_tag_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"name": gorm.Expr(
				"CASE WHEN VALUES(updated_at) > updated_at AND name != VALUES(name) THEN VALUES(name) ELSE name END",
			),
			"updated_at": gorm.Expr(
				"CASE WHEN VALUES(updated_at) > updated_at THEN VALUES(updated_at) ELSE updated_at END",
			),
		}),
	}).Create(&tagModels)

	if result.Error != nil {
		return fmt.Errorf("batch upsert tags failed: %w", result.Error)
	}
	return nil
}

func (r *TagRepository) FindByGlobalIDs(
	ctx context.Context,
	globalIDs []string,
) ([]*entity.Tag, error) {
	var tagModels []*models.Tag

	result := r.db.WithContext(ctx).
		Where("global_tag_id IN ?", globalIDs).
		Where("deleted_at IS NULL"). // 如果使用了软删除，确保只查询未删除的记录
		Find(&tagModels)
	if result.Error != nil {
		return nil, fmt.Errorf("find tags by global ids failed: %w", result.Error)
	}

	// Map models to domain entities
	tags := make([]*entity.Tag, len(tagModels))
	for i, tagModel := range tagModels {
		tags[i] = mapToDomainTag(tagModel)
	}

	return tags, nil
}

func mapToDBTag(tag *entity.Tag) *models.Tag {
	dbTag := &models.Tag{
		ID:          tag.GetID(),
		MerchantID:  tag.GetMerchantID(),
		GlobalTagID: tag.GetGlobalTagID(),
		Name:        tag.GetName(),
		CreatedAt:   tag.GetCreatedAt(),
		UpdatedAt:   tag.GetUpdatedAt(),
	}
	if tag.GetDeletedAt() != nil {
		dbTag.DeletedAt = gorm.DeletedAt{
			Time:  *tag.GetDeletedAt(),
			Valid: true,
		}
	}
	return dbTag
}

// mapToDomainTag maps from models.Tag to entity.Tag
func mapToDomainTag(tag *models.Tag) *entity.Tag {
	var deletedAt *time.Time
	if tag.DeletedAt.Valid {
		deletedTime := tag.DeletedAt.Time
		deletedAt = &deletedTime
	}

	tagEntity := entity.NewTagWithTimes(
		tag.MerchantID,
		tag.Name,
		tag.GlobalTagID,
		tag.UpdatedAt,
	)
	tagEntity.SetID(tag.ID)
	tagEntity.SetCreatedAt(tag.CreatedAt)
	if deletedAt != nil {
		tagEntity.SetDeletedAt(deletedAt)
	}
	return tagEntity
}
