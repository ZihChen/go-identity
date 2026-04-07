package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-identity-cat/test/factories"
	"github.com/jvdiamondtech/ms-identity-cat/test/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// Helper functions
func setupTest(
	t *testing.T,
) (*mocks.MerchantUseCaseMock, *mocks.PlayerUseCaseMock, *mocks.ManagerUseCaseMock, *mocks.MockLogger, *mocks.JWTServiceMock, *HTTPHandler, *gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)

	merchantUseCase := mocks.NewMerchantUseCaseMock(t)
	playerUseCase := mocks.NewPlayerUseCaseMock(t)
	managerUseCase := mocks.NewManagerUseCaseMock(t)
	mockLogger := mocks.NewMockLogger(t)
	mockTracing := mocks.NewTracingServiceMock(t)
	mockEventProducer := mocks.NewEventProducerMock(t)
	mockJWTService := mocks.NewJWTServiceMock(t)

	handler := &HTTPHandler{
		merchantUseCase: merchantUseCase,
		playerUseCase:   playerUseCase,
		managerUseCase:  managerUseCase,
		logger:          mockLogger,
		tracing:         mockTracing,
		eventProducer:   mockEventProducer,
		jwtService:      mockJWTService,
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	return merchantUseCase, playerUseCase, managerUseCase, mockLogger, mockJWTService, handler, c, w
}

// Test helper functions are now in test/factories/factory.go

// Tests for GetMerchantByID
func TestHTTPHandler_GetMerchantByID(t *testing.T) {
	merchantUseCase, _, _, _, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("GET", "/api/v1/merchants/1", nil)
	c.Params = []gin.Param{{Key: "id", Value: "1"}}

	// Setup mock
	merchant := factories.CreateTestMerchant()
	merchantUseCase.On("GetMerchantByID", mock.Anything, uint64(1)).Return(merchant, nil)

	// Execute
	handler.GetMerchantByID(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response entity.Merchant
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, merchant.GetID(), response.GetID())
	assert.Equal(t, merchant.GetGlobalMerchantID(), response.GetGlobalMerchantID())
	assert.Equal(t, merchant.GetName(), response.GetName())

	merchantUseCase.AssertExpectations()
}

func TestHTTPHandler_GetMerchantByID_InvalidID(t *testing.T) {
	_, _, _, _, _, handler, c, w := setupTest(t)

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
	merchantUseCase, _, _, _, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("GET", "/api/v1/merchants/999", nil)
	c.Params = []gin.Param{{Key: "id", Value: "999"}}

	// Setup mock to return not found error
	merchantUseCase.On("GetMerchantByID", mock.Anything, uint64(999)).
		Return(nil, errmsg.ErrRepoMerchantNotFound)

	// Execute
	handler.GetMerchantByID(c)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Merchant not found")

	merchantUseCase.AssertExpectations()
}

func TestHTTPHandler_GetMerchantByID_InternalError(t *testing.T) {
	merchantUseCase, _, _, _, _, handler, c, w := setupTest(t)

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

	merchantUseCase.AssertExpectations()
}

// Tests for GetMerchantByGlobalID
func TestHTTPHandler_GetMerchantByGlobalID(t *testing.T) {
	merchantUseCase, _, _, _, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("GET", "/api/v1/merchants/global/FATCAT-MERCHANT-1", nil)
	c.Params = []gin.Param{{Key: "global_id", Value: "FATCAT-MERCHANT-1"}}

	// Setup mock
	merchant := factories.CreateTestMerchant()
	merchantUseCase.On("GetMerchantByGlobalID", mock.Anything, "FATCAT-MERCHANT-1").
		Return(merchant, nil)

	// Execute
	handler.GetMerchantByGlobalID(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response entity.Merchant
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, merchant.GetID(), response.GetID())
	assert.Equal(t, merchant.GetGlobalMerchantID(), response.GetGlobalMerchantID())
	assert.Equal(t, merchant.GetName(), response.GetName())

	merchantUseCase.AssertExpectations()
}

func TestHTTPHandler_GetMerchantByGlobalID_EmptyID(t *testing.T) {
	_, _, _, _, _, handler, c, w := setupTest(t)

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
	merchantUseCase, _, _, _, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("GET", "/api/v1/merchants/global/NONEXISTENT", nil)
	c.Params = []gin.Param{{Key: "global_id", Value: "NONEXISTENT"}}

	// Setup mock to return not found error
	merchantUseCase.On("GetMerchantByGlobalID", mock.Anything, "NONEXISTENT").
		Return(nil, errmsg.ErrRepoMerchantNotFound)

	// Execute
	handler.GetMerchantByGlobalID(c)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Merchant not found")

	merchantUseCase.AssertExpectations()
}

