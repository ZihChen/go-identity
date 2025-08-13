package repository

import (
	"context"
	"database/sql"
	"errors"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/errmsg"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type PlayerTestCase struct {
	name           string
	id             uint64
	setupMock      func(sqlmock.Sqlmock)
	expectedPlayer *entity.Player
	expectedError  error
}

func setupPlayerMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
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

func TestPlayerRepository_FindByID(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	defer func() {
		_ = sqlDB.Close()
	}()

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

func TestPlayerRepository_FindByGlobalID(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	defer func() {
		_ = sqlDB.Close()
	}()

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

func TestPlayerRepository_FirstOrCreate(t *testing.T) {
	now := time.Now()
	testCases := []PlayerTestCase{
		{
			name: "player first or create",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "merchant_id", "global_player_id", "account", "api_key", "email", "created_at", "updated_at", "deleted_at"}).
					AddRow(2, 1, "Test-Player-01", "test-player-01", "abc123", "", now, now, nil)

				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `players`")).
					WithArgs("Test-Player-01", 2, 1).
					WillReturnRows(rows)
			},
			expectedPlayer: &entity.Player{
				ID:             2,
				MerchantID:     1,
				GlobalPlayerID: "Test-Player-01",
				Account:        "test-player-01",
				APIKey:         "abc123",
				CreatedAt:      now,
				UpdatedAt:      now,
				DeletedAt:      nil,
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupPlayerMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)
			repo := NewPlayerRepository(db)

			err := repo.FirstOrCreate(context.Background(), tc.expectedPlayer)
			if tc.expectedError != nil {
				assert.Error(t, err)
				if errors.Is(tc.expectedError, errmsg.ErrRepoPlayerNotFound) {
					assert.ErrorIs(t, err, errmsg.ErrRepoPlayerNotFound)
				}
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestPlayerRepository_Create(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	defer func() {
		_ = sqlDB.Close()
	}()

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

	// Convert to domain entity for the test
	domainPlayer := &entity.Player{
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

func TestPlayerRepository_Update(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	defer func() {
		_ = sqlDB.Close()
	}()

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
	player := &entity.Player{
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

func TestPlayerRepository_Delete(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	defer func() {
		_ = sqlDB.Close()
	}()

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

func TestPlayerRepository_Upsert(t *testing.T) {
	now := time.Now()
	testCases := []PlayerTestCase{
		{
			name: "player upsert",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				pattern := regexp.QuoteMeta(
					"INSERT INTO `players`",
				) + ".*" + regexp.QuoteMeta(
					"ON DUPLICATE KEY UPDATE",
				)
				mock.ExpectExec(pattern).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedPlayer: &entity.Player{
				ID:             2,
				MerchantID:     1,
				GlobalPlayerID: "Test-Player-01",
				Account:        "test-player-01",
				APIKey:         "abc123",
				CreatedAt:      now,
				UpdatedAt:      now,
				DeletedAt:      nil,
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupPlayerMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)
			repo := NewPlayerRepository(db)

			err := repo.Upsert(context.Background(), tc.expectedPlayer)
			if tc.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
