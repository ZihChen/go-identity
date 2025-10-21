# 測試需求模板 - Fat Identity Cat

## 基本資訊

- **專案**: Fat Identity Cat
- **模組名稱**: [Merchant / Player / Manager / Level / Tag]
- **測試類型**: [Repository單元測試 / UseCase單元測試 / Handler單元測試 / Consumer單元測試 / Worker單元測試]
- **建立日期**: [YYYY-MM-DD]
- **覆蓋率目標**: ≥ 80%

## 測試架構

### Logger Mock 標準規則

**統一使用 `test/factories` 創建測試資料**
**統一使用 `test/mocks` 創建測試物件**

```go
// ✅ 正確做法
func createMockDependencies(t *testing.T) (*MockRepository, *mocks.MockLogger) {
    repo := mocks.NewRepositoryMock(t)
    logger := mocks.NewMockLogger(t)  // 統一使用此函數
    return repo, logger
}
```

**重要規則：**
1. **統一調用**: 所有測試都必須使用 `mocks.NewMockLogger(t)` 創建 Logger Mock
2. **無需驗證**: **不要** 使用 `logger.AssertExpectations(t)` 進行驗證
3. **獨立 Mock**: 每個 logger 方法都已獨立 mock，支援所有日誌級別和字段方法
4. **自動設置**: 函數已預設所有必要的 Mock 期望值

**支援的方法：**
- **日誌方法**: `DebugLog`, `InfoLog`, `ErrorLog`, `WarnLog`, `FatalLog`
- **Context 方法**: `DebugWithContext`, `InfoWithContext`, `ErrorWithContext`, `WarnWithContext`, `FatalWithContext`
- **字段方法**: `Error()`, `String()`, `Int()`, `Int64()`, `UInt64()`, `Float64()`, `Bool()`, `Any()`
- **生命週期**: `Close()`

### Repository 層測試

#### 核心實體 Repository 測試
針對 Fat Identity Cat 的主要實體進行測試：

```go
// 測試結構體
type [Module]TestCase struct {
    name           string
    setupMock      func(sqlmock.Sqlmock)
    input          interface{}
    expected[Entity] *entity.[Entity]
    expectedError  error
}

// Mock 資料庫設置
func setup[Module]MockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
    mockDB, mock, err := sqlmock.New()
    require.NoError(t, err)
    
    dialector := mysql.New(mysql.Config{
        Conn:                      mockDB,
        SkipInitializeWithVersion: true,
    })
    
    db, err := gorm.Open(dialector, &gorm.Config{})
    require.NoError(t, err)
    
    return db, mock, mockDB
}
```

#### 身份管理特有測試案例

