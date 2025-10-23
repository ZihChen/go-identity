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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func setupMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
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

func TestLevelRepository_Upsert(t *testing.T) {
	testCases := []struct {
		name          string
		level         *entity.Level
		setupMock     func(sqlmock.Sqlmock)
		expectedError bool
	}{
		{
			name: "successful upsert",
			level: func() *entity.Level {
				now := time.Now()
				level := entity.NewLevelWithTimes(100, "VIP", "global-level-1", now, now)
				level.SetID(1)
				return level
			}(),
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `level` (`merchant_id`,`name`,`global_player_level_id`,`deleted_at`,`id`,`created_at`,`updated_at`) VALUES (?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE `name`=CASE WHEN VALUES(updated_at) > updated_at AND name != VALUES(name) THEN VALUES(name) ELSE name END,`updated_at`=CASE WHEN VALUES(updated_at) > updated_at THEN VALUES(updated_at) ELSE updated_at END")).
					WithArgs(
						sqlmock.AnyArg(), // MerchantID
						sqlmock.AnyArg(), // Name
						sqlmock.AnyArg(), // GlobalPlayerLevelID
						sqlmock.AnyArg(), // DeletedAt
						sqlmock.AnyArg(), // ID
						sqlmock.AnyArg(), // CreatedAt
						sqlmock.AnyArg(), // UpdatedAt
					).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedError: false,
		},
		{
			name: "upsert error",
			level: func() *entity.Level {
				now := time.Now()
				level := entity.NewLevelWithTimes(100, "VIP", "global-level-1", now, now)
				level.SetID(1)
				return level
			}(),
			setupMock: func(mock sqlmock.Sqlmock) {
				// Expect the SQL query for upsert but return an error
				mock.ExpectBegin()
				mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `level` (`merchant_id`,`name`,`global_player_level_id`,`deleted_at`,`id`,`created_at`,`updated_at`) VALUES (?,?,?,?,?,?,?) ON DUPLICATE KEY UPDATE `name`=CASE WHEN VALUES(updated_at) > updated_at AND name != VALUES(name) THEN VALUES(name) ELSE name END,`updated_at`=CASE WHEN VALUES(updated_at) > updated_at THEN VALUES(updated_at) ELSE updated_at END")).
					WithArgs(
						sqlmock.AnyArg(), // MerchantID
						sqlmock.AnyArg(), // Name
						sqlmock.AnyArg(), // GlobalPlayerLevelID
						sqlmock.AnyArg(), // DeletedAt
						sqlmock.AnyArg(), // ID
						sqlmock.AnyArg(), // CreatedAt
						sqlmock.AnyArg(), // UpdatedAt
					).
					WillReturnError(errors.New("database error"))
				mock.ExpectRollback()
			},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)

			repo := NewLevelRepository(db)

			err := repo.Upsert(context.Background(), tc.level)

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestLevelRepository_FindByGlobalID(t *testing.T) {
	// Setup test cases
	now := time.Now()
	testCases := []struct {
		name          string
		globalID      string
		setupMock     func(sqlmock.Sqlmock)
		expectedLevel *entity.Level
		expectedError error
	}{
		{
			name:     "level found",
			globalID: "global-level-1",
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "merchant_id", "name", "global_player_level_id", "created_at", "updated_at", "deleted_at"}).
					AddRow(1, 100, "VIP", "global-level-1", now, now, nil)

				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `level` WHERE global_player_level_id = ? AND `level`.`deleted_at` IS NULL ORDER BY `level`.`id` LIMIT ?")).
					WithArgs("global-level-1", 1).
					WillReturnRows(rows)
			},
			expectedLevel: func() *entity.Level {
				level := entity.NewLevelWithTimes(100, "VIP", "global-level-1", now, now)
				level.SetID(1)
				return level
			}(),
			expectedError: nil,
		},
		{
			name:     "level not found",
			globalID: "non-existent",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `level` WHERE global_player_level_id = ? AND `level`.`deleted_at` IS NULL ORDER BY `level`.`id` LIMIT ?")).
					WithArgs("non-existent", 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			expectedLevel: &entity.Level{},
			expectedError: errmsg.ErrRepoLevelNotFound,
		},
		{
			name:     "database error",
			globalID: "global-level-1",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `level` WHERE global_player_level_id = ? AND `level`.`deleted_at` IS NULL ORDER BY `level`.`id` LIMIT ?")).
					WithArgs("global-level-1", 1).
					WillReturnError(errors.New("database error"))
			},
			expectedLevel: &entity.Level{},
			expectedError: errors.New("database error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)

			repo := NewLevelRepository(db)

			level, err := repo.FindByGlobalID(context.Background(), tc.globalID)

			if tc.expectedError != nil {
				assert.Error(t, err)
				if tc.expectedError == errmsg.ErrRepoLevelNotFound {
					assert.ErrorIs(t, err, errmsg.ErrRepoLevelNotFound)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedLevel.GetID(), level.GetID())
				assert.Equal(t, tc.expectedLevel.GetMerchantID(), level.GetMerchantID())
				assert.Equal(t, tc.expectedLevel.GetGlobalPlayerLevelID(), level.GetGlobalPlayerLevelID())
				assert.Equal(t, tc.expectedLevel.GetName(), level.GetName())
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
