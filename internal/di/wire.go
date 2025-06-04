//go:build wireinject
// +build wireinject

package di

import (
	"github.com/google/wire"
	"github.com/hibiken/asynq"
	"github.com/jvdiamondtech/ms-identity-cat/internal/adapter/handler"
	"github.com/jvdiamondtech/ms-identity-cat/internal/adapter/repository"
	adapterUsecase "github.com/jvdiamondtech/ms-identity-cat/internal/adapter/usecase"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/infraport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/serviceport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/database"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/deduplication"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/kds"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/queue"
	redisInfra "github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/redis"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// WorkerComponents 包含 worker 所需的所有組件
type WorkerComponents struct {
	Handler *handler.WorkerHandler
	Server  *asynq.Server
}

var baseSet = wire.NewSet(
	// 基礎設施層
	provideDatabaseConnection,
	queue.NewQueueService,
	provideRedisClient,
	provideDeduplicationService,

	// 資料庫
	repository.NewMerchantRepository,
	repository.NewPlayerRepository,
	repository.NewManagerRepository,

	// 服務
	provideEventProducer,

	// 用例層
	adapterUsecase.NewMerchantUseCase,
	adapterUsecase.NewPlayerUseCase,
	adapterUsecase.NewManagerUseCase,
)

// 資料庫連接提供者
func provideDatabaseConnection(cfg *config.Config) (*gorm.DB, error) {
	db, err := database.NewDatabase(cfg)
	if err != nil {
		return nil, err
	}
	return db.DB, nil
}

// 事件生產者提供者
func provideEventProducer(kdsService *kds.KDSService, logger *zap.Logger) serviceport.EventProducer {
	return kdsService
}

// InitializeWebServer 初始化 Web 服務的 HTTP 處理器
func InitializeWebServer(cfg *config.Config, logger *zap.Logger, serviceLog infraport.Logger) (*handler.HTTPHandler, error) {
	wire.Build(
		baseSet,
		kds.NewKDSService,
		handler.NewHTTPHandler,
	)
	return nil, nil
}

// InitializeWorkerServer 初始化 Worker 服務的處理器
func InitializeWorkerServer(cfg *config.Config, logger *zap.Logger) (*handler.WorkerHandler, error) {
	wire.Build(
		baseSet,
		kds.NewKDSService,
		handler.NewWorkerHandler,
	)
	return nil, nil
}

// InitializeWorkerComponents 初始化 Worker 服務的所有組件
func InitializeWorkerComponents(cfg *config.Config, logger *zap.Logger) (*WorkerComponents, error) {
	wire.Build(
		wire.Struct(new(WorkerComponents), "*"),
		baseSet,
		kds.NewKDSService,
		handler.NewWorkerHandler,
		provideWorkerServer,
	)
	return nil, nil
}

// 提供 worker 服務器
func provideWorkerServer(cfg *config.Config, logger *zap.Logger) (*asynq.Server, error) {
	return queue.NewWorkerServer(cfg, logger)
}

// InitializeConsumer 初始化 Consumer 服務的 KDS 服務
func InitializeConsumer(cfg *config.Config, logger *zap.Logger) (*kds.KDSService, error) {
	wire.Build(
		queue.NewQueueService,
		provideRedisClient,
		kds.NewKDSService,
	)
	return nil, nil
}

// 提供 Redis 客戶端
func provideRedisClient(cfg *config.Config) (*redis.Client, error) {
	return redisInfra.NewRedis(cfg)
}

// 提供事件去重服務
func provideDeduplicationService(redisClient *redis.Client, logger *zap.Logger) serviceport.EventDeduplicationService {
	return deduplication.NewRedisDeduplicationService(redisClient, logger)
}
