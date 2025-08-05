package usecase

import (
	"context"
	"errors"
	"fmt"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/errmsg"
	"time"

	"github.com/google/uuid"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/infraport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/repositoryport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/serviceport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/usecaseport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
)

// PlayerUseCase 玩家用例
type PlayerUseCase struct {
	playerRepo    repositoryport.PlayerRepository
	merchantRepo  repositoryport.MerchantRepository
	eventProducer serviceport.EventProducer
	logger        infraport.Logger
	redis         *redis.Client
}

// NewPlayerUseCase 創建玩家用例
func NewPlayerUseCase(
	playerRepo repositoryport.PlayerRepository,
	merchantRepo repositoryport.MerchantRepository,
	eventProducer serviceport.EventProducer,
	logger infraport.Logger,
	redis *redis.Client,
) usecaseport.PlayerUseCase {
	return &PlayerUseCase{
		playerRepo:    playerRepo,
		merchantRepo:  merchantRepo,
		eventProducer: eventProducer,
		logger:        logger,
		redis:         redis,
	}
}

func (u *PlayerUseCase) SyncPlayer(
	ctx context.Context,
	playerData *event.PlayerData,
	globalMerchantID string,
) error {
	ctx, span := tracing.StartSpan(ctx, "PlayerUseCase.SyncPlayer")
	defer tracing.SpanEnd(span)

	tracing.TraceEvent(span, "Checking if merchant exists")
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, globalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("find merchant: %w", err)
	}

	var email *string
	if playerData.Email != "" {
		email = &playerData.Email
	}
	player := entity.Player{
		MerchantID:     merchant.ID,
		GlobalPlayerID: playerData.GlobalPlayerID,
		LevelID:        playerData.LevelID,
		APIKey:         uuid.New().String(), // 生成新的API密鑰
		Account:        playerData.Account,
		Email:          email,
		CreatedAt:      playerData.UpdatedAt,
		UpdatedAt:      playerData.UpdatedAt,
		DeletedAt: func() *time.Time {
			if playerData.DeletedAt == "" {
				return nil
			}
			nowTime := time.Now()
			return &nowTime
		}(),
	}

	tracing.TraceEvent(span, "Upsert player")
	if err = u.playerRepo.Upsert(ctx, &player); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("upsert player: %w", err)
	}

	// 記錄資料庫操作完成
	u.logger.InfoLog("Player upserted successfully",
		u.logger.String("global_id", player.GlobalPlayerID),
		u.logger.String("account", player.Account))
	tracing.TraceEvent(span, "Database operation completed")
	tracing.RecordSpanAttributes(span,
		attribute.String("player.global_id", player.GlobalPlayerID),
		attribute.String("player.account", player.Account))

	// 發布玩家同步事件到KDS
	tracing.TraceEvent(span, "Publishing player sync event to KDS")
	if err = u.publishPlayerSyncEvent(ctx, &player, globalMerchantID); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("publish player sync event: %w", err)
	}

	// 記錄處理完成
	tracing.TraceEvent(span, "Player sync completed successfully")
	return nil
}

