package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap/zaptest"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/model"
	"github.com/jvdiamondtech/ms-identity-cat/internal/usecase"
)

// 資料庫模擬
type MockPlayerRepository struct {
	mock.Mock
}

func (m *MockPlayerRepository) FindByID(ctx context.Context, id uint64) (*model.Player, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Player), args.Error(1)
}

func (m *MockPlayerRepository) FindByGlobalID(ctx context.Context, globalID string) (*model.Player, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Player), args.Error(1)
}

func (m *MockPlayerRepository) Create(ctx context.Context, player *model.Player) error {
	args := m.Called(ctx, player)
	player.ID = 1 // 為新創建的玩家設置 ID
	return args.Error(0)
}

func (m *MockPlayerRepository) Update(ctx context.Context, player *model.Player) error {
	args := m.Called(ctx, player)
	return args.Error(0)
}

func (m *MockPlayerRepository) Delete(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// 測試 SyncPlayer 方法 - 創建新玩家
func TestPlayerUseCase_SyncPlayer_Create(t *testing.T) {
	// 準備測試數據
	globalMerchantID := "FATCAT-MERCHANT-1"
	globalPlayerID := "FATCAT-PLAYER-7241"
	playerAccount := "testplayer"
	playerEmail := "test@example.com"

	// 創建模擬資料庫
	playerRepo := new(MockPlayerRepository)
	playerRepo.On("FindByGlobalID", mock.Anything, globalPlayerID).Return(nil, fmt.Errorf("record not found"))
	playerRepo.On("Create", mock.Anything, mock.AnythingOfType("*model.Player")).Return(nil)

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
	eventProducer.On("PublishPlayerSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).Return(nil)

	// 創建記錄器
	logger := zaptest.NewLogger(t)

	// 創建用例
	useCase := usecase.NewPlayerUseCase(playerRepo, merchantRepo, eventProducer, logger)

	// 創建測試事件
	playerEvent := event.PlayerSyncEvent{
		GlobalMerchantID: globalMerchantID,
		Player: event.PlayerData{
			GlobalPlayerID: globalPlayerID,
			Account:        playerAccount,
			Email:          playerEmail,
			Status:         "active",
		},
	}

	cloudEvent := event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "tw.jvd.fatcat.player.sync.v1",
		Source:          "/fatcat/FATCAT",
		Subject:         "player_sync",
		ID:              uuid.New().String(),
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     "00-3119f4c8eac427ec128029e5b6d056a4-ddfb98d07cce2587-01",
		Data:            playerEvent,
	}

	// 序列化事件
	eventData, err := json.Marshal(cloudEvent)
	assert.NoError(t, err)

	// 執行測試
	err = useCase.SyncPlayer(context.Background(), eventData)

	// 驗證結果
	assert.NoError(t, err)
	playerRepo.AssertExpectations(t)
	merchantRepo.AssertExpectations(t)
	eventProducer.AssertExpectations(t)
}

// 測試 SyncPlayer 方法 - 更新現有玩家
func TestPlayerUseCase_SyncPlayer_Update(t *testing.T) {
	// 準備測試數據
	globalMerchantID := "FATCAT-MERCHANT-1"
	globalPlayerID := "FATCAT-PLAYER-7241"
	playerAccount := "testplayer"
	playerEmail := "updated@example.com" // 更新的郵箱
	apiKey := "test-api-key"
	oldEmail := "old@example.com"

	// 準備現有玩家
	existingPlayer := &model.Player{
		ID:             1,
		MerchantID:     1,
		GlobalPlayerID: globalPlayerID,
		APIKey:         apiKey,
		Account:        "oldaccount",
		Email:          &oldEmail,
		CreatedAt:      time.Now().Add(-24 * time.Hour),
		UpdatedAt:      time.Now().Add(-24 * time.Hour),
	}

	// 創建模擬資料庫
	playerRepo := new(MockPlayerRepository)
	playerRepo.On("FindByGlobalID", mock.Anything, globalPlayerID).Return(existingPlayer, nil)
	playerRepo.On("Update", mock.Anything, mock.AnythingOfType("*model.Player")).Return(nil)

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
	eventProducer.On("PublishPlayerSync", mock.Anything, mock.AnythingOfType("*event.CloudEvent")).Return(nil)

	// 創建記錄器
	logger := zaptest.NewLogger(t)

	// 創建用例
	useCase := usecase.NewPlayerUseCase(playerRepo, merchantRepo, eventProducer, logger)

	// 創建測試事件
	playerEvent := event.PlayerSyncEvent{
		GlobalMerchantID: globalMerchantID,
		Player: event.PlayerData{
			GlobalPlayerID: globalPlayerID,
			Account:        playerAccount,
			Email:          playerEmail,
			Status:         "active",
		},
	}

	cloudEvent := event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "tw.jvd.fatcat.player.sync.v1",
		Source:          "/fatcat/FATCAT",
		Subject:         "player_sync",
		ID:              uuid.New().String(),
		Time:            time.Now(),
		DataContentType: "application/json",
		TraceParent:     "00-3119f4c8eac427ec128029e5b6d056a4-ddfb98d07cce2587-01",
		Data:            playerEvent,
	}

	// 序列化事件
	eventData, err := json.Marshal(cloudEvent)
	assert.NoError(t, err)

	// 執行測試
	err = useCase.SyncPlayer(context.Background(), eventData)

	// 驗證結果
	assert.NoError(t, err)
	playerRepo.AssertExpectations(t)
	merchantRepo.AssertExpectations(t)
	eventProducer.AssertExpectations(t)

	// 驗證更新時保留了原始的 API 密鑰
	updated := playerRepo.Calls[1].Arguments.Get(1).(*model.Player)
	assert.Equal(t, apiKey, updated.APIKey)
	assert.Equal(t, playerAccount, updated.Account)
	assert.Equal(t, playerEmail, *updated.Email)
}

