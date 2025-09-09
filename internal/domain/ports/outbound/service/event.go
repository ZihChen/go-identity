package service

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

	// PublishTagSync 發布標籤同步事件
	PublishTagSync(ctx context.Context, event *event.CloudEvent) error
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