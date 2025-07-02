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

func TestManagerRepository_FindByID_Unit(t *testing.T) {
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

	repo := NewManagerRepository(db)

	managerID := uint64(1)
	email := "test@example.com"
	expectedManager := &models.Manager{
		ID:              managerID,
		MerchantID:      uint64(2),
		GlobalManagerID: "FATCAT-MANAGER-1",
		Account:         "TestManager",
		Email:           &email,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// 塞入假資料到mock db
	rows := sqlmock.NewRows([]string{"id", "merchant_id", "global_manager_id", "account", "email", "created_at", "updated_at"}).
		AddRow(expectedManager.ID, expectedManager.MerchantID, expectedManager.GlobalManagerID, expectedManager.Account, expectedManager.Email, expectedManager.CreatedAt, expectedManager.UpdatedAt)

	mock.ExpectQuery("SELECT"). // 寬鬆模式
					WithArgs(managerID, 1).
					WillReturnRows(rows)

	// 執行測試
	result, err := repo.FindByID(context.Background(), managerID)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedManager.MerchantID, result.MerchantID)
	assert.Equal(t, expectedManager.GlobalManagerID, result.GlobalManagerID)
	assert.Equal(t, expectedManager.Account, result.Account)
	assert.Equal(t, *expectedManager.Email, *result.Email)

	// 驗證所有 SQL 期望都被滿足
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestManagerRepository_FindByGlobalID_Unit(t *testing.T) {
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

	repo := NewManagerRepository(db)

	globalID := "FATCAT-MANAGER-1"
	email := "test@example.com"
	expectedManager := &models.Manager{
		ID:              uint64(1),
		MerchantID:      uint64(2),
		GlobalManagerID: globalID,
		Account:         "TestManager",
		Email:           &email,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// 塞入假資料到mock db
	rows := sqlmock.NewRows([]string{"id", "merchant_id", "global_manager_id", "account", "email", "created_at", "updated_at"}).
		AddRow(expectedManager.ID, expectedManager.MerchantID, expectedManager.GlobalManagerID, expectedManager.Account, expectedManager.Email, expectedManager.CreatedAt, expectedManager.UpdatedAt)

	mock.ExpectQuery("SELECT").
		WithArgs(globalID, 1).
		WillReturnRows(rows)

	// 執行測試
	result, err := repo.FindByGlobalID(context.Background(), globalID)

	// 驗證結果
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedManager.ID, result.ID)
	assert.Equal(t, expectedManager.MerchantID, result.MerchantID)
	assert.Equal(t, expectedManager.GlobalManagerID, result.GlobalManagerID)
	assert.Equal(t, expectedManager.Account, result.Account)
	assert.Equal(t, *expectedManager.Email, *result.Email)

	// 驗證所有 SQL 期望都被滿足
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestManagerRepository_Create_Unit(t *testing.T) {
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

	repo := NewManagerRepository(db)

	now := time.Now()
	email := "new@example.com"
	manager := &models.Manager{
		MerchantID:      uint64(2),
		GlobalManagerID: "FATCAT-MANAGER-NEW",
		Account:         "NewManager",
		Email:           &email,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Convert to domain entity for the test
	domainManager := &entity.Manager{
		MerchantID:      manager.MerchantID,
		GlobalManagerID: manager.GlobalManagerID,
		Account:         manager.Account,
		Email:           manager.Email,
		CreatedAt:       manager.CreatedAt,
		UpdatedAt:       manager.UpdatedAt,
	}

	// Mock the SQL execution
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `managers`").
		WillReturnResult(sqlmock.NewResult(1, 1)) // ID = 1, affected rows = 1
	mock.ExpectCommit()

	// 執行測試
	err = repo.Create(context.Background(), domainManager)

	// 驗證結果
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), domainManager.ID) // ID should be updated

	// 驗證所有 SQL 期望都被滿足
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestManagerRepository_Update_Unit(t *testing.T) {
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

	repo := NewManagerRepository(db)

	now := time.Now()
	managerID := uint64(1)
	email := "updated@example.com"
	manager := &entity.Manager{
		ID:              managerID,
		MerchantID:      uint64(2),
		GlobalManagerID: "FATCAT-MANAGER-1",
		Account:         "UpdatedManager",
		Email:           &email,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Mock the SQL execution
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE").
		WillReturnResult(sqlmock.NewResult(0, 1)) // No new ID, 1 row affected
	mock.ExpectCommit()

	// 執行測試
	err = repo.Update(context.Background(), manager)

	// 驗證結果
	assert.NoError(t, err)

	// 驗證所有 SQL 期望都被滿足
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestManagerRepository_Delete_Unit(t *testing.T) {
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

	repo := NewManagerRepository(db)

	managerID := uint64(1)

	// Mock the SQL execution
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE").
		WithArgs(sqlmock.AnyArg(), managerID).
		WillReturnResult(sqlmock.NewResult(0, 1)) // No new ID, 1 row affected
	mock.ExpectCommit()

	// 執行測試
	err = repo.Delete(context.Background(), managerID)

	// 驗證結果
	assert.NoError(t, err)

	// 驗證所有 SQL 期望都被滿足
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}
