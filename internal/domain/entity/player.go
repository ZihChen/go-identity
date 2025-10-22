package entity

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Player 玩家模型
type Player struct {
	// 私有欄位
	id             uint64
	merchantID     uint64
	globalPlayerID string
	levelID        uint64
	apiKey         string
	account        string
	email          *string
	lastActiveAt   *time.Time
	createdAt      time.Time
	updatedAt      time.Time
	deletedAt      *time.Time
	playerLevel    PlayerLevel
}

type PlayerLevel struct {
	GlobalPlayerLevelID string `json:"global_player_level_id"`
	Name                string `json:"name"`
}

// NewPlayer 建立新的Player實體
func NewPlayer(
	merchantID uint64,
	globalPlayerID string,
	account string,
	levelID uint64,
	email *string,
) *Player {
	now := time.Now()
	player := &Player{
		id:             0, // 將在資料庫中自動分配
		merchantID:     merchantID,
		globalPlayerID: globalPlayerID,
		levelID:        levelID,
		apiKey:         uuid.New().String(),
		account:        account,
		email:          email,
		createdAt:      now,
		updatedAt:      now,
	}

	return player
}

// NewPlayerWithTimes 建立新的Player實體（包含指定時間）
func NewPlayerWithTimes(
	merchantID uint64,
	globalPlayerID string,
	account string,
	levelID uint64,
	email *string,
	createdAt, updatedAt time.Time,
) *Player {
	player := &Player{
		id:             0,
		merchantID:     merchantID,
		globalPlayerID: globalPlayerID,
		levelID:        levelID,
		apiKey:         uuid.New().String(),
		account:        account,
		email:          email,
		createdAt:      createdAt,
		updatedAt:      updatedAt,
	}

	return player
}

// Getter methods
func (p *Player) GetID() uint64               { return p.id }
func (p *Player) GetMerchantID() uint64       { return p.merchantID }
func (p *Player) GetGlobalPlayerID() string   { return p.globalPlayerID }
func (p *Player) GetLevelID() uint64          { return p.levelID }
func (p *Player) GetAPIKey() string           { return p.apiKey }
func (p *Player) GetAccount() string          { return p.account }
func (p *Player) GetEmail() *string           { return p.email }
func (p *Player) GetLastActiveAt() *time.Time { return p.lastActiveAt }
func (p *Player) GetCreatedAt() time.Time     { return p.createdAt }
func (p *Player) GetUpdatedAt() time.Time     { return p.updatedAt }
func (p *Player) GetDeletedAt() *time.Time    { return p.deletedAt }
func (p *Player) GetPlayerLevel() PlayerLevel { return p.playerLevel }

// UpdateLastActive 業務方法
func (p *Player) UpdateLastActive() {
	now := time.Now()
	p.lastActiveAt = &now
	p.updatedAt = now
}

func (p *Player) ChangeLevel(newLevelID uint64) error {
	if newLevelID == 0 {
		return errors.New("invalid level ID")
	}
	p.levelID = newLevelID
	p.updatedAt = time.Now()
	return nil
}

func (p *Player) SetEmail(email *string) {
	p.email = email
	p.updatedAt = time.Now()
}

func (p *Player) SetPlayerLevel(playerLevel PlayerLevel) {
	p.playerLevel = playerLevel
	p.updatedAt = time.Now()
}

func (p *Player) SetLastActiveAt(lastActiveAt *time.Time) {
	p.lastActiveAt = lastActiveAt
	p.updatedAt = time.Now()
}

func (p *Player) SetDeletedAt(deletedAt *time.Time) {
	p.deletedAt = deletedAt
	p.updatedAt = time.Now()
}

func (p *Player) RegenerateAPIKey() {
	p.apiKey = uuid.New().String()
	p.updatedAt = time.Now()
}

// IsValid 驗證方法
func (p *Player) IsValid() error {
	if p.account == "" {
		return errors.New("player account cannot be empty")
	}
	if p.merchantID == 0 {
		return errors.New("player must belong to a merchant")
	}
	if p.globalPlayerID == "" {
		return errors.New("player global ID cannot be empty")
	}
	return nil
}

