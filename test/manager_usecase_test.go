package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jvdiamondtech/ms-identity-cat/internal/adapter/usecase"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"
)

// 資料庫模擬
type MockManagerRepository struct {
	mock.Mock
}

func (m *MockManagerRepository) FindByID(ctx context.Context, id uint64) (*model.Manager, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Manager), args.Error(1)
}

func (m *MockManagerRepository) FindByGlobalID(ctx context.Context, globalID string) (*model.Manager, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Manager), args.Error(1)
}

func (m *MockManagerRepository) Create(ctx context.Context, manager *model.Manager) error {
	args := m.Called(ctx, manager)
	manager.ID = 1 // 為新創建的管理員設置 ID
	return args.Error(0)
}

func (m *MockManagerRepository) Update(ctx context.Context, manager *model.Manager) error {
	args := m.Called(ctx, manager)
	return args.Error(0)
}

func (m *MockManagerRepository) Delete(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// 測試 SyncManager 方法 - 創建新管理員
func TestManagerUseCase_SyncManager_Create(t *testing.T) {
	// 準備測試數據
	globalMerchantID := "FATCAT-MERCHANT-1"
	globalManagerID := "FATCAT-MANAGER-231"
	managerAccount := "testmanager"
	managerEmail := "manager@example.com"

	// 創建模擬資料庫
	managerRepo := new(MockManagerRepository)
	managerRepo.On("FindByGlobalID", mock.Anything, globalManagerID).Return(nil, fmt.Errorf("record not found"))
	managerRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Manager")).Return(nil)

	merchantRepo := new(MockMerchantRepository)
	merchantRepo.On("FindByGlobalID", mock.Anything, globalMerchantID).Return(&model.Merchant{
		ID:               1,
		GlobalMerchantID: globalMerchantID,
		Name:             "Test Merchant",
		DisplayName:      "Test Merchant Display",
		APIKey:           "merchant-api-key",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}, nil)

	// 創建模擬事件生產者
	eventProducer := new(MockEventProducer)
	eventProducer.On("PublishManagerSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).Return(nil)

	// 創建記錄器
	logger := zaptest.NewLogger(t)

	// 創建用例
	useCase := usecase.NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// 創建測試事件
	managerEvent := event.ManagerSyncEvent{
		GlobalMerchantID: globalMerchantID,
		Manager: event.ManagerData{
			ID:              231,
			GlobalManagerID: globalManagerID,
			Account:         managerAccount,
			Email:           managerEmail,
		},
	}

	cloudEvent := event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "tw.jvd.fatcat.manager.sync.v1",
		Source:          "/fatcat/FATCAT",
		Subject:         "manager_sync",
		ID:              uuid.New().String(),
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     "00-3119f4c8eac427ec128029e5b6d056a4-ddfb98d07cce2587-01",
		Data:            managerEvent,
	}

	// 序列化事件
	eventData, err := json.Marshal(cloudEvent)
	assert.NoError(t, err)

	// 執行測試
	err = useCase.SyncManager(context.Background(), eventData)

	// 驗證結果
	assert.NoError(t, err)
	managerRepo.AssertExpectations(t)
	merchantRepo.AssertExpectations(t)
	eventProducer.AssertExpectations(t)
}

// 測試 SyncManager 方法 - 更新現有管理員
func TestManagerUseCase_SyncManager_Update(t *testing.T) {
	// 準備測試數據
	globalMerchantID := "FATCAT-MERCHANT-1"
	globalManagerID := "FATCAT-MANAGER-231"
	managerAccount := "testmanager"
	managerEmail := "updated@example.com" // 更新的郵箱
	oldEmail := "old@example.com"
	// 準備現有管理員
	existingManager := &model.Manager{
		ID:              1,
		MerchantID:      1,
		GlobalManagerID: globalManagerID,
		Account:         "oldaccount",
		Email:           &oldEmail,
		CreatedAt:       time.Now().Add(-24 * time.Hour),
		UpdatedAt:       time.Now().Add(-24 * time.Hour),
	}

	// 創建模擬資料庫
	managerRepo := new(MockManagerRepository)
	managerRepo.On("FindByGlobalID", mock.Anything, globalManagerID).Return(existingManager, nil)
	managerRepo.On("Update", mock.Anything, mock.AnythingOfType("*model.Manager")).Return(nil)

	merchantRepo := new(MockMerchantRepository)
	merchantRepo.On("FindByGlobalID", mock.Anything, globalMerchantID).Return(&model.Merchant{
		ID:               1,
		GlobalMerchantID: globalMerchantID,
		Name:             "Test Merchant",
		DisplayName:      "Test Merchant Display",
		APIKey:           "merchant-api-key",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}, nil)

	// 創建模擬事件生產者
	eventProducer := new(MockEventProducer)
	eventProducer.On("PublishManagerSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).Return(nil)

	// 創建記錄器
	logger := zaptest.NewLogger(t)

	// 創建用例
	useCase := usecase.NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// 創建測試事件
	managerEvent := event.ManagerSyncEvent{
		GlobalMerchantID: globalMerchantID,
		Manager: event.ManagerData{
			ID:              231,
			GlobalManagerID: globalManagerID,
			Account:         managerAccount,
			Email:           managerEmail,
		},
	}

	cloudEvent := event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "tw.jvd.fatcat.manager.sync.v1",
		Source:          "/fatcat/FATCAT",
		Subject:         "manager_sync",
		ID:              uuid.New().String(),
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     "00-3119f4c8eac427ec128029e5b6d056a4-ddfb98d07cce2587-01",
		Data:            managerEvent,
	}

	// 序列化事件
	eventData, err := json.Marshal(cloudEvent)
	assert.NoError(t, err)

	// 執行測試
	err = useCase.SyncManager(context.Background(), eventData)

	// 驗證結果
	assert.NoError(t, err)
	managerRepo.AssertExpectations(t)
	merchantRepo.AssertExpectations(t)
	eventProducer.AssertExpectations(t)

	// 驗證更新後的數據
	updated := managerRepo.Calls[1].Arguments.Get(1).(*model.Manager)
	assert.Equal(t, managerAccount, updated.Account)
	assert.Equal(t, managerEmail, *updated.Email)
}

// 測試 GetManagerByID 方法
func TestManagerUseCase_GetManagerByID(t *testing.T) {
	// 準備測試數據
	managerID := uint64(1)
	merchantID := uint64(2)
	globalManagerID := "FATCAT-MANAGER-231"
	managerAccount := "testmanager"
	managerEmail := "manager@example.com"

	// 準備現有管理員
	existingManager := &model.Manager{
		ID:              managerID,
		MerchantID:      merchantID,
		GlobalManagerID: globalManagerID,
		Account:         managerAccount,
		Email:           &managerEmail,
		CreatedAt:       time.Now().Add(-24 * time.Hour),
		UpdatedAt:       time.Now().Add(-24 * time.Hour),
	}

	// 創建模擬資料庫
	managerRepo := new(MockManagerRepository)
	managerRepo.On("FindByID", mock.Anything, managerID).Return(existingManager, nil)

	// 創建模擬商戶資料庫
	merchantRepo := new(MockMerchantRepository)

	// 創建模擬事件生產者
	eventProducer := new(MockEventProducer)

	// 創建記錄器
	logger := zaptest.NewLogger(t)

	// 創建用例
	useCase := usecase.NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// 執行測試
	manager, err := useCase.GetManagerByID(context.Background(), managerID)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, manager)
	assert.Equal(t, managerID, manager.ID)
	assert.Equal(t, merchantID, manager.MerchantID)
	assert.Equal(t, globalManagerID, manager.GlobalManagerID)
	assert.Equal(t, managerAccount, manager.Account)
	assert.Equal(t, managerEmail, *manager.Email)

	managerRepo.AssertExpectations(t)
}

