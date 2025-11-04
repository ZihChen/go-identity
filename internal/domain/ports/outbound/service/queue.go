package service

import (
	"context"
)

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

	// EnqueueAgentSync 將代理同步任務加入隊列
	EnqueueAgentSync(ctx context.Context, data []byte) error

	// Close 關閉隊列服務並釋放資源
	Close() error
}
