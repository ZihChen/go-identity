package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/models"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PlayerRepository GORM 實現的玩家資料庫
type PlayerRepository struct {
	db    *gorm.DB
	cache infrastructure.CacheManager
}

// NewPlayerRepository 創建玩家資料庫
func NewPlayerRepository(
	db *gorm.DB,
	cache infrastructure.CacheManager,
) repository.PlayerRepository {
	return &PlayerRepository{
		db:    db,
		cache: cache,
	}
}

// 快取相關常數
const (
	playerCacheKeyPrefix = "player:global_id:"
	playerCacheTTL       = 5 * time.Minute // 5分鐘快取TTL
)

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

// FindByGlobalID 通過全局ID查找玩家（帶快取）
func (r *PlayerRepository) FindByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Player, error) {
	cacheKey := playerCacheKeyPrefix + globalID

	// 1. 先嘗試從快取獲取
	if r.cache != nil {
		if cachedData, err := r.cache.Get(ctx, cacheKey); err == nil {
			var player *entity.Player
			if unmarshalErr := json.Unmarshal([]byte(cachedData), &player); unmarshalErr == nil {
				return player, nil
			}
			// 快取數據格式錯誤，繼續查詢資料庫
		} else if !errors.Is(err, redis.Nil) {
			// 快取服務錯誤（非 key 不存在），記錄但不中斷，繼續查詢資料庫
			fmt.Printf("Cache get error for key %s: %v\n", cacheKey, err)
		}
	}

	// 2. 從資料庫查詢
	var player models.Player
	result := r.db.WithContext(ctx).Where("global_player_id = ?", globalID).First(&player)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return &entity.Player{}, errmsg.ErrRepoPlayerNotFound
		}
		return &entity.Player{}, result.Error
	}

	playerEntity := mapToDomainPlayer(&player)

	// 3. 更新快取（異步進行，不影響主流程）
	if r.cache != nil {
		go r.updateCache(ctx, cacheKey, playerEntity)
	}

	return playerEntity, nil
}

// FirstOrCreate 取得或創建，避免重複插入
func (r *PlayerRepository) FirstOrCreate(ctx context.Context, player *entity.Player) error {
	playerModel := mapToDBPlayer(player)
	result := r.db.WithContext(ctx).Where("global_player_id = ?", player.GetGlobalPlayerID()).
		FirstOrCreate(playerModel)
	if result.Error != nil {
		return result.Error
	}

	player.SetID(playerModel.ID)
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
	player.SetID(playerModel.ID)

	return nil
}

// Update 更新玩家
func (r *PlayerRepository) Update(ctx context.Context, player *entity.Player) error {
	playerModel := mapToDBPlayer(player)
	result := r.db.WithContext(ctx).Save(playerModel)
	if result.Error != nil {
		return result.Error
	}

	// 更新後使快取失效
	r.invalidateCache(ctx, player.GetGlobalPlayerID())

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
	const maxRetries = 5
	const baseDelay = 100 * time.Millisecond

	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := r.upsertWithoutRetry(ctx, player)

		if err == nil {
			return nil
		}

		// 檢查是否為 deadlock 錯誤
		if r.isDeadlockError(err) && attempt < maxRetries {
			delay := baseDelay * time.Duration(1<<uint(attempt))
			time.Sleep(delay)
			continue
		}

		// 非 deadlock 錯誤或達到最大重試次數
		return err
	}

	return fmt.Errorf("player upsert failed after %d retries", maxRetries)
}

// upsertWithoutRetry 原始的 Upsert 邏輯，不含 retry
func (r *PlayerRepository) upsertWithoutRetry(ctx context.Context, player *entity.Player) error {
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

	// Upsert 後使快取失效
	r.invalidateCache(ctx, player.GetGlobalPlayerID())

	return nil
}

// BatchUpsert 批次更新插入玩家
func (r *PlayerRepository) BatchUpsert(ctx context.Context, players []*entity.Player) error {
	if len(players) == 0 {
		return nil
	}

	const maxRetries = 5
	const baseDelay = 100 * time.Millisecond

	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := r.batchUpsertWithoutRetry(ctx, players)

		if err == nil {
			// BatchUpsert 成功後使相關快取失效
			for _, player := range players {
				r.invalidateCache(ctx, player.GetGlobalPlayerID())
			}
			return nil
		}

		// 檢查是否為 deadlock 錯誤
		if r.isDeadlockError(err) && attempt < maxRetries {
			delay := baseDelay * time.Duration(1<<uint(attempt))
			time.Sleep(delay)
			continue
		}

		// 非 deadlock 錯誤或達到最大重試次數
		return err
	}

	return fmt.Errorf("batch upsert failed after %d retries", maxRetries)
}

// batchUpsertWithoutRetry 原始的批次 Upsert 邏輯，不含 retry
func (r *PlayerRepository) batchUpsertWithoutRetry(
	ctx context.Context,
	players []*entity.Player,
) error {
	// 將領域實體轉換為資料庫模型
	playerModels := make([]*models.Player, len(players))
	for i, player := range players {
		playerModels[i] = mapToDBPlayer(player)
	}

	// 批次分塊處理，避免單次SQL過大
	const batchSize = 100
	for i := 0; i < len(playerModels); i += batchSize {
		end := i + batchSize
		if end > len(playerModels) {
			end = len(playerModels)
		}

		batchChunk := playerModels[i:end]
		if err := r.processBatchChunk(ctx, batchChunk); err != nil {
			return fmt.Errorf("process batch chunk %d-%d: %w", i, end-1, err)
		}
	}

	return nil
}

