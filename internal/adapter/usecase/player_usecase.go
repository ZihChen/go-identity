package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/model"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/repositoryport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/serviceport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/usecaseport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
)

// PlayerUseCase 玩家用例
type PlayerUseCase struct {
	playerRepo    repositoryport.PlayerRepository
	merchantRepo  repositoryport.MerchantRepository
	eventProducer serviceport.EventProducer
	logger        *zap.Logger
}

// NewPlayerUseCase 創建玩家用例
func NewPlayerUseCase(
	playerRepo repositoryport.PlayerRepository,
	merchantRepo repositoryport.MerchantRepository,
	eventProducer serviceport.EventProducer,
	logger *zap.Logger,
) usecaseport.PlayerUseCase {
	return &PlayerUseCase{
		playerRepo:    playerRepo,
		merchantRepo:  merchantRepo,
		eventProducer: eventProducer,
		logger:        logger,
	}
}

// SyncPlayer 同步玩家信息
func (u *PlayerUseCase) SyncPlayer(ctx context.Context, eventData []byte) error {
	// 獲取當前 span
	span := trace.SpanFromContext(ctx)

	// 將事件解析為 CloudEvent
	var cloudEvent event.CloudEvent
	if err := json.Unmarshal(eventData, &cloudEvent); err != nil {
		span.RecordError(err)
		return fmt.Errorf("unmarshal cloud event: %w", err)
	}

	// 添加事件信息到 span
	span.SetAttributes(
		attribute.String("event.id", cloudEvent.ID),
		attribute.String("event.type", cloudEvent.Type),
		attribute.String("event.source", cloudEvent.Source),
	)

	// 記錄事件開始處理
	tracing.TraceEvent(span, "Starting player sync processing")

	// 將 data 部分解析為 PlayerSyncEvent
	dataBytes, err := json.Marshal(cloudEvent.Data)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("marshal event data: %w", err)
	}

	var playerEvent event.PlayerSyncEvent
	if err := json.Unmarshal(dataBytes, &playerEvent); err != nil {
		span.RecordError(err)
		return fmt.Errorf("unmarshal player event: %w", err)
	}

	// 添加玩家信息到 span
	span.SetAttributes(
		attribute.String("player.global_id", playerEvent.Player.GlobalPlayerID),
		attribute.String("player.account", playerEvent.Player.Account),
		attribute.String("merchant.global_id", playerEvent.GlobalMerchantID),
	)

	// 查找商戶是否存在
	tracing.TraceEvent(span, "Checking if merchant exists")
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, playerEvent.GlobalMerchantID)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("find merchant: %w", err)
	}

	// 查找玩家是否存在
	tracing.TraceEvent(span, "Checking if player exists")
	existing, err := u.playerRepo.FindByGlobalID(ctx, playerEvent.Player.GlobalPlayerID)
	if err != nil && err.Error() != "record not found" {
		span.RecordError(err)
		return fmt.Errorf("find player: %w", err)
	}

	// 創建或更新玩家
	var player model.Player
	if existing == nil {
		// 創建新玩家
		tracing.TraceEvent(span, "Creating new player")
		var email *string
		if playerEvent.Player.Email != "" {
			email = &playerEvent.Player.Email
		}

		player = model.Player{
			MerchantID:     merchant.ID,
			GlobalPlayerID: playerEvent.Player.GlobalPlayerID,
			APIKey:         uuid.New().String(), // 生成新的API密鑰
			Account:        playerEvent.Player.Account,
			Email:          email,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		if err := u.playerRepo.Create(ctx, &player); err != nil {
			span.RecordError(err)
			return fmt.Errorf("create player: %w", err)
		}
		u.logger.Info("Player created",
			zap.String("global_id", player.GlobalPlayerID),
			zap.String("account", player.Account))
	} else {
		// 更新現有玩家
		tracing.TraceEvent(span, "Updating existing player")
		player = *existing
		player.Account = playerEvent.Player.Account
		if playerEvent.Player.Email != "" {
			player.Email = &playerEvent.Player.Email
		}
		player.UpdatedAt = time.Now()

		if err := u.playerRepo.Update(ctx, &player); err != nil {
			span.RecordError(err)
			return fmt.Errorf("update player: %w", err)
		}
		u.logger.Info("Player updated",
			zap.String("global_id", player.GlobalPlayerID),
			zap.String("account", player.Account))
	}

	// 記錄資料庫操作完成
	tracing.TraceEvent(span, "Database operation completed")

	// 發布玩家同步事件到KDS
	tracing.TraceEvent(span, "Publishing player sync event to KDS")
	if err := u.publishPlayerSyncEvent(ctx, &player, playerEvent.GlobalMerchantID, cloudEvent.TraceParent); err != nil {
		span.RecordError(err)
		return fmt.Errorf("publish player sync event: %w", err)
	}

	// 記錄處理完成
	tracing.TraceEvent(span, "Player sync completed successfully")

	return nil
}