func TestHTTPHandler_GetMerchantByGlobalID_InternalError(t *testing.T) {
	merchantUseCase, _, _, _, _, handler, c, w := setupTest(t)

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

	merchantUseCase.AssertExpectations()
}

// Tests for GetPlayerByID
func TestHTTPHandler_GetPlayerByID(t *testing.T) {
	_, playerUseCase, _, _, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("GET", "/api/v1/players/1", nil)
	c.Params = []gin.Param{{Key: "id", Value: "1"}}

	// Setup mock
	player := factories.CreateTestPlayer()
	playerUseCase.On("GetPlayerByID", mock.Anything, uint64(1)).Return(player, nil)

	// Execute
	handler.GetPlayerByID(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response entity.Player
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, player.GetID(), response.GetID())
	assert.Equal(t, player.GetGlobalPlayerID(), response.GetGlobalPlayerID())
	assert.Equal(t, player.GetAccount(), response.GetAccount())

	playerUseCase.AssertExpectations()
}

func TestHTTPHandler_GetPlayerByID_InvalidID(t *testing.T) {
	_, _, _, _, _, handler, c, w := setupTest(t)

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
	_, playerUseCase, _, _, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("GET", "/api/v1/players/999", nil)
	c.Params = []gin.Param{{Key: "id", Value: "999"}}

	// Setup mock to return not found error
	playerUseCase.On("GetPlayerByID", mock.Anything, uint64(999)).
		Return(nil, errmsg.ErrRepoPlayerNotFound)

	// Execute
	handler.GetPlayerByID(c)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Player not found")

	playerUseCase.AssertExpectations()
}

func TestHTTPHandler_GetPlayerByID_InternalError(t *testing.T) {
	_, playerUseCase, _, _, _, handler, c, w := setupTest(t)

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

	playerUseCase.AssertExpectations()
}

// Tests for GetPlayerByGlobalID
func TestHTTPHandler_GetPlayerByGlobalID(t *testing.T) {
	_, playerUseCase, _, _, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("GET", "/api/v1/players/global/FATCAT-PLAYER-1", nil)
	c.Params = []gin.Param{{Key: "global_id", Value: "FATCAT-PLAYER-1"}}

	// Setup mock
	player := factories.CreateTestPlayer()
	playerUseCase.On("GetPlayerByGlobalID", mock.Anything, "FATCAT-PLAYER-1").Return(player, nil)

	// Execute
	handler.GetPlayerByGlobalID(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response entity.Player
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, player.GetID(), response.GetID())
	assert.Equal(t, player.GetGlobalPlayerID(), response.GetGlobalPlayerID())
	assert.Equal(t, player.GetAccount(), response.GetAccount())

	playerUseCase.AssertExpectations()
}

func TestHTTPHandler_GetPlayerByGlobalID_EmptyID(t *testing.T) {
	_, _, _, _, _, handler, c, w := setupTest(t)

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
	_, playerUseCase, _, _, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("GET", "/api/v1/players/global/NONEXISTENT", nil)
	c.Params = []gin.Param{{Key: "global_id", Value: "NONEXISTENT"}}

	// Setup mock to return not found error
	playerUseCase.On("GetPlayerByGlobalID", mock.Anything, "NONEXISTENT").
		Return(nil, errmsg.ErrRepoPlayerNotFound)

	// Execute
	handler.GetPlayerByGlobalID(c)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Player not found")

	playerUseCase.AssertExpectations()
}

func TestHTTPHandler_GetPlayerByGlobalID_InternalError(t *testing.T) {
	_, playerUseCase, _, _, _, handler, c, w := setupTest(t)

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

	playerUseCase.AssertExpectations()
}

func TestHTTPHandler_UpdatePlayerLastActive(t *testing.T) {
	_, playerUseCase, _, _, _, handler, c, w := setupTest(t)

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

	playerUseCase.AssertExpectations()
}

