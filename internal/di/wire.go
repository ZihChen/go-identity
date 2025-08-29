//go:build wireinject
// +build wireinject

package di

import (
	"github.com/google/wire"
	"github.com/hibiken/asynq"
	"github.com/jvdiamondtech/ms-identity-cat/internal/adapter/handler/api"
	"github.com/jvdiamondtech/ms-identity-cat/internal/adapter/handler/worker"
	"github.com/jvdiamondtech/ms-identity-cat/internal/adapter/repository"
	adapterUsecase "github.com/jvdiamondtech/ms-identity-cat/internal/adapter/usecase"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/infraport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/serviceport"
	redisCache "github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/cache/redis"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/kds"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/queue"
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

	// 資料庫
	repository.NewMerchantRepository,
	repository.NewPlayerRepository,
	repository.NewManagerRepository,
	repository.NewTagRepository,
	repository.NewLevelRepository,
	repository.NewPlayerTagRepository,

	// 服務
	provideEventProducer,

	// 用例層
	adapterUsecase.NewMerchantUseCase,
	adapterUsecase.NewPlayerUseCase,
	adapterUsecase.NewManagerUseCase,
	adapterUsecase.NewTagUseCase,
	adapterUsecase.NewLevelUseCase,
)

// 事件生產者提供者
func provideEventProducer(kdsService *kds.KDSService, logger infraport.Logger) serviceport.EventProducer {
	return kdsService
}

// InitializeWebServer 初始化 Web 服務的 HTTP 處理器
func InitializeWebServer(cfg *config.Config, logger infraport.Logger, redisManager *redisCache.Manager, db *gorm.DB) (*api.HTTPHandler, error) {
	wire.Build(
		baseSet,
		kds.NewKDSService,
		api.NewHTTPHandler,
	)
	return nil, nil
}

// InitializeWorkerServer 初始化 Worker 服務的處理器
func InitializeWorkerServer(cfg *config.Config, logger infraport.Logger, redisManager *redisCache.Manager, db *gorm.DB) (*worker.WorkerHandler, error) {
	wire.Build(
		baseSet,
		kds.NewKDSService,
		worker.NewWorkerHandler,
	)
	return nil, nil
}

// InitializeWorkerComponents 初始化 Worker 服務的所有組件
func InitializeWorkerComponents(cfg *config.Config, logger infraport.Logger, redisManager *redisCache.Manager, db *gorm.DB) (*WorkerComponents, error) {
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
func provideWorkerServer(cfg *config.Config, logger infraport.Logger) (*asynq.Server, error) {
	return queue.NewWorkerServer(cfg, logger)
}

// InitializeConsumer 初始化 Consumer 服務的 KDS 服務
func InitializeConsumer(cfg *config.Config, logger infraport.Logger, redisManager *redisCache.Manager) (*kds.KDSService, error) {
	wire.Build(
		queue.NewQueueService,
		kds.NewKDSService,
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
