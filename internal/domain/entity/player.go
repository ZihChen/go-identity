package entity

import (
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

	// 向後兼容的公共欄位（標記為 deprecated）
	ID             uint64      `json:"id"                       deprecated:"use ID() method instead"`
	MerchantID     uint64      `json:"merchant_id"              deprecated:"use MerchantID() method instead"`
	GlobalPlayerID string      `json:"global_player_id"         deprecated:"use GlobalPlayerID() method instead"`
	LevelID        uint64      `json:"level_id"                 deprecated:"use LevelID() method instead"`
	APIKey         string      `json:"api_key"                  deprecated:"use APIKey() method instead"`
	Account        string      `json:"account"                  deprecated:"use Account() method instead"`
	Email          *string     `json:"email,omitempty"          deprecated:"use Email() method instead"`
	LastActiveAt   *time.Time  `json:"last_active_at,omitempty" deprecated:"use LastActiveAt() method instead"`
	CreatedAt      time.Time   `json:"created_at"               deprecated:"use CreatedAt() method instead"`
	UpdatedAt      time.Time   `json:"updated_at"               deprecated:"use UpdatedAt() method instead"`
	DeletedAt      *time.Time  `json:"deleted_at,omitempty"     deprecated:"use DeletedAt() method instead"`
	PlayerLevel    PlayerLevel `json:"player_level,omitempty"   deprecated:"use PlayerLevel() method instead"`
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

	// 同步到公共欄位以保持向後兼容性
	player.syncFields()
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

	player.syncFields()
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
	p.syncFields()
}

func (p *Player) ChangeLevel(newLevelID uint64) error {
	if newLevelID == 0 {
		return errors.New("invalid level ID")
	}
	p.levelID = newLevelID
	p.updatedAt = time.Now()
	p.syncFields()
	return nil
}

func (p *Player) SetEmail(email *string) {
	p.email = email
	p.updatedAt = time.Now()
	p.syncFields()
}

func (p *Player) SetPlayerLevel(playerLevel PlayerLevel) {
	p.playerLevel = playerLevel
	p.updatedAt = time.Now()
	p.syncFields()
}

func (p *Player) SetLastActiveAt(lastActiveAt *time.Time) {
	p.lastActiveAt = lastActiveAt
	p.updatedAt = time.Now()
	p.syncFields()
}

func (p *Player) SetDeletedAt(deletedAt *time.Time) {
	p.deletedAt = deletedAt
	p.updatedAt = time.Now()
	p.syncFields()
}

func (p *Player) RegenerateAPIKey() {
	p.apiKey = uuid.New().String()
	p.updatedAt = time.Now()
	p.syncFields()
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

// 同步方法確保資料一致性
func (p *Player) syncFields() {
	p.ID = p.id
	p.MerchantID = p.merchantID
	p.GlobalPlayerID = p.globalPlayerID
	p.LevelID = p.levelID
	p.APIKey = p.apiKey
	p.Account = p.account
	p.Email = p.email
	p.LastActiveAt = p.lastActiveAt
	p.CreatedAt = p.createdAt
	p.UpdatedAt = p.updatedAt
	p.DeletedAt = p.deletedAt
	p.PlayerLevel = p.playerLevel
}

// SetID 設置ID（用於資料庫操作）
func (p *Player) SetID(id uint64) {
	p.id = id
	p.syncFields()
}