```go
// Merchant Repository 測試
func TestMerchantRepository_FindByGlobalID(t *testing.T) {
    testCases := []MerchantTestCase{
        {
            name: "merchant found by global ID",
            setupMock: func(mock sqlmock.Sqlmock) {
                rows := sqlmock.NewRows([]string{"id", "global_id", "name", "api_key"}).
                    AddRow(1, "FATCAT-MERCHANT-001", "Test Merchant", "test-api-key")
                mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `merchants` WHERE global_id = ?")).
                    WithArgs("FATCAT-MERCHANT-001").
                    WillReturnRows(rows)
            },
            expectedMerchant: &entity.Merchant{
                ID:       1,
                GlobalID: "FATCAT-MERCHANT-001",
                Name:     "Test Merchant",
                APIKey:   "test-api-key",
            },
        },
        {
            name: "merchant not found",
            setupMock: func(mock sqlmock.Sqlmock) {
                mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `merchants` WHERE global_id = ?")).
                    WithArgs("NONEXISTENT").
                    WillReturnError(gorm.ErrRecordNotFound)
            },
            expectedError: errmsg.ErrRepoMerchantNotFound,
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            db, mock, sqlDB := setupMerchantMockDB(t)
            defer sqlDB.Close()
            
            tc.setupMock(mock)
            repo := NewMerchantRepository(db)
            
            result, err := repo.FindByGlobalID(context.Background(), tc.input.(string))
            
            if tc.expectedError != nil {
                assert.Error(t, err)
                assert.ErrorIs(t, err, tc.expectedError)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tc.expectedMerchant.GlobalID, result.GlobalID)
            }
            assert.NoError(t, mock.ExpectationsWereMet())
        })
    }
}

// Player Repository 測試（包含 Tag 關聯）
func TestPlayerRepository_FindWithTags(t *testing.T) {
    testCases := []PlayerTestCase{
        {
            name: "player found with tags",
            setupMock: func(mock sqlmock.Sqlmock) {
                // Player 查詢
                playerRows := sqlmock.NewRows([]string{"id", "global_id", "level", "last_active_at"}).
                    AddRow(1, "FATCAT-PLAYER-001", 5, time.Now())
                mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `players` WHERE id = ?")).
                    WithArgs(uint64(1)).
                    WillReturnRows(playerRows)
                
                // PlayerTag 關聯查詢
                tagRows := sqlmock.NewRows([]string{"tag_id", "tag_name"}).
                    AddRow(1, "VIP").
                    AddRow(2, "Active")
                mock.ExpectQuery(regexp.QuoteMeta("SELECT t.id as tag_id, t.name as tag_name FROM `player_tags` pt INNER JOIN `tags` t")).
                    WithArgs(uint64(1)).
                    WillReturnRows(tagRows)
            },
            expectedPlayer: &entity.Player{
                ID:           1,
                GlobalID:     "FATCAT-PLAYER-001",
                Level:        5,
                Tags:         []entity.Tag{{ID: 1, Name: "VIP"}, {ID: 2, Name: "Active"}},
            },
        },
    }
    
    // 測試實現...
}
```

### UseCase 層測試

#### 身份同步 UseCase 測試

```go
// Mock Repository 設計
type MockMerchantRepository struct {
    mock.Mock
}

func (m *MockMerchantRepository) FindByGlobalID(ctx context.Context, globalID string) (*entity.Merchant, error) {
    args := m.Called(ctx, globalID)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*entity.Merchant), args.Error(1)
}

func (m *MockMerchantRepository) Upsert(ctx context.Context, merchant *entity.Merchant) error {
    args := m.Called(ctx, merchant)
    return args.Error(0)
}

// 依賴創建函數
func createMockDependencies(t *testing.T) (*MockMerchantRepository, *helper.MockLogger, *redis.Client) {
    repo := new(MockMerchantRepository)
    logger := helper.SetupLoggerMock(t)
    redisClient, _ := redismock.NewClientMock()
    return repo, logger, redisClient
}

// 同步業務邏輯測試
func TestMerchantUseCase_SyncMerchant(t *testing.T) {
    // 準備測試資料
    merchantEvent := &event.MerchantEvent{
        GlobalMerchantID: "FATCAT-MERCHANT-001",
        Name:            "Updated Merchant Name",
        APIKey:          "updated-api-key",
        Status:          "active",
    }
    
    // 創建 Mock 依賴
    repo, logger, redisClient := createMockDependencies(t)
    
    // 設置 Mock 預期 - 新商戶同步
    repo.On("FindByGlobalID", mock.Anything, "FATCAT-MERCHANT-001").
        Return(nil, errmsg.ErrRepoMerchantNotFound)
    repo.On("Upsert", mock.Anything, mock.MatchedBy(func(merchant *entity.Merchant) bool {
        return merchant.GlobalID == "FATCAT-MERCHANT-001" && merchant.Name == "Updated Merchant Name"
    })).Return(nil)
    
    // 創建 UseCase
    useCase := NewMerchantUseCase(repo, logger)
    
    // 執行測試
    err := useCase.SyncMerchant(context.Background(), merchantEvent)
    
    // 驗證結果
    assert.NoError(t, err)
    
    // 驗證 Mock 調用
    repo.AssertExpectations(t)
    // 注意：不需要驗證 logger.AssertExpectations(t)
}

// 玩家最後活躍時間更新測試
func TestPlayerUseCase_UpdateLastActive(t *testing.T) {
    testCases := []struct {
        name          string
        globalPlayerID string
        setupMock     func(*MockPlayerRepository)
        expectedError error
    }{
        {
            name:          "update success",
            globalPlayerID: "FATCAT-PLAYER-001",
            setupMock: func(repo *MockPlayerRepository) {
                repo.On("UpdateLastActiveByGlobalID", mock.Anything, "FATCAT-PLAYER-001").
                    Return(nil)
            },
        },
        {
            name:          "player not found",
            globalPlayerID: "NONEXISTENT-PLAYER",
            setupMock: func(repo *MockPlayerRepository) {
                repo.On("UpdateLastActiveByGlobalID", mock.Anything, "NONEXISTENT-PLAYER").
                    Return(errmsg.ErrRepoPlayerNotFound)
            },
            expectedError: errmsg.ErrRepoPlayerNotFound,
        },
    }
    
    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            repo, logger, _ := createMockDependencies(t)
            tc.setupMock(repo)
            
            useCase := NewPlayerUseCase(repo, logger)
            
            err := useCase.UpdateLastActive(context.Background(), tc.globalPlayerID)
            
            if tc.expectedError != nil {
                assert.Error(t, err)
                assert.ErrorIs(t, err, tc.expectedError)
            } else {
                assert.NoError(t, err)
            }
            
            repo.AssertExpectations(t)
        })
    }
}
```

