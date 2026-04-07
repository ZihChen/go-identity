package entity

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/errmsg"
)

// Merchant 商戶模型
type Merchant struct {
	id               uint64
	globalMerchantID string
	name             string
	displayName      string
	apiKey           string
	createdAt        time.Time
	updatedAt        time.Time
	deletedAt        *time.Time
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

	return merchant
}

// NewMerchantWithTimes 建立新的Merchant實體（包含指定時間和顯示名稱）
func NewMerchantWithTimes(
	globalMerchantID, name, displayName string,
	updatedAt time.Time,
) *Merchant {
	merchant := &Merchant{
		id:               0,
		globalMerchantID: globalMerchantID,
		name:             name,
		displayName:      displayName,
		apiKey:           uuid.New().String(),
		createdAt:        updatedAt,
		updatedAt:        updatedAt,
	}

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
		return errmsg.ErrMerchantNameEmpty
	}
	m.name = newName
	m.updatedAt = time.Now()
	return nil
}

func (m *Merchant) UpdateDisplayName(newDisplayName string) error {
	if newDisplayName == "" {
		return errmsg.ErrMerchantDisplayNameEmpty
	}
	m.displayName = newDisplayName
	m.updatedAt = time.Now()
	return nil
}

func (m *Merchant) RegenerateAPIKey() {
	m.apiKey = uuid.New().String()
	m.updatedAt = time.Now()
}

func (m *Merchant) SetDeletedAt(deletedAt *time.Time) {
	m.deletedAt = deletedAt
	m.updatedAt = time.Now()
}

// Additional setter methods for repository mapping
func (m *Merchant) SetGlobalMerchantID(globalMerchantID string) {
	m.globalMerchantID = globalMerchantID
}

func (m *Merchant) SetName(name string) {
	m.name = name
}

func (m *Merchant) SetDisplayName(displayName string) {
	m.displayName = displayName
}

func (m *Merchant) SetAPIKey(apiKey string) {
	m.apiKey = apiKey
}

func (m *Merchant) SetCreatedAt(createdAt time.Time) {
	m.createdAt = createdAt
}

func (m *Merchant) SetUpdatedAt(updatedAt time.Time) {
	m.updatedAt = updatedAt
}

// IsValid 驗證方法
func (m *Merchant) IsValid() error {
	if m.globalMerchantID == "" {
		return errmsg.ErrMerchantGlobalIDEmpty
	}
	if m.name == "" {
		return errmsg.ErrMerchantNameEmpty
	}
	if m.apiKey == "" {
		return errmsg.ErrMerchantAPIKeyEmpty
	}
	return nil
}

func (m *Merchant) IsDeleted() bool {
	return m.deletedAt != nil
}

// SetID 設置ID（用於資料庫操作） for Merchant
func (m *Merchant) SetID(id uint64) {
	m.id = id
}

// MarshalJSON
//  1. Go 的 json 包無法序列化私有欄位
//  2. 當物件實作了 json.Marshaler 和 json.Unmarshaler interface 時，json.Marshal 和 json.Unmarshal 會自動使用這些方法
//  3. 沒有這些方法，HTTP API 返回的會是零值
func (m *Merchant) MarshalJSON() ([]byte, error) {
	type Alias struct {
		ID               uint64     `json:"id"`
		GlobalMerchantID string     `json:"global_merchant_id"`
		Name             string     `json:"name"`
		DisplayName      string     `json:"display_name"`
		APIKey           string     `json:"api_key"`
		CreatedAt        time.Time  `json:"created_at"`
		UpdatedAt        time.Time  `json:"updated_at"`
		DeletedAt        *time.Time `json:"deleted_at,omitempty"`
	}
	return json.Marshal(Alias{
		ID:               m.id,
		GlobalMerchantID: m.globalMerchantID,
		Name:             m.name,
		DisplayName:      m.displayName,
		APIKey:           m.apiKey,
		CreatedAt:        m.createdAt,
		UpdatedAt:        m.updatedAt,
		DeletedAt:        m.deletedAt,
	})
}

// UnmarshalJSON implements custom JSON unmarshaling
func (m *Merchant) UnmarshalJSON(data []byte) error {
	type Alias struct {
		ID               uint64     `json:"id"`
		GlobalMerchantID string     `json:"global_merchant_id"`
		Name             string     `json:"name"`
		DisplayName      string     `json:"display_name"`
		APIKey           string     `json:"api_key"`
		CreatedAt        time.Time  `json:"created_at"`
		UpdatedAt        time.Time  `json:"updated_at"`
		DeletedAt        *time.Time `json:"deleted_at,omitempty"`
	}
	var aux Alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	m.id = aux.ID
	m.globalMerchantID = aux.GlobalMerchantID
	m.name = aux.Name
	m.displayName = aux.DisplayName
	m.apiKey = aux.APIKey
	m.createdAt = aux.CreatedAt
	m.updatedAt = aux.UpdatedAt
	m.deletedAt = aux.DeletedAt
	return nil
}
