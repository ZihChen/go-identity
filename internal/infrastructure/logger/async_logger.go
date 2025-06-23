package logger

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/infraport"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// TODO:Log改由Fluent-bit蒐集(棄用)

type LogEntry struct {
	Timestamp string   `json:"timestamp"`
	Level     LogLevel `json:"level"`
	Message   string   `json:"message"`
	TraceID   string   `json:"trace_id,omitempty"`
	SpanID    string   `json:"span_id,omitempty"`
	Fields    string   `json:"fields,omitempty"`
}

type Field struct {
	Key   string
	Value interface{}
}

type AsyncLogger struct {
	Logger     *zap.Logger
	Endpoint   string
	AuthHeader string
	LogChannel chan LogEntry
	wg         sync.WaitGroup
}

// NewAsyncLogger workerCount擴增worker數量，default為1
func NewAsyncLogger(cfg *config.Config, workerCount int) infraport.Logger {
	var zapConfig zap.Config

	if cfg.App.Debug {
		zapConfig = zap.NewDevelopmentConfig()
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		zapConfig = zap.NewProductionConfig()
		zapConfig.EncoderConfig.TimeKey = "timestamp"
		zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	logger, err := zapConfig.Build()
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.Logs.Username+":"+cfg.Logs.Password))

	al := &AsyncLogger{
		Logger:     logger,
		Endpoint:   cfg.Logs.Endpoint,
		AuthHeader: auth,
		LogChannel: make(chan LogEntry, 5000),
	}

	for i := 0; i < workerCount; i++ {
		al.wg.Add(1)
		go al.worker()
	}
	return al
}

func (al *AsyncLogger) worker() {
	defer al.wg.Done()
	buffer := make([]LogEntry, 0, 100)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case logEntry, ok := <-al.LogChannel:
			// 確保Channel關閉後剩下的Logs被消耗完
			if !ok {
				al.flush(buffer)
				return
			}
			buffer = append(buffer, logEntry)
			// Logs大於100筆就開始消耗
			if len(buffer) >= 100 {
				al.flush(buffer)
				buffer = buffer[:0]
			}
			// Logs固定每2秒消耗一次
		case <-ticker.C:
			if len(buffer) > 0 {
				al.flush(buffer)
				buffer = buffer[:0]
			}
		}
	}
}

func (al *AsyncLogger) flush(logs []LogEntry) {
	body, _ := json.Marshal(logs)
	req, err := http.NewRequest(http.MethodPost, al.Endpoint, bytes.NewBuffer(body))

	if err != nil {
		log.Println("[flush error] creating request:", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", al.AuthHeader)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("[flush error] sending logs:", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Println("[flush error] status:", resp.Status)
	}
}

func (al *AsyncLogger) Close() {
	close(al.LogChannel)
	al.wg.Wait()
}

func (al *AsyncLogger) DebugWithContext(ctx context.Context, msg string, fields ...*entity.LoggerFiled) {
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     LevelDebug,
		Message:   msg,
		TraceID: func() string {
			if v, ok := ctx.Value("trace_id").(string); ok {
				return v
			}
			return ""
		}(),
		SpanID: func() string {
			if v, ok := ctx.Value("span_id").(string); ok {
				return v
			}
			return ""
		}(),
	}

	zapFields := make([]zap.Field, 0, len(fields)+2)
	loggerFields := make([]map[string]interface{}, 0, len(fields)+2)

	for _, field := range fields {
		loggerFields = append(loggerFields, map[string]interface{}{
			field.Key: field.Value,
		})
		zapFields = append(zapFields, zap.Any(field.Key, field.Value))
	}
	b, _ := jsoniter.Marshal(loggerFields)
	entry.Fields = string(b)

	al.Logger.Debug(msg, zapFields...)
	al.LogChannel <- entry
}

func (al *AsyncLogger) InfoWithContext(ctx context.Context, msg string, fields ...*entity.LoggerFiled) {
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     LevelInfo,
		Message:   msg,
		TraceID: func() string {
			if v, ok := ctx.Value("trace_id").(string); ok {
				return v
			}
			return ""
		}(),
		SpanID: func() string {
			if v, ok := ctx.Value("span_id").(string); ok {
				return v
			}
			return ""
		}(),
	}

	zapFields := make([]zap.Field, 0, len(fields)+2)
	loggerFields := make([]map[string]interface{}, 0, len(fields)+2)

	for _, field := range fields {
		loggerFields = append(loggerFields, map[string]interface{}{
			field.Key: field.Value,
		})
		zapFields = append(zapFields, zap.Any(field.Key, field.Value))
	}
	b, _ := jsoniter.Marshal(loggerFields)
	entry.Fields = string(b)

	al.Logger.Info(msg, zapFields...)
	al.LogChannel <- entry
}

