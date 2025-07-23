package models

import "time"

// PlayerTag 玩家對標籤多對多關聯表
type PlayerTag struct {
	PlayerID  uint64    `gorm:"uniqueIndex:idx_player_tag"`
	TagID     uint64    `gorm:"uniqueIndex:idx_player_tag"`
	CreatedAt time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP" json:"created_at"`
}

func (*PlayerTag) TableName() string {
	return "player_tags"
}
