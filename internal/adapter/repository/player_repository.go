package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PlayerRepository GORM 實現的玩家資料庫
type PlayerRepository struct {
	db *gorm.DB
}

// NewPlayerRepository 創建玩家資料庫
func NewPlayerRepository(db *gorm.DB) repository.PlayerRepository {
	return &PlayerRepository{db: db}
}

// FindByID 通過ID查找玩家
func (r *PlayerRepository) FindByID(ctx context.Context, id uint64) (*entity.Player, error) {
	var player models.Player
	result := r.db.WithContext(ctx).First(&player, id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, errmsg.ErrRepoPlayerNotFound
		}
		return &entity.Player{}, result.Error
	}

	return mapToDomainPlayer(&player), nil
}

// FindByGlobalID 通過全局ID查找玩家
func (r *PlayerRepository) FindByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Player, error) {
	var player models.Player
	result := r.db.WithContext(ctx).Where("global_player_id = ?", globalID).First(&player)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return &entity.Player{}, errmsg.ErrRepoPlayerNotFound
		}
		return &entity.Player{}, result.Error
	}

	return mapToDomainPlayer(&player), nil
}

// FirstOrCreate 取得或創建，避免重複插入
func (r *PlayerRepository) FirstOrCreate(ctx context.Context, player *entity.Player) error {
	playerModel := mapToDBPlayer(player)
	result := r.db.WithContext(ctx).Where("global_player_id = ?", player.GlobalPlayerID).
		FirstOrCreate(playerModel)
	if result.Error != nil {
		return result.Error
	}

	player.ID = playerModel.ID
	return nil
}

// Create 創建玩家
func (r *PlayerRepository) Create(ctx context.Context, player *entity.Player) error {
	playerModel := mapToDBPlayer(player)
	result := r.db.WithContext(ctx).Create(playerModel)
	if result.Error != nil {
		return result.Error
	}

	// 更新ID
	player.ID = playerModel.ID

	return nil
}

// Update 更新玩家
func (r *PlayerRepository) Update(ctx context.Context, player *entity.Player) error {
	playerModel := mapToDBPlayer(player)
	result := r.db.WithContext(ctx).Save(playerModel)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

// Delete 刪除玩家
func (r *PlayerRepository) Delete(ctx context.Context, id uint64) error {
	result := r.db.WithContext(ctx).Delete(&models.Player{}, id)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errmsg.ErrRepoDeletePlayerNotFound
	}
	return nil
}

// Upsert 資料冪等性設計：只有當新資料的UpdatedAt要大於當前資料，並且內容要不同時才更新
func (r *PlayerRepository) Upsert(ctx context.Context, player *entity.Player) error {
	playerModel := mapToDBPlayer(player)

	updates := map[string]interface{}{
		"account": gorm.Expr(
			"CASE WHEN ? > updated_at AND account != ? THEN ? ELSE account END",
			playerModel.UpdatedAt, playerModel.Account, playerModel.Account),
		"api_key": gorm.Expr(
			"CASE WHEN ? > updated_at AND api_key != ? THEN ? ELSE api_key END",
			playerModel.UpdatedAt, playerModel.APIKey, playerModel.APIKey),
		"level_id": gorm.Expr(
			"CASE WHEN ? > updated_at AND level_id != ? THEN ? ELSE level_id END",
			playerModel.UpdatedAt, playerModel.LevelID, playerModel.LevelID),
		"email": gorm.Expr(
			"CASE WHEN ? > updated_at THEN ? ELSE email END",
			playerModel.UpdatedAt, playerModel.Email),
		"last_active_at": gorm.Expr(
			"CASE WHEN ? > updated_at THEN ? ELSE last_active_at END",
			playerModel.LastActiveAt, playerModel.LastActiveAt),
		"updated_at": gorm.Expr(
			"CASE WHEN VALUES(updated_at) >= updated_at THEN VALUES(updated_at) ELSE updated_at END",
		),
		"deleted_at": gorm.Expr(
			"CASE WHEN ? > updated_at AND deleted_at IS NULL THEN ? ELSE deleted_at END",
			playerModel.UpdatedAt, playerModel.DeletedAt),
	}

	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "global_player_id"}},
		DoUpdates: clause.Assignments(updates),
	}).Create(playerModel)

	if result.Error != nil {
		return fmt.Errorf("timestamp-based upsert failed: %w", result.Error)
	}
	return nil
}

// 將DB模型映射到領域模型
func mapToDomainPlayer(player *models.Player) *entity.Player {
	var deletedAt *time.Time
	if player.DeletedAt.Valid {
		deletedTime := player.DeletedAt.Time
		deletedAt = &deletedTime
	}

	return &entity.Player{
		ID:             player.ID,
		MerchantID:     player.MerchantID,
		GlobalPlayerID: player.GlobalPlayerID,
		APIKey:         player.APIKey,
		Account:        player.Account,
		Email:          player.Email,
		LastActiveAt:   player.LastActiveAt,
		CreatedAt:      player.CreatedAt,
		UpdatedAt:      player.UpdatedAt,
		DeletedAt:      deletedAt,
	}
}

// 將領域模型映射到DB模型
func mapToDBPlayer(player *entity.Player) *models.Player {
	dbPlayer := &models.Player{
		ID:             player.ID,
		MerchantID:     player.MerchantID,
		GlobalPlayerID: player.GlobalPlayerID,
		LevelID:        player.LevelID,
		APIKey:         player.APIKey,
		Account:        player.Account,
		Email:          player.Email,
		LastActiveAt:   player.LastActiveAt,
		CreatedAt:      player.CreatedAt,
		UpdatedAt:      player.UpdatedAt,
	}

	if player.DeletedAt != nil {
		dbPlayer.DeletedAt = gorm.DeletedAt{
			Time:  *player.DeletedAt,
			Valid: true,
		}
	}

	return dbPlayer
}
