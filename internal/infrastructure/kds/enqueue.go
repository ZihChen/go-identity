package kds

import (
	"context"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/errmsg"
)

func (k *KDSService) eventEnqueueProcess(ctx context.Context, eventType string, data []byte) error {
	switch eventType {
	case k.config.Events.MerchantSync:
		return k.queueService.EnqueueMerchantSync(ctx, data)
	case k.config.Events.PlayerSync:
		return k.queueService.EnqueuePlayerSync(ctx, data)
	case k.config.Events.ManagerSync:
		return k.queueService.EnqueueManagerSync(ctx, data)
	case k.config.Events.TagSync:
		return k.queueService.EnqueueTagSync(ctx, data)
	case k.config.Events.LevelSync:
		return k.queueService.EnqueueLevelSync(ctx, data)
	case k.config.Events.AgentSync:
		return k.queueService.EnqueueAgentSync(ctx, data)
	default:
		return errmsg.ErrUnknownEventType
	}
}
