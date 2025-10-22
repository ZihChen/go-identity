package entity

import (
	"errors"
	"time"
)

type Level struct {
	// 私有欄位
	id                  uint64
	merchantID          uint64
	name                string
	globalPlayerLevelID string
	globalMerchantID    string
	createdAt           time.Time
	updatedAt           time.Time
	deletedAt           *time.Time

	// 向後兼容的公共欄位
	ID                  uint64     `json:"id"                     deprecated:"use GetID() method instead"`
	MerchantID          uint64     `json:"merchant_id"            deprecated:"use GetMerchantID() method instead"`
	Name                string     `json:"name"                   deprecated:"use GetName() method instead"`
	GlobalPlayerLevelID string     `json:"global_player_level_id" deprecated:"use GetGlobalPlayerLevelID() method instead"`
	GlobalMerchantID    string     `json:"global_merchant_id"     deprecated:"use GetGlobalMerchantID() method instead"`
	CreatedAt           time.Time  `json:"created_at"             deprecated:"use GetCreatedAt() method instead"`
	UpdatedAt           time.Time  `json:"updated_at"             deprecated:"use GetUpdatedAt() method instead"`
	DeletedAt           *time.Time `json:"deleted_at,omitempty"   deprecated:"use GetDeletedAt() method instead"`
}

// NewLevel 建立新的Level實體
func NewLevel(merchantID uint64, name, globalPlayerLevelID, globalMerchantID string) *Level {
	now := time.Now()
	level := &Level{
		id:                  0,
		merchantID:          merchantID,
		name:                name,
		globalPlayerLevelID: globalPlayerLevelID,
		globalMerchantID:    globalMerchantID,
		createdAt:           now,
		updatedAt:           now,
	}

	level.syncLevelFields()
	return level
}

// NewLevelWithTimes 建立帶有指定時間戳的Level實體
func NewLevelWithTimes(
	merchantID uint64,
	name, globalPlayerLevelID, globalMerchantID string,
	createdAt, updatedAt time.Time,
) *Level {
	level := &Level{
		id:                  0,
		merchantID:          merchantID,
		name:                name,
		globalPlayerLevelID: globalPlayerLevelID,
		globalMerchantID:    globalMerchantID,
		createdAt:           createdAt,
		updatedAt:           updatedAt,
	}

	level.syncLevelFields()
	return level
}

// Getter methods for Level
func (l *Level) GetID() uint64                  { return l.id }
func (l *Level) GetMerchantID() uint64          { return l.merchantID }
func (l *Level) GetName() string                { return l.name }
func (l *Level) GetGlobalPlayerLevelID() string { return l.globalPlayerLevelID }
func (l *Level) GetGlobalMerchantID() string    { return l.globalMerchantID }
func (l *Level) GetCreatedAt() time.Time        { return l.createdAt }
func (l *Level) GetUpdatedAt() time.Time        { return l.updatedAt }
func (l *Level) GetDeletedAt() *time.Time       { return l.deletedAt }

// 業務方法 for Level
func (l *Level) UpdateName(newName string) error {
	if newName == "" {
		return errors.New("level name cannot be empty")
	}
	l.name = newName
	l.updatedAt = time.Now()
	l.syncLevelFields()
	return nil
}

func (l *Level) SetDeletedAt(deletedAt *time.Time) {
	l.deletedAt = deletedAt
	l.updatedAt = time.Now()
	l.syncLevelFields()
}

// 驗證方法 for Level
func (l *Level) IsValid() error {
	if l.merchantID == 0 {
		return errors.New("level must belong to a merchant")
	}
	if l.name == "" {
		return errors.New("level name cannot be empty")
	}
	if l.globalPlayerLevelID == "" {
		return errors.New("level global player level ID cannot be empty")
	}
	if l.globalMerchantID == "" {
		return errors.New("level global merchant ID cannot be empty")
	}
	return nil
}

func (l *Level) IsDeleted() bool {
	return l.deletedAt != nil
}

// 同步方法 for Level
func (l *Level) syncLevelFields() {
	l.ID = l.id
	l.MerchantID = l.merchantID
	l.Name = l.name
	l.GlobalPlayerLevelID = l.globalPlayerLevelID
	l.GlobalMerchantID = l.globalMerchantID
	l.CreatedAt = l.createdAt
	l.UpdatedAt = l.updatedAt
	l.DeletedAt = l.deletedAt
}

func (l *Level) SetID(id uint64) {
	l.id = id
	l.syncLevelFields()
}