### Handler 層測試

#### HTTP API 測試

```go
// Mock UseCase
type MockMerchantUseCase struct {
    mock.Mock
}

func (m *MockMerchantUseCase) GetMerchantByID(ctx context.Context, id uint64) (*dto.MerchantResponse, error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*dto.MerchantResponse), args.Error(1)
}

func (m *MockMerchantUseCase) GetMerchantByGlobalID(ctx context.Context, globalID string) (*dto.MerchantResponse, error) {
    args := m.Called(ctx, globalID)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*dto.MerchantResponse), args.Error(1)
}

// 測試環境設置
func setupTestRouter() *gin.Engine {
    gin.SetMode(gin.TestMode)
    router := gin.New()
    // 添加錯誤處理中間件來捕獲 ResponseSentError panic
    router.Use(middleware.ErrorHandler())
    return router
}

// 創建 Mock 依賴
func createMockDependencies(t *testing.T) (*MockMerchantUseCase, *helper.MockLogger) {
    useCase := new(MockMerchantUseCase)
    logger := helper.SetupLoggerMock(t)
    return useCase, logger
}

// 身份管理 API 測試
func TestHTTPHandler_GetMerchantByGlobalID_Success(t *testing.T) {
    // Setup
    useCase, logger := createMockDependencies(t)
    handler := createTestHandler(useCase, logger)
    
    router := setupTestRouter()
    router.GET("/api/v1/merchants/global/:global_id", handler.GetMerchantByGlobalID)
    
    // 準備測試資料
    testData := createMerchantTestData()
    useCase.On("GetMerchantByGlobalID", mock.Anything, "FATCAT-MERCHANT-001").
        Return(testData, nil)
    
    // Execute
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("GET", "/api/v1/merchants/global/FATCAT-MERCHANT-001", nil)
    router.ServeHTTP(w, req)
    
    // Verify
    assert.Equal(t, http.StatusOK, w.Code)
    
    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    assert.NoError(t, err)
    assert.Equal(t, true, response["success"])
    assert.Contains(t, response, "data")
    
    // 驗證 merchant 特定字段
    data := response["data"].(map[string]interface{})
    assert.Equal(t, "FATCAT-MERCHANT-001", data["global_id"])
    assert.Equal(t, "Test Merchant", data["name"])
    
    useCase.AssertExpectations(t)
}

// 玩家最後活躍時間更新 API 測試
func TestHTTPHandler_UpdatePlayerLastActive_Success(t *testing.T) {
    // Setup
    useCase, logger := createMockDependencies(t)
    handler := createTestHandler(useCase, logger)
    
    router := setupTestRouter()
    router.PUT("/api/v1/players/:id/active", handler.UpdatePlayerLastActive)
    
    useCase.On("UpdateLastActive", mock.Anything, "FATCAT-PLAYER-001").
        Return(nil)
    
    // Execute
    w := httptest.NewRecorder()
    req, _ := http.NewRequest("PUT", "/api/v1/players/FATCAT-PLAYER-001/active", nil)
    router.ServeHTTP(w, req)
    
    // Verify
    assert.Equal(t, http.StatusOK, w.Code)
    
    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    assert.NoError(t, err)
    assert.Equal(t, true, response["success"])
    
    useCase.AssertExpectations(t)
}
```

