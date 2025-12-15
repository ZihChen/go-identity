package usecase

import (
	"context"
	"errors"
	"fmt"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/consts"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/service"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/cache"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// PlayerUseCase 玩家用例
type PlayerUseCase struct {
	playerRepo     repository.PlayerRepository
	merchantRepo   repository.MerchantRepository
	levelRepo      repository.LevelRepository
	eventProducer  service.EventProducer
	logger         infrastructure.Logger
	redis          *redis.Client
	tracing        infrastructure.TracingService
	cache          infrastructure.CacheManager
	batchProcessor *PlayerBatchProcessor
}

// NewPlayerUseCase 創建玩家用例
func NewPlayerUseCase(
	playerRepo repository.PlayerRepository,
	merchantRepo repository.MerchantRepository,
	levelRepo repository.LevelRepository,
	eventProducer service.EventProducer,
	logger infrastructure.Logger,
	redis *redis.Client,
	tracing infrastructure.TracingService,
	cache infrastructure.CacheManager,
) inbound.PlayerUseCase {
	usecase := &PlayerUseCase{
		playerRepo:    playerRepo,
		merchantRepo:  merchantRepo,
		levelRepo:     levelRepo,
		eventProducer: eventProducer,
		logger:        logger,
		redis:         redis,
		tracing:       tracing,
		cache:         cache,
		// 初始化批次處理器
		batchProcessor: NewPlayerBatchProcessor(
			playerRepo,
			eventProducer,
			logger,
			tracing,
			cache,
		),
	}

	return usecase
}

// StartBatchProcessor 啟動批次處理器
func (u *PlayerUseCase) StartBatchProcessor(ctx context.Context) error {
	return u.batchProcessor.Start(ctx)
}

// StopBatchProcessor 停止批次處理器
func (u *PlayerUseCase) StopBatchProcessor(ctx context.Context) error {
	return u.batchProcessor.Stop(ctx)
}

func (u *PlayerUseCase) SyncPlayer(
	ctx context.Context,
	data *event.PlayerSyncEvent,
) error {
	ctx, span := u.tracing.StartSpan(ctx, "PlayerUseCase.SyncPlayer")
	defer u.tracing.SpanEnd(span)

	u.tracing.TraceEvent(span, "Checking if merchant exists")
	cacheKey := fmt.Sprintf(consts.RedisMerchantGlobalIDKey, data.GlobalMerchantID)
	merchant, err := cache.QueryWithCache(
		ctx,
		u.cache,
		cacheKey,
		5*time.Minute,
		"merchant",
		func(ctx context.Context) (*entity.Merchant, error) {
			return u.merchantRepo.FindByGlobalID(ctx, data.GlobalMerchantID)
		},
	)
	if err != nil {
		u.tracing.RecordSpanError(span, err)
		return fmt.Errorf("find merchant: %w", err)
	}

	level := &entity.Level{}
	if data.PlayerLevel.GlobalPlayerLevelID != "" {
		// 檢查有無Level，沒有則建立
		level, err = u.findOrCreateLevel(ctx, span, data, merchant.GetID())
		if err != nil {
			u.tracing.RecordSpanError(span, err)
			return fmt.Errorf("find or create level: %w", err)
		}
	}

	var email *string
	if data.Player.Email != "" {
		email = &data.Player.Email
	}

	if data.Player.UpdatedAt.IsZero() {
		data.Player.UpdatedAt = time.Now()
	}

	// 使用 NewPlayerWithTimes 建構子建立 Player 實體
	player := entity.NewPlayerWithTimes(
		merchant.GetID(),
		data.Player.GlobalPlayerID,
		data.Player.Account,
		level.GetID(),
		email,
		data.Player.UpdatedAt,
		data.Player.UpdatedAt,
	)

	if err = player.IsValid(); err != nil {
		u.tracing.RecordSpanError(span, err)
		return fmt.Errorf("player is invalid: %w", err)
	}

	// 設置 LastActiveAt
	if !data.Player.LastActiveAt.IsZero() {
		player.SetLastActiveAt(&data.Player.LastActiveAt)
	}

	// 設置 DeletedAt
	if data.Player.DeletedAt != "" {
		nowTime := time.Now()
		player.SetDeletedAt(&nowTime)
	}

	// 設置 PlayerLevel
	player.SetPlayerLevel(entity.PlayerLevel{
		GlobalPlayerLevelID: data.PlayerLevel.GlobalPlayerLevelID,
		Name:                data.PlayerLevel.Name,
	})

	u.tracing.TraceEvent(span, "Submit player to batch processor")

	// 使用批次處理器進行異步處理
	resultChannel := u.batchProcessor.SubmitPlayer(player, data.GlobalMerchantID)

	// 等待批次處理結果
	u.tracing.TraceEvent(span, "Waiting for batch processing result")
	select {
	case err = <-resultChannel:
		if err != nil {
			u.tracing.RecordSpanError(span, err)
			return fmt.Errorf("batch process player: %w", err)
		}
	case <-ctx.Done():
		u.tracing.RecordSpanError(span, ctx.Err())
		return fmt.Errorf("context cancelled while waiting for batch processing: %w", ctx.Err())
	}

	// 記錄處理完成
	u.logger.InfoWithContext(ctx, "Player processed via batch successfully",
		u.logger.String("global_id", player.GetGlobalPlayerID()),
		u.logger.String("account", player.GetAccount()))
	u.tracing.TraceEvent(span, "Player sync completed successfully via batch processing")
	u.tracing.RecordSpanAttributes(span,
		attribute.String("player.global_id", player.GetGlobalPlayerID()),
		attribute.String("player.account", player.GetAccount()))

	return nil
}

func (u *PlayerUseCase) findOrCreateLevel(
	ctx context.Context,
	span trace.Span,
	data *event.PlayerSyncEvent,
	merchantID uint64,
) (*entity.Level, error) {
	level, err := u.findByGlobalID(ctx, span, data.PlayerLevel.GlobalPlayerLevelID)
	if err == nil {
		return level, nil
	}

	if !errors.Is(err, errmsg.ErrRepoLevelNotFound) {
		return nil, fmt.Errorf("find level: %w", err)
	}

	u.tracing.TraceEvent(span, "Upsert player")
	newLevel := entity.NewLevelWithTimes(
		merchantID,
		data.PlayerLevel.Name,
		data.PlayerLevel.GlobalPlayerLevelID,
		data.Player.UpdatedAt,
		data.Player.UpdatedAt,
	)

	if err = u.levelRepo.Upsert(ctx, newLevel); err != nil {
		u.tracing.RecordSpanError(span, err)
		return nil, fmt.Errorf("upsert level: %w", err)
	}

	// 取得新的level
	level, err = u.findByGlobalID(ctx, span, data.PlayerLevel.GlobalPlayerLevelID)
	if err != nil {
		return level, nil
	}
	return level, nil
}

func (u *PlayerUseCase) findByGlobalID(
	ctx context.Context,
	span trace.Span,
	globalPlayerLevelID string,
) (*entity.Level, error) {
	level, err := u.levelRepo.FindByGlobalID(ctx, globalPlayerLevelID)
	if err == nil {
		return level, nil
	}

	if !errors.Is(err, errmsg.ErrRepoLevelNotFound) {
		u.tracing.RecordSpanError(span, err)
		return nil, fmt.Errorf("find level: %w", err)
	}
	return nil, errmsg.ErrRepoLevelNotFound
}

// GetPlayerByID 通過ID獲取玩家
func (u *PlayerUseCase) GetPlayerByID(ctx context.Context, id uint64) (*entity.Player, error) {
	// 創建 span 並跟踪此操作
	ctx, span := u.tracing.StartSpan(ctx, "PlayerUseCase.GetPlayerByID")
	defer u.tracing.SpanEnd(span)

	u.tracing.RecordSpanAttributes(span, attribute.Int64("player.id", int64(id)))

	player, err := u.playerRepo.FindByID(ctx, id)
	if err != nil {
		u.tracing.RecordSpanError(span, err)
		return nil, fmt.Errorf("find player: %w", err)
	}

	// 添加玩家信息到 span
	u.tracing.RecordSpanAttributes(span,
		attribute.String("player.global_id", player.GetGlobalPlayerID()),
		attribute.String("player.account", player.GetAccount()),
	)

	return player, nil
}

// GetPlayerByGlobalID 通過全局ID獲取玩家（帶快取）
func (u *PlayerUseCase) GetPlayerByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Player, error) {
	// 創建 span 並跟踪此操作
	ctx, span := u.tracing.StartSpan(ctx, "PlayerUseCase.GetPlayerByGlobalID")
	defer u.tracing.SpanEnd(span)

	u.tracing.RecordSpanAttributes(span, attribute.String("player.global_id", globalID))

	// 獲取玩家
	cacheKey := fmt.Sprintf(consts.RedisPlayerGlobalIDKey, globalID)
	player, err := cache.QueryWithCache(
		ctx,
		u.cache,
		cacheKey,
		5*time.Minute,
		"player",
		func(ctx context.Context) (*entity.Player, error) {
			return u.playerRepo.FindByGlobalID(ctx, globalID)
		},
	)
	if err != nil {
		u.tracing.RecordSpanError(span, err)
		return nil, fmt.Errorf("find player: %w", err)
	}

	// 添加玩家信息到 span
	u.tracing.RecordSpanAttributes(span,
		attribute.String("player.account", player.GetAccount()),
	)

	return player, nil
}

