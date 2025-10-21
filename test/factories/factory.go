package factories

import (
	"context"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
)

// TestError 測試用錯誤結構
type TestError struct {
	Message string
}

func (e *TestError) Error() string {
	return e.Message
}

// Test helper functions
func CreateTestContext() context.Context {
	return context.Background()
}

func CreateTestPlayer() *entity.Player {
	email := "test@example.com"
	now := time.Now()
	lastActive := now.Add(-1 * time.Hour)
	return &entity.Player{
		ID:             1,
		MerchantID:     2,
		GlobalPlayerID: "FATCAT-PLAYER-1",
		APIKey:         "player-api-key",
		Account:        "TestPlayer",
		Email:          &email,
		LastActiveAt:   &lastActive,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
}

func CreateTestMerchant() *entity.Merchant {
	now := time.Now()
	return &entity.Merchant{
		ID:               2,
		GlobalMerchantID: "FATCAT-MERCHANT-1",
		Name:             "TestMerchant",
		DisplayName:      "Test Merchant",
		APIKey:           "merchant-api-key",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func CreateTestManager() *entity.Manager {
	email := "manager@example.com"
	now := time.Now()
	return &entity.Manager{
		ID:              1,
		MerchantID:      2,
		GlobalManagerID: "FATCAT-MANAGER-1",
		Account:         "TestManager",
		Email:           &email,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func CreateTestLevel() *entity.Level {
	now := time.Now()
	return &entity.Level{
		ID:                  1,
		MerchantID:          2,
		GlobalPlayerLevelID: "FATCAT-LEVEL-1",
		GlobalMerchantID:    "FATCAT-MERCHANT-1",
		Name:                "Bronze",
		CreatedAt:           now,
		UpdatedAt:           now,
	}
}

func CreateTestTag() *entity.Tag {
	now := time.Now()
	return &entity.Tag{
		ID:          1,
		MerchantID:  2,
		GlobalTagID: "FATCAT-TAG-1",
		Name:        "VIP",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func CreateTestPlayerTag() *entity.PlayerTag {
	now := time.Now()
	return &entity.PlayerTag{
		PlayerID:  1,
		TagID:     1,
		CreatedAt: now,
	}
}
