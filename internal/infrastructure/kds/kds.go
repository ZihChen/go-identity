package kds

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/kinesis"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/infraport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/serviceport"
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
	queueService  serviceport.QueueService
	logger        infraport.Logger
}

// NewKDSService 創建KDS服務
func NewKDSService(
	config *cfg.Config,
	queueService serviceport.QueueService,
	redisManager *redisCache.Manager,
	logger infraport.Logger,
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

	return &KDSService{
		client:        kinesisClient,
		dynamoClient:  dynamoClient,
		redisManager:  redisManager,
		streamName:    streamName,
		consumeStream: config.AWS.ConsumeStream,
		produceStream: config.AWS.ProduceStream,
		tableName:     config.AWS.DynamoDBTable,
		partitionKey:  config.AWS.PartitionKey,
		sortKey:       config.AWS.SortKey,
		config:        config,
		queueService:  queueService,
		logger:        logger,
	}, nil
}
