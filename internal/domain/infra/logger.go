package infra

import (
	"context"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/model"
)

type Logger interface {
	DebugWithContext(ctx context.Context, msg string, fields ...*model.LoggerFiled)
	InfoWithContext(ctx context.Context, msg string, fields ...*model.LoggerFiled)
	ErrorWithContext(ctx context.Context, msg string, fields ...*model.LoggerFiled)
	WarnWithContext(ctx context.Context, msg string, fields ...*model.LoggerFiled)
	FatalWithContext(ctx context.Context, msg string, fields ...*model.LoggerFiled)
	DebugLog(msg string, fields ...*model.LoggerFiled)
	InfoLog(msg string, fields ...*model.LoggerFiled)
	ErrorLog(msg string, fields ...*model.LoggerFiled)
	WarnLog(msg string, fields ...*model.LoggerFiled)
	FatalLog(msg string, fields ...*model.LoggerFiled)
	Error(key string, value error) *model.LoggerFiled
	String(key string, value string) *model.LoggerFiled
	Int(key string, value int) *model.LoggerFiled
	Int64(key string, value int64) *model.LoggerFiled
	UInt64(key string, value uint64) *model.LoggerFiled
	Float64(key string, value float64) *model.LoggerFiled
	Bool(key string, value bool) *model.LoggerFiled
	Any(key string, value interface{}) *model.LoggerFiled
	Close()
}
