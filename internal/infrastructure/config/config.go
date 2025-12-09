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
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	AWS      AWSConfig
	Tracing  TracingConfig
	Logs     LogsConfig
	Events   EventsConfig
	Consumer ConsumerConfig
	CORS     CORSConfig
}

// AppConfig 應用程序基本配置
type AppConfig struct {
	Name  string
	Env   string
	Port  int
	Debug bool
}

// ServerConfig HTTP服務器配置
type ServerConfig struct {
	HttpDomain   string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
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
	Domain       string
	Port         int
	Password     string
	DB           int
	PoolSize     int           // 連線池大小
	MinIdleConns int           // 最小空閒連線數
	MaxRetries   int           // 最大重試次數
	DialTimeout  time.Duration // 連線超時
	ReadTimeout  time.Duration // 讀取超時
	WriteTimeout time.Duration // 寫入超時
	PoolTimeout  time.Duration // 連線池等待超時
	IdleTimeout  time.Duration // 連線最大空閒時間
	MaxConnAge   time.Duration // 連線最大生命週期
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
	AgentSync            string
	IdentityMerchantSync string
	IdentityPlayerSync   string
	IdentityManagerSync  string
}

// ConsumerConfig Consumer 服務配置
type ConsumerConfig struct {
	// 批次處理配置
	BatchSize int // 每批次記錄數 (用於 Worker Pool 處理)

	// KDS 獲取配置
	KDSRecordLimit int // 每次從 KDS GetRecords 獲取的記錄數上限

	// Worker Pool 配置
	WorkerPoolSize   int // Worker 數量
	WorkerBufferSize int // Worker 通道緩衝區大小

	// 退避策略配置
	MinBackoff time.Duration // 最小退避時間
	MaxBackoff time.Duration // 最大退避時間

	// 分片處理配置
	MaxShardConcurrency int           // 最大並行分片數
	ShardLockTimeout    time.Duration // 分片鎖超時時間
}

