package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// Mock implementations of the use cases
type MockMerchantUseCase struct {
	mock.Mock
}

func (m *MockMerchantUseCase) GetMerchantByID(
	ctx context.Context,
	id uint64,
) (*entity.Merchant, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Merchant), args.Error(1)
}

func (m *MockMerchantUseCase) GetMerchantByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Merchant, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Merchant), args.Error(1)
}

func (m *MockMerchantUseCase) SyncMerchant(ctx context.Context, eventData []byte) error {
	args := m.Called(ctx, eventData)
	return args.Error(0)
}

type MockPlayerUseCase struct {
	mock.Mock
}

func (m *MockPlayerUseCase) GetPlayerByID(ctx context.Context, id uint64) (*entity.Player, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Player), args.Error(1)
}

func (m *MockPlayerUseCase) GetPlayerByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Player, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Player), args.Error(1)
}

func (m *MockPlayerUseCase) UpdatePlayerLastActive(ctx context.Context, id uint64) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPlayerUseCase) SyncPlayer(ctx context.Context, eventData []byte) error {
	args := m.Called(ctx, eventData)
	return args.Error(0)
}

type MockManagerUseCase struct {
	mock.Mock
}

func (m *MockManagerUseCase) GetManagerByID(
	ctx context.Context,
	id uint64,
) (*entity.Manager, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Manager), args.Error(1)
}

func (m *MockManagerUseCase) GetManagerByGlobalID(
	ctx context.Context,
	globalID string,
) (*entity.Manager, error) {
	args := m.Called(ctx, globalID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*entity.Manager), args.Error(1)
}

func (m *MockManagerUseCase) SyncManager(ctx context.Context, eventData []byte) error {
	args := m.Called(ctx, eventData)
	return args.Error(0)
}

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

// Helper functions
func setupTest(
	t *testing.T,
) (*MockMerchantUseCase, *MockPlayerUseCase, *MockManagerUseCase, *MockLogger, *HTTPHandler, *gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)

	merchantUseCase := new(MockMerchantUseCase)
	playerUseCase := new(MockPlayerUseCase)
	managerUseCase := new(MockManagerUseCase)
	mockLogger := new(MockLogger)

	logger := zaptest.NewLogger(t)

	handler := &HTTPHandler{
		merchantUseCase: merchantUseCase,
		playerUseCase:   playerUseCase,
		managerUseCase:  managerUseCase,
		logger:          logger,
		serviceLog:      mockLogger,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	return merchantUseCase, playerUseCase, managerUseCase, mockLogger, handler, c, w
}

func createTestMerchant() *entity.Merchant {
	return &entity.Merchant{
		ID:               1,
		GlobalMerchantID: "FATCAT-MERCHANT-1",
		Name:             "Test Merchant",
		DisplayName:      "Test Merchant Display",
		APIKey:           "api-key-123",
	}
}

func createTestPlayer() *entity.Player {
	return &entity.Player{
		ID:             1,
		GlobalPlayerID: "FATCAT-PLAYER-1",
		MerchantID:     1,
		Account:        "testplayer",
		APIKey:         "api-key-456",
	}
}

func createTestManager() *entity.Manager {
	return &entity.Manager{
		ID:              1,
		GlobalManagerID: "FATCAT-MANAGER-1",
		MerchantID:      1,
		Account:         "testmanager",
	}
}

// Tests for GetMerchantByID
func TestHTTPHandler_GetMerchantByID(t *testing.T) {
	merchantUseCase, _, _, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("GET", "/api/v1/merchants/1", nil)
	c.Params = []gin.Param{{Key: "id", Value: "1"}}

	// Setup mock
	merchant := createTestMerchant()
	merchantUseCase.On("GetMerchantByID", mock.Anything, uint64(1)).Return(merchant, nil)

	// Execute
	handler.GetMerchantByID(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response entity.Merchant
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, merchant.ID, response.ID)
	assert.Equal(t, merchant.GlobalMerchantID, response.GlobalMerchantID)
	assert.Equal(t, merchant.Name, response.Name)

	merchantUseCase.AssertExpectations(t)
}