func TestHTTPHandler_UpdatePlayerLastActive_InvalidID(t *testing.T) {
	_, _, _, _, _, handler, c, w := setupTest(t)

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
	_, playerUseCase, _, _, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("PUT", "/api/v1/players/999/active", nil)
	c.Params = []gin.Param{{Key: "id", Value: "999"}}

	// Setup mock to return not found error
	playerUseCase.On("UpdatePlayerLastActive", mock.Anything, uint64(999)).
		Return(errmsg.ErrRepoPlayerNotFound)

	// Execute
	handler.UpdatePlayerLastActive(c)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Player not found")

	playerUseCase.AssertExpectations()
}

func TestHTTPHandler_UpdatePlayerLastActive_InternalError(t *testing.T) {
	_, playerUseCase, _, _, _, handler, c, w := setupTest(t)

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

	playerUseCase.AssertExpectations()
}

// Tests for GetManagerByID
func TestHTTPHandler_GetManagerByID(t *testing.T) {
	_, _, managerUseCase, _, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("GET", "/api/v1/managers/1", nil)
	c.Params = []gin.Param{{Key: "id", Value: "1"}}

	// Setup mock
	manager := factories.CreateTestManager()
	managerUseCase.On("GetManagerByID", mock.Anything, uint64(1)).Return(manager, nil)

	// Execute
	handler.GetManagerByID(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response entity.Manager
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, manager.GetID(), response.GetID())
	assert.Equal(t, manager.GetGlobalManagerID(), response.GetGlobalManagerID())
	assert.Equal(t, manager.GetAccount(), response.GetAccount())

	managerUseCase.AssertExpectations()
}

func TestHTTPHandler_GetManagerByID_InvalidID(t *testing.T) {
	_, _, _, _, _, handler, c, w := setupTest(t)

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
	_, _, managerUseCase, _, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("GET", "/api/v1/managers/999", nil)
	c.Params = []gin.Param{{Key: "id", Value: "999"}}

	// Setup mock to return not found error
	managerUseCase.On("GetManagerByID", mock.Anything, uint64(999)).
		Return(nil, errmsg.ErrRepoManagerNotFound)

	// Execute
	handler.GetManagerByID(c)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Manager not found")

	managerUseCase.AssertExpectations()
}

func TestHTTPHandler_GetManagerByID_InternalError(t *testing.T) {
	_, _, managerUseCase, _, _, handler, c, w := setupTest(t)

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

	managerUseCase.AssertExpectations()
}

// Tests for GetManagerByGlobalID
func TestHTTPHandler_GetManagerByGlobalID(t *testing.T) {
	_, _, managerUseCase, _, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("GET", "/api/v1/managers/global/FATCAT-MANAGER-1", nil)
	c.Params = []gin.Param{{Key: "global_id", Value: "FATCAT-MANAGER-1"}}

	// Setup mock
	manager := factories.CreateTestManager()
	managerUseCase.On("GetManagerByGlobalID", mock.Anything, "FATCAT-MANAGER-1").
		Return(manager, nil)

	// Execute
	handler.GetManagerByGlobalID(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response entity.Manager
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, manager.GetID(), response.GetID())
	assert.Equal(t, manager.GetGlobalManagerID(), response.GetGlobalManagerID())
	assert.Equal(t, manager.GetAccount(), response.GetAccount())

	managerUseCase.AssertExpectations()
}

func TestHTTPHandler_GetManagerByGlobalID_EmptyID(t *testing.T) {
	_, _, _, _, _, handler, c, w := setupTest(t)

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
	_, _, managerUseCase, _, _, handler, c, w := setupTest(t)

	// Setup request
	c.Request = httptest.NewRequest("GET", "/api/v1/managers/global/NONEXISTENT", nil)
	c.Params = []gin.Param{{Key: "global_id", Value: "NONEXISTENT"}}

	// Setup mock to return not found error
	managerUseCase.On("GetManagerByGlobalID", mock.Anything, "NONEXISTENT").
		Return(nil, errmsg.ErrRepoManagerNotFound)

	// Execute
	handler.GetManagerByGlobalID(c)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Manager not found")

	managerUseCase.AssertExpectations()
}

func TestHTTPHandler_GetManagerByGlobalID_InternalError(t *testing.T) {
	_, _, managerUseCase, _, _, handler, c, w := setupTest(t)

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

	managerUseCase.AssertExpectations()
}