// publishPlayerSyncEvent 發布玩家同步事件
func (u *PlayerUseCase) publishPlayerSyncEvent(ctx context.Context, player *model.Player, globalMerchantID string, traceParent string) error {
	// 獲取當前 span
	ctx, span := tracing.StartSpan(ctx, "PlayerUseCase.publishPlayerSyncEvent")
	defer span.End()

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

	// 添加事件信息到 span
	span.SetAttributes(
		attribute.String("outgoing.event.id", eventID),
		attribute.String("outgoing.event.type", cloudEvent.Type),
	)

	// 發布事件
	if err := u.eventProducer.PublishPlayerSync(ctx, &cloudEvent); err != nil {
		span.RecordError(err)
		return fmt.Errorf("publish player sync: %w", err)
	}

	// 記錄事件發布成功
	tracing.TraceEvent(span, "Player sync event published successfully")

	u.logger.Info("Player sync event published",
		zap.String("global_id", player.GlobalPlayerID),
		zap.String("event_id", cloudEvent.ID))

	return nil
}

// GetPlayerByID 通過ID獲取玩家
func (u *PlayerUseCase) GetPlayerByID(ctx context.Context, id uint64) (*model.Player, error) {
	// 創建 span 並跟踪此操作
	ctx, span := tracing.StartSpan(ctx, "PlayerUseCase.GetPlayerByID")
	defer span.End()

	span.SetAttributes(attribute.Int64("player.id", int64(id)))

	player, err := u.playerRepo.FindByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("find player: %w", err)
	}

	// 添加玩家信息到 span
	span.SetAttributes(
		attribute.String("player.global_id", player.GlobalPlayerID),
		attribute.String("player.account", player.Account),
	)

	if err = u.publishPlayerSyncEvent(ctx, player, "123", "321"); err != nil {
		return nil, err
	}

	return player, nil
}

// GetPlayerByGlobalID 通過全局ID獲取玩家
func (u *PlayerUseCase) GetPlayerByGlobalID(ctx context.Context, globalID string) (*model.Player, error) {
	// 創建 span 並跟踪此操作
	ctx, span := tracing.StartSpan(ctx, "PlayerUseCase.GetPlayerByGlobalID")
	defer span.End()

	span.SetAttributes(attribute.String("player.global_id", globalID))

	player, err := u.playerRepo.FindByGlobalID(ctx, globalID)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("find player: %w", err)
	}

	// 添加玩家信息到 span
	span.SetAttributes(
		attribute.String("player.account", player.Account),
	)

	return player, nil
}

// UpdatePlayerLastActive 更新玩家最後活躍時間
func (u *PlayerUseCase) UpdatePlayerLastActive(ctx context.Context, id uint64) error {
	// 創建 span 並跟踪此操作
	ctx, span := tracing.StartSpan(ctx, "PlayerUseCase.UpdatePlayerLastActive")
	defer span.End()

	span.SetAttributes(attribute.Int64("player.id", int64(id)))

	// 查找玩家
	player, err := u.playerRepo.FindByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("find player: %w", err)
	}

	// 添加玩家信息到 span
	span.SetAttributes(
		attribute.String("player.global_id", player.GlobalPlayerID),
		attribute.String("player.account", player.Account),
	)

	// 更新最後活躍時間
	now := time.Now()
	player.LastActiveAt = &now
	player.UpdatedAt = now

	if err := u.playerRepo.Update(ctx, player); err != nil {
		span.RecordError(err)
		return fmt.Errorf("update player: %w", err)
	}

	u.logger.Info("Player last active time updated",
		zap.String("global_id", player.GlobalPlayerID),
		zap.Time("last_active_at", *player.LastActiveAt))

	return nil
}
