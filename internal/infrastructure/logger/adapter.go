package logger

import (
	"fmt"

	"go.uber.org/zap"
)

// AsynqLoggerAdapter 是 asynq.Logger 接口
type AsynqLoggerAdapter struct {
	logger *zap.Logger
}

// NewAsynqLoggerAdapter 創建一個新的 asynq.Logger 適配
func NewAsynqLoggerAdapter(logger *zap.Logger) *AsynqLoggerAdapter {
	return &AsynqLoggerAdapter{
		logger: logger,
	}
}

// Debug 實現 asynq.Logger 接口的 Debug 方法
func (a *AsynqLoggerAdapter) Debug(args ...interface{}) {
	a.logger.Debug(fmt.Sprint(args...))
}

// Info 實現 asynq.Logger 接口的 Info 方法
func (a *AsynqLoggerAdapter) Info(args ...interface{}) {
	a.logger.Info(fmt.Sprint(args...))
}

// Warn 實現 asynq.Logger 接口的 Warn 方法
func (a *AsynqLoggerAdapter) Warn(args ...interface{}) {
	a.logger.Warn(fmt.Sprint(args...))
}

// Error 實現 asynq.Logger 接口的 Error 方法
func (a *AsynqLoggerAdapter) Error(args ...interface{}) {
	a.logger.Error(fmt.Sprint(args...))
}

// Fatal 實現 asynq.Logger 接口的 Fatal 方法
func (a *AsynqLoggerAdapter) Fatal(args ...interface{}) {
	a.logger.Fatal(fmt.Sprint(args...))
}