// Tests for PlayerLogin
func TestHTTPHandler_PlayerLogin_Success(t *testing.T) {
	_, _, _, _, mockJWT, handler, c, w := setupTest(t)

	// Setup request
	requestBody := map[string]interface{}{
		"global_merchant_id": "FATCAT-MERCHANT-001",
		"metadata": map[string]interface{}{
			"player_id": "883",
			"username":  "winston",
		},
	}
	body, _ := json.Marshal(requestBody)
	c.Request = httptest.NewRequest(
		"POST",
		"/api/v1/players/login",
		strings.NewReader(string(body)),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	// Set merchant ID in context (simulating auth middleware)
	c.Set("global_merchant_id", "FATCAT-MERCHANT-001")

	// Setup mock
	expectedToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test"
	mockJWT.On("GenerateTokenWithMetadata", "FATCAT-MERCHANT-001", mock.MatchedBy(func(metadata map[string]interface{}) bool {
		return metadata["player_id"] == "883" && metadata["username"] == "winston"
	})).
		Return(expectedToken, nil)

	// Execute
	handler.PlayerLogin(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response["success"].(bool))
	assert.Equal(t, expectedToken, response["jwt_token"])

	mockJWT.AssertExpectations()
}

func TestHTTPHandler_PlayerLogin_InvalidRequestFormat(t *testing.T) {
	_, _, _, _, _, handler, c, w := setupTest(t)

	// Setup request with invalid JSON
	c.Request = httptest.NewRequest(
		"POST",
		"/api/v1/players/login",
		strings.NewReader("invalid json"),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	// Execute
	handler.PlayerLogin(c)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Invalid request format")
}

func TestHTTPHandler_PlayerLogin_MissingMerchantAuth(t *testing.T) {
	_, _, _, _, _, handler, c, w := setupTest(t)

	// Setup request
	requestBody := map[string]interface{}{
		"global_merchant_id": "FATCAT-MERCHANT-001",
		"metadata": map[string]interface{}{
			"player_id": "883",
		},
	}
	body, _ := json.Marshal(requestBody)
	c.Request = httptest.NewRequest(
		"POST",
		"/api/v1/players/login",
		strings.NewReader(string(body)),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	// NOT setting merchant ID in context

	// Execute
	handler.PlayerLogin(c)

	// Assert
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Missing merchant authentication")
}

func TestHTTPHandler_PlayerLogin_MerchantIDMismatch(t *testing.T) {
	_, _, _, mockLogger, _, handler, c, w := setupTest(t)

	// Setup request
	requestBody := map[string]interface{}{
		"global_merchant_id": "FATCAT-MERCHANT-001",
		"metadata": map[string]interface{}{
			"player_id": "883",
		},
	}
	body, _ := json.Marshal(requestBody)
	c.Request = httptest.NewRequest(
		"POST",
		"/api/v1/players/login",
		strings.NewReader(string(body)),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	// Set different merchant ID in context
	c.Set("global_merchant_id", "FATCAT-MERCHANT-002")

	// Execute
	handler.PlayerLogin(c)

	// Assert
	assert.Equal(t, http.StatusForbidden, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Merchant ID does not match API key")

	mockLogger.AssertNumberOfCalls(t, "WarnWithContext", 1)
	mockLogger.AssertExpectations()
}

func TestHTTPHandler_PlayerLogin_JWTGenerationError(t *testing.T) {
	_, _, _, mockLogger, mockJWT, handler, c, w := setupTest(t)

	// Setup request
	requestBody := map[string]interface{}{
		"global_merchant_id": "FATCAT-MERCHANT-001",
		"metadata": map[string]interface{}{
			"player_id": "883",
		},
	}
	body, _ := json.Marshal(requestBody)
	c.Request = httptest.NewRequest(
		"POST",
		"/api/v1/players/login",
		strings.NewReader(string(body)),
	)
	c.Request.Header.Set("Content-Type", "application/json")

	// Set merchant ID in context
	c.Set("global_merchant_id", "FATCAT-MERCHANT-001")

	// Setup mock to return error
	mockJWT.On("GenerateTokenWithMetadata", mock.Anything, mock.Anything).
		Return("", errors.New("jwt generation failed"))
	// Execute
	handler.PlayerLogin(c)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Contains(t, response["error"], "Failed to generate token")

	mockJWT.AssertExpectations()
	mockLogger.AssertNumberOfCalls(t, "ErrorWithContext", 1)
	mockLogger.AssertExpectations()
}
