package entity

import (
	"errors"
	"time"
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

	// 向後兼容的公共欄位
	ID          uint64     `json:"id" deprecated:"use GetID() method instead"`
	MerchantID  uint64     `json:"merchant_id" deprecated:"use GetMerchantID() method instead"`
	Name        string     `json:"name" deprecated:"use GetName() method instead"`
	GlobalTagID string     `json:"global_tag_id" deprecated:"use GetGlobalTagID() method instead"`
	CreatedAt   time.Time  `json:"created_at" deprecated:"use GetCreatedAt() method instead"`
	UpdatedAt   time.Time  `json:"updated_at" deprecated:"use GetUpdatedAt() method instead"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty" deprecated:"use GetDeletedAt() method instead"`
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

	tag.syncTagFields()
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

// 業務方法 for Tag
func (t *Tag) UpdateName(newName string) error {
	if newName == "" {
		return errors.New("tag name cannot be empty")
	}
	t.name = newName
	t.updatedAt = time.Now()
	t.syncTagFields()
	return nil
}

func (t *Tag) SetDeletedAt(deletedAt *time.Time) {
	t.deletedAt = deletedAt
	t.updatedAt = time.Now()
	t.syncTagFields()
}

// 驗證方法 for Tag
func (t *Tag) IsValid() error {
	if t.merchantID == 0 {
		return errors.New("tag must belong to a merchant")
	}
	if t.name == "" {
		return errors.New("tag name cannot be empty")
	}
	if t.globalTagID == "" {
		return errors.New("tag global ID cannot be empty")
	}
	return nil
}

func (t *Tag) IsValidForMerchant(merchantID uint64) bool {
	return t.merchantID == merchantID && !t.IsDeleted()
}

func (t *Tag) IsDeleted() bool {
	return t.deletedAt != nil
}

// 同步方法 for Tag
func (t *Tag) syncTagFields() {
	t.ID = t.id
	t.MerchantID = t.merchantID
	t.Name = t.name
	t.GlobalTagID = t.globalTagID
	t.CreatedAt = t.createdAt
	t.UpdatedAt = t.updatedAt
	t.DeletedAt = t.deletedAt
}

func (t *Tag) SetID(id uint64) {
	t.id = id
	t.syncTagFields()
}
