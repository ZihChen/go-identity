package serviceport

import (
	"context"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
)

// EventProducer 事件生產者接口
type EventProducer interface {
	// PublishMerchantSync 發布商戶同步事件
	PublishMerchantSync(ctx context.Context, event *event.CloudEvent) error

	// PublishPlayerSync 發布玩家同步事件
	PublishPlayerSync(ctx context.Context, event *event.CloudEvent) error

	// PublishManagerSync 發布管理員同步事件
	PublishManagerSync(ctx context.Context, event *event.CloudEvent) error

	// PublishPlayerLevelSync 發布玩家等級同步事件
	PublishPlayerLevelSync(ctx context.Context, event *event.CloudEvent) error

	// PublishPlayerTagsSync 發布玩家標籤同步事件
	PublishPlayerTagsSync(ctx context.Context, event *event.CloudEvent) error
}

// EventConsumer 事件Consumer接口
type EventConsumer interface {
	// ConsumeMerchantSync 消費商戶同步事件
	ConsumeMerchantSync(ctx context.Context) error

	// ConsumePlayerSync 消費玩家同步事件
	ConsumePlayerSync(ctx context.Context) error

	// ConsumeManagerSync 消費管理員同步事件
	ConsumeManagerSync(ctx context.Context) error
}

// QueueService 隊列服務接口
type QueueService interface {
	// EnqueueMerchantSync 將商戶同步任務加入隊列
	EnqueueMerchantSync(ctx context.Context, data []byte) error

	// EnqueuePlayerSync 將玩家同步任務加入隊列
	EnqueuePlayerSync(ctx context.Context, data []byte) error

	// EnqueueManagerSync 將管理員同步任務加入隊列
	EnqueueManagerSync(ctx context.Context, data []byte) error

	// EnqueueLevelSync 將會員等級同步任務加入佇列
	EnqueueLevelSync(ctx context.Context, data []byte) error

	// EnqueueTagSync 將會員標籤同步任務加入佇列
	EnqueueTagSync(ctx context.Context, data []byte) error
}