### Consumer 和 Worker 層測試

#### KDS Consumer 測試

```go
// Mock KDS Service
type MockKDSService struct {
    mock.Mock
}

func (m *MockKDSService) ConsumeEvents(ctx context.Context, shardID string) error {
    args := m.Called(ctx, shardID)
    return args.Error(0)
}

// Consumer Handler 測試
func TestConsumerHandler_ProcessMerchantEvent(t *testing.T) {
    // Setup
    merchantUseCase, logger := createMockDependencies(t)
    handler := NewConsumerHandler(merchantUseCase, logger)
    
    // 準備測試 Event
    merchantEvent := &event.MerchantEvent{
        GlobalMerchantID: "FATCAT-MERCHANT-001",
        Name:            "Test Merchant",
        APIKey:          "test-api-key",
        Status:          "active",
    }
    
    merchantUseCase.On("SyncMerchant", mock.Anything, merchantEvent).
        Return(nil)
    
    // Execute
    err := handler.ProcessMerchantEvent(context.Background(), merchantEvent)
    
    // Verify
    assert.NoError(t, err)
    merchantUseCase.AssertExpectations(t)
}

// Worker 測試
func TestWorkerHandler_ProcessIdentitySync(t *testing.T) {
    // Setup
    playerUseCase, logger := createMockDependencies(t)
    handler := NewWorkerHandler(playerUseCase, logger)
    
    // 準備測試任務
    syncTask := &dto.IdentitySyncTask{
        Type:     "player_sync",
        EntityID: "FATCAT-PLAYER-001",
        Action:   "update_last_active",
    }
    
    playerUseCase.On("UpdateLastActive", mock.Anything, "FATCAT-PLAYER-001").
        Return(nil)
    
    // Execute
    err := handler.ProcessIdentitySync(context.Background(), syncTask)
    
    // Verify
    assert.NoError(t, err)
    playerUseCase.AssertExpectations(t)
}
```

## 測試案例設計

### Fat Identity Cat 特有測試案例

#### 核心身份實體測試

```go
// Repository 層測試方法
func TestMerchantRepository_Methods(t *testing.T) {
    // 必須測試的方法：
    // - FindByID
    // - FindByGlobalID  
    // - Create
    // - Update
    // - Upsert
    // - FindByAPIKey (商戶特有)
}

func TestPlayerRepository_Methods(t *testing.T) {
    // 必須測試的方法：
    // - FindByID
    // - FindByGlobalID
    // - UpdateLastActiveByGlobalID (玩家特有)
    // - FindWithTags (關聯查詢)
    // - AddTag / RemoveTag (標籤管理)
}

func TestManagerRepository_Methods(t *testing.T) {
    // 必須測試的方法：
    // - FindByID
    // - FindByGlobalID
    // - FindByMerchantID (管理員特有)
}

func TestLevelRepository_Methods(t *testing.T) {
    // 必須測試的方法：
    // - FindByID
    // - FindByLevel (等級查詢)
    // - CreateOrUpdate
}

func TestTagRepository_Methods(t *testing.T) {
    // 必須測試的方法：
    // - FindByID
    // - FindByName
    // - FindByMerchantID (商戶標籤)
    // - CreateOrUpdate
}
```

#### UseCase 業務邏輯測試

```go
// 身份同步 UseCase
func TestIdentitySyncUseCases(t *testing.T) {
    // SyncMerchant - 商戶資料同步
    // SyncPlayer - 玩家資料同步  
    // SyncManager - 管理員資料同步
    // UpdatePlayerLastActive - 玩家活躍度更新
    // AssignPlayerTag - 玩家標籤分配
    // UpdatePlayerLevel - 玩家等級更新
}
```

