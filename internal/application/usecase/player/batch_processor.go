package usecase

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/consts"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/service"
)

// PlayerBatchRequest 批次處理請求
type PlayerBatchRequest struct {
	Player            *entity.Player
	GlobalMerchantID  string
	CompletionChannel chan<- error
}

// PlayerBatchProcessor 玩家批次處理器
type PlayerBatchProcessor struct {
	playerRepo    repository.PlayerRepository
	eventProducer service.EventProducer
	logger        infrastructure.Logger
	tracing       infrastructure.TracingService
	cache         infrastructure.CacheManager

	// 批次處理配置
	batchSize    int           // 批次大小
	batchTimeout time.Duration // 批次超時時間
	bufferSize   int           // Channel 緩衝大小
	blockOnFull  bool          // Channel 滿時是否阻塞等待

	// 內部狀態
	requestChannel chan *PlayerBatchRequest
	stopChannel    chan struct{}
	wg             sync.WaitGroup
	started        bool
	mu             sync.Mutex
}

// NewPlayerBatchProcessor 創建玩家批次處理器
func NewPlayerBatchProcessor(
	playerRepo repository.PlayerRepository,
	eventProducer service.EventProducer,
	logger infrastructure.Logger,
	tracing infrastructure.TracingService,
	cache infrastructure.CacheManager,
) *PlayerBatchProcessor {
	bufferSize := 1000 // Channel 緩衝大小

	return &PlayerBatchProcessor{
		playerRepo:    playerRepo,
		eventProducer: eventProducer,
		logger:        logger,
		tracing:       tracing,
		cache:         cache,

		// 批次配置：300筆或2秒超時
		batchSize:    300,
		batchTimeout: 2 * time.Second,
		bufferSize:   bufferSize,
		blockOnFull:  false, // 預設不阻塞，使用降級處理

		requestChannel: make(chan *PlayerBatchRequest, bufferSize),
		stopChannel:    make(chan struct{}),
	}
}

// Start 啟動批次處理器
func (p *PlayerBatchProcessor) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		return nil
	}

	p.started = true
	p.wg.Add(1)

	go p.processBatches(ctx)

	p.logger.InfoLog("Player batch processor started",
		p.logger.Int("batch_size", p.batchSize),
		p.logger.String("batch_timeout", p.batchTimeout.String()))

	return nil
}

// Stop 停止批次處理器
func (p *PlayerBatchProcessor) Stop(ctx context.Context) error {
	p.mu.Lock()
	if !p.started {
		p.mu.Unlock()
		return nil
	}
	p.mu.Unlock()

	close(p.stopChannel)
	p.wg.Wait()

	p.mu.Lock()
	p.started = false
	p.mu.Unlock()

	p.logger.InfoLog("Player batch processor stopped")
	return nil
}

// SetBlockOnFull 設置當Channel滿時的行為
func (p *PlayerBatchProcessor) SetBlockOnFull(block bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.blockOnFull = block

	if block {
		p.logger.InfoLog("Batch processor configured to block when channel is full")
	} else {
		p.logger.InfoLog(
			"Batch processor configured to use fallback processing when channel is full",
		)
	}
}

// SetBatchConfig 設置批次處理配置
func (p *PlayerBatchProcessor) SetBatchConfig(batchSize int, batchTimeout time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.started {
		p.logger.WarnLog("Cannot change batch config while processor is running")
		return
	}

	p.batchSize = batchSize
	p.batchTimeout = batchTimeout

	p.logger.InfoLog("Batch processor config updated",
		p.logger.Int("batch_size", p.batchSize),
		p.logger.String("batch_timeout", p.batchTimeout.String()))
}

// GetStats 獲取批次處理器統計信息
func (p *PlayerBatchProcessor) GetStats() map[string]interface{} {
	p.mu.Lock()
	defer p.mu.Unlock()

	return map[string]interface{}{
		"batch_size":       p.batchSize,
		"batch_timeout":    p.batchTimeout,
		"buffer_size":      p.bufferSize,
		"block_on_full":    p.blockOnFull,
		"started":          p.started,
		"channel_length":   len(p.requestChannel),
		"channel_capacity": cap(p.requestChannel),
	}
}

// SubmitPlayer 提交玩家進行批次處理
func (p *PlayerBatchProcessor) SubmitPlayer(
	player *entity.Player,
	globalMerchantID string,
) <-chan error {
	completionChannel := make(chan error, 1)

	request := &PlayerBatchRequest{
		Player:            player,
		GlobalMerchantID:  globalMerchantID,
		CompletionChannel: completionChannel,
	}

	if p.blockOnFull {
		// 阻塞模式：等待 Channel 有空位，確保所有資料都進入批次處理
		select {
		case p.requestChannel <- request:
			// 請求已成功提交到批次處理器
		case <-time.After(30 * time.Second): // 防止無限阻塞，30秒超時
			p.logger.ErrorLog("Batch processor submission timeout after 30 seconds",
				p.logger.String("global_player_id", player.GetGlobalPlayerID()))
			completionChannel <- fmt.Errorf("batch processor submission timeout")
			close(completionChannel)
		}
	} else {
		// 非阻塞模式：Channel 滿時使用降級處理
		select {
		case p.requestChannel <- request:
			// 請求已成功提交到批次處理器
		default:
			// Channel 已滿，使用降級處理：直接同步處理
			p.logger.WarnLog("Batch processor channel full, falling back to synchronous processing",
				p.logger.String("global_player_id", player.GetGlobalPlayerID()))

			go p.handleSyncFallback(request)
		}
	}

	return completionChannel
}

