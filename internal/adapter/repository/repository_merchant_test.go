package repository

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/jvdiamondtech/ms-identity-cat/internal/infrastructure/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestMerchantRepository_FindByID_Unit(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	decorator := mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	})
	db, err := gorm.Open(decorator, &gorm.Config{})
	require.NoError(t, err)

	repo := NewMerchantRepository(db)

	merchantID := uint64(1)
	expectedMerchant := &models.Merchant{
		ID:               merchantID,
		GlobalMerchantID: "FATCAT-MERCHANT-1",
		Name:             "TestMerchant",
		DisplayName:      "Test Merchant",
		APIKey:           "test-api-key",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// 塞入假資料到mock db
	rows := sqlmock.NewRows([]string{"id", "name", "display_name", "global_merchant_id", "api_key", "created_at", "updated_at"}).
		AddRow(expectedMerchant.ID, expectedMerchant.Name, expectedMerchant.DisplayName, expectedMerchant.GlobalMerchantID, expectedMerchant.APIKey, expectedMerchant.CreatedAt, expectedMerchant.UpdatedAt)

	mock.ExpectQuery("SELECT"). // 寬鬆模式
					WithArgs(merchantID, 1).
					WillReturnRows(rows)

	// 執行測試
	result, err := repo.FindByID(context.Background(), merchantID)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedMerchant.Name, result.Name)
	assert.Equal(t, expectedMerchant.DisplayName, result.DisplayName)
	assert.Equal(t, expectedMerchant.GlobalMerchantID, result.GlobalMerchantID)
	assert.Equal(t, expectedMerchant.APIKey, result.APIKey)

	// 驗證所有 SQL 期望都被滿足
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestMerchantRepository_FindByGlobalID_Unit(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	decorator := mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	})
	db, err := gorm.Open(decorator, &gorm.Config{})
	require.NoError(t, err)

	repo := NewMerchantRepository(db)

	globalID := "FATCAT-MERCHANT-1"
	expectedMerchant := &models.Merchant{
		ID:               uint64(1),
		GlobalMerchantID: globalID,
		Name:             "TestMerchant",
		DisplayName:      "Test Merchant",
		APIKey:           "test-api-key",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	// 塞入假資料到mock db
	rows := sqlmock.NewRows([]string{"id", "name", "display_name", "global_merchant_id", "api_key", "created_at", "updated_at"}).
		AddRow(expectedMerchant.ID, expectedMerchant.Name, expectedMerchant.DisplayName, expectedMerchant.GlobalMerchantID, expectedMerchant.APIKey, expectedMerchant.CreatedAt, expectedMerchant.UpdatedAt)

	mock.ExpectQuery("SELECT").
		WithArgs(globalID, 1).
		WillReturnRows(rows)

	// 執行測試
	result, err := repo.FindByGlobalID(context.Background(), globalID)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedMerchant.ID, result.ID)
	assert.Equal(t, expectedMerchant.Name, result.Name)
	assert.Equal(t, expectedMerchant.DisplayName, result.DisplayName)
	assert.Equal(t, expectedMerchant.GlobalMerchantID, result.GlobalMerchantID)
	assert.Equal(t, expectedMerchant.APIKey, result.APIKey)

	// 驗證所有 SQL 期望都被滿足
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestMerchantRepository_Create_Unit(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	decorator := mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	})
	db, err := gorm.Open(decorator, &gorm.Config{})
	require.NoError(t, err)

	repo := NewMerchantRepository(db)

	now := time.Now()
	merchant := &models.Merchant{
		GlobalMerchantID: "FATCAT-MERCHANT-NEW",
		Name:             "NewMerchant",
		DisplayName:      "New Merchant",
		APIKey:           "new-api-key",
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Convert to domain entity for the test
	domainMerchant := &entity.Merchant{
		GlobalMerchantID: merchant.GlobalMerchantID,
		Name:             merchant.Name,
		DisplayName:      merchant.DisplayName,
		APIKey:           merchant.APIKey,
		CreatedAt:        merchant.CreatedAt,
		UpdatedAt:        merchant.UpdatedAt,
	}

	// Mock the SQL execution
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `merchants`").
		WillReturnResult(sqlmock.NewResult(1, 1)) // ID = 1, affected rows = 1
	mock.ExpectCommit()

	// 執行測試
	err = repo.Create(context.Background(), domainMerchant)

	// 驗證結果
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), domainMerchant.ID) // ID should be updated

	// 驗證所有 SQL 期望都被滿足
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestMerchantRepository_Update_Unit(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	decorator := mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	})
	db, err := gorm.Open(decorator, &gorm.Config{})
	require.NoError(t, err)

	repo := NewMerchantRepository(db)

	now := time.Now()
	merchantID := uint64(1)
	merchant := &entity.Merchant{
		ID:               merchantID,
		GlobalMerchantID: "FATCAT-MERCHANT-1",
		Name:             "UpdatedMerchant",
		DisplayName:      "Updated Merchant",
		APIKey:           "updated-api-key",
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Mock the SQL execution
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE").
		WillReturnResult(sqlmock.NewResult(0, 1)) // No new ID, 1 row affected
	mock.ExpectCommit()

	// 執行測試
	err = repo.Update(context.Background(), merchant)

	// 驗證結果
	assert.NoError(t, err)

	// 驗證所有 SQL 期望都被滿足
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestMerchantRepository_Delete_Unit(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer sqlDB.Close()

	decorator := mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	})
	db, err := gorm.Open(decorator, &gorm.Config{})
	require.NoError(t, err)

	repo := NewMerchantRepository(db)

	merchantID := uint64(1)

	// Mock the SQL execution
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE").
		WithArgs(sqlmock.AnyArg(), merchantID).
		WillReturnResult(sqlmock.NewResult(0, 1)) // No new ID, 1 row affected
	mock.ExpectCommit()

	// 執行測試
	err = repo.Delete(context.Background(), merchantID)

	// 驗證結果
	assert.NoError(t, err)

	// 驗證所有 SQL 期望都被滿足
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}