// 測試 GetPlayerByID 方法
func TestPlayerUseCase_GetPlayerByID(t *testing.T) {
	// 準備測試數據
	playerID := uint64(1)
	merchantID := uint64(2)
	globalPlayerID := "FATCAT-PLAYER-7241"
	playerAccount := "testplayer"
	playerEmail := "test@example.com"
	apiKey := "test-api-key"

	// 準備現有玩家
	existingPlayer := &model.Player{
		ID:             playerID,
		MerchantID:     merchantID,
		GlobalPlayerID: globalPlayerID,
		APIKey:         apiKey,
		Account:        playerAccount,
		Email:          &playerEmail,
		CreatedAt:      time.Now().Add(-24 * time.Hour),
		UpdatedAt:      time.Now().Add(-24 * time.Hour),
	}

	// 創建模擬資料庫
	playerRepo := new(MockPlayerRepository)
	playerRepo.On("FindByID", mock.Anything, playerID).Return(existingPlayer, nil)

	// 創建模擬商戶資料庫
	merchantRepo := new(MockMerchantRepository)

	// 創建模擬事件生產者
	eventProducer := new(MockEventProducer)

	// 創建記錄器
	logger := zaptest.NewLogger(t)

	// 創建用例
	useCase := usecase.NewPlayerUseCase(playerRepo, merchantRepo, eventProducer, logger)

	// 執行測試
	player, err := useCase.GetPlayerByID(context.Background(), playerID)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, player)
	assert.Equal(t, playerID, player.ID)
	assert.Equal(t, merchantID, player.MerchantID)
	assert.Equal(t, globalPlayerID, player.GlobalPlayerID)
	assert.Equal(t, apiKey, player.APIKey)
	assert.Equal(t, playerAccount, player.Account)
	assert.Equal(t, playerEmail, *player.Email)

	playerRepo.AssertExpectations(t)
}

