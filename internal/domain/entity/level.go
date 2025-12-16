package entity

import (
	"encoding/json"
	"errors"
	"time"
)

type Level struct {
	// 私有欄位
	id                  uint64
	merchantID          uint64
	name                string
	globalPlayerLevelID string
	createdAt           time.Time
	updatedAt           time.Time
	deletedAt           *time.Time
}

// NewLevel 建立新的Level實體
func NewLevel(merchantID uint64, name, globalPlayerLevelID string) *Level {
	now := time.Now()
	level := &Level{
		id:                  0,
		merchantID:          merchantID,
		name:                name,
		globalPlayerLevelID: globalPlayerLevelID,
		createdAt:           now,
		updatedAt:           now,
	}

	return level
}

// NewLevelWithTimes 建立帶有指定時間戳的Level實體
func NewLevelWithTimes(
	merchantID uint64,
	name, globalPlayerLevelID string,
	createdAt, updatedAt time.Time,
) *Level {
	level := &Level{
		id:                  0,
		merchantID:          merchantID,
		name:                name,
		globalPlayerLevelID: globalPlayerLevelID,
		createdAt:           createdAt,
		updatedAt:           updatedAt,
	}

	return level
}

// Getter methods for Level
func (l *Level) GetID() uint64                  { return l.id }
func (l *Level) GetMerchantID() uint64          { return l.merchantID }
func (l *Level) GetName() string                { return l.name }
func (l *Level) GetGlobalPlayerLevelID() string { return l.globalPlayerLevelID }
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
	return nil
}

func (l *Level) SetDeletedAt(deletedAt *time.Time) {
	l.deletedAt = deletedAt
	l.updatedAt = time.Now()
}

// IsValid 驗證方法 for Level
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
	return nil
}

func (l *Level) IsDeleted() bool {
	return l.deletedAt != nil
}

func (l *Level) SetID(id uint64) {
	l.id = id
}

func (l *Level) SetMerchantID(merchantID uint64) {
	l.merchantID = merchantID
}

func (l *Level) SetName(name string) {
	l.name = name
}

func (l *Level) SetGlobalPlayerLevelID(globalPlayerLevelID string) {
	l.globalPlayerLevelID = globalPlayerLevelID
}

func (l *Level) SetCreatedAt(createdAt time.Time) {
	l.createdAt = createdAt
}

func (l *Level) SetUpdatedAt(updatedAt time.Time) {
	l.updatedAt = updatedAt
}

// MarshalJSON 自定義 JSON 序列化
// Level entity 需要 JSON 序列化以支援 Redis 快取
func (l *Level) MarshalJSON() ([]byte, error) {
	type Alias struct {
		ID                  uint64     `json:"id"`
		MerchantID          uint64     `json:"merchant_id"`
		Name                string     `json:"name"`
		GlobalPlayerLevelID string     `json:"global_player_level_id"`
		CreatedAt           time.Time  `json:"created_at"`
		UpdatedAt           time.Time  `json:"updated_at"`
		DeletedAt           *time.Time `json:"deleted_at,omitempty"`
	}
	return json.Marshal(Alias{
		ID:                  l.id,
		MerchantID:          l.merchantID,
		Name:                l.name,
		GlobalPlayerLevelID: l.globalPlayerLevelID,
		CreatedAt:           l.createdAt,
		UpdatedAt:           l.updatedAt,
		DeletedAt:           l.deletedAt,
	})
}

// UnmarshalJSON 自定義 JSON 反序列化
// 用於從 Redis 快取或 HTTP 請求中恢復 Level 實體
func (l *Level) UnmarshalJSON(data []byte) error {
	type Alias struct {
		ID                  uint64     `json:"id"`
		MerchantID          uint64     `json:"merchant_id"`
		Name                string     `json:"name"`
		GlobalPlayerLevelID string     `json:"global_player_level_id"`
		CreatedAt           time.Time  `json:"created_at"`
		UpdatedAt           time.Time  `json:"updated_at"`
		DeletedAt           *time.Time `json:"deleted_at,omitempty"`
	}

	aux := &Alias{}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	l.id = aux.ID
	l.merchantID = aux.MerchantID
	l.name = aux.Name
	l.globalPlayerLevelID = aux.GlobalPlayerLevelID
	l.createdAt = aux.CreatedAt
	l.updatedAt = aux.UpdatedAt
	l.deletedAt = aux.DeletedAt

	return nil
}
