package repository

import (
	"context"
	"database/sql"
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

type AgentTestCase struct {
	name          string
	id            uint64
	setupMock     func(sqlmock.Sqlmock)
	expectedAgent *entity.Agent
	expectedError error
}

func setupAgentMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
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

func TestAgentRepository_FindByID(t *testing.T) {
	sqlDB, mock, err := sqlmock.New()
	require.NoError(t, err)

	defer func() {
		_ = sqlDB.Close()
	}()

	// Create GORM DB instance
	dialector := mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)

	repo := NewAgentRepository(db)

	testCases := []AgentTestCase{
		{
			name: "Success - Agent found",
			id:   1,
			setupMock: func(mock sqlmock.Sqlmock) {
				now := time.Now()
				expectedAgent := &models.Agent{
					ID:              1,
					MerchantID:      123,
					GlobalAgentID:   "agent_123",
					Account:         "test_agent",
					Ancestry:        "1/2/3",
					CurrentSignInAt: &now,
					CreatedAt:       now,
					UpdatedAt:       now,
				}

				rows := sqlmock.NewRows([]string{"id", "merchant_id", "global_agent_id", "account", "ancestry", "current_sign_in_at", "created_at", "updated_at", "deleted_at"}).
					AddRow(expectedAgent.ID, expectedAgent.MerchantID, expectedAgent.GlobalAgentID, expectedAgent.Account, expectedAgent.Ancestry, expectedAgent.CurrentSignInAt, expectedAgent.CreatedAt, expectedAgent.UpdatedAt, nil)

				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `agents` WHERE `agents`.`id` = ? AND `agents`.`deleted_at` IS NULL ORDER BY `agents`.`id` LIMIT ?")).
					WithArgs(1, 1).
					WillReturnRows(rows)
			},
			expectedError: nil,
		},
		{
			name: "Error - Agent not found",
			id:   999,
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `agents` WHERE `agents`.`id` = ? AND `agents`.`deleted_at` IS NULL ORDER BY `agents`.`id` LIMIT ?")).
					WithArgs(999, 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			expectedError: errmsg.ErrRepoAgentNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupMock(mock)

			agent, err := repo.FindByID(context.Background(), tc.id)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedError, err)
				assert.Nil(t, agent)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, agent)
				assert.Equal(t, tc.id, agent.GetID())
			}

			// Verify all expectations were met
			err = mock.ExpectationsWereMet()
			assert.NoError(t, err)
		})
	}
}

func TestAgentRepository_FindByGlobalID(t *testing.T) {
	db, mock, sqlDB := setupAgentMockDB(t)
	defer func() {
		_ = sqlDB.Close()
	}()

	repo := NewAgentRepository(db)

	testCases := []struct {
		name          string
		globalID      string
		setupMock     func(sqlmock.Sqlmock)
		expectedError error
	}{
		{
			name:     "Success - Agent found",
			globalID: "agent_123",
			setupMock: func(mock sqlmock.Sqlmock) {
				now := time.Now()
				expectedAgent := &models.Agent{
					ID:              1,
					MerchantID:      123,
					GlobalAgentID:   "agent_123",
					Account:         "test_agent",
					Ancestry:        "1/2/3",
					CurrentSignInAt: &now,
					CreatedAt:       now,
					UpdatedAt:       now,
				}

				rows := sqlmock.NewRows([]string{"id", "merchant_id", "global_agent_id", "account", "ancestry", "current_sign_in_at", "created_at", "updated_at", "deleted_at"}).
					AddRow(expectedAgent.ID, expectedAgent.MerchantID, expectedAgent.GlobalAgentID, expectedAgent.Account, expectedAgent.Ancestry, expectedAgent.CurrentSignInAt, expectedAgent.CreatedAt, expectedAgent.UpdatedAt, nil)

				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `agents` WHERE global_agent_id = ? AND `agents`.`deleted_at` IS NULL ORDER BY `agents`.`id` LIMIT ?")).
					WithArgs("agent_123", 1).
					WillReturnRows(rows)
			},
			expectedError: nil,
		},
		{
			name:     "Error - Agent not found",
			globalID: "nonexistent_agent",
			setupMock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `agents` WHERE global_agent_id = ? AND `agents`.`deleted_at` IS NULL ORDER BY `agents`.`id` LIMIT ?")).
					WithArgs("nonexistent_agent", 1).
					WillReturnError(gorm.ErrRecordNotFound)
			},
			expectedError: errmsg.ErrRepoAgentNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupMock(mock)

			agent, err := repo.FindByGlobalID(context.Background(), tc.globalID)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedError, err)
				assert.Nil(t, agent)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, agent)
				assert.Equal(t, tc.globalID, agent.GetGlobalAgentID())
			}

			// Verify all expectations were met
			err = mock.ExpectationsWereMet()
			assert.NoError(t, err)
		})
	}
}

