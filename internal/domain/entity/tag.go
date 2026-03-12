package entity

import (
	"encoding/json"
	"time"

	"github.com/jvdiamondtech/ms-identity-cat/internal/domain/errmsg"
)

type Tag struct {
	// 私有欄位
	id          uint64
	merchantID  uint64
	name        string
	globalTagID string
	createdAt   time.Time
	updatedAt   time.Time
	deletedAt   *time.Time
}

// NewTag 建立新的Tag實體
func NewTag(merchantID uint64, name, globalTagID string) *Tag {
	now := time.Now()
	tag := &Tag{
		id:          0,
		merchantID:  merchantID,
		name:        name,
		globalTagID: globalTagID,
		createdAt:   now,
		updatedAt:   now,
	}

	return tag
}

// NewTagWithTimes 建立帶有指定時間戳的Tag實體
func NewTagWithTimes(merchantID uint64, name, globalTagID string, updatedAt time.Time) *Tag {
	tag := &Tag{
		id:          0,
		merchantID:  merchantID,
		name:        name,
		globalTagID: globalTagID,
		createdAt:   updatedAt,
		updatedAt:   updatedAt,
	}

	return tag
}

// Getter methods for Tag
func (t *Tag) GetID() uint64            { return t.id }
func (t *Tag) GetMerchantID() uint64    { return t.merchantID }
func (t *Tag) GetName() string          { return t.name }
func (t *Tag) GetGlobalTagID() string   { return t.globalTagID }
func (t *Tag) GetCreatedAt() time.Time  { return t.createdAt }
func (t *Tag) GetUpdatedAt() time.Time  { return t.updatedAt }
func (t *Tag) GetDeletedAt() *time.Time { return t.deletedAt }

// UpdateName 業務方法 for Tag
func (t *Tag) UpdateName(newName string) error {
	if newName == "" {
		return errmsg.ErrTagNameEmpty
	}
	t.name = newName
	t.updatedAt = time.Now()
	return nil
}

func (t *Tag) SetDeletedAt(deletedAt *time.Time) {
	t.deletedAt = deletedAt
	t.updatedAt = time.Now()
}

// IsValid 驗證方法 for Tag
func (t *Tag) IsValid() error {
	if t.merchantID == 0 {
		return errmsg.ErrTagNoMerchant
	}
	if t.name == "" {
		return errmsg.ErrTagNameEmpty
	}
	if t.globalTagID == "" {
		return errmsg.ErrTagGlobalIDEmpty
	}
	return nil
}

func (t *Tag) IsValidForMerchant(merchantID uint64) bool {
	return t.merchantID == merchantID && !t.IsDeleted()
}

func (t *Tag) IsDeleted() bool {
	return t.deletedAt != nil
}

func (t *Tag) SetID(id uint64) {
	t.id = id
}

// Additional setter methods for repository mapping
func (t *Tag) SetMerchantID(merchantID uint64) {
	t.merchantID = merchantID
}

func (t *Tag) SetName(name string) {
	t.name = name
}

func (t *Tag) SetGlobalTagID(globalTagID string) {
	t.globalTagID = globalTagID
}

func (t *Tag) SetCreatedAt(createdAt time.Time) {
	t.createdAt = createdAt
}

func (t *Tag) SetUpdatedAt(updatedAt time.Time) {
	t.updatedAt = updatedAt
}

// MarshalJSON implements custom JSON marshaling for Redis cache storage
// 這個方法使 Tag 可以被正確序列化為 JSON 格式存入 Redis
func (t *Tag) MarshalJSON() ([]byte, error) {
	type Alias struct {
		ID          uint64     `json:"id"`
		MerchantID  uint64     `json:"merchant_id"`
		Name        string     `json:"name"`
		GlobalTagID string     `json:"global_tag_id"`
		CreatedAt   time.Time  `json:"created_at"`
		UpdatedAt   time.Time  `json:"updated_at"`
		DeletedAt   *time.Time `json:"deleted_at,omitempty"`
	}
	return json.Marshal(Alias{
		ID:          t.id,
		MerchantID:  t.merchantID,
		Name:        t.name,
		GlobalTagID: t.globalTagID,
		CreatedAt:   t.createdAt,
		UpdatedAt:   t.updatedAt,
		DeletedAt:   t.deletedAt,
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for Redis cache retrieval
// 這個方法使 Tag 可以從 JSON 格式正確反序列化
func (t *Tag) UnmarshalJSON(data []byte) error {
	type Alias struct {
		ID          uint64     `json:"id"`
		MerchantID  uint64     `json:"merchant_id"`
		Name        string     `json:"name"`
		GlobalTagID string     `json:"global_tag_id"`
		CreatedAt   time.Time  `json:"created_at"`
		UpdatedAt   time.Time  `json:"updated_at"`
		DeletedAt   *time.Time `json:"deleted_at,omitempty"`
	}
	var aux Alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	t.id = aux.ID
	t.merchantID = aux.MerchantID
	t.name = aux.Name
	t.globalTagID = aux.GlobalTagID
	t.createdAt = aux.CreatedAt
	t.updatedAt = aux.UpdatedAt
	t.deletedAt = aux.DeletedAt
	return nil
}
