package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/infra"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/model"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"time"
)

const (
	LevelDebug = "DEBUG"
	LevelInfo  = "INFO"
	LevelWarn  = "WARN"
	LevelError = "ERROR"
	LevelFatal = "FATAL"
)

type ServiceLogger struct {
	Logger  *zap.Logger
	AppName string
	Env     string
}

// NewServiceLogger creates a new ServiceLogger
func NewServiceLogger(cfg *config.Config) infra.Logger {
	var config zap.Config

	config.EncoderConfig.NameKey = "app"

	if cfg.App.Debug {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

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
		Logger:  logger,
		AppName: cfg.App.Name,
		Env:     cfg.App.Env,
	}
}

func (s *ServiceLogger) InfoWithContext(ctx context.Context, msg string, fields ...*model.LoggerFiled) {
	traceID := ""
	if v, ok := ctx.Value("trace_id").(string); ok {
		traceID = v
	}

	spanID := ""
	if v, ok := ctx.Value("span_id").(string); ok {
		spanID = v
	}

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

	s.Logger.Info(msg, zapFields...)

	logData := map[string]interface{}{
		"app":       s.AppName,
		"env":       s.Env,
		"timestamp": time.Now().Format(time.RFC3339),
		"level":     LevelInfo,
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
		s.Logger.Error("Failed to marshal log data to JSON", zap.Error(err))
		return
	}

	fmt.Println(string(jsonData))
}

func (s *ServiceLogger) DebugWithContext(ctx context.Context, msg string, fields ...*model.LoggerFiled) {
	traceID := ""
	if v, ok := ctx.Value("trace_id").(string); ok {
		traceID = v
	}

	spanID := ""
	if v, ok := ctx.Value("span_id").(string); ok {
		spanID = v
	}

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

	s.Logger.Debug(msg, zapFields...)

	logData := map[string]interface{}{
		"app":       s.AppName,
		"env":       s.Env,
		"timestamp": time.Now().Format(time.RFC3339),
		"level":     LevelDebug,
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
		s.Logger.Error("Failed to marshal log data to JSON", zap.Error(err))
		return
	}

	fmt.Println(string(jsonData))
}

func (s *ServiceLogger) ErrorWithContext(ctx context.Context, msg string, fields ...*model.LoggerFiled) {
	traceID := ""
	if v, ok := ctx.Value("trace_id").(string); ok {
		traceID = v
	}
	spanID := ""
	if v, ok := ctx.Value("span_id").(string); ok {
		spanID = v
	}

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

	s.Logger.Error(msg, zapFields...)

	logData := map[string]interface{}{
		"app":       s.AppName,
		"env":       s.Env,
		"timestamp": time.Now().Format(time.RFC3339),
		"level":     LevelError,
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
		s.Logger.Error("Failed to marshal log data to JSON", zap.Error(err))
		return
	}

	fmt.Println(string(jsonData))
}

func (s *ServiceLogger) WarnWithContext(ctx context.Context, msg string, fields ...*model.LoggerFiled) {
	traceID := ""
	if v, ok := ctx.Value("trace_id").(string); ok {
		traceID = v
	}
	spanID := ""
	if v, ok := ctx.Value("span_id").(string); ok {
		spanID = v
	}

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

	s.Logger.Warn(msg, zapFields...)

	logData := map[string]interface{}{
		"app":       s.AppName,
		"env":       s.Env,
		"timestamp": time.Now().Format(time.RFC3339),
		"level":     LevelWarn,
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
		s.Logger.Error("Failed to marshal log data to JSON", zap.Error(err))
		return
	}

	fmt.Println(string(jsonData))
}

func (s *ServiceLogger) FatalWithContext(ctx context.Context, msg string, fields ...*model.LoggerFiled) {
	traceID := ""
	if v, ok := ctx.Value("trace_id").(string); ok {
		traceID = v
	}
	spanID := ""
	if v, ok := ctx.Value("span_id").(string); ok {
		spanID = v
	}

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

	s.Logger.Fatal(msg, zapFields...)

	logData := map[string]interface{}{
		"app":       s.AppName,
		"env":       s.Env,
		"timestamp": time.Now().Format(time.RFC3339),
		"level":     LevelFatal,
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
		s.Logger.Error("Failed to marshal log data to JSON", zap.Error(err))
		return
	}

	fmt.Println(string(jsonData))
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
