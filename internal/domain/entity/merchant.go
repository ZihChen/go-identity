package entity

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Merchant 商戶模型
type Merchant struct {
	// 私有欄位
	id               uint64
	globalMerchantID string
	name             string
	displayName      string
	apiKey           string
	createdAt        time.Time
	updatedAt        time.Time
	deletedAt        *time.Time

	// 向後兼容的公共欄位（標記為 deprecated）
	ID               uint64     `json:"id" deprecated:"use GetID() method instead"`
	GlobalMerchantID string     `json:"global_merchant_id" deprecated:"use GetGlobalMerchantID() method instead"`
	Name             string     `json:"name" deprecated:"use GetName() method instead"`
	DisplayName      string     `json:"display_name" deprecated:"use GetDisplayName() method instead"`
	APIKey           string     `json:"api_key" deprecated:"use GetAPIKey() method instead"`
	CreatedAt        time.Time  `json:"created_at" deprecated:"use GetCreatedAt() method instead"`
	UpdatedAt        time.Time  `json:"updated_at" deprecated:"use GetUpdatedAt() method instead"`
	DeletedAt        *time.Time `json:"deleted_at,omitempty" deprecated:"use GetDeletedAt() method instead"`
}

// NewMerchant 建立新的Merchant實體
func NewMerchant(globalMerchantID, name string) *Merchant {
	now := time.Now()
	merchant := &Merchant{
		id:               0, // 將在資料庫中自動分配
		globalMerchantID: globalMerchantID,
		name:             name,
		displayName:      name, // 預設與name相同
		apiKey:           uuid.New().String(),
		createdAt:        now,
		updatedAt:        now,
	}

	merchant.syncMerchantFields()
	return merchant
}

// NewMerchantWithTimes 建立新的Merchant實體（包含指定時間）
func NewMerchantWithTimes(globalMerchantID, name string, createdAt, updatedAt time.Time) *Merchant {
	merchant := &Merchant{
		id:               0,
		globalMerchantID: globalMerchantID,
		name:             name,
		displayName:      name,
		apiKey:           uuid.New().String(),
		createdAt:        createdAt,
		updatedAt:        updatedAt,
	}

	merchant.syncMerchantFields()
	return merchant
}

// Getter methods for Merchant
func (m *Merchant) GetID() uint64               { return m.id }
func (m *Merchant) GetGlobalMerchantID() string { return m.globalMerchantID }
func (m *Merchant) GetName() string             { return m.name }
func (m *Merchant) GetDisplayName() string      { return m.displayName }
func (m *Merchant) GetAPIKey() string           { return m.apiKey }
func (m *Merchant) GetCreatedAt() time.Time     { return m.createdAt }
func (m *Merchant) GetUpdatedAt() time.Time     { return m.updatedAt }
func (m *Merchant) GetDeletedAt() *time.Time    { return m.deletedAt }

// UpdateName 業務方法
func (m *Merchant) UpdateName(newName string) error {
	if newName == "" {
		return errors.New("merchant name cannot be empty")
	}
	m.name = newName
	m.updatedAt = time.Now()
	m.syncMerchantFields()
	return nil
}

func (m *Merchant) UpdateDisplayName(newDisplayName string) error {
	if newDisplayName == "" {
		return errors.New("merchant display name cannot be empty")
	}
	m.displayName = newDisplayName
	m.updatedAt = time.Now()
	m.syncMerchantFields()
	return nil
}

func (m *Merchant) RegenerateAPIKey() {
	m.apiKey = uuid.New().String()
	m.updatedAt = time.Now()
	m.syncMerchantFields()
}

func (m *Merchant) SetDeletedAt(deletedAt *time.Time) {
	m.deletedAt = deletedAt
	m.updatedAt = time.Now()
	m.syncMerchantFields()
}

// IsValid 驗證方法
func (m *Merchant) IsValid() error {
	if m.globalMerchantID == "" {
		return errors.New("merchant global ID cannot be empty")
	}
	if m.name == "" {
		return errors.New("merchant name cannot be empty")
	}
	if m.apiKey == "" {
		return errors.New("merchant API key cannot be empty")
	}
	return nil
}

func (m *Merchant) IsDeleted() bool {
	return m.deletedAt != nil
}

// 同步方法確保資料一致性 for Merchant
func (m *Merchant) syncMerchantFields() {
	m.ID = m.id
	m.GlobalMerchantID = m.globalMerchantID
	m.Name = m.name
	m.DisplayName = m.displayName
	m.APIKey = m.apiKey
	m.CreatedAt = m.createdAt
	m.UpdatedAt = m.updatedAt
	m.DeletedAt = m.deletedAt
}

// SetID 設置ID（用於資料庫操作） for Merchant
func (m *Merchant) SetID(id uint64) {
	m.id = id
	m.syncMerchantFields()
}