func (al *AsyncLogger) ErrorWithContext(ctx context.Context, msg string, fields ...*entity.LoggerFiled) {
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     LevelError,
		Message:   msg,
		TraceID: func() string {
			if v, ok := ctx.Value("trace_id").(string); ok {
				return v
			}
			return ""
		}(),
		SpanID: func() string {
			if v, ok := ctx.Value("span_id").(string); ok {
				return v
			}
			return ""
		}(),
	}

	zapFields := make([]zap.Field, 0, len(fields)+2)
	loggerFields := make([]map[string]interface{}, 0, len(fields)+2)

	for _, field := range fields {
		loggerFields = append(loggerFields, map[string]interface{}{
			field.Key: field.Value,
		})
		zapFields = append(zapFields, zap.Any(field.Key, field.Value))
	}
	b, _ := jsoniter.Marshal(loggerFields)
	entry.Fields = string(b)

	al.Logger.Error(msg, zapFields...)
	al.LogChannel <- entry
}

func (al *AsyncLogger) WarnWithContext(ctx context.Context, msg string, fields ...*entity.LoggerFiled) {
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     LevelWarn,
		Message:   msg,
		TraceID: func() string {
			if v, ok := ctx.Value("trace_id").(string); ok {
				return v
			}
			return ""
		}(),
		SpanID: func() string {
			if v, ok := ctx.Value("span_id").(string); ok {
				return v
			}
			return ""
		}(),
	}

	zapFields := make([]zap.Field, 0, len(fields)+2)
	loggerFields := make([]map[string]interface{}, 0, len(fields)+2)

	for _, field := range fields {
		loggerFields = append(loggerFields, map[string]interface{}{
			field.Key: field.Value,
		})
		zapFields = append(zapFields, zap.Any(field.Key, field.Value))
	}
	b, _ := jsoniter.Marshal(loggerFields)
	entry.Fields = string(b)

	al.Logger.Warn(msg, zapFields...)
	al.LogChannel <- entry
}

func (al *AsyncLogger) FatalWithContext(ctx context.Context, msg string, fields ...*entity.LoggerFiled) {
	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     LevelFatal,
		Message:   msg,
		TraceID: func() string {
			if v, ok := ctx.Value("trace_id").(string); ok {
				return v
			}
			return ""
		}(),
		SpanID: func() string {
			if v, ok := ctx.Value("span_id").(string); ok {
				return v
			}
			return ""
		}(),
	}

	zapFields := make([]zap.Field, 0, len(fields)+2)
	loggerFields := make([]map[string]interface{}, 0, len(fields)+2)

	for _, field := range fields {
		loggerFields = append(loggerFields, map[string]interface{}{
			field.Key: field.Value,
		})
		zapFields = append(zapFields, zap.Any(field.Key, field.Value))
	}
	b, _ := jsoniter.Marshal(loggerFields)
	entry.Fields = string(b)

	al.Logger.Fatal(msg, zapFields...)
	al.LogChannel <- entry
}

func (al *AsyncLogger) DebugLog(msg string, fields ...*entity.LoggerFiled) {}

func (al *AsyncLogger) InfoLog(msg string, fields ...*entity.LoggerFiled) {}

func (al *AsyncLogger) ErrorLog(msg string, fields ...*entity.LoggerFiled) {}

func (al *AsyncLogger) WarnLog(msg string, fields ...*entity.LoggerFiled) {}

func (al *AsyncLogger) FatalLog(msg string, fields ...*entity.LoggerFiled) {}

func (al *AsyncLogger) Error(key string, value error) *entity.LoggerFiled {
	return &entity.LoggerFiled{Key: key, Value: value}
}

func (al *AsyncLogger) String(key string, value string) *entity.LoggerFiled {
	return &entity.LoggerFiled{Key: key, Value: value}
}

func (al *AsyncLogger) Int(key string, value int) *entity.LoggerFiled {
	return &entity.LoggerFiled{Key: key, Value: value}
}

func (al *AsyncLogger) Int64(key string, value int64) *entity.LoggerFiled {
	return &entity.LoggerFiled{Key: key, Value: value}
}

func (al *AsyncLogger) UInt64(key string, value uint64) *entity.LoggerFiled {
	return &entity.LoggerFiled{Key: key, Value: value}
}

func (al *AsyncLogger) Float64(key string, value float64) *entity.LoggerFiled {
	return &entity.LoggerFiled{Key: key, Value: value}
}

func (al *AsyncLogger) Bool(key string, value bool) *entity.LoggerFiled {
	return &entity.LoggerFiled{Key: key, Value: value}
}

func (al *AsyncLogger) Any(key string, value interface{}) *entity.LoggerFiled {
	return &entity.LoggerFiled{Key: key, Value: value}
}