// handleSyncFallback 當批次處理器滿載時的降級同步處理
func (p *PlayerBatchProcessor) handleSyncFallback(request *PlayerBatchRequest) {
	defer close(request.CompletionChannel)

	ctx := context.Background()

	// 降級為單筆處理，確保資料不遺失
	err := p.playerRepo.Upsert(ctx, request.Player)
	if err != nil {
		p.logger.ErrorLog("Fallback sync upsert failed",
			p.logger.String("global_player_id", request.Player.GetGlobalPlayerID()),
			p.logger.Error("error", err))
		request.CompletionChannel <- err
		return
	}

	// Upsert 成功後，使快取失效
	cacheKey := fmt.Sprintf(consts.RedisPlayerGlobalIDKey, request.Player.GetGlobalPlayerID())
	if err = p.cache.Del(ctx, cacheKey); err != nil {
		p.logger.WarnWithContext(ctx, "Cache invalidation failed",
			p.logger.String("cache_key", cacheKey),
			p.logger.Error("error", err))
	}

	// 單筆事件發送
	err = p.eventProducer.PublishPlayerSync(ctx, request.Player, request.GlobalMerchantID)
	if err != nil {
		p.logger.ErrorLog("Fallback sync publish failed",
			p.logger.String("global_player_id", request.Player.GetGlobalPlayerID()),
			p.logger.Error("error", err))
		request.CompletionChannel <- err
		return
	}

	p.logger.InfoLog("Fallback sync processing completed",
		p.logger.String("global_player_id", request.Player.GetGlobalPlayerID()))

	request.CompletionChannel <- nil
}

// processBatches 批次處理主循環
func (p *PlayerBatchProcessor) processBatches(ctx context.Context) {
	defer p.wg.Done()

	batch := make([]*PlayerBatchRequest, 0, p.batchSize)
	timer := time.NewTimer(p.batchTimeout)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			// 上下文取消，處理剩餘批次
			if len(batch) > 0 {
				p.processBatch(ctx, batch)
			}
			return

		case <-p.stopChannel:
			// 停止信號，處理剩餘批次
			if len(batch) > 0 {
				p.processBatch(ctx, batch)
			}
			return

		case request := <-p.requestChannel:
			batch = append(batch, request)

			// 檢查是否達到批次大小
			if len(batch) >= p.batchSize {
				p.processBatch(ctx, batch)
				batch = batch[:0] // 重置批次
				timer.Reset(p.batchTimeout)
			}

		case <-timer.C:
			// 超時，處理當前批次
			if len(batch) > 0 {
				p.processBatch(ctx, batch)
				batch = batch[:0] // 重置批次
			}
			timer.Reset(p.batchTimeout)
		}
	}
}

// processBatch 處理單個批次
func (p *PlayerBatchProcessor) processBatch(ctx context.Context, batch []*PlayerBatchRequest) {
	ctx, span := p.tracing.StartSpan(ctx, "PlayerBatchProcessor.processBatch")
	defer p.tracing.SpanEnd(span)

	batchSize := len(batch)
	p.logger.InfoLog("Processing player batch",
		p.logger.Int("batch_size", batchSize))

	// 第一步：批次寫入資料庫
	dbErrors := p.batchUpsertPlayers(ctx, batch)

	// 第二步：批次發送事件（只為成功的玩家）
	eventErrors := p.batchPublishEvents(ctx, batch, dbErrors)

	// 第三步：回應每個請求的結果
	for i, request := range batch {
		var err error
		if dbErrors[i] != nil {
			err = dbErrors[i]
		} else if eventErrors[i] != nil {
			err = eventErrors[i]
		}

		request.CompletionChannel <- err
		close(request.CompletionChannel)
	}

	p.logger.InfoLog("Player batch processed",
		p.logger.Int("batch_size", batchSize),
		p.logger.Int("successful_count", p.countSuccessful(dbErrors, eventErrors)))
}

// batchUpsertPlayers 批次寫入玩家資料
func (p *PlayerBatchProcessor) batchUpsertPlayers(
	ctx context.Context,
	batch []*PlayerBatchRequest,
) []error {
	ctx, span := p.tracing.StartSpan(ctx, "PlayerBatchProcessor.batchUpsertPlayers")
	defer p.tracing.SpanEnd(span)

	// 提取所有玩家實體
	players := make([]*entity.Player, len(batch))
	for i, request := range batch {
		players[i] = request.Player
	}

	// 使用真正的批次SQL操作
	err := p.playerRepo.BatchUpsert(ctx, players)

	// 準備錯誤返回
	errors := make([]error, len(batch))

	if err != nil {
		// 如果批次操作失敗，所有請求都標記為失敗
		p.logger.ErrorLog("Batch upsert failed for entire batch",
			p.logger.Int("batch_size", len(batch)),
			p.logger.Error("error", err))

		for i := range errors {
			errors[i] = err
		}
	} else {
		// 批次操作成功，所有請求都成功
		p.logger.InfoLog("Batch upsert succeeded",
			p.logger.Int("batch_size", len(batch)))

		// 批次操作成功後，使用 Pipeline 批次失效相關快取
		if err = p.batchInvalidateCache(ctx, players); err != nil {
			p.logger.WarnWithContext(ctx, "Batch cache invalidation failed",
				p.logger.Int("player_count", len(players)),
				p.logger.Error("error", err))
		}
	}

	return errors
}

