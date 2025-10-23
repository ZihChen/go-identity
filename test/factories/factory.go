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
	player := entity.NewPlayerWithTimes(2, "FATCAT-PLAYER-1", "TestPlayer", 0, &email, now, now)
	player.SetID(1)
	player.SetAPIKey("player-api-key")
	player.SetLastActiveAt(&lastActive)
	return player
}

func CreateTestMerchant() *entity.Merchant {
	now := time.Now()
	merchant := entity.NewMerchantWithTimes(
		"FATCAT-MERCHANT-1",
		"TestMerchant",
		"Test Merchant",
		now,
	)
	merchant.SetID(2)
	merchant.SetAPIKey("merchant-api-key")
	merchant.SetCreatedAt(now)
	return merchant
}

func CreateTestManager() *entity.Manager {
	email := "manager@example.com"
	return entity.NewManager(1, "FATCAT-MANAGER-1", "TestManager", &email)
}

func CreateTestLevel() *entity.Level {
	now := time.Now()
	level := entity.NewLevelWithTimes(
		2,
		"Bronze",
		"FATCAT-LEVEL-1",
		now,
		now,
	)
	level.SetID(1)
	return level
}

func CreateTestTag() *entity.Tag {
	now := time.Now()
	tag := entity.NewTagWithTimes(
		2,
		"VIP",
		"FATCAT-TAG-1",
		now,
	)
	tag.SetID(1)
	return tag
}

func CreateTestPlayerTag() *entity.PlayerTag {
	now := time.Now()
	return &entity.PlayerTag{
		PlayerID:  1,
		TagID:     1,
		CreatedAt: now,
	}
}