func TestAgentRepository_FindByMerchantID(t *testing.T) {
	db, mock, sqlDB := setupAgentMockDB(t)
	defer func() {
		_ = sqlDB.Close()
	}()

	repo := NewAgentRepository(db)

	testCases := []struct {
		name          string
		merchantID    uint64
		setupMock     func(sqlmock.Sqlmock)
		expectedCount int
		expectedError error
	}{
		{
			name:       "Success - Multiple agents found",
			merchantID: 123,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "merchant_id", "global_agent_id", "account", "ancestry", "current_sign_in_at", "created_at", "updated_at", "deleted_at"}).
					AddRow(1, 123, "agent_123", "agent1", "1/2/3", time.Now(), time.Now(), time.Now(), nil).
					AddRow(2, 123, "agent_124", "agent2", "1/2/4", time.Now(), time.Now(), time.Now(), nil)

				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `agents` WHERE merchant_id = ? AND `agents`.`deleted_at` IS NULL")).
					WithArgs(123).
					WillReturnRows(rows)
			},
			expectedCount: 2,
			expectedError: nil,
		},
		{
			name:       "Success - No agents found",
			merchantID: 999,
			setupMock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows(
					[]string{
						"id",
						"merchant_id",
						"global_agent_id",
						"account",
						"ancestry",
						"current_sign_in_at",
						"created_at",
						"updated_at",
						"deleted_at",
					},
				)

				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `agents` WHERE merchant_id = ? AND `agents`.`deleted_at` IS NULL")).
					WithArgs(999).
					WillReturnRows(rows)
			},
			expectedCount: 0,
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupMock(mock)

			agents, err := repo.FindByMerchantID(context.Background(), tc.merchantID)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tc.expectedError, err)
				assert.Nil(t, agents)
			} else {
				assert.NoError(t, err)
				assert.Len(t, agents, tc.expectedCount)
				for _, agent := range agents {
					assert.Equal(t, tc.merchantID, agent.GetMerchantID())
				}
			}

			// Verify all expectations were met
			err = mock.ExpectationsWereMet()
			assert.NoError(t, err)
		})
	}
}

func TestAgentRepository_Upsert(t *testing.T) {
	db, mock, sqlDB := setupAgentMockDB(t)
	defer func() {
		_ = sqlDB.Close()
	}()

	repo := NewAgentRepository(db)

	testCases := []struct {
		name          string
		agent         *entity.Agent
		setupMock     func(sqlmock.Sqlmock, *entity.Agent)
		expectedError error
	}{
		{
			name: "Success - Upsert agent",
			agent: func() *entity.Agent {
				now := time.Now()
				return entity.NewAgentWithTimes(
					123,
					"agent_123",
					"test_agent",
					"1/2/3",
					&now,
					now,
					now,
				)
			}(),
			setupMock: func(mock sqlmock.Sqlmock, agent *entity.Agent) {
				// Expect transaction
				mock.ExpectBegin()
				// Expect INSERT with ON DUPLICATE KEY UPDATE - use simpler pattern
				mock.ExpectExec("INSERT INTO").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			expectedError: nil,
		},
		{
			name: "Error - Database error",
			agent: func() *entity.Agent {
				now := time.Now()
				return entity.NewAgentWithTimes(
					123,
					"agent_123",
					"test_agent",
					"1/2/3",
					&now,
					now,
					now,
				)
			}(),
			setupMock: func(mock sqlmock.Sqlmock, agent *entity.Agent) {
				// Expect transaction
				mock.ExpectBegin()
				// Expect INSERT to fail
				mock.ExpectExec("INSERT INTO").
					WillReturnError(gorm.ErrInvalidDB)
				mock.ExpectRollback()
			},
			expectedError: gorm.ErrInvalidDB,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupMock(mock, tc.agent)

			err := repo.Upsert(context.Background(), tc.agent)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "timestamp-based upsert failed")
			} else {
				assert.NoError(t, err)
			}

			// Verify all expectations were met
			err = mock.ExpectationsWereMet()
			assert.NoError(t, err)
		})
	}
}

func TestMapToDomainAgent(t *testing.T) {
	now := time.Now()
	agentModel := &models.Agent{
		ID:              1,
		MerchantID:      123,
		GlobalAgentID:   "agent_123",
		Account:         "test_agent",
		Ancestry:        "1/2/3",
		CurrentSignInAt: &now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	domainAgent := mapToDomainAgent(agentModel)

	assert.Equal(t, agentModel.ID, domainAgent.GetID())
	assert.Equal(t, agentModel.MerchantID, domainAgent.GetMerchantID())
	assert.Equal(t, agentModel.GlobalAgentID, domainAgent.GetGlobalAgentID())
	assert.Equal(t, agentModel.Account, domainAgent.GetAccount())
	assert.Equal(t, agentModel.Ancestry, domainAgent.GetAncestry())
	assert.Equal(t, agentModel.CurrentSignInAt, domainAgent.GetCurrentSignInAt())
	assert.Equal(t, agentModel.CreatedAt, domainAgent.GetCreatedAt())
	assert.Equal(t, agentModel.UpdatedAt, domainAgent.GetUpdatedAt())
	assert.Nil(t, domainAgent.GetDeletedAt())
}

func TestMapToDBAgent(t *testing.T) {
	now := time.Now()
	domainAgent := entity.NewAgentWithTimes(
		123,
		"agent_123",
		"test_agent",
		"1/2/3",
		&now,
		now,
		now,
	)
	domainAgent.SetID(1)

	dbAgent := mapToDBAgent(domainAgent)

	assert.Equal(t, domainAgent.GetID(), dbAgent.ID)
	assert.Equal(t, domainAgent.GetMerchantID(), dbAgent.MerchantID)
	assert.Equal(t, domainAgent.GetGlobalAgentID(), dbAgent.GlobalAgentID)
	assert.Equal(t, domainAgent.GetAccount(), dbAgent.Account)
	assert.Equal(t, domainAgent.GetAncestry(), dbAgent.Ancestry)
	assert.Equal(t, domainAgent.GetCurrentSignInAt(), dbAgent.CurrentSignInAt)
	assert.Equal(t, domainAgent.GetCreatedAt(), dbAgent.CreatedAt)
	assert.Equal(t, domainAgent.GetUpdatedAt(), dbAgent.UpdatedAt)
	assert.False(t, dbAgent.DeletedAt.Valid)
}
