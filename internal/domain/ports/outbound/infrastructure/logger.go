package infrastructure

import (
	"context"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
)

type Logger interface {
	DebugWithContext(ctx context.Context, msg string, fields ...*entity.LoggerFiled)
	InfoWithContext(ctx context.Context, msg string, fields ...*entity.LoggerFiled)
	ErrorWithContext(ctx context.Context, msg string, fields ...*entity.LoggerFiled)
	WarnWithContext(ctx context.Context, msg string, fields ...*entity.LoggerFiled)
	FatalWithContext(ctx context.Context, msg string, fields ...*entity.LoggerFiled)
	DebugLog(msg string, fields ...*entity.LoggerFiled)
	InfoLog(msg string, fields ...*entity.LoggerFiled)
	ErrorLog(msg string, fields ...*entity.LoggerFiled)
	WarnLog(msg string, fields ...*entity.LoggerFiled)
	FatalLog(msg string, fields ...*entity.LoggerFiled)
	Error(key string, value error) *entity.LoggerFiled
	String(key string, value string) *entity.LoggerFiled
	Int(key string, value int) *entity.LoggerFiled
	Int64(key string, value int64) *entity.LoggerFiled
	UInt64(key string, value uint64) *entity.LoggerFiled
	Float64(key string, value float64) *entity.LoggerFiled
	Bool(key string, value bool) *entity.LoggerFiled
	Any(key string, value interface{}) *entity.LoggerFiled
	Close()
}