func (p *Player) IsDeleted() bool {
	return p.deletedAt != nil
}

// SetID 設置ID（用於資料庫操作）
func (p *Player) SetID(id uint64) {
	p.id = id
}

// Additional setter methods for repository mapping
func (p *Player) SetAPIKey(apiKey string) {
	p.apiKey = apiKey
}

func (p *Player) SetMerchantID(merchantID uint64) {
	p.merchantID = merchantID
}

func (p *Player) SetGlobalPlayerID(globalPlayerID string) {
	p.globalPlayerID = globalPlayerID
}

func (p *Player) SetLevelID(levelID uint64) {
	p.levelID = levelID
}

func (p *Player) SetAccount(account string) {
	p.account = account
}

func (p *Player) SetCreatedAt(createdAt time.Time) {
	p.createdAt = createdAt
}

func (p *Player) SetUpdatedAt(updatedAt time.Time) {
	p.updatedAt = updatedAt
}

// MarshalJSON
//  1. Go 的 json 包無法序列化私有欄位
//  2. 當物件實作了 json.Marshaler 和 json.Unmarshaler interface 時，json.Marshal 和 json.Unmarshal 會自動使用這些方法
//  3. 沒有這些方法，HTTP API 返回的會是零值
func (p *Player) MarshalJSON() ([]byte, error) {
	type Alias struct {
		ID             uint64      `json:"id"`
		MerchantID     uint64      `json:"merchant_id"`
		GlobalPlayerID string      `json:"global_player_id"`
		LevelID        uint64      `json:"level_id"`
		APIKey         string      `json:"api_key"`
		Account        string      `json:"account"`
		Email          *string     `json:"email,omitempty"`
		LastActiveAt   *time.Time  `json:"last_active_at,omitempty"`
		CreatedAt      time.Time   `json:"created_at"`
		UpdatedAt      time.Time   `json:"updated_at"`
		DeletedAt      *time.Time  `json:"deleted_at,omitempty"`
		PlayerLevel    PlayerLevel `json:"player_level,omitempty"`
	}
	return json.Marshal(Alias{
		ID:             p.id,
		MerchantID:     p.merchantID,
		GlobalPlayerID: p.globalPlayerID,
		LevelID:        p.levelID,
		APIKey:         p.apiKey,
		Account:        p.account,
		Email:          p.email,
		LastActiveAt:   p.lastActiveAt,
		CreatedAt:      p.createdAt,
		UpdatedAt:      p.updatedAt,
		DeletedAt:      p.deletedAt,
		PlayerLevel:    p.playerLevel,
	})
}

// UnmarshalJSON implements custom JSON unmarshaling for HTTP API requests
func (p *Player) UnmarshalJSON(data []byte) error {
	type Alias struct {
		ID             uint64      `json:"id"`
		MerchantID     uint64      `json:"merchant_id"`
		GlobalPlayerID string      `json:"global_player_id"`
		LevelID        uint64      `json:"level_id"`
		APIKey         string      `json:"api_key"`
		Account        string      `json:"account"`
		Email          *string     `json:"email,omitempty"`
		LastActiveAt   *time.Time  `json:"last_active_at,omitempty"`
		CreatedAt      time.Time   `json:"created_at"`
		UpdatedAt      time.Time   `json:"updated_at"`
		DeletedAt      *time.Time  `json:"deleted_at,omitempty"`
		PlayerLevel    PlayerLevel `json:"player_level,omitempty"`
	}
	var aux Alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	p.id = aux.ID
	p.merchantID = aux.MerchantID
	p.globalPlayerID = aux.GlobalPlayerID
	p.levelID = aux.LevelID
	p.apiKey = aux.APIKey
	p.account = aux.Account
	p.email = aux.Email
	p.lastActiveAt = aux.LastActiveAt
	p.createdAt = aux.CreatedAt
	p.updatedAt = aux.UpdatedAt
	p.deletedAt = aux.DeletedAt
	p.playerLevel = aux.PlayerLevel
	return nil
}