// 測試 GetManagerByGlobalID 方法
func TestManagerUseCase_GetManagerByGlobalID(t *testing.T) {
	// 準備測試數據
	managerID := uint64(1)
	merchantID := uint64(2)
	globalManagerID := "FATCAT-MANAGER-231"
	managerAccount := "testmanager"
	managerEmail := "manager@example.com"

	// 準備現有管理員
	existingManager := &model.Manager{
		ID:              managerID,
		MerchantID:      merchantID,
		GlobalManagerID: globalManagerID,
		Account:         managerAccount,
		Email:           &managerEmail,
		CreatedAt:       time.Now().Add(-24 * time.Hour),
		UpdatedAt:       time.Now().Add(-24 * time.Hour),
	}

	// 創建模擬資料庫
	managerRepo := new(MockManagerRepository)
	managerRepo.On("FindByGlobalID", mock.Anything, globalManagerID).Return(existingManager, nil)

	// 創建模擬商戶資料庫
	merchantRepo := new(MockMerchantRepository)

	// 創建模擬事件生產者
	eventProducer := new(MockEventProducer)

	// 創建記錄器
	logger := zaptest.NewLogger(t)

	// 創建用例
	useCase := usecase.NewManagerUseCase(managerRepo, merchantRepo, eventProducer, logger)

	// 執行測試
	manager, err := useCase.GetManagerByGlobalID(context.Background(), globalManagerID)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, manager)
	assert.Equal(t, managerID, manager.ID)
	assert.Equal(t, merchantID, manager.MerchantID)
	assert.Equal(t, globalManagerID, manager.GlobalManagerID)
	assert.Equal(t, managerAccount, manager.Account)
	assert.Equal(t, managerEmail, *manager.Email)

	managerRepo.AssertExpectations(t)
}
