package service

import "context"

// QueueService 隊列服務接口
type QueueService interface {
	EnqueueMerchantSync(ctx context.Context, data []byte) error
	EnqueuePlayerSync(ctx context.Context, data []byte) error
	EnqueueManagerSync(ctx context.Context, data []byte) error
	EnqueueLevelSync(ctx context.Context, data []byte) error
	EnqueueTagSync(ctx context.Context, data []byte) error
	EnqueueAgentSync(ctx context.Context, data []byte) error
	Close() error
}