// processBatchChunk 處理單個批次分塊
func (r *PlayerRepository) processBatchChunk(ctx context.Context, players []*models.Player) error {
	// 使用 GORM 的 Clauses(clause.OnConflict{}) 處理批次 upsert
	updates := map[string]interface{}{
		"account": gorm.Expr(
			"CASE WHEN VALUES(updated_at) > updated_at AND VALUES(account) != account THEN VALUES(account) ELSE account END",
		),
		"api_key": gorm.Expr(
			"CASE WHEN VALUES(updated_at) > updated_at AND VALUES(api_key) != api_key THEN VALUES(api_key) ELSE api_key END",
		),
		"level_id": gorm.Expr(
			"CASE WHEN VALUES(updated_at) > updated_at AND VALUES(level_id) != level_id THEN VALUES(level_id) ELSE level_id END",
		),
		"email": gorm.Expr(
			"CASE WHEN VALUES(updated_at) > updated_at THEN VALUES(email) ELSE email END"),
		"last_active_at": gorm.Expr(
			"CASE WHEN VALUES(updated_at) > updated_at THEN VALUES(last_active_at) ELSE last_active_at END",
		),
		"updated_at": gorm.Expr(
			"CASE WHEN VALUES(updated_at) >= updated_at THEN VALUES(updated_at) ELSE updated_at END",
		),
		"deleted_at": gorm.Expr(
			"CASE WHEN VALUES(updated_at) > updated_at AND deleted_at IS NULL THEN VALUES(deleted_at) ELSE deleted_at END",
		),
	}

	result := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "global_player_id"}},
		DoUpdates: clause.Assignments(updates),
	}).CreateInBatches(players, len(players))

	if result.Error != nil {
		return fmt.Errorf("batch upsert failed: %w", result.Error)
	}

	return nil
}

// isDeadlockError 檢查是否為 MySQL deadlock 錯誤
func (r *PlayerRepository) isDeadlockError(err error) bool {
	if err == nil {
		return false
	}
	errorStr := err.Error()
	return strings.Contains(errorStr, "Deadlock found") ||
		strings.Contains(errorStr, "1213") ||
		strings.Contains(errorStr, "40001")
}

// 將DB模型映射到領域模型
func mapToDomainPlayer(player *models.Player) *entity.Player {
	var deletedAt *time.Time
	if player.DeletedAt.Valid {
		deletedTime := player.DeletedAt.Time
		deletedAt = &deletedTime
	}

	// 使用時間感知建構子建立Player實體
	playerEntity := entity.NewPlayerWithTimes(
		player.MerchantID,
		player.GlobalPlayerID,
		player.Account,
		player.LevelID,
		player.Email,
		player.CreatedAt,
		player.UpdatedAt,
	)

	// 設定其他欄位
	playerEntity.SetID(player.ID)
	if player.APIKey != "" {
		playerEntity.SetAPIKey(player.APIKey)
	}
	if player.LastActiveAt != nil {
		playerEntity.SetLastActiveAt(player.LastActiveAt)
	}
	if deletedAt != nil {
		playerEntity.SetDeletedAt(deletedAt)
	}

	return playerEntity
}

// 將領域模型映射到DB模型
func mapToDBPlayer(player *entity.Player) *models.Player {
	dbPlayer := &models.Player{
		ID:             player.GetID(),
		MerchantID:     player.GetMerchantID(),
		GlobalPlayerID: player.GetGlobalPlayerID(),
		LevelID:        player.GetLevelID(),
		APIKey:         player.GetAPIKey(),
		Account:        player.GetAccount(),
		Email:          player.GetEmail(),
		LastActiveAt:   player.GetLastActiveAt(),
		CreatedAt:      player.GetCreatedAt(),
		UpdatedAt:      player.GetUpdatedAt(),
	}

	if player.GetDeletedAt() != nil {
		dbPlayer.DeletedAt = gorm.DeletedAt{
			Time:  *player.GetDeletedAt(),
			Valid: true,
		}
	}

	return dbPlayer
}

// updateCache 更新快取數據
func (r *PlayerRepository) updateCache(
	ctx context.Context,
	cacheKey string,
	player *entity.Player,
) {
	if playerData, err := json.Marshal(player); err == nil {
		if _, err := r.cache.Set(ctx, cacheKey, string(playerData), playerCacheTTL); err != nil {
			fmt.Printf("Cache set error for key %s: %v\n", cacheKey, err)
		}
	}
}

// invalidateCache 使快取失效
func (r *PlayerRepository) invalidateCache(ctx context.Context, globalID string) {
	if r.cache != nil {
		cacheKey := playerCacheKeyPrefix + globalID
		if client, err := r.cache.GetClient(); err == nil && client != nil {
			client.Del(ctx, cacheKey)
		}
	}
}
