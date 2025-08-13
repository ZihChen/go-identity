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
			mock.ExpectBegin()

			mock.ExpectExec(regexp.QuoteMeta("DELETE")).
				WithArgs(2).
				WillReturnResult(sqlmock.NewResult(0, 2))

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
