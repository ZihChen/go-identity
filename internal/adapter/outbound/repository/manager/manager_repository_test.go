package repository

import (
	"context"
	"database/sql"
	"regexp"
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

type ManagerTestCase struct {
	name            string
	setupMock       func(sqlmock.Sqlmock)
	expectedManager *entity.Manager
	expectedError   error
}

func setupManagerMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
	// Create a new SQL mock
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	// Create a GORM DB instance using the mock database
	dialector := mysql.New(mysql.Config{
		Conn:                      mockDB,
		SkipInitializeWithVersion: true,
	})

	db, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)

	return db, mock, mockDB
}

func TestManagerRepository_FindByID(t *testing.T) {
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
	assert.Equal(t, expectedManager.MerchantID, result.GetMerchantID())
	assert.Equal(t, expectedManager.GlobalManagerID, result.GetGlobalManagerID())
	assert.Equal(t, expectedManager.Account, result.GetAccount())
	assert.Equal(t, *expectedManager.Email, *result.GetEmail())

	// 驗證所有 SQL 期望都被滿足
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestManagerRepository_FindByGlobalID(t *testing.T) {
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
	assert.Equal(t, expectedManager.ID, result.GetID())
	assert.Equal(t, expectedManager.MerchantID, result.GetMerchantID())
	assert.Equal(t, expectedManager.GlobalManagerID, result.GetGlobalManagerID())
	assert.Equal(t, expectedManager.Account, result.GetAccount())
	assert.Equal(t, *expectedManager.Email, *result.GetEmail())

	// 驗證所有 SQL 期望都被滿足
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestManagerRepository_Create(t *testing.T) {
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

	// Convert to domain entity for the test using constructor
	domainManager := entity.NewManagerWithTimes(
		manager.MerchantID,
		manager.GlobalManagerID,
		manager.Account,
		manager.Email,
		manager.CreatedAt,
		manager.UpdatedAt,
	)

	// Mock the SQL execution
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `managers`").
		WillReturnResult(sqlmock.NewResult(1, 1)) // ID = 1, affected rows = 1
	mock.ExpectCommit()

	// 執行測試
	err = repo.Create(context.Background(), domainManager)

	// 驗證結果
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), domainManager.GetID()) // ID should be updated

	// 驗證所有 SQL 期望都被滿足
	err = mock.ExpectationsWereMet()
	assert.NoError(t, err)
}

func TestManagerRepository_Update(t *testing.T) {
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
	manager := entity.NewManagerWithTimes(
		uint64(2),
		"FATCAT-MANAGER-1",
		"UpdatedManager",
		&email,
		now,
		now,
	)
	manager.SetID(managerID)

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

func TestManagerRepository_Delete(t *testing.T) {
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

func TestManagerRepository_Upsert(t *testing.T) {
	// Setup test cases
	now := time.Now()
	email := "test@example.com"
	testCases := []ManagerTestCase{
		{
			name: "successful upsert",
			expectedManager: func() *entity.Manager {
				manager := entity.NewManagerWithTimes(
					100,
					"global-manager-1",
					"testaccount",
					&email,
					now,
					now,
				)
				return manager
			}(),
			setupMock: func(mock sqlmock.Sqlmock) {
				// Expect upsert
				mock.ExpectBegin()
				pattern := regexp.QuoteMeta(
					"INSERT INTO `managers`",
				) + ".*" + regexp.QuoteMeta(
					"ON DUPLICATE KEY UPDATE",
				)
				mock.ExpectExec(pattern).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Setup mock DB
			db, mock, sqlDB := setupManagerMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)

			repo := NewManagerRepository(db)

			err := repo.Upsert(context.Background(), tc.expectedManager)

			if tc.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestManagerRepository_FirstOrCreate(t *testing.T) {
	now := time.Now()
	email := "test@example.com"
	testCases := ManagerTestCase{
		name: "manager first or create",
		setupMock: func(mock sqlmock.Sqlmock) {
			rows := sqlmock.NewRows([]string{"id", "merchant_id", "global_manager_id", "account", "email", "created_at", "updated_at", "deleted_at"}).
				AddRow(1, 100, "global-manager-1", "testaccount", email, now, now, nil)

			mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `managers`")).
				WithArgs("global-manager-1", 1, 1).
				WillReturnRows(rows)
		},
		expectedManager: func() *entity.Manager {
			manager := entity.NewManagerWithTimes(
				100,
				"global-manager-1",
				"testaccount",
				&email,
				now,
				now,
			)
			manager.SetID(1)
			return manager
		}(),
		expectedError: nil,
	}

	db, mock, sqlDB := setupManagerMockDB(t)
	defer func() {
		_ = sqlDB.Close()
	}()

	testCases.setupMock(mock)
	repo := NewManagerRepository(db)

	err := repo.FirstOrCreate(context.Background(), testCases.expectedManager)
	if testCases.expectedError != nil {
		assert.Error(t, err)
	} else {
		assert.NoError(t, err)
	}
	assert.NoError(t, mock.ExpectationsWereMet())
}
