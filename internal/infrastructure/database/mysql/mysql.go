package mysql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Database struct {
	dbInstance *gorm.DB
	cfg        *config.Config
	logger     infrastructure.Logger
	isClose    chan struct{}
}

func NewDatabase(cfg *config.Config, logger infrastructure.Logger) (*Database, error) {
	db := &Database{
		cfg:     cfg,
		logger:  logger,
		isClose: make(chan struct{}),
	}
	if err := db.connect(); err != nil {
		return nil, err
	}

	go db.startHealthChecker()

	return db, nil
}

func (d *Database) connect() error {
	logLevel := logger.Silent
	if d.cfg.App.Debug {
		logLevel = logger.Info
	}

	gormCfg := &gorm.Config{
		PrepareStmt:            true, // 啟用預編譯語句 (語法快取)
		SkipDefaultTransaction: true, // 關閉默認Transaction，隨應用場景自行加入Transaction
		Logger:                 logger.Default.LogMode(logLevel),
	}

	dsn := buildDSN(d.cfg)
	d.logger.InfoLog(fmt.Sprintf("Connecting to database DSN:%s", dsn))

	db, err := gorm.Open(mysql.Open(dsn), gormCfg)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection pool: %w", err)
	}

	// 使用配置文件中的連接池設置
	sqlDB.SetMaxIdleConns(d.cfg.Database.MaxIdle)        // 限制最大閒置連線數
	sqlDB.SetMaxOpenConns(d.cfg.Database.MaxOpen)        // 限制最大開啟的連線數
	sqlDB.SetConnMaxLifetime(d.cfg.Database.MaxLifetime) // 連線最大生命週期
	sqlDB.SetConnMaxIdleTime(d.cfg.Database.MaxIdleTime) // 連線最大空閒時間
	d.dbInstance = db
	return nil

}

func (d *Database) Close() error {
	// 關閉Health Checker
	close(d.isClose)

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

func (d *Database) startHealthChecker() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := d.Ping(); err != nil {
				d.logger.ErrorLog(
					"Failed to ping database, retrying to reconnect...",
					d.logger.Error("err", err),
				)
				// 嘗試重新連線
				if err = d.connect(); err != nil {
					d.logger.ErrorLog("Failed to reconnect to database", d.logger.Error("err", err))
				}
			} else {
				d.logger.InfoLog("Database connection is up!")
			}
		case <-d.isClose:
			return
		}
	}
}