// 測試 GetPlayerByGlobalID 方法
func TestPlayerUseCase_GetPlayerByGlobalID(t *testing.T) {
	// 準備測試數據
	playerID := uint64(1)
	merchantID := uint64(2)
	globalPlayerID := "FATCAT-PLAYER-7241"
	playerAccount := "testplayer"
	playerEmail := "test@example.com"
	apiKey := "test-api-key"

	// 準備現有玩家
	existingPlayer := &model.Player{
		ID:             playerID,
		MerchantID:     merchantID,
		GlobalPlayerID: globalPlayerID,
		APIKey:         apiKey,
		Account:        playerAccount,
		Email:          &playerEmail,
		CreatedAt:      time.Now().Add(-24 * time.Hour),
		UpdatedAt:      time.Now().Add(-24 * time.Hour),
	}

	// 創建模擬資料庫
	playerRepo := new(MockPlayerRepository)
	playerRepo.On("FindByGlobalID", mock.Anything, globalPlayerID).Return(existingPlayer, nil)

	// 創建模擬商戶資料庫
	merchantRepo := new(MockMerchantRepository)

	// 創建模擬事件生產者
	eventProducer := new(MockEventProducer)

	// 創建記錄器
	logger := zaptest.NewLogger(t)

	// 創建用例
	useCase := usecase.NewPlayerUseCase(playerRepo, merchantRepo, eventProducer, logger)

	// 執行測試
	player, err := useCase.GetPlayerByGlobalID(context.Background(), globalPlayerID)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, player)
	assert.Equal(t, playerID, player.ID)
	assert.Equal(t, merchantID, player.MerchantID)
	assert.Equal(t, globalPlayerID, player.GlobalPlayerID)
	assert.Equal(t, apiKey, player.APIKey)
	assert.Equal(t, playerAccount, player.Account)
	assert.Equal(t, playerEmail, *player.Email)

	playerRepo.AssertExpectations(t)
}

// 測試 UpdatePlayerLastActive 方法
func TestPlayerUseCase_UpdatePlayerLastActive(t *testing.T) {
	// 準備測試數據
	playerID := uint64(1)
	merchantID := uint64(2)
	globalPlayerID := "FATCAT-PLAYER-7241"
	playerAccount := "testplayer"
	playerEmail := "test@example.com"
	apiKey := "test-api-key"

	// 準備現有玩家
	existingPlayer := &model.Player{
		ID:             playerID,
		MerchantID:     merchantID,
		GlobalPlayerID: globalPlayerID,
		APIKey:         apiKey,
		Account:        playerAccount,
		Email:          &playerEmail,
		LastActiveAt:   nil, // 初始為 nil
		CreatedAt:      time.Now().Add(-24 * time.Hour),
		UpdatedAt:      time.Now().Add(-24 * time.Hour),
	}

	// 創建模擬資料庫
	playerRepo := new(MockPlayerRepository)
	playerRepo.On("FindByID", mock.Anything, playerID).Return(existingPlayer, nil)
	playerRepo.On("Update", mock.Anything, mock.AnythingOfType("*model.Player")).Return(nil)

	// 創建模擬商戶資料庫
	merchantRepo := new(MockMerchantRepository)

	// 創建模擬事件生產者
	eventProducer := new(MockEventProducer)

	// 創建記錄器
	logger := zaptest.NewLogger(t)

	// 創建用例
	useCase := usecase.NewPlayerUseCase(playerRepo, merchantRepo, eventProducer, logger)

	// 執行測試
	err := useCase.UpdatePlayerLastActive(context.Background(), playerID)

	// 驗證結果
	assert.NoError(t, err)

	// 驗證最後活躍時間被更新
	updated := playerRepo.Calls[1].Arguments.Get(1).(*model.Player)
	assert.NotNil(t, updated.LastActiveAt)

	playerRepo.AssertExpectations(t)
}
