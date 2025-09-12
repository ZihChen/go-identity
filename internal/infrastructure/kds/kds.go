package kds

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/service"
	redisCache "github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/cache/redis"
	cfg "github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
)

// KDSService KDS服務實現
type KDSService struct {
	client        *kinesis.Client
	dynamoClient  *dynamodb.Client
	redisManager  *redisCache.Manager
	streamName    string
	consumeStream string
	produceStream string
	tableName     string
	partitionKey  string
	sortKey       string
	config        *cfg.Config
	queueService  service.QueueService
	logger        infrastructure.Logger

	// 新增：錯誤分類器用於批次處理
	errorClassifier *ErrorClassifier
}

// NewKDSService 創建KDS服務
func NewKDSService(
	config *cfg.Config,
	queueService service.QueueService,
	redisManager *redisCache.Manager,
	logger infrastructure.Logger,
) (*KDSService, error) {
	// 創建AWS配置
	awsConfig, err := config.LoadAWSConfig(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// 創建Kinesis客戶端
	kinesisClient := kinesis.NewFromConfig(awsConfig)

	// 創建DynamoDB客戶端
	dynamoClient := dynamodb.NewFromConfig(awsConfig)

	// 從ARN中提取stream名稱
	streamARN := config.AWS.KinesisStream
	streamName := streamARN
	if len(streamARN) > 0 {
		// 處理可能的ARN格式
		for i := len(streamARN) - 1; i >= 0; i-- {
			if streamARN[i] == '/' || streamARN[i] == ':' {
				streamName = streamARN[i+1:]
				break
			}
		}
	}

	logger.InfoLog("Initialized KDS service",
		logger.String("stream_name", streamName),
		logger.String("stream_arn", streamARN),
		logger.String("dynamodb_table", config.AWS.DynamoDBTable))

	// 創建錯誤分類器
	errorClassifier := NewErrorClassifier()

	return &KDSService{
		client:          kinesisClient,
		dynamoClient:    dynamoClient,
		redisManager:    redisManager,
		streamName:      streamName,
		consumeStream:   config.AWS.ConsumeStream,
		produceStream:   config.AWS.ProduceStream,
		tableName:       config.AWS.DynamoDBTable,
		partitionKey:    config.AWS.PartitionKey,
		sortKey:         config.AWS.SortKey,
		config:          config,
		queueService:    queueService,
		logger:          logger,
		errorClassifier: errorClassifier,
	}, nil
}

// Close 釋放 KDSService 持有的所有資源
func (k *KDSService) Close() error {
	if err := k.queueService.Close(); err != nil {
		k.logger.ErrorLog("Failed to close queue service",
			k.logger.Error("error", err))
		return fmt.Errorf("failed to close queue service: %w", err)
	}
	k.logger.InfoLog("KDS service and queue service closed successfully")
	return nil
}

// ConsumeAllEventsEnhanced 使用簡化版批次處理消費所有事件
// 這是一個簡化版本，直接使用現有的 ConsumeAllEvents 方法
// 該方法現在已經包含了批次處理和 worker pool 優化
func (k *KDSService) ConsumeAllEventsEnhanced(ctx context.Context) error {
	k.logger.InfoWithContext(
		ctx,
		"Starting enhanced event consumption with batch processing",
		k.logger.String("stream", k.consumeStream),
		k.logger.Int("batch_size", k.config.Consumer.BatchSize),
		k.logger.Int("worker_pool_size", k.config.Consumer.WorkerPoolSize),
	)

	// 直接調用已經優化的 ConsumeAllEvents 方法
	return k.ConsumeAllEvents(ctx)
}
