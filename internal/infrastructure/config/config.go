package config

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/spf13/viper"
)

// Config 應用程序配置
type Config struct {
	App      AppConfig
	Database DatabaseConfig
	Redis    RedisConfig
	AWS      AWSConfig
	Tracing  TracingConfig
	Logs     LogsConfig
	Events   EventsConfig
}

// AppConfig 應用程序基本配置
type AppConfig struct {
	Name  string
	Env   string
	Port  int
	Debug bool
}

// DatabaseConfig 資料庫配置
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	Options  string
	MaxIdle  int
	MaxOpen  int
	Timeout  time.Duration
}

// RedisConfig Redis配置
type RedisConfig struct {
	Domain   string
	Port     int
	Password string
	DB       int
}

// AWSConfig AWS配置
type AWSConfig struct {
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	KinesisStream   string
	DynamoDBTable   string
	PartitionKey    string
	SortKey         string
}

// TracingConfig 追踪配置
type TracingConfig struct {
	Endpoint   string
	APIKey     string
	StreamName string
}

type LogsConfig struct {
	Endpoint string
	Username string
	Password string
}

// EventsConfig 事件配置
type EventsConfig struct {
	MerchantSync         string
	PlayerSync           string
	ManagerSync          string
	IdentityMerchantSync string
	IdentityPlayerSync   string
	IdentityManagerSync  string
}

// LoadConfig 加載配置
func LoadConfig() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
	}

	config := &Config{
		App: AppConfig{
			Name:  viper.GetString("APP_NAME"),
			Env:   viper.GetString("APP_ENV"),
			Port:  viper.GetInt("APP_PORT"),
			Debug: viper.GetBool("APP_DEBUG"),
		},
		Database: DatabaseConfig{
			Host:     viper.GetString("DB_HOST"),
			Port:     viper.GetInt("DB_PORT"),
			User:     viper.GetString("DB_USER"),
			Password: viper.GetString("DB_PASSWORD"),
			DBName:   viper.GetString("DB_NAME"),
			Options:  viper.GetString("DB_OPTIONS"),
			MaxIdle:  viper.GetInt("DB_MAX_IDLE"),
			MaxOpen:  viper.GetInt("DB_MAX_OPEN"),
			Timeout:  viper.GetDuration("DB_TIMEOUT"),
		},
		Redis: RedisConfig{
			Domain:   viper.GetString("REDIS_DOMAIN"),
			Port:     viper.GetInt("REDIS_PORT"),
			Password: viper.GetString("REDIS_PWD"),
			DB:       viper.GetInt("REDIS_DB"),
		},
		AWS: AWSConfig{
			AccessKeyID:     viper.GetString("AWS_ACCESS_KEY_ID"),
			SecretAccessKey: viper.GetString("AWS_SECRET_ACCESS_KEY"),
			Region:          viper.GetString("AWS_REGION"),
			KinesisStream:   viper.GetString("KINESIS_STREAM_ARN"),
			DynamoDBTable:   viper.GetString("DYNAMODB_TABLE"),
			PartitionKey:    viper.GetString("DYNAMODB_PARTITION_KEY"),
			SortKey:         viper.GetString("DYNAMODB_SORT_KEY"),
		},
		Tracing: TracingConfig{
			Endpoint:   viper.GetString("OPENOBSERVE_TRACE_API_ENDPOINT"),
			APIKey:     viper.GetString("OPENOBSERVE_TRACE_API_KEY"),
			StreamName: viper.GetString("OPENOBSERVE_TRACE_STREAM_NAME"),
		},
		Logs: LogsConfig{
			Endpoint: viper.GetString("OPENOBSERVE_LOGS_API_ENDPOINT"),
			Username: viper.GetString("OPENOBSERVE_LOGS_USERNAME"),
			Password: viper.GetString("OPENOBSERVE_LOGS_PASSWORD"),
		},
		Events: EventsConfig{
			MerchantSync:         viper.GetString("EVENT_MERCHANT_SYNC"),
			PlayerSync:           viper.GetString("EVENT_PLAYER_SYNC"),
			ManagerSync:          viper.GetString("EVENT_MANAGER_SYNC"),
			IdentityMerchantSync: viper.GetString("EVENT_IDENTITY_MERCHANT_SYNC"),
			IdentityPlayerSync:   viper.GetString("EVENT_IDENTITY_PLAYER_SYNC"),
			IdentityManagerSync:  viper.GetString("EVENT_IDENTITY_MANAGER_SYNC"),
		},
	}

	return config, nil
}

// LoadAWSConfig 加載AWS配置
func (c *Config) LoadAWSConfig(ctx context.Context) (aws.Config, error) {
	return awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(c.AWS.Region),
		awsconfig.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return aws.Credentials{
				AccessKeyID:     c.AWS.AccessKeyID,
				SecretAccessKey: c.AWS.SecretAccessKey,
			}, nil
		})),
		awsconfig.WithRetryer(func() aws.Retryer {
			return retry.NewStandard(func(o *retry.StandardOptions) {
				o.MaxAttempts = 3               // 最大重試次數
				o.MaxBackoff = 10 * time.Second // 最大重試間隔
			})
		}),
	)
}
