package database

import (
	"fmt"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database 資料庫連接封裝
type Database struct {
	DB *gorm.DB
}

// NewDatabase 創建新的資料庫連接
func NewDatabase(cfg *config.Config) (*Database, error) {
	// 構建 DSN 字串
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=True",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.DBName,
	)

	// 添加額外選項
	if cfg.Database.Options != "" {
		dsn = fmt.Sprintf("%s&%s", dsn, cfg.Database.Options)
	}

	// 添加 TLS 設置
	dsn = fmt.Sprintf("%s&tls=true", dsn)

	// 設置日誌級別
	logLevel := logger.Silent
	if cfg.App.Debug {
		logLevel = logger.Info
	}

	// 創建 GORM 配置
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	}

	// 連接資料庫
	db, err := gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// 獲取底層 SQL 連接池
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection pool: %w", err)
	}

	// 設置連接池參數
	maxIdle := cfg.Database.MaxIdle
	if maxIdle <= 0 {
		maxIdle = 10
	}

	maxOpen := cfg.Database.MaxOpen
	if maxOpen <= 0 {
		maxOpen = 100
	}

	timeout := cfg.Database.Timeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetConnMaxLifetime(timeout)

	return &Database{DB: db}, nil
}

// Close 關閉資料庫連接
func (d *Database) Close() error {
	sqlDB, err := d.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}

	return nil
}