#### Handler API 測試

```go
// Fat Identity Cat API 端點
func TestIdentityAPIs(t *testing.T) {
    // Merchant APIs
    // GET /api/v1/merchants/:id
    // GET /api/v1/merchants/global/:global_id
    
    // Player APIs  
    // GET /api/v1/players/:id
    // GET /api/v1/players/global/:global_id
    // PUT /api/v1/players/:id/active
    
    // Manager APIs
    // GET /api/v1/managers/:id
    // GET /api/v1/managers/global/:global_id
    
    // System APIs
    // GET /health
}
```

## 測試執行

### Fat Identity Cat 特定測試命令

```bash
# Repository 測試
go test ./internal/adapter/outbound/repository/merchant/
go test ./internal/adapter/outbound/repository/player/
go test ./internal/adapter/outbound/repository/manager/
go test ./internal/adapter/outbound/repository/level/
go test ./internal/adapter/outbound/repository/tag/

# UseCase 測試  
go test ./internal/application/usecase/merchant/
go test ./internal/application/usecase/player/
go test ./internal/application/usecase/manager/

# Handler 測試
go test ./internal/adapter/inbound/handler/api/
go test ./internal/adapter/inbound/handler/consumer/
go test ./internal/adapter/inbound/handler/worker/

# 基礎設施測試
go test ./internal/infrastructure/kds/
go test ./internal/infrastructure/queue/

# 整合測試
go test ./test/...

# 特定功能測試
go test -run TestMerchantRepository_FindByGlobalID ./internal/adapter/outbound/repository/merchant/
go test -run TestPlayerUseCase_UpdateLastActive ./internal/application/usecase/player/
go test -run TestHTTPHandler_GetMerchantByGlobalID ./internal/adapter/inbound/handler/api/
```

### 覆蓋率檢查

```bash
# 按模組檢查覆蓋率
go test -cover ./internal/adapter/outbound/repository/...
go test -cover ./internal/application/usecase/...
go test -cover ./internal/adapter/inbound/handler/...

# 完整覆蓋率報告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# 關鍵模組詳細覆蓋率
go test -coverprofile=merchant.out ./internal/adapter/outbound/repository/merchant/
go test -coverprofile=player.out ./internal/adapter/outbound/repository/player/
go tool cover -html=merchant.out
go tool cover -html=player.out
```

## 必要測試案例檢查清單

### Repository 層
- [ ] **Merchant Repository**
  - [ ] FindByID - 正常查詢、記錄不存在
  - [ ] FindByGlobalID - 正常查詢、記錄不存在  
  - [ ] FindByAPIKey - API Key 查詢、無效 Key
  - [ ] Create - 成功創建、重複創建錯誤
  - [ ] Update - 成功更新、記錄不存在錯誤
  - [ ] Upsert - 插入新記錄、更新現有記錄

- [ ] **Player Repository**
  - [ ] FindByID - 正常查詢、記錄不存在
  - [ ] FindByGlobalID - 正常查詢、記錄不存在
  - [ ] UpdateLastActiveByGlobalID - 更新成功、玩家不存在
  - [ ] FindWithTags - 關聯查詢、空標籤、多標籤
  - [ ] AddTag / RemoveTag - 標籤操作成功、重複操作

- [ ] **Manager Repository**
  - [ ] FindByID - 正常查詢、記錄不存在
  - [ ] FindByGlobalID - 正常查詢、記錄不存在
  - [ ] FindByMerchantID - 商戶管理員查詢

- [ ] **Level Repository**
  - [ ] FindByID - 正常查詢、記錄不存在
  - [ ] FindByLevel - 等級查詢、無效等級
  - [ ] CreateOrUpdate - 創建新等級、更新現有等級

- [ ] **Tag Repository**
  - [ ] FindByID - 正常查詢、記錄不存在
  - [ ] FindByName - 名稱查詢、名稱不存在
  - [ ] FindByMerchantID - 商戶標籤查詢
  - [ ] CreateOrUpdate - 創建新標籤、更新現有標籤