func TestHTTPHandler_GetMerchantByID_InvalidID(t *testing.T) {
	_, _, _, _, handler, c, w := setupTest(t)

	// Setup request with invalid ID
	c.Request = httptest.NewRequest("GET", "/api/v1/merchants/invalid", nil)
	c.Params = []gin.Param{{Key: "id", Value: "invalid"}}

	// Execute
	handler.GetMerchantByID(c)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Invalid merchant ID")
}

func TestHTTPHandler_GetMerchantByID_NotFound(t *testing.T) {
	merchantUseCase, _, _, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("GET", "/api/v1/merchants/999", nil)
	c.Params = []gin.Param{{Key: "id", Value: "999"}}

	// Setup mock to return not found error
	merchantUseCase.On("GetMerchantByID", mock.Anything, uint64(999)).
		Return(nil, errors.New("record not found"))

	// Execute
	handler.GetMerchantByID(c)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Merchant not found")

	merchantUseCase.AssertExpectations(t)
}

func TestHTTPHandler_GetMerchantByID_InternalError(t *testing.T) {
	merchantUseCase, _, _, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("GET", "/api/v1/merchants/1", nil)
	c.Params = []gin.Param{{Key: "id", Value: "1"}}

	// Setup mock to return internal error
	merchantUseCase.On("GetMerchantByID", mock.Anything, uint64(1)).
		Return(nil, errors.New("database error"))

	// Execute
	handler.GetMerchantByID(c)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Failed to get merchant")

	merchantUseCase.AssertExpectations(t)
}

// Tests for GetMerchantByGlobalID
func TestHTTPHandler_GetMerchantByGlobalID(t *testing.T) {
	merchantUseCase, _, _, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("GET", "/api/v1/merchants/global/FATCAT-MERCHANT-1", nil)
	c.Params = []gin.Param{{Key: "global_id", Value: "FATCAT-MERCHANT-1"}}

	// Setup mock
	merchant := createTestMerchant()
	merchantUseCase.On("GetMerchantByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").
		Return(merchant, nil)

	// Execute
	handler.GetMerchantByGlobalID(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response entity.Merchant
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, merchant.ID, response.ID)
	assert.Equal(t, merchant.GlobalMerchantID, response.GlobalMerchantID)
	assert.Equal(t, merchant.Name, response.Name)

	merchantUseCase.AssertExpectations(t)
}

func TestHTTPHandler_GetMerchantByGlobalID_EmptyID(t *testing.T) {
	_, _, _, _, handler, c, w := setupTest(t)

	// Setup request with empty global ID
	c.Request = httptest.NewRequest("GET", "/api/v1/merchants/global/", nil)
	c.Params = []gin.Param{{Key: "global_id", Value: ""}}

	// Execute
	handler.GetMerchantByGlobalID(c)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Global merchant ID is required")
}

func TestHTTPHandler_GetMerchantByGlobalID_NotFound(t *testing.T) {
	merchantUseCase, _, _, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("GET", "/api/v1/merchants/global/NONEXISTENT", nil)
	c.Params = []gin.Param{{Key: "global_id", Value: "NONEXISTENT"}}

	// Setup mock to return not found error
	merchantUseCase.On("GetMerchantByGlobalID", mock.Anything, "NONEXISTENT").
		Return(nil, errors.New("record not found"))

	// Execute
	handler.GetMerchantByGlobalID(c)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Merchant not found")

	merchantUseCase.AssertExpectations(t)
}

func TestHTTPHandler_GetMerchantByGlobalID_InternalError(t *testing.T) {
	merchantUseCase, _, _, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("GET", "/api/v1/merchants/global/FATCAT-MERCHANT-1", nil)
	c.Params = []gin.Param{{Key: "global_id", Value: "FATCAT-MERCHANT-1"}}

	// Setup mock to return internal error
	merchantUseCase.On("GetMerchantByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").
		Return(nil, errors.New("database error"))

	// Execute
	handler.GetMerchantByGlobalID(c)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Failed to get merchant")

	merchantUseCase.AssertExpectations(t)
}

