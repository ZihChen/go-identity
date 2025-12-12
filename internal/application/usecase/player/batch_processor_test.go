package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestPlayerBatchProcessor_BasicFlow(t *testing.T) {
	// 準備模擬依賴
	mockPlayerRepo := mocks.NewPlayerRepositoryMock(t)
	mockEventProducer := mocks.NewEventProducerMock(t)
	mockLogger := mocks.NewMockLogger(t)
	mockTracing := mocks.NewNilTracingService()

	// 設定模擬行為
	mockPlayerRepo.On("BatchUpsert", mock.Anything, mock.AnythingOfType("[]*entity.Player")).
		Return(nil)
	mockEventProducer.On("BatchPublishPlayerSync", mock.Anything,
		mock.AnythingOfType("[]*entity.Player"),
		mock.AnythingOfType("[]string")).Return(nil)
	mockLogger.On("InfoLog", mock.AnythingOfType("string"), mock.Anything).Return()
	mockLogger.On("ErrorLog", mock.AnythingOfType("string"), mock.Anything).Return()

	// 創建批次處理器
	processor := NewPlayerBatchProcessor(
		mockPlayerRepo,
		mockEventProducer,
		mockLogger,
		mockTracing,
	)

	// 啟動處理器
	ctx := context.Background()
	err := processor.Start(ctx)
	assert.NoError(t, err)

	// 準備測試玩家
	player := entity.NewPlayer(1, "test-global-id", "test-account", 1, nil)
	globalMerchantID := "test-merchant-id"

	// 提交玩家
	resultChan := processor.SubmitPlayer(player, globalMerchantID)

	// 等待結果
	select {
	case err := <-resultChan:
		assert.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for batch processing result")
	}

	// 停止處理器
	err = processor.Stop(ctx)
	assert.NoError(t, err)

	// 驗證模擬調用
	mockPlayerRepo.AssertExpectations()
	mockEventProducer.AssertExpectations()
}

func TestPlayerBatchProcessor_BatchSizeLimit(t *testing.T) {
	// 準備模擬依賴
	mockPlayerRepo := mocks.NewPlayerRepositoryMock(t)
	mockEventProducer := mocks.NewEventProducerMock(t)
	mockLogger := mocks.NewMockLogger(t)
	mockTracing := mocks.NewNilTracingService()

	// 設定模擬行為
	mockPlayerRepo.On("BatchUpsert", mock.Anything, mock.MatchedBy(func(players []*entity.Player) bool {
		return len(players) <= 500 // 檢查批次大小不超過配置值
	})).
		Return(nil)
	mockEventProducer.On("BatchPublishPlayerSync", mock.Anything,
		mock.AnythingOfType("[]*entity.Player"),
		mock.AnythingOfType("[]string")).Return(nil)
	mockLogger.On("InfoLog", mock.AnythingOfType("string"), mock.Anything).Return()
	mockLogger.On("ErrorLog", mock.AnythingOfType("string"), mock.Anything).Return()

	// 創建批次處理器
	processor := NewPlayerBatchProcessor(
		mockPlayerRepo,
		mockEventProducer,
		mockLogger,
		mockTracing,
	)

	// 設定較小的批次大小進行測試
	processor.batchSize = 3

	// 啟動處理器
	ctx := context.Background()
	err := processor.Start(ctx)
	assert.NoError(t, err)

	// 提交多個玩家
	resultChannels := make([]<-chan error, 5)
	for i := 0; i < 5; i++ {
		player := entity.NewPlayer(
			uint64(i+1),
			"test-global-id-"+string(rune(i+'1')),
			"test-account-"+string(rune(i+'1')),
			1,
			nil,
		)
		globalMerchantID := "test-merchant-id"
		resultChannels[i] = processor.SubmitPlayer(player, globalMerchantID)
	}

	// 等待所有結果
	for i, resultChan := range resultChannels {
		select {
		case err := <-resultChan:
			assert.NoError(t, err, "Player %d should process successfully", i)
		case <-time.After(10 * time.Second):
			t.Fatalf("Timeout waiting for player %d processing result", i)
		}
	}

	// 停止處理器
	err = processor.Stop(ctx)
	assert.NoError(t, err)

	// 驗證模擬調用
	mockPlayerRepo.AssertExpectations()
	mockEventProducer.AssertExpectations()
}
