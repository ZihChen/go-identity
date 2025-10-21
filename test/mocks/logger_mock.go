package mocks

import (
	"context"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/stretchr/testify/mock"
)

type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) DebugWithContext(
	ctx context.Context,
	msg string,
	fields ...*entity.LoggerFiled,
) {
	m.Called(ctx, msg, fields)
}

func (m *MockLogger) InfoWithContext(
	ctx context.Context,
	msg string,
	fields ...*entity.LoggerFiled,
) {
	m.Called(ctx, msg, fields)
}

func (m *MockLogger) ErrorWithContext(
	ctx context.Context,
	msg string,
	fields ...*entity.LoggerFiled,
) {
	m.Called(ctx, msg, fields)
}

func (m *MockLogger) WarnWithContext(
	ctx context.Context,
	msg string,
	fields ...*entity.LoggerFiled,
) {
	m.Called(ctx, msg, fields)
}

func (m *MockLogger) FatalWithContext(
	ctx context.Context,
	msg string,
	fields ...*entity.LoggerFiled,
) {
	m.Called(ctx, msg, fields)
}

func (m *MockLogger) DebugLog(msg string, fields ...*entity.LoggerFiled) {
	m.Called(msg, fields)
}

func (m *MockLogger) InfoLog(msg string, fields ...*entity.LoggerFiled) {
	m.Called(msg, fields)
}

func (m *MockLogger) ErrorLog(msg string, fields ...*entity.LoggerFiled) {
	m.Called(msg, fields)
}

func (m *MockLogger) WarnLog(msg string, fields ...*entity.LoggerFiled) {
	m.Called(msg, fields)
}

func (m *MockLogger) FatalLog(msg string, fields ...*entity.LoggerFiled) {
	m.Called(msg, fields)
}

func (m *MockLogger) Error(key string, value error) *entity.LoggerFiled {
	args := m.Called(key, value)
	return args.Get(0).(*entity.LoggerFiled)
}

func (m *MockLogger) String(key string, value string) *entity.LoggerFiled {
	args := m.Called(key, value)
	return args.Get(0).(*entity.LoggerFiled)
}

func (m *MockLogger) Int(key string, value int) *entity.LoggerFiled {
	args := m.Called(key, value)
	return args.Get(0).(*entity.LoggerFiled)
}

func (m *MockLogger) Int64(key string, value int64) *entity.LoggerFiled {
	args := m.Called(key, value)
	return args.Get(0).(*entity.LoggerFiled)
}

func (m *MockLogger) UInt64(key string, value uint64) *entity.LoggerFiled {
	args := m.Called(key, value)
	return args.Get(0).(*entity.LoggerFiled)
}

func (m *MockLogger) Float64(key string, value float64) *entity.LoggerFiled {
	args := m.Called(key, value)
	return args.Get(0).(*entity.LoggerFiled)
}

func (m *MockLogger) Bool(key string, value bool) *entity.LoggerFiled {
	args := m.Called(key, value)
	return args.Get(0).(*entity.LoggerFiled)
}

func (m *MockLogger) Any(key string, value interface{}) *entity.LoggerFiled {
	args := m.Called(key, value)
	return args.Get(0).(*entity.LoggerFiled)
}

func (m *MockLogger) Close() {
	m.Called()
}

// NewMockLogger 創建新的 Mock Logger 實例
func NewMockLogger() *MockLogger {
	logger := new(MockLogger)

	// 字段方法
	logger.On("Error",
		mock.AnythingOfType("string"),
		mock.MatchedBy(func(e interface{}) bool {
			_, ok := e.(error)
			return ok
		}),
	).Return(&entity.LoggerFiled{}).Maybe()

	logger.On("String",
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
	).Return(&entity.LoggerFiled{}).Maybe()

	logger.On("Int",
		mock.AnythingOfType("string"),
		mock.AnythingOfType("int"),
	).Return(&entity.LoggerFiled{}).Maybe()

	logger.On("Int64",
		mock.AnythingOfType("string"),
		mock.AnythingOfType("int64"),
	).Return(&entity.LoggerFiled{}).Maybe()

	logger.On("UInt64",
		mock.AnythingOfType("string"),
		mock.AnythingOfType("uint64"),
	).Return(&entity.LoggerFiled{}).Maybe()

	logger.On("Float64",
		mock.AnythingOfType("string"),
		mock.AnythingOfType("float64"),
	).Return(&entity.LoggerFiled{}).Maybe()

	logger.On("Bool",
		mock.AnythingOfType("string"),
		mock.AnythingOfType("bool"),
	).Return(&entity.LoggerFiled{}).Maybe()

	logger.On("Any",
		mock.AnythingOfType("string"),
		mock.Anything,
	).Return(&entity.LoggerFiled{}).Maybe()

	// Close 方法
	logger.On("Close").Return().Maybe()

	// Context相關的日誌方法
	logger.On("DebugWithContext",
		mock.Anything,                 // context
		mock.AnythingOfType("string"), // msg
		mock.Anything,                 // fields
	).Return().Maybe()

	logger.On("InfoWithContext",
		mock.Anything,
		mock.AnythingOfType("string"),
		mock.Anything,
	).Return().Maybe()

	logger.On("ErrorWithContext",
		mock.Anything,
		mock.AnythingOfType("string"),
		mock.Anything,
	).Return().Maybe()

	logger.On("WarnWithContext",
		mock.Anything,
		mock.AnythingOfType("string"),
		mock.Anything,
	).Return().Maybe()

	logger.On("FatalWithContext",
		mock.Anything,
		mock.AnythingOfType("string"),
		mock.Anything,
	).Return().Maybe()

	// 一般日誌方法
	logger.On("DebugLog",
		mock.AnythingOfType("string"),
		mock.Anything,
	).Return().Maybe()

	logger.On("InfoLog",
		mock.AnythingOfType("string"),
		mock.Anything,
	).Return().Maybe()

	logger.On("ErrorLog",
		mock.AnythingOfType("string"),
		mock.Anything,
	).Return().Maybe()

	logger.On("WarnLog",
		mock.AnythingOfType("string"),
		mock.Anything,
	).Return().Maybe()

	logger.On("FatalLog",
		mock.AnythingOfType("string"),
		mock.Anything,
	).Return().Maybe()

	return logger
}
