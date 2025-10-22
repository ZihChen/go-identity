package entity

import (
	"errors"
	"time"
)

// Manager 管理員模型
type Manager struct {
	// 私有欄位
	id              uint64
	merchantID      uint64
	globalManagerID string
	account         string
	email           *string
	createdAt       time.Time
	updatedAt       time.Time
	deletedAt       *time.Time

	// 向後兼容的公共欄位
	ID              uint64     `json:"id" deprecated:"use GetID() method instead"`
	MerchantID      uint64     `json:"merchant_id" deprecated:"use GetMerchantID() method instead"`
	GlobalManagerID string     `json:"global_manager_id" deprecated:"use GetGlobalManagerID() method instead"`
	Account         string     `json:"account" deprecated:"use GetAccount() method instead"`
	Email           *string    `json:"email,omitempty" deprecated:"use GetEmail() method instead"`
	CreatedAt       time.Time  `json:"created_at" deprecated:"use GetCreatedAt() method instead"`
	UpdatedAt       time.Time  `json:"updated_at" deprecated:"use GetUpdatedAt() method instead"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty" deprecated:"use GetDeletedAt() method instead"`
}

// NewManager 建立新的Manager實體
func NewManager(merchantID uint64, globalManagerID, account string, email *string) *Manager {
	now := time.Now()
	manager := &Manager{
		id:              0,
		merchantID:      merchantID,
		globalManagerID: globalManagerID,
		account:         account,
		email:           email,
		createdAt:       now,
		updatedAt:       now,
	}

	manager.syncManagerFields()
	return manager
}

// Getter methods for Manager
func (m *Manager) GetID() uint64              { return m.id }
func (m *Manager) GetMerchantID() uint64      { return m.merchantID }
func (m *Manager) GetGlobalManagerID() string { return m.globalManagerID }
func (m *Manager) GetAccount() string         { return m.account }
func (m *Manager) GetEmail() *string          { return m.email }
func (m *Manager) GetCreatedAt() time.Time    { return m.createdAt }
func (m *Manager) GetUpdatedAt() time.Time    { return m.updatedAt }
func (m *Manager) GetDeletedAt() *time.Time   { return m.deletedAt }

// UpdateEmail 業務方法 for Manager
func (m *Manager) UpdateEmail(newEmail *string) {
	m.email = newEmail
	m.updatedAt = time.Now()
	m.syncManagerFields()
}

func (m *Manager) SetDeletedAt(deletedAt *time.Time) {
	m.deletedAt = deletedAt
	m.updatedAt = time.Now()
	m.syncManagerFields()
}

// IsValid 驗證方法 for Manager
func (m *Manager) IsValid() error {
	if m.merchantID == 0 {
		return errors.New("manager must belong to a merchant")
	}
	if m.globalManagerID == "" {
		return errors.New("manager global ID cannot be empty")
	}
	if m.account == "" {
		return errors.New("manager account cannot be empty")
	}
	return nil
}

func (m *Manager) IsDeleted() bool {
	return m.deletedAt != nil
}

// 同步方法 for Manager
func (m *Manager) syncManagerFields() {
	m.ID = m.id
	m.MerchantID = m.merchantID
	m.GlobalManagerID = m.globalManagerID
	m.Account = m.account
	m.Email = m.email
	m.CreatedAt = m.createdAt
	m.UpdatedAt = m.updatedAt
	m.DeletedAt = m.deletedAt
}

func (m *Manager) SetID(id uint64) {
	m.id = id
	m.syncManagerFields()
}
