package repository

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type TagTestCase struct {
	name          string
	id            uint64
	setupMock     func(sqlmock.Sqlmock)
	expectedTag   *entity.Tag
	expectedTags  []*entity.Tag
	expectedError error
}

func setupTagMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
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

func TestTagRepository_Upsert(t *testing.T) {
	now := time.Now()
	testCases := []TagTestCase{
		{
			name: "tag upsert",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				pattern := regexp.QuoteMeta(
					"INSERT INTO `tags`",
				) + ".*" + regexp.QuoteMeta(
					"ON DUPLICATE KEY UPDATE",
				)
				mock.ExpectExec(pattern).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedTag: &entity.Tag{
				ID:          2,
				MerchantID:  1,
				Name:        "test-tag-01",
				GlobalTagID: "Test-Tag-01",
				CreatedAt:   now,
				UpdatedAt:   now,
				DeletedAt:   nil,
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, sqlDB := setupTagMockDB(t)
			defer func() {
				_ = sqlDB.Close()
			}()

			tc.setupMock(mock)
			repo := NewTagRepository(db)

			err := repo.Upsert(context.Background(), tc.expectedTag)
			if tc.expectedError != nil {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestTagRepository_BatchUpsert(t *testing.T) {
	now := time.Now()
	testCase := TagTestCase{
		name: "tags upsert",
		id:   1,
		setupMock: func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			pattern := regexp.QuoteMeta(
				"INSERT INTO `tags`",
			) + ".*" + regexp.QuoteMeta(
				"ON DUPLICATE KEY UPDATE",
			)
			mock.ExpectExec(pattern).
				WillReturnResult(sqlmock.NewResult(1, 2))
			mock.ExpectCommit()
		},
		expectedTags: []*entity.Tag{
			{
				ID:          2,
				MerchantID:  1,
				GlobalTagID: "Test-Tag-01",
				Name:        "test-tag-01",
				CreatedAt:   now,
				UpdatedAt:   now,
				DeletedAt:   nil,
			},
			{
				ID:          3,
				MerchantID:  1,
				GlobalTagID: "Test-Tag-02",
				Name:        "test-tag-02",
				CreatedAt:   now,
				UpdatedAt:   now,
				DeletedAt:   nil,
			},
		},
		expectedError: nil,
	}

	db, mock, sqlDB := setupTagMockDB(t)
	defer func() {
		_ = sqlDB.Close()
	}()
	testCase.setupMock(mock)
	repo := NewTagRepository(db)

	err := repo.BatchUpsert(context.Background(), testCase.expectedTags)
	if testCase.expectedError != nil {
		assert.Error(t, err)
	} else {
		assert.NoError(t, err)
	}
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestTagRepository_FindByGlobalIDs(t *testing.T) {
	now := time.Now()
	testCase := TagTestCase{
		name: "tag found",
		id:   1,
		setupMock: func(mock sqlmock.Sqlmock) {
			rows := sqlmock.NewRows([]string{"id", "merchant_id", "global_tag_id", "name", "created_at", "updated_at", "deleted_at"}).
				AddRow(2, 1, "Test-Tag-01", "test-tag-01", now, now, nil).
				AddRow(3, 1, "Test-Tag-02", "test-tag-02", now, now, nil)

			mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `tags`")).
				WithArgs("Test-Tag-01", "Test-Tag-02").
				WillReturnRows(rows)
		},
		expectedTags: []*entity.Tag{
			{
				ID:          2,
				MerchantID:  1,
				GlobalTagID: "Test-Tag-01",
				Name:        "test-tag-01",
				CreatedAt:   now,
				UpdatedAt:   now,
				DeletedAt:   nil,
			},
			{
				ID:          3,
				MerchantID:  1,
				GlobalTagID: "Test-Tag-02",
				Name:        "test-tag-02",
				CreatedAt:   now,
				UpdatedAt:   now,
				DeletedAt:   nil,
			},
		},
		expectedError: nil,
	}

	db, mock, sqlDB := setupTagMockDB(t)
	defer func() {
		_ = sqlDB.Close()
	}()
	testCase.setupMock(mock)
	repo := NewTagRepository(db)

	tags, err := repo.FindByGlobalIDs(context.Background(), []string{"Test-Tag-01", "Test-Tag-02"})
	for k, tag := range tags {
		assert.Equal(t, testCase.expectedTags[k].ID, tag.ID)
		assert.Equal(t, testCase.expectedTags[k].GlobalTagID, tag.GlobalTagID)
		assert.Equal(t, testCase.expectedTags[k].Name, tag.Name)
	}
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
