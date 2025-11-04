package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	domainModel "github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/event"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/inbound"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/ports/outbound/service"
)

// 為 Swagger 提供的類型別名
type Merchant = domainModel.Merchant
type Player = domainModel.Player
type Manager = domainModel.Manager

// 錯誤響應模型
type ErrorResponse struct {
	Error string `json:"error" example:"An error occurred"`
}

// 成功響應模型
type SuccessResponse struct {
	Status string `json:"status" example:"success"`
}

// HTTPHandler HTTP接口處理器
type HTTPHandler struct {
	merchantUseCase inbound.MerchantUseCase
	playerUseCase   inbound.PlayerUseCase
	managerUseCase  inbound.ManagerUseCase
	logger          infrastructure.Logger
	tracing         infrastructure.TracingService
	eventProducer   service.EventProducer
}

// NewHTTPHandler 創建HTTP處理器
func NewHTTPHandler(
	merchantUseCase inbound.MerchantUseCase,
	playerUseCase inbound.PlayerUseCase,
	managerUseCase inbound.ManagerUseCase,
	logger infrastructure.Logger,
	tracing infrastructure.TracingService,
	eventProducer service.EventProducer,
) *HTTPHandler {
	return &HTTPHandler{
		merchantUseCase: merchantUseCase,
		playerUseCase:   playerUseCase,
		managerUseCase:  managerUseCase,
		logger:          logger,
		tracing:         tracing,
		eventProducer:   eventProducer,
	}
}

// GetTracingService 獲取TracingService實例
func (h *HTTPHandler) GetTracingService() infrastructure.TracingService {
	return h.tracing
}

// HealthCheck 健康檢查
// @Summary 健康檢查
// @Description 檢查服務是否正常運行
// @Tags 系統
// @Produce json
// @Success 200 {object} SuccessResponse
// @Router /health [get]
func (h *HTTPHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

// GetMerchantByID 通過ID獲取商戶
// @Summary 通過ID獲取商戶
// @Description 根據商戶ID獲取商戶信息
// @Tags 商戶
// @Accept json
// @Produce json
// @Param id path uint64 true "商戶ID"
// @Success 200 {object} Merchant
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/merchants/{id} [get]
func (h *HTTPHandler) GetMerchantByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid merchant ID",
		})
		return
	}

	merchant, err := h.merchantUseCase.GetMerchantByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Merchant not found",
			})
			return
		}

		h.logger.ErrorLog("Failed to get merchant by ID",
			h.logger.UInt64("id", id),
			h.logger.Error("err", err))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get merchant",
		})
		return
	}

	c.JSON(http.StatusOK, merchant)
}

// GetMerchantByGlobalID 通過全局ID獲取商戶
// @Summary 通過全局ID獲取商戶
// @Description 根據商戶全局ID獲取商戶信息
// @Tags 商戶
// @Accept json
// @Produce json
// @Param global_id path string true "商戶全局ID"
// @Success 200 {object} Merchant
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/merchants/global/{global_id} [get]
func (h *HTTPHandler) GetMerchantByGlobalID(c *gin.Context) {
	globalID := c.Param("global_id")
	if globalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Global merchant ID is required",
		})
		return
	}

	merchant, err := h.merchantUseCase.GetMerchantByGlobalID(c.Request.Context(), globalID)
	if err != nil {
		if errors.Is(err, errmsg.ErrRepoMerchantNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Merchant not found",
			})
			return
		}

		h.logger.ErrorLog("Failed to get merchant by global ID",
			h.logger.String("global_id", globalID),
			h.logger.Error("err", err))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get merchant",
		})
		return
	}

	c.JSON(http.StatusOK, merchant)
}

// GetPlayerByID 通過ID獲取玩家
// @Summary 通過ID獲取玩家
// @Description 根據玩家ID獲取玩家信息
// @Tags 玩家
// @Accept json
// @Produce json
// @Param id path uint64 true "玩家ID"
// @Success 200 {object} Player
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/players/{id} [get]
func (h *HTTPHandler) GetPlayerByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid player ID",
		})
		return
	}

	player, err := h.playerUseCase.GetPlayerByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, errmsg.ErrRepoPlayerNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Player not found",
			})
			return
		}

		h.logger.ErrorLog("Failed to get player by ID",
			h.logger.UInt64("id", id),
			h.logger.Error("err", err))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get player",
		})
		return
	}

	c.JSON(http.StatusOK, player)
}

// GetPlayerByGlobalID 通過全局ID獲取玩家
// @Summary 通過全局ID獲取玩家
// @Description 根據玩家全局ID獲取玩家信息
// @Tags 玩家
// @Accept json
// @Produce json
// @Param global_id path string true "玩家全局ID"
// @Success 200 {object} Player
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/players/global/{global_id} [get]
func (h *HTTPHandler) GetPlayerByGlobalID(c *gin.Context) {
	globalID := c.Param("global_id")
	if globalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Global player ID is required",
		})
		return
	}

	player, err := h.playerUseCase.GetPlayerByGlobalID(c.Request.Context(), globalID)
	if err != nil {
		if errors.Is(err, errmsg.ErrRepoPlayerNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Player not found",
			})
			return
		}

		h.logger.ErrorLog("Failed to get player by global ID",
			h.logger.String("global_id", globalID),
			h.logger.Error("err", err))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get player",
		})
		return
	}
	c.JSON(http.StatusOK, player)
}

