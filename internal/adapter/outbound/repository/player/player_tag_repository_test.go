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

type PlayerTagTestCase struct {
	name               string
	id                 uint64
	setupMock          func(sqlmock.Sqlmock)
	expectedPlayerTags []*entity.PlayerTag
	expectedError      error
}

func setupPlayerTagMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
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

func TestPlayerTagRepository_BatchUpdate(t *testing.T) {
	now := time.Now()
	testCase := PlayerTagTestCase{
		name: "player tag upsert",
		id:   1,
		setupMock: func(mock sqlmock.Sqlmock) {
			// 優化後的邏輯：先查詢現有標籤
			mock.ExpectQuery(regexp.QuoteMeta("SELECT `tag_id` FROM `player_tags` WHERE player_id = ?")).
				WithArgs(2).
				WillReturnRows(sqlmock.NewRows([]string{"tag_id"}).AddRow(1).AddRow(2))

			mock.ExpectBegin()

			// 刪除不需要的標籤 (1, 2 不在新的 [3, 4] 中)
			mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `player_tags` WHERE player_id = ? AND tag_id IN")).
				WithArgs(2, 1, 2).
				WillReturnResult(sqlmock.NewResult(0, 2))

			// 插入新標籤 ([3, 4] 不在現有的 [1, 2] 中)
			mock.ExpectExec(regexp.QuoteMeta("INSERT INTO `player_tags`")).
				WillReturnResult(sqlmock.NewResult(0, 2))

			mock.ExpectCommit()
		},
		expectedPlayerTags: []*entity.PlayerTag{
			{
				PlayerID:  2,
				TagID:     3,
				CreatedAt: now,
			},
			{
				PlayerID:  2,
				TagID:     4,
				CreatedAt: now,
			},
		},
		expectedError: nil,
	}

	db, mock, sqlDB := setupPlayerTagMockDB(t)
	defer func() {
		_ = sqlDB.Close()
	}()
	testCase.setupMock(mock)
	repo := NewPlayerTagRepository(db)

	err := repo.BatchUpdate(context.Background(), 2, []uint64{3, 4})

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestPlayerTagRepository_DeleteByPlayerID(t *testing.T) {
	now := time.Now()
	testCase := PlayerTagTestCase{
		name: "player tag delete",
		id:   1,
		setupMock: func(mock sqlmock.Sqlmock) {
			mock.ExpectBegin()
			mock.ExpectExec(regexp.QuoteMeta("DELETE FROM `player_tags`")).
				WithArgs(2).
				WillReturnResult(sqlmock.NewResult(0, 2))
			mock.ExpectCommit()
		},
		expectedPlayerTags: []*entity.PlayerTag{
			{
				PlayerID:  2,
				TagID:     3,
				CreatedAt: now,
			},
			{
				PlayerID:  2,
				TagID:     4,
				CreatedAt: now,
			},
		},
		expectedError: nil,
	}

	db, mock, sqlDB := setupPlayerTagMockDB(t)
	defer func() {
		_ = sqlDB.Close()
	}()
	testCase.setupMock(mock)
	repo := NewPlayerTagRepository(db)

	err := repo.DeleteByPlayerID(context.Background(), 2)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
