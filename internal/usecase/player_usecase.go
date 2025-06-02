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
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/infra"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/model"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/repository"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/service"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
)

// PlayerUseCase 玩家用例
type PlayerUseCase struct {
	playerRepo    repository.PlayerRepository
	merchantRepo  repository.MerchantRepository
	eventProducer service.EventProducer
	logger        *zap.Logger
	asyncLogger   infra.Logger
}

// NewPlayerUseCase 創建玩家用例
func NewPlayerUseCase(
	playerRepo repository.PlayerRepository,
	merchantRepo repository.MerchantRepository,
	eventProducer service.EventProducer,
	logger *zap.Logger,
	asyncLogger infra.Logger,
) *PlayerUseCase {
	return &PlayerUseCase{
		playerRepo:    playerRepo,
		merchantRepo:  merchantRepo,
		eventProducer: eventProducer,
		logger:        logger,
		asyncLogger:   asyncLogger,
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
		attribute.String("merchant.global_id", playerEvent.GlobalMerchantID),
		attribute.String("player.global_id", playerEvent.Player.GlobalPlayerID),
		attribute.String("player.account", playerEvent.Player.Account),
	)

	// 查找對應的商戶
	tracing.TraceEvent(span, "Finding merchant")
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, playerEvent.GlobalMerchantID)
	if err != nil {
		span.RecordError(err)
		// 如果找不到商戶，將事件放回隊列延遲處理
		if err.Error() == "record not found" {
			u.logger.Warn("Merchant not found, re-queuing player sync event",
				zap.String("global_merchant_id", playerEvent.GlobalMerchantID),
				zap.String("global_player_id", playerEvent.Player.GlobalPlayerID))
			return fmt.Errorf("find merchant (will retry): %w", err)
		}
		return fmt.Errorf("find merchant: %w", err)
	}

	span.SetAttributes(attribute.Int64("merchant.id", int64(merchant.ID)))

	// 查找玩家是否存在
	tracing.TraceEvent(span, "Checking if player exists")
	existing, err := u.playerRepo.FindByGlobalID(ctx, playerEvent.Player.GlobalPlayerID)
	if err != nil && err.Error() != "record not found" {
		span.RecordError(err)
		return fmt.Errorf("find player: %w", err)
	}

	// 設置電子郵件
	var email *string
	if playerEvent.Player.Email != "" {
		email = &playerEvent.Player.Email
	}

	// 創建或更新玩家
	var player model.Player
	if existing == nil {
		// 創建新玩家
		tracing.TraceEvent(span, "Creating new player")
		player = model.Player{
			MerchantID:     merchant.ID,
			GlobalPlayerID: playerEvent.Player.GlobalPlayerID,
			APIKey:         uuid.New().String(), // 生成新的API密鑰
			Account:        playerEvent.Player.Account,
			Email:          email,
			LastActiveAt:   nil,
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

		// 確保幂等性：檢查更新時間，只有更新的數據才會覆蓋現有數據
		// 從事件中獲取最後更新時間
		var eventTime time.Time
		if cloudEvent.Time.After(time.Time{}) {
			eventTime = cloudEvent.Time
		} else {
			eventTime = time.Now()
		}

		// 如果現有記錄的更新時間較新，則跳過更新（確保幂等性）
		if existing.UpdatedAt.After(eventTime) {
			u.logger.Info("Skipping player update as existing data is newer",
				zap.String("global_id", existing.GlobalPlayerID),
				zap.Time("existing_updated_at", existing.UpdatedAt),
				zap.Time("event_time", eventTime))

			// 發布玩家同步事件到KDS確認我們已處理
			tracing.TraceEvent(span, "Publishing player sync confirmation event to KDS")
			if err := u.publishPlayerSyncEvent(ctx, existing, playerEvent.GlobalMerchantID, cloudEvent.TraceParent); err != nil {
				u.logger.Warn("Failed to publish player sync confirmation event",
					zap.String("global_id", existing.GlobalPlayerID),
					zap.Error(err))
			}

			return nil
		}

		player = *existing
		player.Account = playerEvent.Player.Account
		player.Email = email
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
	span.SetAttributes(attribute.Int64("player.id", int64(player.ID)))

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

// 發布玩家同步事件
func (u *PlayerUseCase) publishPlayerSyncEvent(ctx context.Context, player *model.Player, globalMerchantID, traceParent string) error {
	// 獲取當前 span
	span := trace.SpanFromContext(ctx)

	// 記錄發布事件開始
	tracing.TraceEvent(span, "Preparing player sync event for KDS")

	// 構建事件數據
	var lastActiveAt string
	if player.LastActiveAt != nil {
		lastActiveAt = player.LastActiveAt.Format(time.RFC3339)
	}

	var deletedAt string
	if player.DeletedAt != nil {
		deletedAt = player.DeletedAt.Format(time.RFC3339)
	}

	syncEvent := event.IdentityPlayerSyncEvent{
		GlobalMerchantID: globalMerchantID,
		GlobalPlayerID:   player.GlobalPlayerID,
		ID:               player.ID,
		MerchantID:       player.MerchantID,
		APIKey:           player.APIKey,
		Account:          player.Account,
		Email:            player.Email,
		LastActiveAt:     lastActiveAt,
		CreatedAt:        player.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        player.UpdatedAt.Format(time.RFC3339),
		DeletedAt:        deletedAt,
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
		attribute.Int64("merchant.id", int64(player.MerchantID)),
	)

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
		attribute.Int64("player.id", int64(player.ID)),
		attribute.String("player.account", player.Account),
		attribute.Int64("merchant.id", int64(player.MerchantID)),
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
	tracing.TraceEvent(span, "Finding player")
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

	// 更新玩家
	tracing.TraceEvent(span, "Updating player last active time")
	if err := u.playerRepo.Update(ctx, player); err != nil {
		span.RecordError(err)
		return fmt.Errorf("update player: %w", err)
	}

	// 記錄更新成功
	tracing.TraceEvent(span, "Player last active time updated successfully",
		attribute.String("last_active_at", now.Format(time.RFC3339)))

	return nil
}