### UseCase 層  
- [ ] **Merchant UseCase**
  - [ ] GetMerchantByID - 正常查詢、記錄不存在
  - [ ] GetMerchantByGlobalID - 正常查詢、記錄不存在
  - [ ] SyncMerchant - 新增同步、更新同步
  - [ ] ValidateAPIKey - API Key 驗證

- [ ] **Player UseCase**
  - [ ] GetPlayerByID - 正常查詢、記錄不存在
  - [ ] GetPlayerByGlobalID - 正常查詢、記錄不存在
  - [ ] UpdateLastActive - 更新最後活躍時間成功、玩家不存在
  - [ ] SyncPlayer - 新增同步、更新同步
  - [ ] AssignPlayerTag - 標籤分配、重複分配
  - [ ] UpdatePlayerLevel - 等級更新、無效等級

- [ ] **Manager UseCase**
  - [ ] GetManagerByID - 正常查詢、記錄不存在
  - [ ] GetManagerByGlobalID - 正常查詢、記錄不存在
  - [ ] SyncManager - 新增同步、更新同步

### Handler 層 (HTTP API)
- [ ] **健康檢查**
  - [ ] HealthCheck - 成功案例、數據庫錯誤案例

- [ ] **Merchant API**
  - [ ] GetMerchantByID - 成功案例、InvalidID、NotFound、InternalServerError
  - [ ] GetMerchantByGlobalID - 成功案例、EmptyID、NotFound

- [ ] **Player API** 
  - [ ] GetPlayerByID - 成功案例、InvalidID、NotFound
  - [ ] GetPlayerByGlobalID - 成功案例、EmptyID、NotFound
  - [ ] UpdatePlayerLastActive - 成功案例、InvalidID、NotFound

- [ ] **Manager API**
  - [ ] GetManagerByID - 成功案例、InvalidID、NotFound
  - [ ] GetManagerByGlobalID - 成功案例、EmptyID、NotFound

### Consumer/Worker 層
- [ ] **Consumer Handler**
  - [ ] ProcessMerchantEvent - 成功處理、事件格式錯誤
  - [ ] ProcessPlayerEvent - 成功處理、事件格式錯誤
  - [ ] ProcessManagerEvent - 成功處理、事件格式錯誤
  - [ ] ErrorHandling - 重試機制、死信隊列

- [ ] **Worker Handler**
  - [ ] ProcessIdentitySync - 同步任務成功、任務失敗
  - [ ] ProcessPlayerActiveUpdate - 活躍度更新成功、失敗重試
  - [ ] ProcessLevelUpdate - 等級更新成功、失敗處理

### 基礎設施層
- [ ] **KDS Integration**
  - [ ] ConsumeEvents - 事件消費成功、連接錯誤
  - [ ] ProduceEvents - 事件發送成功、發送失敗
  - [ ] BatchProcessing - 批量處理成功、部分失敗

- [ ] **Queue Integration** 
  - [ ] EnqueueTask - 任務入隊成功、隊列滿
  - [ ] DequeueTask - 任務出隊成功、隊列空
  - [ ] RetryMechanism - 重試成功、達到最大重試次數

## 測試資料準備

### Fat Identity Cat 特定測試資料