// publishPlayerSyncEvent 發布玩家同步事件
func (u *PlayerUseCase) publishPlayerSyncEvent(
	ctx context.Context,
	player *entity.Player,
	globalMerchantID string,
) error {
	ctx, span := tracing.StartSpan(ctx, "PlayerUseCase.publishPlayerSyncEvent")
	defer tracing.SpanEnd(span)

	// 記錄發布事件開始
	tracing.TraceEvent(span, "Preparing player sync event for KDS")

	// 構建事件數據
	syncEvent := event.IdentityPlayerSyncEvent{
		GlobalMerchantID: globalMerchantID,
		GlobalPlayerID:   player.GlobalPlayerID,
		ID:               player.ID,
		MerchantID:       player.MerchantID,
		APIKey:           player.APIKey,
		Account:          player.Account,
		Email:            player.Email,
		CreatedAt:        player.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        player.UpdatedAt.Format(time.RFC3339),
	}

	if player.LastActiveAt != nil {
		syncEvent.LastActiveAt = player.LastActiveAt.Format(time.RFC3339)
	}

	if player.DeletedAt != nil {
		syncEvent.DeletedAt = player.DeletedAt.Format(time.RFC3339)
	}

	// 構建CloudEvent
	eventID := uuid.New().String()
	cloudEvent := event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "tw.jvd.fatidentitycat.player.sync.v1",
		Source:          "/fatidentitycat/FATCAT",
		Subject:         "player_sync",
		ID:              eventID,
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     tracing.GetTraceparent(ctx),
		Data:            syncEvent,
	}

	tracing.RecordSpanAttributes(span,
		attribute.String("outgoing.event.id", eventID),
		attribute.String("outgoing.event.type", cloudEvent.Type),
	)

	// 發布事件
	if err := u.eventProducer.PublishPlayerSync(ctx, &cloudEvent); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("publish player sync: %w", err)
	}

	// 記錄事件發布成功
	tracing.TraceEvent(span, "Player sync event published successfully")

	u.logger.InfoLog("Player sync event published",
		u.logger.String("global_id", player.GlobalPlayerID),
		u.logger.String("event_id", cloudEvent.ID))

	return nil
}

// GetPlayerByID 通過ID獲取玩家
func (u *PlayerUseCase) GetPlayerByID(ctx context.Context, id uint64) (*entity.Player, error) {
	// 創建 span 並跟踪此操作
	ctx, span := tracing.StartSpan(ctx, "PlayerUseCase.GetPlayerByID")
	defer tracing.SpanEnd(span)

	tracing.RecordSpanAttributes(span, attribute.Int64("player.id", int64(id)))

	player, err := u.playerRepo.FindByID(ctx, id)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return nil, fmt.Errorf("find player: %w", err)
	}

	// 添加玩家信息到 span
	tracing.RecordSpanAttributes(span,
		attribute.String("player.global_id", player.GlobalPlayerID),
		attribute.String("player.account", player.Account),
	)

	if err = u.publishPlayerSyncEvent(ctx, player, "123"); err != nil {
		return nil, err
	}

	return player, nil
}

// GetPlayerByGlobalID 通過全局ID獲取玩家
func (u *PlayerUseCase) GetPlayerByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Player, error) {
	// 創建 span 並跟踪此操作
	ctx, span := tracing.StartSpan(ctx, "PlayerUseCase.GetPlayerByGlobalID")
	defer tracing.SpanEnd(span)

	tracing.RecordSpanAttributes(span, attribute.String("player.global_id", globalID))

	player, err := u.playerRepo.FindByGlobalID(ctx, globalID)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return nil, fmt.Errorf("find player: %w", err)
	}

	// 添加玩家信息到 span
	tracing.RecordSpanAttributes(span,
		attribute.String("player.account", player.Account),
	)

	return player, nil
}

// UpdatePlayerLastActive 更新玩家最後活躍時間
func (u *PlayerUseCase) UpdatePlayerLastActive(ctx context.Context, id uint64) error {
	// 創建 span 並跟踪此操作
	ctx, span := tracing.StartSpan(ctx, "PlayerUseCase.UpdatePlayerLastActive")
	defer tracing.SpanEnd(span)

	tracing.RecordSpanAttributes(span, attribute.Int64("player.id", int64(id)))

	// 查找玩家
	player, err := u.playerRepo.FindByID(ctx, id)
	if err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("find player: %w", err)
	}

	// 添加玩家信息到 span
	tracing.RecordSpanAttributes(span,
		attribute.String("player.global_id", player.GlobalPlayerID),
		attribute.String("player.account", player.Account),
	)

	// 更新最後活躍時間
	now := time.Now()
	player.LastActiveAt = &now
	player.UpdatedAt = now

	if err := u.playerRepo.Update(ctx, player); err != nil {
		tracing.RecordSpanError(span, err)
		return fmt.Errorf("update player: %w", err)
	}

	u.logger.InfoLog("Player last active time updated",
		u.logger.String("global_id", player.GlobalPlayerID),
		u.logger.String("last_active_at", player.LastActiveAt.String()))

	return nil
}