// UpdatePlayerLastActive 更新玩家最後活躍時間
func (u *PlayerUseCase) UpdatePlayerLastActive(ctx context.Context, id uint64) error {
	// 創建 span 並跟踪此操作
	ctx, span := u.tracing.StartSpan(ctx, "PlayerUseCase.UpdatePlayerLastActive")
	defer u.tracing.SpanEnd(span)

	u.tracing.RecordSpanAttributes(span, attribute.Int64("player.id", int64(id)))

	// 查找玩家
	player, err := u.playerRepo.FindByID(ctx, id)
	if err != nil {
		u.tracing.RecordSpanError(span, err)
		return fmt.Errorf("find player: %w", err)
	}

	// 添加玩家信息到 span
	u.tracing.RecordSpanAttributes(span,
		attribute.String("player.global_id", player.GetGlobalPlayerID()),
		attribute.String("player.account", player.GetAccount()),
	)

	// 更新最後活躍時間
	player.UpdateLastActive()

	if err := u.playerRepo.Update(ctx, player); err != nil {
		u.tracing.RecordSpanError(span, err)
		return fmt.Errorf("update player: %w", err)
	}

	u.logger.InfoLog("Player last active time updated",
		u.logger.String("global_id", player.GetGlobalPlayerID()),
		u.logger.String("last_active_at", player.GetLastActiveAt().String()))

	return nil
}
