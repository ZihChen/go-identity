package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/repository"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/service"
	"go.opentelemetry.io/otel/attribute"
)

// AgentUseCase 代理用例
type AgentUseCase struct {
	agentRepo     repository.AgentRepository
	merchantRepo  repository.MerchantRepository
	eventProducer service.EventProducer
	logger        infrastructure.Logger
	tracing       infrastructure.TracingService
}

// NewAgentUseCase 創建代理用例
func NewAgentUseCase(
	agentRepo repository.AgentRepository,
	merchantRepo repository.MerchantRepository,
	eventProducer service.EventProducer,
	logger infrastructure.Logger,
	tracing infrastructure.TracingService,
) inbound.AgentUseCase {
	return &AgentUseCase{
		agentRepo:     agentRepo,
		merchantRepo:  merchantRepo,
		eventProducer: eventProducer,
		logger:        logger,
		tracing:       tracing,
	}
}

// SyncAgentData 同步代理數據
func (u *AgentUseCase) SyncAgentData(ctx context.Context, event *event.AgentSyncEvent) error {
	ctx, span := u.tracing.StartSpan(ctx, "AgentUseCase.SyncAgentData")
	defer u.tracing.SpanEnd(span)

	// 記錄事件開始處理
	u.tracing.TraceEvent(span, "Starting agent sync processing")
	u.tracing.RecordSpanAttributes(span,
		attribute.String("agent.global_id", event.Agent.GlobalAgentID),
		attribute.String("agent.account", event.Agent.Account),
		attribute.String("merchant.global_id", event.Merchant.GlobalMerchantID))

	// 透過 GlobalMerchantID 取得 merchant 的資料庫 ID
	u.tracing.TraceEvent(span, "Checking if merchant exists")
	merchant, err := u.merchantRepo.FindByGlobalID(ctx, event.Merchant.GlobalMerchantID)
	if err != nil && !errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
		u.tracing.RecordSpanError(span, err)
		u.logger.ErrorWithContext(ctx, "Failed to find merchant",
			u.logger.Error("err", err),
			u.logger.String("global_merchant_id", event.Merchant.GlobalMerchantID))
		return fmt.Errorf("find merchant: %w", err)
	}

	if event.EventTime.IsZero() {
		event.EventTime = time.Now()
	}

	// 構建Agent實體，使用從資料庫查詢到的 merchant ID
	agent := entity.NewAgentWithTimes(
		merchant.GetID(),
		event.Agent.GlobalAgentID,
		event.Agent.Account,
		event.Agent.Ancestry,
		event.Agent.CurrentSignInAt,
		event.EventTime,
		event.EventTime,
	)

	// 驗證Agent實體
	if err = agent.IsValid(); err != nil {
		u.tracing.RecordSpanError(span, err)
		u.logger.ErrorWithContext(ctx, "Invalid agent entity",
			u.logger.Error("err", err),
			u.logger.String("global_agent_id", event.Agent.GlobalAgentID))
		return fmt.Errorf("invalid agent entity: %w", err)
	}

	// 執行 Upsert 操作 (時間戳判斷的冪等性更新)
	u.tracing.TraceEvent(span, "Upsert agent")
	if err = u.agentRepo.Upsert(ctx, agent); err != nil {
		u.tracing.RecordSpanError(span, err)
		u.logger.ErrorWithContext(ctx, "Failed to upsert agent",
			u.logger.Error("err", err),
			u.logger.String("global_agent_id", event.Agent.GlobalAgentID))
		return fmt.Errorf("upsert agent: %w", err)
	}

	u.logger.InfoWithContext(ctx, "Agent upserted successfully",
		u.logger.String("global_agent_id", agent.GetGlobalAgentID()),
		u.logger.String("account", agent.GetAccount()),
		u.logger.UInt64("merchant_id", agent.GetMerchantID()),
		u.logger.String("global_merchant_id", event.Merchant.GlobalMerchantID))

	// 發布代理同步事件到 KDS
	if err = u.eventProducer.PublishAgentSync(ctx, agent, merchant.GetGlobalMerchantID()); err != nil {
		u.tracing.RecordSpanError(span, err)
		u.logger.ErrorWithContext(ctx, "Failed to publish agent sync event",
			u.logger.Error("err", err),
			u.logger.String("global_agent_id", agent.GetGlobalAgentID()))
		return fmt.Errorf("publish agent sync event: %w", err)
	}

	u.tracing.TraceEvent(span, "Agent sync completed successfully")
	return nil
}

// GetAgentByID 通過ID獲取代理
func (u *AgentUseCase) GetAgentByID(ctx context.Context, id uint64) (*entity.Agent, error) {
	ctx, span := u.tracing.StartSpan(ctx, "AgentUseCase.GetAgentByID")
	defer u.tracing.SpanEnd(span)

	u.tracing.RecordSpanAttributes(span, attribute.Int64("agent.id", int64(id)))

	agent, err := u.agentRepo.FindByID(ctx, id)
	if err != nil {
		u.tracing.RecordSpanError(span, err)
		return nil, fmt.Errorf("find agent: %w", err)
	}

	u.tracing.RecordSpanAttributes(span,
		attribute.String("agent.global_id", agent.GetGlobalAgentID()),
		attribute.String("agent.account", agent.GetAccount()),
	)

	return agent, nil
}

// GetAgentByGlobalID 通過全局ID獲取代理
func (u *AgentUseCase) GetAgentByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Agent, error) {
	ctx, span := u.tracing.StartSpan(ctx, "AgentUseCase.GetAgentByGlobalID")
	defer u.tracing.SpanEnd(span)

	u.tracing.RecordSpanAttributes(span, attribute.String("agent.global_id", globalID))

	agent, err := u.agentRepo.FindByGlobalID(ctx, globalID)
	if err != nil {
		u.tracing.RecordSpanError(span, err)
		return nil, fmt.Errorf("find agent: %w", err)
	}

	u.tracing.RecordSpanAttributes(span,
		attribute.String("agent.account", agent.GetAccount()),
		attribute.Int64("agent.merchant_id", int64(agent.GetMerchantID())),
	)

	return agent, nil
}

// GetAgentsByMerchantID 通過商戶ID獲取代理列表
func (u *AgentUseCase) GetAgentsByMerchantID(
	ctx context.Context,
	merchantID uint64,
) ([]*entity.Agent, error) {
	ctx, span := u.tracing.StartSpan(ctx, "AgentUseCase.GetAgentsByMerchantID")
	defer u.tracing.SpanEnd(span)

	u.tracing.RecordSpanAttributes(span, attribute.Int64("merchant.id", int64(merchantID)))

	agents, err := u.agentRepo.FindByMerchantID(ctx, merchantID)
	if err != nil {
		u.tracing.RecordSpanError(span, err)
		return nil, fmt.Errorf("find agents by merchant: %w", err)
	}

	u.tracing.RecordSpanAttributes(span, attribute.Int("agents.count", len(agents)))

	return agents, nil
}
