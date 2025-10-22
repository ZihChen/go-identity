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

	expectedTag := entity.NewTagWithTimes(
		1,
		"test-tag-01",
		"Test-Tag-01",
		now,
	)
	expectedTag.SetID(2)

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
			expectedTag:   expectedTag,
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

	expectedTag1 := entity.NewTagWithTimes(
		1,
		"test-tag-01",
		"Test-Tag-01",
		now,
	)
	expectedTag1.SetID(2)

	expectedTag2 := entity.NewTagWithTimes(
		1,
		"test-tag-02",
		"Test-Tag-02",
		now,
	)
	expectedTag2.SetID(3)

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
		expectedTags:  []*entity.Tag{expectedTag1, expectedTag2},
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

	expectedTag1 := entity.NewTagWithTimes(
		1,
		"test-tag-01",
		"Test-Tag-01",
		now,
	)
	expectedTag1.SetID(2)

	expectedTag2 := entity.NewTagWithTimes(
		1,
		"test-tag-02",
		"Test-Tag-02",
		now,
	)
	expectedTag2.SetID(3)

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
		expectedTags:  []*entity.Tag{expectedTag1, expectedTag2},
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
		assert.Equal(t, testCase.expectedTags[k].GetID(), tag.GetID())
		assert.Equal(t, testCase.expectedTags[k].GetGlobalTagID(), tag.GetGlobalTagID())
		assert.Equal(t, testCase.expectedTags[k].GetName(), tag.GetName())
	}
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