```go
// Merchant 測試資料
func createMerchantTestData() *dto.MerchantResponse {
    return &dto.MerchantResponse{
        ID:       1,
        GlobalID: "FATCAT-MERCHANT-001",
        Name:     "Test Merchant",
        APIKey:   "test-api-key-12345",
        Status:   "active",
        CreatedAt: time.Now().Add(-24 * time.Hour),
        UpdatedAt: time.Now().Add(-1 * time.Hour),
    }
}

// Player 測試資料
func createPlayerTestData() *dto.PlayerResponse {
    return &dto.PlayerResponse{
        ID:           1,
        GlobalID:     "FATCAT-PLAYER-001",
        GlobalMerchantID: "FATCAT-MERCHANT-001",
        Level:        5,
        LastActiveAt: time.Now().Add(-30 * time.Minute),
        Tags:         []string{"VIP", "Active"},
        CreatedAt:    time.Now().Add(-7 * 24 * time.Hour),
        UpdatedAt:    time.Now().Add(-30 * time.Minute),
    }
}

// Manager 測試資料
func createManagerTestData() *dto.ManagerResponse {
    return &dto.ManagerResponse{
        ID:       1,
        GlobalID: "FATCAT-MANAGER-001",
        GlobalMerchantID: "FATCAT-MERCHANT-001",
        Name:     "Test Manager",
        Role:     "admin",
        CreatedAt: time.Now().Add(-24 * time.Hour),
        UpdatedAt: time.Now().Add(-1 * time.Hour),
    }
}

// Event 測試資料
func createMerchantEvent() *event.MerchantEvent {
    return &event.MerchantEvent{
        GlobalMerchantID: "FATCAT-MERCHANT-001",
        Name:            "Test Merchant Updated",
        APIKey:          "updated-api-key-12345",
        Status:          "active",
        UpdatedAt:       time.Now(),
    }
}

func createPlayerEvent() *event.PlayerEvent {
    return &event.PlayerEvent{
        GlobalPlayerID:   "FATCAT-PLAYER-001",
        GlobalMerchantID: "FATCAT-MERCHANT-001",
        Level:           6,
        LastActiveAt:    time.Now(),
        Tags:            []string{"VIP", "Active", "Premium"},
    }
}

// Level 和 Tag 測試資料
func createLevelTestData() *entity.Level {
    return &entity.Level{
        ID:    1,
        Level: 5,
        Name:  "Silver",
        Description: "Silver level player",
        CreatedAt: time.Now().Add(-24 * time.Hour),
    }
}

func createTagTestData() *entity.Tag {
    return &entity.Tag{
        ID:   1,
        Name: "VIP",
        GlobalMerchantID: "FATCAT-MERCHANT-001",
        Description: "VIP customer tag",
        CreatedAt: time.Now().Add(-24 * time.Hour),
    }
}
```

## 驗收標準

### Fat Identity Cat 特定覆蓋率要求
- [ ] **Repository 層覆蓋率 ≥ 85%**
  - [ ] Merchant Repository ≥ 90%
  - [ ] Player Repository ≥ 90% 
  - [ ] Manager Repository ≥ 85%
  - [ ] Level Repository ≥ 80%
  - [ ] Tag Repository ≥ 80%

- [ ] **UseCase 層覆蓋率 ≥ 80%**
  - [ ] 身份同步邏輯 = 100%
  - [ ] 關鍵業務邏輯 ≥ 95%
  - [ ] 錯誤處理 ≥ 90%

- [ ] **Handler 層覆蓋率 ≥ 75%**
  - [ ] HTTP API Handler ≥ 80%
  - [ ] Consumer Handler ≥ 75%
  - [ ] Worker Handler ≥ 70%

- [ ] **基礎設施層覆蓋率 ≥ 70%**
  - [ ] KDS Integration ≥ 75%
  - [ ] Queue Integration ≥ 70%

### 測試品質要求
- [ ] 所有測試案例通過
- [ ] Mock 期望完全滿足 (`AssertExpectations`)
- [ ] SQL Mock 期望完全滿足 (`ExpectationsWereMet`)
- [ ] 錯誤處理測試完整
- [ ] 邊界條件測試覆蓋

### Logger Mock 要求
- [ ] **必須使用** `helper.SetupLoggerMock(t)` 創建 Logger Mock
- [ ] **禁止使用** `logger.AssertExpectations(t)` 驗證
- [ ] **禁止手動設置** Logger Mock 期望值
- [ ] Logger Mock 支援所有必要方法且自動配置

### Fat Identity Cat 特定要求
- [ ] **身份實體關聯測試** - Merchant、Player、Manager 關聯正確
- [ ] **標籤系統測試** - Player-Tag 多對多關聯完整
- [ ] **等級系統測試** - Player Level 關聯和更新邏輯
- [ ] **API Key 認證測試** - Merchant API Key 驗證邏輯
- [ ] **同步邏輯測試** - KDS 事件同步完整性
- [ ] **活躍度更新測試** - Player LastActive 更新邏輯

