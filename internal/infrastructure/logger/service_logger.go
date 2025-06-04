package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/infraport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/model"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"time"
)

type LogLevel string

const (
	LevelDebug LogLevel = "DEBUG"
	LevelInfo  LogLevel = "INFO"
	LevelWarn  LogLevel = "WARN"
	LevelError LogLevel = "ERROR"
	LevelFatal LogLevel = "FATAL"
)

type ServiceLogger struct {
	logger  *zap.Logger
	AppName string
	Env     string
}

// NewServiceLogger creates a new ServiceLogger
func NewServiceLogger(cfg *config.Config) infraport.Logger {
	config := createZapConfig(cfg.App.Debug)
	logger, err := config.Build()
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger = logger.With(
		zap.String("app", cfg.App.Name),
		zap.String("env", cfg.App.Env),
	)

	return &ServiceLogger{
		logger:  logger,
		AppName: cfg.App.Name,
		Env:     cfg.App.Env,
	}
}

func createZapConfig(debug bool) zap.Config {
	var config zap.Config
	config.EncoderConfig.NameKey = "app"

	if debug {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	return config
}

func (s *ServiceLogger) logWithLevel(ctx context.Context, level LogLevel, msg string, fields ...*model.LoggerFiled) {
	traceID, spanID := s.extractTraceInfo(ctx)
	zapFields := s.createZapFields(traceID, spanID, fields)

	switch level {
	case LevelDebug:
		s.logger.Debug(msg, zapFields...)
	case LevelInfo:
		s.logger.Info(msg, zapFields...)
	case LevelWarn:
		s.logger.Warn(msg, zapFields...)
	case LevelError:
		s.logger.Error(msg, zapFields...)
	case LevelFatal:
		s.logger.Fatal(msg, zapFields...)
	}

	// 创建并输出JSON格式日志
	s.outputJSONLog(level, msg, traceID, spanID, fields)
}

func (s *ServiceLogger) extractTraceInfo(ctx context.Context) (traceID, spanID string) {
	if ctx == nil {
		return "", ""
	}
	if v, ok := ctx.Value("trace_id").(string); ok {
		traceID = v
	}
	if v, ok := ctx.Value("span_id").(string); ok {
		spanID = v
	}
	return
}

func (s *ServiceLogger) createZapFields(traceID, spanID string, fields []*model.LoggerFiled) []zap.Field {
	zapFields := make([]zap.Field, 0, len(fields)+2)
	if traceID != "" {
		zapFields = append(zapFields, zap.String("trace_id", traceID))
	}
	if spanID != "" {
		zapFields = append(zapFields, zap.String("span_id", spanID))
	}
	for _, field := range fields {
		zapFields = append(zapFields, zap.Any(field.Key, field.Value))
	}
	return zapFields
}

func (s *ServiceLogger) outputJSONLog(level LogLevel, msg, traceID, spanID string, fields []*model.LoggerFiled) {
	logData := map[string]interface{}{
		"app":       s.AppName,
		"env":       s.Env,
		"timestamp": time.Now().Format(time.RFC3339),
		"level":     level,
		"message":   msg,
	}

	if traceID != "" {
		logData["trace_id"] = traceID
	}
	if spanID != "" {
		logData["span_id"] = spanID
	}
	for _, field := range fields {
		logData[field.Key] = field.Value
	}

	jsonData, err := json.Marshal(logData)
	if err != nil {
		s.logger.Error("Failed to marshal log data to JSON", zap.Error(err))
		return
	}
	fmt.Println(string(jsonData))
}

func (s *ServiceLogger) InfoWithContext(ctx context.Context, msg string, fields ...*model.LoggerFiled) {
	s.logWithLevel(ctx, LevelInfo, msg, fields...)
}

func (s *ServiceLogger) DebugWithContext(ctx context.Context, msg string, fields ...*model.LoggerFiled) {
	s.logWithLevel(ctx, LevelDebug, msg, fields...)
}

func (s *ServiceLogger) ErrorWithContext(ctx context.Context, msg string, fields ...*model.LoggerFiled) {
	s.logWithLevel(ctx, LevelError, msg, fields...)
}

func (s *ServiceLogger) WarnWithContext(ctx context.Context, msg string, fields ...*model.LoggerFiled) {
	s.logWithLevel(ctx, LevelWarn, msg, fields...)
}

func (s *ServiceLogger) FatalWithContext(ctx context.Context, msg string, fields ...*model.LoggerFiled) {
	s.logWithLevel(ctx, LevelFatal, msg, fields...)
}

func (s *ServiceLogger) DebugLog(msg string, fields ...*model.LoggerFiled) {
	s.logWithLevel(nil, LevelDebug, msg, fields...)
}

func (s *ServiceLogger) InfoLog(msg string, fields ...*model.LoggerFiled) {
	s.logWithLevel(nil, LevelInfo, msg, fields...)
}

func (s *ServiceLogger) ErrorLog(msg string, fields ...*model.LoggerFiled) {
	s.logWithLevel(nil, LevelError, msg, fields...)
}

func (s *ServiceLogger) WarnLog(msg string, fields ...*model.LoggerFiled) {
	s.logWithLevel(nil, LevelWarn, msg, fields...)
}

func (s *ServiceLogger) FatalLog(msg string, fields ...*model.LoggerFiled) {
	s.logWithLevel(nil, LevelFatal, msg, fields...)
}

func (s *ServiceLogger) Error(key string, value error) *model.LoggerFiled {
	return &model.LoggerFiled{Key: key, Value: value}
}

func (s *ServiceLogger) String(key string, value string) *model.LoggerFiled {
	return &model.LoggerFiled{Key: key, Value: value}
}

func (s *ServiceLogger) Int(key string, value int) *model.LoggerFiled {
	return &model.LoggerFiled{Key: key, Value: value}
}

func (s *ServiceLogger) Int64(key string, value int64) *model.LoggerFiled {
	return &model.LoggerFiled{Key: key, Value: value}
}

func (s *ServiceLogger) UInt64(key string, value uint64) *model.LoggerFiled {
	return &model.LoggerFiled{Key: key, Value: value}
}

func (s *ServiceLogger) Float64(key string, value float64) *model.LoggerFiled {
	return &model.LoggerFiled{Key: key, Value: value}
}

func (s *ServiceLogger) Bool(key string, value bool) *model.LoggerFiled {
	return &model.LoggerFiled{Key: key, Value: value}
}

func (s *ServiceLogger) Any(key string, value interface{}) *model.LoggerFiled {
	return &model.LoggerFiled{Key: key, Value: value}
}

func (s *ServiceLogger) Close() {

}
