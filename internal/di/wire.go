//go:build wireinject

package di

import (
	"github.com/google/wire"
	"github.com/hibiken/asynq"
	"github.com/jvdiamondtech/ms-identity-cat/internal/adapter/inbound/handler/api"
	"github.com/jvdiamondtech/ms-identity-cat/internal/adapter/inbound/handler/consumer"
	"github.com/jvdiamondtech/ms-identity-cat/internal/adapter/inbound/handler/worker"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/inbound"
	agentRepo "github.com/jvdiamondtech/ms-identity-cat/internal/adapter/outbound/repository/agent"
	failedTaskEventRepo "github.com/jvdiamondtech/ms-identity-cat/internal/adapter/outbound/repository/failed_task_event"
	levelRepo "github.com/jvdiamondtech/ms-identity-cat/internal/adapter/outbound/repository/level"
	managerRepo "github.com/jvdiamondtech/ms-identity-cat/internal/adapter/outbound/repository/manager"
	merchantRepo "github.com/jvdiamondtech/ms-identity-cat/internal/adapter/outbound/repository/merchant"
	playerRepo "github.com/jvdiamondtech/ms-identity-cat/internal/adapter/outbound/repository/player"
	tagRepo "github.com/jvdiamondtech/ms-identity-cat/internal/adapter/outbound/repository/tag"
	agentUsecase "github.com/jvdiamondtech/ms-identity-cat/internal/application/usecase/agent"
	failedTaskEventUsecase "github.com/jvdiamondtech/ms-identity-cat/internal/application/usecase/failed_task_event"
	levelUsecase "github.com/jvdiamondtech/ms-identity-cat/internal/application/usecase/level"
	managerUsecase "github.com/jvdiamondtech/ms-identity-cat/internal/application/usecase/manager"
	merchantUsecase "github.com/jvdiamondtech/ms-identity-cat/internal/application/usecase/merchant"
	playerUsecase "github.com/jvdiamondtech/ms-identity-cat/internal/application/usecase/player"
	tagUsecase "github.com/jvdiamondtech/ms-identity-cat/internal/application/usecase/tag"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/service"
	redisCache "github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/cache/redis"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/kds"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/queue"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/tracing"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// WorkerComponents 包含 worker 所需的所有組件
type WorkerComponents struct {
	Handler *worker.WorkerHandler
	Server  *asynq.Server
}

var baseSet = wire.NewSet(
	// 基礎設施層
	queue.NewQueueService,
	provideRedisClient,
	provideTracingService,

	// 資料庫
	merchantRepo.NewMerchantRepository,
	playerRepo.NewPlayerRepository,
	managerRepo.NewManagerRepository,
	tagRepo.NewTagRepository,
	levelRepo.NewLevelRepository,
	agentRepo.NewAgentRepository,
	failedTaskEventRepo.NewFailedTaskEventRepository,
	playerRepo.NewPlayerTagRepository,

	// 服務
	provideEventProducer,

	// 用例層
	merchantUsecase.NewMerchantUseCase,
	playerUsecase.NewPlayerUseCase,
	managerUsecase.NewManagerUseCase,
	tagUsecase.NewTagUseCase,
	levelUsecase.NewLevelUseCase,
	agentUsecase.NewAgentUseCase,
	failedTaskEventUsecase.NewFailedTaskEventUseCase,
)

// 事件生產者提供者
func provideEventProducer(kdsService *kds.KDSService, logger infrastructure.Logger) service.EventProducer {
	return kdsService
}

// TracingService提供者
func provideTracingService(cfg *config.Config) (infrastructure.TracingService, error) {
	tracingService, err := tracing.NewTracingService(cfg)
	if err != nil {
		return nil, err
	}
	return tracingService, nil
}

// InitializeWebServer 初始化 Web 服務的 HTTP 處理器
func InitializeWebServer(cfg *config.Config, logger infrastructure.Logger, redisManager *redisCache.Manager, db *gorm.DB) (*api.HTTPHandler, error) {
	wire.Build(
		baseSet,
		kds.NewKDSService,
		api.NewHTTPHandler,
	)
	return nil, nil
}

// InitializeWorkerServer 初始化 Worker 服務的處理器
func InitializeWorkerServer(cfg *config.Config, logger infrastructure.Logger, redisManager *redisCache.Manager, db *gorm.DB) (*worker.WorkerHandler, error) {
	wire.Build(
		baseSet,
		kds.NewKDSService,
		worker.NewWorkerHandler,
	)
	return nil, nil
}

// InitializeWorkerComponents 初始化 Worker 服務的所有組件
func InitializeWorkerComponents(cfg *config.Config, logger infrastructure.Logger, redisManager *redisCache.Manager, db *gorm.DB) (*WorkerComponents, error) {
	wire.Build(
		wire.Struct(new(WorkerComponents), "*"),
		baseSet,
		kds.NewKDSService,
		worker.NewWorkerHandler,
		provideWorkerServer,
	)
	return nil, nil
}

// 提供 worker 服務器
func provideWorkerServer(cfg *config.Config, logger infrastructure.Logger, failedTaskUseCase inbound.FailedTaskEventUseCase, redisManager *redisCache.Manager, tracing infrastructure.TracingService) (*asynq.Server, error) {
	return queue.NewWorkerServer(cfg, logger, failedTaskUseCase, redisManager, tracing)
}

// InitializeConsumerHandler 初始化 Consumer 服務的 Handler
func InitializeConsumerHandler(cfg *config.Config, logger infrastructure.Logger, redisManager *redisCache.Manager) (*consumer.ConsumerHandler, error) {
	wire.Build(
		provideTracingService,
		queue.NewQueueService,
		kds.NewKDSService,
		consumer.NewConsumerHandler,
	)
	return nil, nil
}

// 提供 Redis 客戶端
func provideRedisClient(manager *redisCache.Manager) (*redis.Client, error) {
	redisInstance, err := manager.GetClient()
	if err != nil {
		return nil, err
	}
	return redisInstance, nil
}