// Tests for GetPlayerByID
func TestHTTPHandler_GetPlayerByID(t *testing.T) {
	_, playerUseCase, _, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("GET", "/api/v1/players/1", nil)
	c.Params = []gin.Param{{Key: "id", Value: "1"}}

	// Setup mock
	player := createTestPlayer()
	playerUseCase.On("GetPlayerByID", mock.Anything, uint64(1)).Return(player, nil)

	// Execute
	handler.GetPlayerByID(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response entity.Player
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, player.ID, response.ID)
	assert.Equal(t, player.GlobalPlayerID, response.GlobalPlayerID)
	assert.Equal(t, player.Account, response.Account)

	playerUseCase.AssertExpectations(t)
}

func TestHTTPHandler_GetPlayerByID_InvalidID(t *testing.T) {
	_, _, _, _, handler, c, w := setupTest(t)

	// Setup request with invalid ID
	c.Request = httptest.NewRequest("GET", "/api/v1/players/invalid", nil)
	c.Params = []gin.Param{{Key: "id", Value: "invalid"}}

	// Execute
	handler.GetPlayerByID(c)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Invalid player ID")
}

func TestHTTPHandler_GetPlayerByID_NotFound(t *testing.T) {
	_, playerUseCase, _, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("GET", "/api/v1/players/999", nil)
	c.Params = []gin.Param{{Key: "id", Value: "999"}}

	// Setup mock to return not found error
	playerUseCase.On("GetPlayerByID", mock.Anything, uint64(999)).
		Return(nil, errors.New("record not found"))

	// Execute
	handler.GetPlayerByID(c)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Player not found")

	playerUseCase.AssertExpectations(t)
}

func TestHTTPHandler_GetPlayerByID_InternalError(t *testing.T) {
	_, playerUseCase, _, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("GET", "/api/v1/players/1", nil)
	c.Params = []gin.Param{{Key: "id", Value: "1"}}

	// Setup mock to return internal error
	playerUseCase.On("GetPlayerByID", mock.Anything, uint64(1)).
		Return(nil, errors.New("database error"))

	// Execute
	handler.GetPlayerByID(c)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Failed to get player")

	playerUseCase.AssertExpectations(t)
}

// Tests for GetPlayerByGlobalID
func TestHTTPHandler_GetPlayerByGlobalID(t *testing.T) {
	_, playerUseCase, _, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("GET", "/api/v1/players/global/FATCAT-PLAYER-1", nil)
	c.Params = []gin.Param{{Key: "global_id", Value: "FATCAT-PLAYER-1"}}

	// Setup mock
	player := createTestPlayer()
	playerUseCase.On("GetPlayerByGlobalID", mock.Anything, "FATCAT-PLAYER-1").Return(player, nil)

	// Execute
	handler.GetPlayerByGlobalID(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response entity.Player
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, player.ID, response.ID)
	assert.Equal(t, player.GlobalPlayerID, response.GlobalPlayerID)
	assert.Equal(t, player.Account, response.Account)

	playerUseCase.AssertExpectations(t)
}

func TestHTTPHandler_GetPlayerByGlobalID_EmptyID(t *testing.T) {
	_, _, _, _, handler, c, w := setupTest(t)

	// Setup request with empty global ID
	c.Request = httptest.NewRequest("GET", "/api/v1/players/global/", nil)
	c.Params = []gin.Param{{Key: "global_id", Value: ""}}

	// Execute
	handler.GetPlayerByGlobalID(c)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Global player ID is required")
}

