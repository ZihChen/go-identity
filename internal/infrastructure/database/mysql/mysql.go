package mysql

import (
	"context"
	"errors"
	"fmt"
	"regexp"
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
		PrepareStmt:            false, // 關閉預編譯語句以降低PlanetScale內存消耗
		SkipDefaultTransaction: true,  // 關閉默認Transaction，隨應用場景自行加入Transaction
		Logger:                 logger.Default.LogMode(logLevel),
	}

	dsn := buildDSN(d.cfg)
	sanitizedDSN := sanitizeDSN(dsn)
	d.logger.InfoLog(fmt.Sprintf("Connecting to database DSN:%s", sanitizedDSN))

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

func sanitizeDSN(dsn string) string {
	re := regexp.MustCompile(`(.*?):(.+?)@(.*)`)
	matches := re.FindStringSubmatch(dsn)
	if len(matches) >= 4 {
		user := matches[1]
		password := matches[2]
		rest := matches[3]

		maskedPassword := "****"
		if len(password) > 0 {
			maskedPassword = "****"
		}

		return fmt.Sprintf("%s:%s@%s", user, maskedPassword, rest)
	}
	return dsn
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
				// 連線正常，輸出連接池狀態監控
				sqlDB, err := d.dbInstance.DB()
				if err == nil {
					stats := sqlDB.Stats()

					// 輸出詳細的連接池統計信息
					d.logger.InfoLog("Database connection pool stats",
						d.logger.Int("max_open_connections", stats.MaxOpenConnections),
						d.logger.Int("open_connections", stats.OpenConnections),
						d.logger.Int("in_use", stats.InUse),
						d.logger.Int("idle", stats.Idle),
						d.logger.Int64("wait_count", stats.WaitCount),
						d.logger.String("wait_duration", stats.WaitDuration.String()),
						d.logger.Int64("max_idle_closed", stats.MaxIdleClosed),
						d.logger.Int64("max_lifetime_closed", stats.MaxLifetimeClosed),
					)

					// 輸出GORM配置狀態（用於驗證PrepareStmt設置）
					d.logger.InfoLog("Database configuration status",
						d.logger.Bool("prepare_stmt_enabled", d.dbInstance.PrepareStmt),
						d.logger.Bool("skip_default_transaction", d.dbInstance.SkipDefaultTransaction),
						d.logger.Int("max_idle_conns_config", d.cfg.Database.MaxIdle),
						d.logger.Int("max_open_conns_config", d.cfg.Database.MaxOpen),
						d.logger.String("max_lifetime_config", d.cfg.Database.MaxLifetime.String()),
						d.logger.String("max_idle_time_config", d.cfg.Database.MaxIdleTime.String()),
					)

					// 檢測異常情況並發出警告
					if stats.WaitCount > 0 {
						d.logger.WarnLog("Database connection pool is experiencing waits",
							d.logger.Int64("wait_count", stats.WaitCount),
							d.logger.String("total_wait_duration", stats.WaitDuration.String()),
							d.logger.String("suggestion", "Consider increasing DB_MAX_OPEN connections"),
						)
					}

					// 檢查是否有過多的空閒連接被關閉
					if stats.MaxIdleClosed > 100 {
						d.logger.WarnLog("High number of idle connections being closed",
							d.logger.Int64("max_idle_closed", stats.MaxIdleClosed),
							d.logger.String("suggestion", "Consider increasing DB_MAX_IDLE or reducing DB_MAX_IDLE_TIME"),
						)
					}

					// 計算連接池使用率
					if stats.MaxOpenConnections > 0 {
						utilizationPercent := float64(stats.OpenConnections) / float64(stats.MaxOpenConnections) * 100
						d.logger.InfoLog("Database connection pool utilization",
							d.logger.Float64("utilization_percent", utilizationPercent),
						)

						// 如果使用率超過80%，發出警告
						if utilizationPercent > 80 {
							d.logger.WarnLog("Database connection pool utilization is high",
								d.logger.Float64("utilization_percent", utilizationPercent),
								d.logger.String("suggestion", "Connection pool may be under pressure"),
							)
						}
					}
				}

				d.logger.InfoLog("Database connection is up!")
			}
		case <-d.isClose:
			return
		}
	}
}
