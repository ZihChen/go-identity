package repository

import (
	"context"
	"fmt"

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
	tags := make([]*entity.Tag, len(globalIDs))

	result := r.db.WithContext(ctx).
		Where("global_tag_id IN ?", globalIDs).
		Where("deleted_at IS NULL"). // 如果使用了软删除，确保只查询未删除的记录
		Find(&tags)
	if result.Error != nil {
		return tags, fmt.Errorf("find tags by global ids failed: %w", result.Error)
	}
	return tags, nil
}

func mapToDBTag(tag *entity.Tag) *models.Tag {
	dbTag := &models.Tag{
		ID:          tag.ID,
		MerchantID:  tag.MerchantID,
		GlobalTagID: tag.GlobalTagID,
		Name:        tag.Name,
		CreatedAt:   tag.CreatedAt,
		UpdatedAt:   tag.UpdatedAt,
	}
	if tag.DeletedAt != nil {
		dbTag.DeletedAt = gorm.DeletedAt{
			Time:  *tag.DeletedAt,
			Valid: true,
		}
	}
	return dbTag
}
