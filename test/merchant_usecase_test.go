package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jvdiamondtech/ms-identity-cat/internal/adapter/usecase"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/model"
)

// 資料庫模擬
type MockMerchantRepository struct {
	mock.Mock
}

func (m *MockMerchantRepository) FindByID(ctx context.Context, id uint64) (*model.Merchant, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Merchant), args.Error(1)
}

func (m *MockMerchantRepository) FindByGlobalID(ctx context.Context, globalID string) (*model.Merchant, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Merchant), args.Error(1)
}

func (m *MockMerchantRepository) Create(ctx context.Context, merchant *model.Merchant) error {
	args := m.Called(ctx, merchant)
	merchant.ID = 1 // 為新創建的商戶設置 ID
	return args.Error(0)
}

func (m *MockMerchantRepository) Update(ctx context.Context, merchant *model.Merchant) error {
	args := m.Called(ctx, merchant)
	return args.Error(0)
}

func (m *MockMerchantRepository) Delete(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// 事件生產者模擬
type MockEventProducer struct {
	mock.Mock
}

func (m *MockEventProducer) PublishMerchantSync(ctx context.Context, event *event.CloudEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockEventProducer) PublishPlayerSync(ctx context.Context, event *event.CloudEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func (m *MockEventProducer) PublishManagerSync(ctx context.Context, event *event.CloudEvent) error {
	args := m.Called(ctx, event)
	return args.Error(0)
}

func TestMerchantUseCase_SyncMerchant_Create(t *testing.T) {
	// 準備測試數據
	globalMerchantID := "FATCAT-MERCHANT-1"
	merchantName := "TestMerchant"
	displayName := "Test Merchant"

	// 創建模擬資料庫
	merchantRepo := new(MockMerchantRepository)
	merchantRepo.On("FindByGlobalID", mock.Anything, globalMerchantID).Return(nil, fmt.Errorf("record not found"))
	merchantRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Merchant")).Return(nil)

	// 創建模擬事件生產者
	eventProducer := new(MockEventProducer)
	eventProducer.On("PublishMerchantSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).Return(nil)

	// 創建記錄器
	logger := zaptest.NewLogger(t)

	// 創建用例
	useCase := usecase.NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// 創建測試事件
	merchantEvent := event.MerchantSyncEvent{
		GlobalMerchantID: globalMerchantID,
		Merchant: event.MerchantData{
			ID:               1,
			Name:             merchantName,
			DisplayName:      displayName,
			GlobalMerchantID: globalMerchantID,
		},
	}

	cloudEvent := event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "tw.jvd.fatcat.merchant.sync.v1",
		Source:          "/fatcat/FATCAT",
		Subject:         "merchant_sync",
		ID:              uuid.New().String(),
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     "00-3119f4c8eac427ec128029e5b6d056a4-ddfb98d07cce2587-01",
		Data:            merchantEvent,
	}

	// 序列化事件
	eventData, err := json.Marshal(cloudEvent)
	assert.NoError(t, err)

	// 執行測試
	err = useCase.SyncMerchant(context.Background(), eventData)

	// 驗證結果
	assert.NoError(t, err)
	merchantRepo.AssertExpectations(t)
	eventProducer.AssertExpectations(t)
}

func TestMerchantUseCase_SyncMerchant_Update(t *testing.T) {
	// 準備測試數據
	globalMerchantID := "FATCAT-MERCHANT-1"
	merchantName := "TestMerchant"
	displayName := "Test Merchant"
	apiKey := "test-api-key"

	// 準備現有商戶
	existingMerchant := &model.Merchant{
		ID:               1,
		GlobalMerchantID: globalMerchantID,
		Name:             "OldName",
		DisplayName:      "Old Display Name",
		APIKey:           apiKey,
		CreatedAt:        time.Now().Add(-24 * time.Hour),
		UpdatedAt:        time.Now().Add(-24 * time.Hour),
	}

	// 創建模擬資料庫
	merchantRepo := new(MockMerchantRepository)
	merchantRepo.On("FindByGlobalID", mock.Anything, globalMerchantID).Return(existingMerchant, nil)
	merchantRepo.On("Update", mock.Anything, mock.AnythingOfType("*model.Merchant")).Return(nil)

	// 創建模擬事件生產者
	eventProducer := new(MockEventProducer)
	eventProducer.On("PublishMerchantSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).Return(nil)

	// 創建記錄器
	logger := zaptest.NewLogger(t)

	// 創建用例
	useCase := usecase.NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// 創建測試事件
	merchantEvent := event.MerchantSyncEvent{
		GlobalMerchantID: globalMerchantID,
		Merchant: event.MerchantData{
			ID:               1,
			Name:             merchantName,
			DisplayName:      displayName,
			GlobalMerchantID: globalMerchantID,
		},
	}

	cloudEvent := event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "tw.jvd.fatcat.merchant.sync.v1",
		Source:          "/fatcat/FATCAT",
		Subject:         "merchant_sync",
		ID:              uuid.New().String(),
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     "00-3119f4c8eac427ec128029e5b6d056a4-ddfb98d07cce2587-01",
		Data:            merchantEvent,
	}

	// 序列化事件
	eventData, err := json.Marshal(cloudEvent)
	assert.NoError(t, err)

	// 執行測試
	err = useCase.SyncMerchant(context.Background(), eventData)

	// 驗證結果
	assert.NoError(t, err)
	merchantRepo.AssertExpectations(t)
	eventProducer.AssertExpectations(t)

	// 驗證更新時保留了原始的 API 密鑰
	updated := merchantRepo.Calls[1].Arguments.Get(1).(*model.Merchant)
	assert.Equal(t, apiKey, updated.APIKey)
	assert.Equal(t, merchantName, updated.Name)
	assert.Equal(t, displayName, updated.DisplayName)
}

func TestMerchantUseCase_GetMerchantByID(t *testing.T) {
	// 準備測試數據
	merchantID := uint64(1)
	globalMerchantID := "FATCAT-MERCHANT-1"
	merchantName := "TestMerchant"
	displayName := "Test Merchant"
	apiKey := "test-api-key"

	// 準備現有商戶
	existingMerchant := &model.Merchant{
		ID:               merchantID,
		GlobalMerchantID: globalMerchantID,
		Name:             merchantName,
		DisplayName:      displayName,
		APIKey:           apiKey,
		CreatedAt:        time.Now().Add(-24 * time.Hour),
		UpdatedAt:        time.Now().Add(-24 * time.Hour),
	}

	// 創建模擬資料庫
	merchantRepo := new(MockMerchantRepository)
	merchantRepo.On("FindByID", mock.Anything, merchantID).Return(existingMerchant, nil)

	// 創建模擬事件生產者
	eventProducer := new(MockEventProducer)

	// 創建記錄器
	logger := zaptest.NewLogger(t)

	// 創建用例
	useCase := usecase.NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// 執行測試
	merchant, err := useCase.GetMerchantByID(context.Background(), merchantID)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, merchant)
	assert.Equal(t, merchantID, merchant.ID)
	assert.Equal(t, globalMerchantID, merchant.GlobalMerchantID)
	assert.Equal(t, merchantName, merchant.Name)
	assert.Equal(t, displayName, merchant.DisplayName)
	assert.Equal(t, apiKey, merchant.APIKey)

	merchantRepo.AssertExpectations(t)
}

func TestMerchantUseCase_GetMerchantByGlobalID(t *testing.T) {
	// 準備測試數據
	merchantID := uint64(1)
	globalMerchantID := "FATCAT-MERCHANT-1"
	merchantName := "TestMerchant"
	displayName := "Test Merchant"
	apiKey := "test-api-key"

	// 準備現有商戶
	existingMerchant := &model.Merchant{
		ID:               merchantID,
		GlobalMerchantID: globalMerchantID,
		Name:             merchantName,
		DisplayName:      displayName,
		APIKey:           apiKey,
		CreatedAt:        time.Now().Add(-24 * time.Hour),
		UpdatedAt:        time.Now().Add(-24 * time.Hour),
	}

	// 創建模擬資料庫
	merchantRepo := new(MockMerchantRepository)
	merchantRepo.On("FindByGlobalID", mock.Anything, globalMerchantID).Return(existingMerchant, nil)

	// 創建模擬事件生產者
	eventProducer := new(MockEventProducer)

	// 創建記錄器
	logger := zaptest.NewLogger(t)

	// 創建用例
	useCase := usecase.NewMerchantUseCase(merchantRepo, eventProducer, logger)

	// 執行測試
	merchant, err := useCase.GetMerchantByGlobalID(context.Background(), globalMerchantID)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, merchant)
	assert.Equal(t, merchantID, merchant.ID)
	assert.Equal(t, globalMerchantID, merchant.GlobalMerchantID)
	assert.Equal(t, merchantName, merchant.Name)
	assert.Equal(t, displayName, merchant.DisplayName)
	assert.Equal(t, apiKey, merchant.APIKey)

	merchantRepo.AssertExpectations(t)
}