func TestHTTPHandler_GetPlayerByGlobalID_NotFound(t *testing.T) {
	_, playerUseCase, _, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("GET", "/api/v1/players/global/NONEXISTENT", nil)
	c.Params = []gin.Param{{Key: "global_id", Value: "NONEXISTENT"}}

	// Setup mock to return not found error
	playerUseCase.On("GetPlayerByGlobalID", mock.Anything, "NONEXISTENT").
		Return(nil, errors.New("record not found"))

	// Execute
	handler.GetPlayerByGlobalID(c)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Player not found")

	playerUseCase.AssertExpectations(t)
}

func TestHTTPHandler_GetPlayerByGlobalID_InternalError(t *testing.T) {
	_, playerUseCase, _, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("GET", "/api/v1/players/global/FATCAT-PLAYER-1", nil)
	c.Params = []gin.Param{{Key: "global_id", Value: "FATCAT-PLAYER-1"}}

	// Setup mock to return internal error
	playerUseCase.On("GetPlayerByGlobalID", mock.Anything, "FATCAT-PLAYER-1").
		Return(nil, errors.New("database error"))

	// Execute
	handler.GetPlayerByGlobalID(c)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Failed to get player")

	playerUseCase.AssertExpectations(t)
}

func TestHTTPHandler_UpdatePlayerLastActive(t *testing.T) {
	_, playerUseCase, _, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("PUT", "/api/v1/players/1/active", nil)
	c.Params = []gin.Param{{Key: "id", Value: "1"}}

	// Setup mock
	playerUseCase.On("UpdatePlayerLastActive", mock.Anything, uint64(1)).Return(nil)

	// Execute
	handler.UpdatePlayerLastActive(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["status"], "success")

	playerUseCase.AssertExpectations(t)
}

func TestHTTPHandler_UpdatePlayerLastActive_InvalidID(t *testing.T) {
	_, _, _, _, handler, c, w := setupTest(t)

	// Setup request with invalid ID
	c.Request = httptest.NewRequest("PUT", "/api/v1/players/invalid/active", nil)
	c.Params = []gin.Param{{Key: "id", Value: "invalid"}}

	// Execute
	handler.UpdatePlayerLastActive(c)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Invalid player ID")
}

func TestHTTPHandler_UpdatePlayerLastActive_NotFound(t *testing.T) {
	_, playerUseCase, _, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("PUT", "/api/v1/players/999/active", nil)
	c.Params = []gin.Param{{Key: "id", Value: "999"}}

	// Setup mock to return not found error
	playerUseCase.On("UpdatePlayerLastActive", mock.Anything, uint64(999)).
		Return(errors.New("record not found"))

	// Execute
	handler.UpdatePlayerLastActive(c)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Player not found")

	playerUseCase.AssertExpectations(t)
}

func TestHTTPHandler_UpdatePlayerLastActive_InternalError(t *testing.T) {
	_, playerUseCase, _, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("PUT", "/api/v1/players/1/active", nil)
	c.Params = []gin.Param{{Key: "id", Value: "1"}}

	// Setup mock to return internal error
	playerUseCase.On("UpdatePlayerLastActive", mock.Anything, uint64(1)).
		Return(errors.New("database error"))

	// Execute
	handler.UpdatePlayerLastActive(c)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Failed to update player")

	playerUseCase.AssertExpectations(t)
}

// Tests for GetManagerByID
func TestHTTPHandler_GetManagerByID(t *testing.T) {
	_, _, managerUseCase, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("GET", "/api/v1/managers/1", nil)
	c.Params = []gin.Param{{Key: "id", Value: "1"}}

	// Setup mock
	manager := createTestManager()
	managerUseCase.On("GetManagerByID", mock.Anything, uint64(1)).Return(manager, nil)

	// Execute
	handler.GetManagerByID(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response entity.Manager
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, manager.ID, response.ID)
	assert.Equal(t, manager.GlobalManagerID, response.GlobalManagerID)
	assert.Equal(t, manager.Account, response.Account)

	managerUseCase.AssertExpectations(t)
}