// UpdatePlayerLastActive 更新玩家最後活躍時間
// @Summary 更新玩家最後活躍時間
// @Description 將玩家的最後活躍時間更新為當前時間
// @Tags 玩家
// @Accept json
// @Produce json
// @Param id path uint64 true "玩家ID"
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/players/{id}/active [put]
func (h *HTTPHandler) UpdatePlayerLastActive(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid player ID",
		})
		return
	}

	if err := h.playerUseCase.UpdatePlayerLastActive(c.Request.Context(), id); err != nil {
		if errors.Is(err, errmsg.ErrRepoPlayerNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Player not found",
			})
			return
		}

		h.logger.ErrorLog("Failed to update player last active time",
			h.logger.UInt64("id", id),
			h.logger.Error("err", err))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update player",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
	})
}

// GetManagerByID 通過ID獲取管理員
// @Summary 通過ID獲取管理員
// @Description 根據管理員ID獲取管理員信息
// @Tags 管理員
// @Accept json
// @Produce json
// @Param id path uint64 true "管理員ID"
// @Success 200 {object} Manager
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/managers/{id} [get]
func (h *HTTPHandler) GetManagerByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid manager ID",
		})
		return
	}

	manager, err := h.managerUseCase.GetManagerByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, errmsg.ErrRepoManagerNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Manager not found",
			})
			return
		}

		h.logger.ErrorLog("Failed to get manager by ID",
			h.logger.UInt64("id", id),
			h.logger.Error("err", err))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get manager",
		})
		return
	}

	c.JSON(http.StatusOK, manager)
}

// GetManagerByGlobalID 通過全局ID獲取管理員
// @Summary 通過全局ID獲取管理員
// @Description 根據管理員全局ID獲取管理員信息
// @Tags 管理員
// @Accept json
// @Produce json
// @Param global_id path string true "管理員全局ID"
// @Success 200 {object} Manager
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/managers/global/{global_id} [get]
func (h *HTTPHandler) GetManagerByGlobalID(c *gin.Context) {
	globalID := c.Param("global_id")
	if globalID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Global manager ID is required",
		})
		return
	}

	manager, err := h.managerUseCase.GetManagerByGlobalID(c.Request.Context(), globalID)
	if err != nil {
		if errors.Is(err, errmsg.ErrRepoManagerNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Manager not found",
			})
			return
		}

		h.logger.ErrorLog("Failed to get manager by global ID",
			h.logger.String("global_id", globalID),
			h.logger.Error("err", err))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get manager",
		})
		return
	}

	c.JSON(http.StatusOK, manager)
}

// SendKDSTestEvent 發送KDS測試事件
// @Summary 發送KDS測試事件
// @Description 發送代理同步測試事件到KDS
// @Tags 測試
// @Accept json
// @Produce json
// @Success 200 {object} SuccessResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/test/kds [post]
func (h *HTTPHandler) SendKDSTestEvent(c *gin.Context) {
	// 創建測試事件
	now := time.Now()
	eventID := uuid.New().String()

	// 構建測試事件payload
	testEvent := &event.CloudEvent{
		SpecVersion:     "1.0",
		Type:            "tw.jvd.fatcat.agent.sync.v1",
		Source:          "/fatcat/FATCAT",
		Subject:         "agent_sync",
		ID:              eventID,
		Time:            now,
		DataContentType: "application/json",
		Data: map[string]interface{}{
			"agent": map[string]interface{}{
				"global_agent_id":    "FATCAT-AGENT-123",
				"account":            "agent001",
				"ancestry":           "FATCAT-AGENT-456",
				"current_sign_in_at": "2025-01-01T00:00:00.000Z",
				"created_at":         "2025-01-01T00:00:00.000Z",
				"updated_at":         now.Format(time.RFC3339Nano),
			},
			"merchant": map[string]interface{}{
				"id":                 2,
				"name":               "FATCAT",
				"global_merchant_id": "FATCAT-MERCHANT-2",
			},
		},
	}

	// 發送事件到KDS Consumer Stream
	ctx := c.Request.Context()

	// 使用KDS的SendToConsumeStream方法發送事件
	kdsService, ok := h.eventProducer.(interface {
		SendToConsumeStream(ctx context.Context, data []byte, eventType string) error
	})
	if !ok {
		h.logger.ErrorLog("Event producer does not support SendToConsumeStream method")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Event producer not supported",
		})
		return
	}

	// 將事件序列化
	eventBytes, err := json.Marshal(testEvent)
	if err != nil {
		h.logger.ErrorLog("Failed to marshal test event",
			h.logger.String("event_id", eventID),
			h.logger.Error("err", err))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to serialize test event",
		})
		return
	}

	if err := kdsService.SendToConsumeStream(ctx, eventBytes, testEvent.Type); err != nil {
		h.logger.ErrorLog("Failed to send KDS test event to consume stream",
			h.logger.String("event_id", eventID),
			h.logger.Error("err", err))

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to send test event to KDS consume stream",
		})
		return
	}

	h.logger.InfoLog("KDS test event sent successfully",
		h.logger.String("event_id", eventID),
		h.logger.String("event_type", testEvent.Type))

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"event_id": eventID,
		"message":  "Test event sent to KDS successfully",
	})
}
