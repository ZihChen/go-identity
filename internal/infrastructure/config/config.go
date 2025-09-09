package config

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
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
	Host        string
	Port        int
	User        string
	Password    string
	DBName      string
	Options     string
	MaxIdle     int
	MaxOpen     int
	Timeout     time.Duration
	MaxLifetime time.Duration // 連線最大生命週期
	MaxIdleTime time.Duration // 連線最大空閒時間
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
	SessionToken    string
	Region          string
	KinesisStream   string
	ConsumeStream   string
	ProduceStream   string
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
	TagSync              string
	LevelSync            string
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
			Host:        viper.GetString("DB_HOST"),
			Port:        viper.GetInt("DB_PORT"),
			User:        viper.GetString("DB_USER"),
			Password:    viper.GetString("DB_PASSWORD"),
			DBName:      viper.GetString("DB_NAME"),
			Options:     viper.GetString("DB_OPTIONS"),
			MaxIdle:     getIntWithDefault("DB_MAX_IDLE", 25),
			MaxOpen:     getIntWithDefault("DB_MAX_OPEN", 100),
			Timeout:     getDurationWithDefault("DB_TIMEOUT", 5*time.Second),
			MaxLifetime: getDurationWithDefault("DB_MAX_LIFETIME", 1*time.Hour),
			MaxIdleTime: getDurationWithDefault("DB_MAX_IDLE_TIME", 30*time.Minute),
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
			SessionToken:    viper.GetString("AWS_SESSION_TOKEN"),
			Region:          viper.GetString("AWS_REGION"),
			KinesisStream:   viper.GetString("KINESIS_STREAM_ARN"),
			ConsumeStream:   viper.GetString("KINESIS_CONSUME_STREAM_NAME"),
			ProduceStream:   viper.GetString("KINESIS_PRODUCE_STREAM_NAME"),
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
			TagSync:              viper.GetString("EVENT_TAG_SYNC"),
			LevelSync:            viper.GetString("EVENT_LEVEL_SYNC"),
			IdentityMerchantSync: viper.GetString("EVENT_IDENTITY_MERCHANT_SYNC"),
			IdentityPlayerSync:   viper.GetString("EVENT_IDENTITY_PLAYER_SYNC"),
			IdentityManagerSync:  viper.GetString("EVENT_IDENTITY_MANAGER_SYNC"),
		},
	}

	return config, nil
}

// LoadAWSConfig 加載AWS配置
func (c *Config) LoadAWSConfig(ctx context.Context) (aws.Config, error) {
	var opts []func(*awsconfig.LoadOptions) error

	opts = append(
		opts,
		awsconfig.WithRegion(c.AWS.Region),
		awsconfig.WithRetryer(func() aws.Retryer {
			return retry.NewStandard(func(o *retry.StandardOptions) {
				o.MaxAttempts = 3               // 最大重試次數
				o.MaxBackoff = 10 * time.Second // 最大重試間隔
			})
		}),
	)

	if c.App.Env == "local" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
				return aws.Credentials{
					AccessKeyID:     c.AWS.AccessKeyID,
					SecretAccessKey: c.AWS.SecretAccessKey,
					SessionToken:    c.AWS.SessionToken,
				}, nil
			}),
		))
	}
	return awsconfig.LoadDefaultConfig(ctx, opts...)
}

// getIntWithDefault 獲取 int 配置值，如果不存在則使用預設值
func getIntWithDefault(key string, defaultValue int) int {
	if viper.IsSet(key) {
		return viper.GetInt(key)
	}
	return defaultValue
}

// getDurationWithDefault 獲取 duration 配置值，如果不存在則使用預設值
func getDurationWithDefault(key string, defaultValue time.Duration) time.Duration {
	if viper.IsSet(key) {
		return viper.GetDuration(key)
	}
	return defaultValue
}