// batchPublishEvents 批次發送事件
func (p *PlayerBatchProcessor) batchPublishEvents(
	ctx context.Context,
	batch []*PlayerBatchRequest,
	dbErrors []error,
) []error {
	ctx, span := p.tracing.StartSpan(ctx, "PlayerBatchProcessor.batchPublishEvents")
	defer p.tracing.SpanEnd(span)

	errors := make([]error, len(batch))

	// 收集成功的玩家和對應的GlobalMerchantID
	var successfulPlayers []*entity.Player
	var successfulMerchantIDs []string
	var successfulIndices []int

	for i, request := range batch {
		if dbErrors[i] == nil {
			successfulPlayers = append(successfulPlayers, request.Player)
			successfulMerchantIDs = append(successfulMerchantIDs, request.GlobalMerchantID)
			successfulIndices = append(successfulIndices, i)
		}
	}

	// 如果有成功的玩家，進行批次事件發送
	if len(successfulPlayers) > 0 {
		p.logger.InfoLog("Batch publishing player sync events",
			p.logger.Int("successful_players_count", len(successfulPlayers)))

		batchErr := p.eventProducer.BatchPublishPlayerSync(
			ctx,
			successfulPlayers,
			successfulMerchantIDs,
		)

		if batchErr != nil {
			p.logger.ErrorLog("Batch publish player sync events failed",
				p.logger.Int("failed_count", len(successfulPlayers)),
				p.logger.Error("error", batchErr))

			// 如果批次發送失敗，對所有成功的玩家標記事件發送錯誤
			for _, idx := range successfulIndices {
				errors[idx] = batchErr
			}
		} else {
			p.logger.InfoLog("Batch publish player sync events succeeded",
				p.logger.Int("successful_count", len(successfulPlayers)))
		}
	}

	return errors
}

// countSuccessful 計算成功處理的數量
func (p *PlayerBatchProcessor) countSuccessful(dbErrors, eventErrors []error) int {
	count := 0
	for i := range dbErrors {
		if dbErrors[i] == nil && eventErrors[i] == nil {
			count++
		}
	}
	return count
}

// batchInvalidateCache 使用 Pipeline 批次失效玩家快取
func (p *PlayerBatchProcessor) batchInvalidateCache(
	ctx context.Context,
	players []*entity.Player,
) error {
	if len(players) == 0 {
		return nil
	}

	// 獲取 Redis Pipeline
	pipeline, err := p.cache.Pipeline()
	if err != nil || pipeline == nil {
		// Pipeline 不可用時，回退到逐個刪除
		return p.fallbackInvalidateCache(ctx, players)
	}

	// 批次添加刪除命令到 Pipeline
	cacheKeys := make([]string, 0, len(players))
	for _, player := range players {
		cacheKey := fmt.Sprintf(consts.RedisPlayerGlobalIDKey, player.GetGlobalPlayerID())
		cacheKeys = append(cacheKeys, cacheKey)
		pipeline.Del(ctx, cacheKey)
	}

	// 執行 Pipeline
	_, err = pipeline.Exec(ctx)
	if err != nil {
		// Pipeline 執行失敗時，回退到逐個刪除
		p.logger.WarnWithContext(
			ctx,
			"Pipeline execution failed, falling back to individual deletions",
			p.logger.Error("error", err),
		)
		return p.fallbackInvalidateCache(ctx, players)
	}

	p.logger.InfoWithContext(ctx, "Batch cache invalidation completed using pipeline",
		p.logger.Int("cache_keys_deleted", len(cacheKeys)))

	return nil
}

// fallbackInvalidateCache 回退機制：逐個失效快取
func (p *PlayerBatchProcessor) fallbackInvalidateCache(
	ctx context.Context,
	players []*entity.Player,
) error {
	successCount := 0
	for _, player := range players {
		cacheKey := fmt.Sprintf(consts.RedisPlayerGlobalIDKey, player.GetGlobalPlayerID())
		if err := p.cache.Del(ctx, cacheKey); err != nil {
			p.logger.WarnWithContext(ctx, "Individual cache invalidation failed",
				p.logger.String("cache_key", cacheKey),
				p.logger.Error("error", err))
		} else {
			successCount++
		}
	}

	p.logger.InfoWithContext(ctx, "Fallback cache invalidation completed",
		p.logger.Int("total_keys", len(players)),
		p.logger.Int("successful_deletions", successCount))

	return nil
}