### 測試命名規範
- [ ] 測試函數: `Test[Module][Method]_[Scenario]`
- [ ] 測試案例: 使用描述性 name 字段
- [ ] Mock 函數: `Mock[Module]Repository` / `Mock[Module]UseCase`

## 常用斷言模式

```go
// Fat Identity Cat 特定斷言
// 身份實體驗證
assert.Equal(t, "FATCAT-MERCHANT-001", result.GlobalID)
assert.Equal(t, "FATCAT-PLAYER-001", result.GlobalID)
assert.Equal(t, "FATCAT-MANAGER-001", result.GlobalID)

// 關聯關係驗證
assert.Len(t, result.Tags, 2)
assert.Contains(t, result.Tags, "VIP")
assert.Equal(t, 5, result.Level)

// 時間戳驗證
assert.WithinDuration(t, time.Now(), result.LastActiveAt, time.Minute)
assert.True(t, result.UpdatedAt.After(result.CreatedAt))

// API Key 驗證
assert.NotEmpty(t, result.APIKey)
assert.True(t, len(result.APIKey) >= 20)

// HTTP Response 驗證 (身份 API)
assert.Equal(t, http.StatusOK, w.Code)
var response map[string]interface{}
err := json.Unmarshal(w.Body.Bytes(), &response)
assert.NoError(t, err)
assert.Equal(t, true, response["success"])
assert.Contains(t, response, "data")

// 身份資料驗證
data := response["data"].(map[string]interface{})
assert.Equal(t, "FATCAT-MERCHANT-001", data["global_id"])
assert.Equal(t, "Test Merchant", data["name"])

// 錯誤處理驗證
assert.Error(t, err)
assert.ErrorIs(t, err, errmsg.ErrRepoMerchantNotFound)
assert.ErrorIs(t, err, errmsg.ErrRepoPlayerNotFound)
assert.ErrorIs(t, err, errmsg.ErrRepoManagerNotFound)

// Mock 驗證
repo.AssertExpectations(t)
useCase.AssertExpectations(t)
assert.NoError(t, mock.ExpectationsWereMet())

// Logger Mock - 不需要驗證
// ❌ logger.AssertExpectations(t)  // 不要這樣做
// ✅ Logger Mock 已自動配置，無需額外驗證
```

---

## 專案架構對應

### Fat Identity Cat 專案結構測試覆蓋

```
測試覆蓋對應：
├── internal/domain/entity/ (實體定義)
│   ├── merchant.go -> Merchant Repository/UseCase 測試
│   ├── player.go -> Player Repository/UseCase 測試  
│   ├── manager.go -> Manager Repository/UseCase 測試
│   ├── level.go -> Level Repository/UseCase 測試
│   ├── tag.go -> Tag Repository/UseCase 測試
│   └── player_tag.go -> PlayerTag Repository 測試
│
├── internal/adapter/outbound/repository/ (Repository 實現)
│   ├── merchant/ -> Merchant Repository 測試
│   ├── player/ -> Player Repository 測試
│   ├── manager/ -> Manager Repository 測試
│   ├── level/ -> Level Repository 測試
│   └── tag/ -> Tag Repository 測試
│
├── internal/application/usecase/ (UseCase 實現)
│   ├── merchant/ -> Merchant UseCase 測試
│   ├── player/ -> Player UseCase 測試
│   └── manager/ -> Manager UseCase 測試
│
├── internal/adapter/inbound/handler/ (Handler 實現)
│   ├── api/ -> HTTP API Handler 測試
│   ├── consumer/ -> Consumer Handler 測試
│   └── worker/ -> Worker Handler 測試
│
└── internal/infrastructure/ (基礎設施)
    ├── kds/ -> KDS Integration 測試
    ├── queue/ -> Queue Integration 測試
    └── database/ -> Database Connection 測試
```

這個測試需求模板專門針對 Fat Identity Cat 專案的身份管理特性設計，涵蓋所有核心功能和架構層次的測試要求。