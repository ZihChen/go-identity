package entity

import (
	"encoding/json"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/errmsg"
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

	return manager
}

// NewManagerWithTimes 建立帶有指定時間戳的Manager實體
func NewManagerWithTimes(
	merchantID uint64,
	globalManagerID, account string,
	email *string,
	createdAt, updatedAt time.Time,
) *Manager {
	manager := &Manager{
		id:              0,
		merchantID:      merchantID,
		globalManagerID: globalManagerID,
		account:         account,
		email:           email,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
	}

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
}

func (m *Manager) SetDeletedAt(deletedAt *time.Time) {
	m.deletedAt = deletedAt
	m.updatedAt = time.Now()
}

// IsValid 驗證方法 for Manager
func (m *Manager) IsValid() error {
	if m.merchantID == 0 {
		return errmsg.ErrManagerNoMerchant
	}
	if m.globalManagerID == "" {
		return errmsg.ErrManagerGlobalIDEmpty
	}
	if m.account == "" {
		return errmsg.ErrManagerAccountEmpty
	}
	return nil
}

func (m *Manager) IsDeleted() bool {
	return m.deletedAt != nil
}

func (m *Manager) SetID(id uint64) {
	m.id = id
}

// MarshalJSON
//  1. Go 的 json 包無法序列化私有欄位
//  2. 當物件實作了 json.Marshaler 和 json.Unmarshaler interface 時，json.Marshal 和 json.Unmarshal 會自動使用這些方法
//  3. 沒有這些方法，HTTP API 返回的會是零值
func (m *Manager) MarshalJSON() ([]byte, error) {
	type Alias struct {
		ID              uint64     `json:"id"`
		MerchantID      uint64     `json:"global_merchant_id"`
		GlobalManagerID string     `json:"name"`
		Account         string     `json:"display_name"`
		Email           *string    `json:"email,omitempty"`
		CreatedAt       time.Time  `json:"created_at"`
		UpdatedAt       time.Time  `json:"updated_at"`
		DeletedAt       *time.Time `json:"deleted_at,omitempty"`
	}
	return json.Marshal(Alias{
		ID:              m.id,
		MerchantID:      m.merchantID,
		GlobalManagerID: m.globalManagerID,
		Account:         m.account,
		Email:           m.email,
		CreatedAt:       m.createdAt,
		UpdatedAt:       m.updatedAt,
		DeletedAt:       m.deletedAt,
	})
}

// UnmarshalJSON implements custom JSON unmarshaling
func (m *Manager) UnmarshalJSON(data []byte) error {
	type Alias struct {
		ID              uint64     `json:"id"`
		MerchantID      uint64     `json:"global_merchant_id"`
		GlobalManagerID string     `json:"name"`
		Account         string     `json:"display_name"`
		Email           *string    `json:"email,omitempty"`
		CreatedAt       time.Time  `json:"created_at"`
		UpdatedAt       time.Time  `json:"updated_at"`
		DeletedAt       *time.Time `json:"deleted_at,omitempty"`
	}
	var aux Alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	m.id = aux.ID
	m.merchantID = aux.MerchantID
	m.globalManagerID = aux.GlobalManagerID
	m.account = aux.Account
	m.email = aux.Email
	m.createdAt = aux.CreatedAt
	m.updatedAt = aux.UpdatedAt
	m.deletedAt = aux.DeletedAt
	return nil
}