func TestHTTPHandler_GetManagerByID_InvalidID(t *testing.T) {
	_, _, _, _, handler, c, w := setupTest(t)

	// Setup request with invalid ID
	c.Request = httptest.NewRequest("GET", "/api/v1/managers/invalid", nil)
	c.Params = []gin.Param{{Key: "id", Value: "invalid"}}

	// Execute
	handler.GetManagerByID(c)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Invalid manager ID")
}

func TestHTTPHandler_GetManagerByID_NotFound(t *testing.T) {
	_, _, managerUseCase, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("GET", "/api/v1/managers/999", nil)
	c.Params = []gin.Param{{Key: "id", Value: "999"}}

	// Setup mock to return not found error
	managerUseCase.On("GetManagerByID", mock.Anything, uint64(999)).
		Return(nil, errors.New("record not found"))

	// Execute
	handler.GetManagerByID(c)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Manager not found")

	managerUseCase.AssertExpectations(t)
}

func TestHTTPHandler_GetManagerByID_InternalError(t *testing.T) {
	_, _, managerUseCase, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("GET", "/api/v1/managers/1", nil)
	c.Params = []gin.Param{{Key: "id", Value: "1"}}

	// Setup mock to return internal error
	managerUseCase.On("GetManagerByID", mock.Anything, uint64(1)).
		Return(nil, errors.New("database error"))

	// Execute
	handler.GetManagerByID(c)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Failed to get manager")

	managerUseCase.AssertExpectations(t)
}

// Tests for GetManagerByGlobalID
func TestHTTPHandler_GetManagerByGlobalID(t *testing.T) {
	_, _, managerUseCase, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("GET", "/api/v1/managers/global/FATCAT-MANAGER-1", nil)
	c.Params = []gin.Param{{Key: "global_id", Value: "FATCAT-MANAGER-1"}}

	// Setup mock
	manager := createTestManager()
	managerUseCase.On("GetManagerByGlobalID", mock.Anything, "FATCAT-MANAGER-1").
		Return(manager, nil)

	// Execute
	handler.GetManagerByGlobalID(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response entity.Manager
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, manager.ID, response.ID)
	assert.Equal(t, manager.GlobalManagerID, response.GlobalManagerID)
	assert.Equal(t, manager.Account, response.Account)

	managerUseCase.AssertExpectations(t)
}

func TestHTTPHandler_GetManagerByGlobalID_EmptyID(t *testing.T) {
	_, _, _, _, handler, c, w := setupTest(t)

	// Setup request with empty global ID
	c.Request = httptest.NewRequest("GET", "/api/v1/managers/global/", nil)
	c.Params = []gin.Param{{Key: "global_id", Value: ""}}

	// Execute
	handler.GetManagerByGlobalID(c)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Global manager ID is required")
}

func TestHTTPHandler_GetManagerByGlobalID_NotFound(t *testing.T) {
	_, _, managerUseCase, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("GET", "/api/v1/managers/global/NONEXISTENT", nil)
	c.Params = []gin.Param{{Key: "global_id", Value: "NONEXISTENT"}}

	// Setup mock to return not found error
	managerUseCase.On("GetManagerByGlobalID", mock.Anything, "NONEXISTENT").
		Return(nil, errors.New("record not found"))

	// Execute
	handler.GetManagerByGlobalID(c)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Manager not found")

	managerUseCase.AssertExpectations(t)
}

func TestHTTPHandler_GetManagerByGlobalID_InternalError(t *testing.T) {
	_, _, managerUseCase, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("GET", "/api/v1/managers/global/FATCAT-MANAGER-1", nil)
	c.Params = []gin.Param{{Key: "global_id", Value: "FATCAT-MANAGER-1"}}

	// Setup mock to return internal error
	managerUseCase.On("GetManagerByGlobalID", mock.Anything, "FATCAT-MANAGER-1").
		Return(nil, errors.New("database error"))

	// Execute
	handler.GetManagerByGlobalID(c)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Failed to get manager")

	managerUseCase.AssertExpectations(t)
}