// CORSConfig CORS配置
type CORSConfig struct {
	Enabled          bool     // 是否啟用CORS
	AllowedOrigins   []string // 允許的來源列表
	AllowedMethods   []string // 允許的HTTP方法
	AllowedHeaders   []string // 允許的請求頭
	ExposedHeaders   []string // 暴露的回應頭
	AllowCredentials bool     // 是否允許憑證
	MaxAge           int      // 預檢請求快取時間(小時)
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
		Server: ServerConfig{
			HttpDomain:   viper.GetString("SERVER_HTTP_DOMAIN"),
			ReadTimeout:  getDurationWithDefault("SERVER_READ_TIMEOUT", 30*time.Second),
			WriteTimeout: getDurationWithDefault("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:  getDurationWithDefault("SERVER_IDLE_TIMEOUT", 120*time.Second),
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
			Domain:       viper.GetString("REDIS_DOMAIN"),
			Port:         viper.GetInt("REDIS_PORT"),
			Password:     viper.GetString("REDIS_PWD"),
			DB:           viper.GetInt("REDIS_DB"),
			PoolSize:     getIntWithDefault("REDIS_POOL_SIZE", 20),
			MinIdleConns: getIntWithDefault("REDIS_MIN_IDLE_CONNS", 5),
			MaxRetries:   getIntWithDefault("REDIS_MAX_RETRIES", 3),
			DialTimeout:  getDurationWithDefault("REDIS_DIAL_TIMEOUT", 5*time.Second),
			ReadTimeout:  getDurationWithDefault("REDIS_READ_TIMEOUT", 3*time.Second),
			WriteTimeout: getDurationWithDefault("REDIS_WRITE_TIMEOUT", 3*time.Second),
			PoolTimeout:  getDurationWithDefault("REDIS_POOL_TIMEOUT", 4*time.Second),
			IdleTimeout:  getDurationWithDefault("REDIS_IDLE_TIMEOUT", 5*time.Minute),
			MaxConnAge:   getDurationWithDefault("REDIS_MAX_CONN_AGE", 30*time.Minute),
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
			AgentSync:            viper.GetString("EVENT_AGENT_SYNC"),
			IdentityMerchantSync: viper.GetString("EVENT_IDENTITY_MERCHANT_SYNC"),
			IdentityPlayerSync:   viper.GetString("EVENT_IDENTITY_PLAYER_SYNC"),
			IdentityManagerSync:  viper.GetString("EVENT_IDENTITY_MANAGER_SYNC"),
		},
		Consumer: ConsumerConfig{
			// 批次處理配置
			BatchSize: getIntWithDefault("CONSUMER_BATCH_SIZE", 100),

			// KDS 獲取配置
			KDSRecordLimit: getIntWithDefault("CONSUMER_KDS_RECORD_LIMIT", 1000),

			// Worker Pool 配置
			WorkerPoolSize:   getIntWithDefault("CONSUMER_WORKER_POOL_SIZE", 10),
			WorkerBufferSize: getIntWithDefault("CONSUMER_WORKER_BUFFER_SIZE", 200),

			// 退避策略配置
			MinBackoff: getDurationWithDefault("CONSUMER_MIN_BACKOFF", 500*time.Millisecond),
			MaxBackoff: getDurationWithDefault("CONSUMER_MAX_BACKOFF", 5*time.Second),

			// 分片處理配置
			MaxShardConcurrency: getIntWithDefault("CONSUMER_MAX_SHARD_CONCURRENCY", 8),
			ShardLockTimeout: getDurationWithDefault(
				"CONSUMER_SHARD_LOCK_TIMEOUT",
				1*time.Minute,
			),
		},
		CORS: CORSConfig{
			Enabled:        getBoolWithDefault("CORS_ENABLED", true),
			AllowedOrigins: getSliceWithDefault("CORS_ALLOWED_ORIGINS", []string{}),
			AllowedMethods: getSliceWithDefault(
				"CORS_ALLOWED_METHODS",
				[]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			),
			AllowedHeaders: getSliceWithDefault(
				"CORS_ALLOWED_HEADERS",
				[]string{"Origin", "Content-Type", "Authorization", "API-Key"},
			),
			ExposedHeaders:   getSliceWithDefault("CORS_EXPOSED_HEADERS", []string{"Content-Type"}),
			AllowCredentials: getBoolWithDefault("CORS_ALLOW_CREDENTIALS", false),
			MaxAge:           getIntWithDefault("CORS_MAX_AGE", 12),
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

// getFloatWithDefault 獲取 float64 配置值，如果不存在則使用預設值
func getFloatWithDefault(key string, defaultValue float64) float64 {
	if viper.IsSet(key) {
		return viper.GetFloat64(key)
	}
	return defaultValue
}

// getBoolWithDefault 獲取 bool 配置值，如果不存在則使用預設值
func getBoolWithDefault(key string, defaultValue bool) bool {
	if viper.IsSet(key) {
		return viper.GetBool(key)
	}
	return defaultValue
}

// getSliceWithDefault 獲取 string slice 配置值，如果不存在則使用預設值
func getSliceWithDefault(key string, defaultValue []string) []string {
	if viper.IsSet(key) {
		return viper.GetStringSlice(key)
	}
	return defaultValue
}

// PrintConfig 輸出所有配置值用於除錯和追蹤
func (c *Config) PrintConfig() {
	fmt.Println("=== Configuration Summary ===")

	fmt.Printf("\n[App]\n")
	fmt.Printf("  Name: %s\n", c.App.Name)
	fmt.Printf("  Environment: %s\n", c.App.Env)
	fmt.Printf("  Port: %d\n", c.App.Port)
	fmt.Printf("  Debug: %t\n", c.App.Debug)

	fmt.Printf("\n[Server]\n")
	fmt.Printf("  ReadTimeout: %v\n", c.Server.ReadTimeout)
	fmt.Printf("  WriteTimeout: %v\n", c.Server.WriteTimeout)
	fmt.Printf("  IdleTimeout: %v\n", c.Server.IdleTimeout)

	fmt.Printf("\n[Database]\n")
	fmt.Printf("  Host: %s\n", c.Database.Host)
	fmt.Printf("  Port: %d\n", c.Database.Port)
	fmt.Printf("  User: %s\n", c.Database.User)
	fmt.Printf("  Password: %s\n", maskPassword(c.Database.Password))
	fmt.Printf("  DBName: %s\n", c.Database.DBName)
	fmt.Printf("  Options: %s\n", c.Database.Options)
	fmt.Printf("  MaxIdle: %d\n", c.Database.MaxIdle)
	fmt.Printf("  MaxOpen: %d\n", c.Database.MaxOpen)
	fmt.Printf("  MaxLifetime: %v\n", c.Database.MaxLifetime)
	fmt.Printf("  MaxIdleTime: %v\n", c.Database.MaxIdleTime)

	fmt.Printf("\n[Redis]\n")
	fmt.Printf("  Domain: %s\n", c.Redis.Domain)
	fmt.Printf("  Port: %d\n", c.Redis.Port)
	fmt.Printf("  Password: %s\n", maskPassword(c.Redis.Password))
	fmt.Printf("  DB: %d\n", c.Redis.DB)
	fmt.Printf("  PoolSize: %d\n", c.Redis.PoolSize)
	fmt.Printf("  MinIdleConns: %d\n", c.Redis.MinIdleConns)
	fmt.Printf("  MaxRetries: %d\n", c.Redis.MaxRetries)
	fmt.Printf("  DialTimeout: %v\n", c.Redis.DialTimeout)
	fmt.Printf("  ReadTimeout: %v\n", c.Redis.ReadTimeout)
	fmt.Printf("  WriteTimeout: %v\n", c.Redis.WriteTimeout)
	fmt.Printf("  PoolTimeout: %v\n", c.Redis.PoolTimeout)
	fmt.Printf("  IdleTimeout: %v\n", c.Redis.IdleTimeout)
	fmt.Printf("  MaxConnAge: %v\n", c.Redis.MaxConnAge)

	fmt.Printf("\n[AWS]\n")
	fmt.Printf("  AccessKeyID: %s\n", maskAPIKey(c.AWS.AccessKeyID))
	fmt.Printf("  SecretAccessKey: %s\n", maskAPIKey(c.AWS.SecretAccessKey))
	fmt.Printf("  SessionToken: %s\n", maskAPIKey(c.AWS.SessionToken))
	fmt.Printf("  Region: %s\n", c.AWS.Region)
	fmt.Printf("  KinesisStream: %s\n", c.AWS.KinesisStream)
	fmt.Printf("  DynamoDBTable: %s\n", c.AWS.DynamoDBTable)
	fmt.Printf("  PartitionKey: %s\n", c.AWS.PartitionKey)
	fmt.Printf("  SortKey: %s\n", c.AWS.SortKey)

	fmt.Printf("\n[Tracing]\n")
	fmt.Printf("  Endpoint: %s\n", c.Tracing.Endpoint)
	fmt.Printf("  APIKey: %s\n", maskAPIKey(c.Tracing.APIKey))
	fmt.Printf("  StreamName: %s\n", c.Tracing.StreamName)

	fmt.Printf("\n[Events]\n")
	fmt.Printf("  MerchantSync: %s\n", c.Events.MerchantSync)
	fmt.Printf("  PlayerSync: %s\n", c.Events.PlayerSync)
	fmt.Printf("  ManagerSync: %s\n", c.Events.ManagerSync)
	fmt.Printf("  LevelSync: %s\n", c.Events.LevelSync)
	fmt.Printf("  TagSync: %s\n", c.Events.TagSync)
	fmt.Printf("  IdentityMerchantSync: %s\n", c.Events.IdentityMerchantSync)
	fmt.Printf("  IdentityPlayerSync: %s\n", c.Events.IdentityPlayerSync)
	fmt.Printf("  IdentityManagerSync: %s\n", c.Events.IdentityManagerSync)

	fmt.Printf("\n[Consumer]\n")
	fmt.Printf("  BatchSize: %d\n", c.Consumer.BatchSize)
	fmt.Printf("  KDSRecordLimit: %d\n", c.Consumer.KDSRecordLimit)
	fmt.Printf("  WorkerPoolSize: %d\n", c.Consumer.WorkerPoolSize)
	fmt.Printf("  WorkerBufferSize: %d\n", c.Consumer.WorkerBufferSize)
	fmt.Printf("  MinBackoff: %v\n", c.Consumer.MinBackoff)
	fmt.Printf("  MaxBackoff: %v\n", c.Consumer.MaxBackoff)

	fmt.Printf("\n[CORS]\n")
	fmt.Printf("  Enabled: %t\n", c.CORS.Enabled)
	fmt.Printf("  AllowedOrigins: %v\n", c.CORS.AllowedOrigins)
	fmt.Printf("  AllowedMethods: %v\n", c.CORS.AllowedMethods)
	fmt.Printf("  AllowedHeaders: %v\n", c.CORS.AllowedHeaders)
	fmt.Printf("  ExposedHeaders: %v\n", c.CORS.ExposedHeaders)
	fmt.Printf("  AllowCredentials: %t\n", c.CORS.AllowCredentials)
	fmt.Printf("  MaxAge: %d hours\n", c.CORS.MaxAge)

	fmt.Println("\n==============================")
}

// maskPassword 遮蔽密碼顯示
func maskPassword(password string) string {
	if password == "" {
		return "<empty>"
	}
	if len(password) <= 4 {
		return "****"
	}
	return password[:2] + "****" + password[len(password)-2:]
}

// maskAPIKey 遮蔽API Key顯示
func maskAPIKey(key string) string {
	if key == "" {
		return "<empty>"
	}
	if len(key) <= 8 {
		return "********"
	}
	return key[:4] + "****" + key[len(key)-4:]
}
