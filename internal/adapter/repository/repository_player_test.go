package repository

import (
	"context"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/model"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"testing"
	"time"
)

func TestPlayerRepository_FindByID_Unit(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	decorator := mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	})
	db, err := gorm.Open(decorator, &gorm.Config{})
	require.NoError(t, err)

	repo := NewPlayerRepository(db)

	playerID := uint64(1)
	email := "player@example.com"
	lastActiveAt := time.Now()
	expectedPlayer := &models.Player{
		ID:             playerID,
		MerchantID:     uint64(2),
		GlobalPlayerID: "FATCAT-PLAYER-1",
		APIKey:         "player-api-key",
		Account:        "TestPlayer",
		Email:          &email,
		LastActiveAt:   &lastActiveAt,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// 塞入假資料到mock db
	rows := sqlmock.NewRows([]string{"id", "merchant_id", "global_player_id", "api_key", "account", "email", "last_active_at", "created_at", "updated_at"}).
		AddRow(expectedPlayer.ID, expectedPlayer.MerchantID, expectedPlayer.GlobalPlayerID, expectedPlayer.APIKey, expectedPlayer.Account, expectedPlayer.Email, expectedPlayer.LastActiveAt, expectedPlayer.CreatedAt, expectedPlayer.UpdatedAt)

	mock.ExpectQuery("SELECT"). // 寬鬆模式
					WithArgs(playerID, 1).
					WillReturnRows(rows)

	// 執行測試
	result, err := repo.FindByID(context.Background(), playerID)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedPlayer.MerchantID, result.MerchantID)
	assert.Equal(t, expectedPlayer.GlobalPlayerID, result.GlobalPlayerID)
	assert.Equal(t, expectedPlayer.APIKey, result.APIKey)
	assert.Equal(t, expectedPlayer.Account, result.Account)
	assert.Equal(t, *expectedPlayer.Email, *result.Email)
	assert.Equal(t, expectedPlayer.LastActiveAt.Unix(), result.LastActiveAt.Unix())

	// 驗證所有 SQL 期望都被滿足
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestPlayerRepository_FindByGlobalID_Unit(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	decorator := mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	})
	db, err := gorm.Open(decorator, &gorm.Config{})
	require.NoError(t, err)

	repo := NewPlayerRepository(db)

	globalID := "FATCAT-PLAYER-1"
	email := "player@example.com"
	lastActiveAt := time.Now()
	expectedPlayer := &models.Player{
		ID:             uint64(1),
		MerchantID:     uint64(2),
		GlobalPlayerID: globalID,
		APIKey:         "player-api-key",
		Account:        "TestPlayer",
		Email:          &email,
		LastActiveAt:   &lastActiveAt,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// 塞入假資料到mock db
	rows := sqlmock.NewRows([]string{"id", "merchant_id", "global_player_id", "api_key", "account", "email", "last_active_at", "created_at", "updated_at"}).
		AddRow(expectedPlayer.ID, expectedPlayer.MerchantID, expectedPlayer.GlobalPlayerID, expectedPlayer.APIKey, expectedPlayer.Account, expectedPlayer.Email, expectedPlayer.LastActiveAt, expectedPlayer.CreatedAt, expectedPlayer.UpdatedAt)

	mock.ExpectQuery("SELECT").
		WithArgs(globalID, 1).
		WillReturnRows(rows)

	// 執行測試
	result, err := repo.FindByGlobalID(context.Background(), globalID)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedPlayer.ID, result.ID)
	assert.Equal(t, expectedPlayer.MerchantID, result.MerchantID)
	assert.Equal(t, expectedPlayer.GlobalPlayerID, result.GlobalPlayerID)
	assert.Equal(t, expectedPlayer.APIKey, result.APIKey)
	assert.Equal(t, expectedPlayer.Account, result.Account)
	assert.Equal(t, *expectedPlayer.Email, *result.Email)
	assert.Equal(t, expectedPlayer.LastActiveAt.Unix(), result.LastActiveAt.Unix())

	// 驗證所有 SQL 期望都被滿足
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestPlayerRepository_Create_Unit(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	decorator := mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	})
	db, err := gorm.Open(decorator, &gorm.Config{})
	require.NoError(t, err)

	repo := NewPlayerRepository(db)

	now := time.Now()
	email := "new@example.com"
	lastActiveAt := now
	player := &models.Player{
		MerchantID:     uint64(2),
		GlobalPlayerID: "FATCAT-PLAYER-NEW",
		APIKey:         "new-player-api-key",
		Account:        "NewPlayer",
		Email:          &email,
		LastActiveAt:   &lastActiveAt,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Convert to domain model for the test
	domainPlayer := &model.Player{
		MerchantID:     player.MerchantID,
		GlobalPlayerID: player.GlobalPlayerID,
		APIKey:         player.APIKey,
		Account:        player.Account,
		Email:          player.Email,
		LastActiveAt:   player.LastActiveAt,
		CreatedAt:      player.CreatedAt,
		UpdatedAt:      player.UpdatedAt,
	}

	// Mock the SQL execution
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `players`").
		WillReturnResult(sqlmock.NewResult(1, 1)) // ID = 1, affected rows = 1
	mock.ExpectCommit()

	// 執行測試
	err = repo.Create(context.Background(), domainPlayer)

	// 驗證結果
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), domainPlayer.ID) // ID should be updated

	// 驗證所有 SQL 期望都被滿足
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestPlayerRepository_Update_Unit(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	decorator := mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	})
	db, err := gorm.Open(decorator, &gorm.Config{})
	require.NoError(t, err)

	repo := NewPlayerRepository(db)

	now := time.Now()
	playerID := uint64(1)
	email := "updated@example.com"
	lastActiveAt := now
	player := &model.Player{
		ID:             playerID,
		MerchantID:     uint64(2),
		GlobalPlayerID: "FATCAT-PLAYER-1",
		APIKey:         "updated-api-key",
		Account:        "UpdatedPlayer",
		Email:          &email,
		LastActiveAt:   &lastActiveAt,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Mock the SQL execution
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE").
		WillReturnResult(sqlmock.NewResult(0, 1)) // No new ID, 1 row affected
	mock.ExpectCommit()

	// 執行測試
	err = repo.Update(context.Background(), player)

	// 驗證結果
	assert.NoError(t, err)

	// 驗證所有 SQL 期望都被滿足
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestPlayerRepository_Delete_Unit(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	decorator := mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	})
	db, err := gorm.Open(decorator, &gorm.Config{})
	require.NoError(t, err)

	repo := NewPlayerRepository(db)

	playerID := uint64(1)

	// Mock the SQL execution
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE").
		WithArgs(sqlmock.AnyArg(), playerID).
		WillReturnResult(sqlmock.NewResult(0, 1)) // No new ID, 1 row affected
	mock.ExpectCommit()

	// 執行測試
	err = repo.Delete(context.Background(), playerID)

	// 驗證結果
	assert.NoError(t, err)

	// 驗證所有 SQL 期望都被滿足
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}