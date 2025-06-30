package mysql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Database struct {
	dbInstance *gorm.DB
}

func NewDatabase(cfg *config.Config) (*Database, error) {
	logLevel := logger.Silent
	if cfg.App.Debug {
		logLevel = logger.Info
	}

	gormCfg := &gorm.Config{
		PrepareStmt:            true, // 啟用預編譯語句 (語法快取)
		SkipDefaultTransaction: true, // 關閉默認Transaction，隨應用場景自行加入Transaction
		Logger:                 logger.Default.LogMode(logLevel),
	}

	db, err := gorm.Open(mysql.Open(buildDSN(cfg)), gormCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection pool: %w", err)
	}

	// 連線池設定
	sqlDB.SetMaxIdleConns(100)                 // 限制最大開啟的連線數
	sqlDB.SetMaxOpenConns(500)                 // 限制最大閒置連線數
	sqlDB.SetConnMaxLifetime(30 * time.Second) // 空閒連線 timeout 時間

	return &Database{dbInstance: db}, nil
}

func (d *Database) Close() error {
	if d.dbInstance != nil {
		sqlDB, err := d.dbInstance.DB()
		if err != nil {
			return fmt.Errorf("DB Connection failed: %w", err)
		}
		return sqlDB.Close()
	}
	return errors.New("DB is empty")
}

func (d *Database) Ping() error {
	sqlDB, err := d.dbInstance.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}

func (d *Database) GetDBConnection() *gorm.DB {
	return d.dbInstance
}

func buildDSN(cfg *config.Config) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=True&tls=true",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.DBName,
	)
}
