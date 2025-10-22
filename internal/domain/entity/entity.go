package entity

import (
	"time"
)

type PlayerTag struct {
	PlayerID  uint64    `json:"player_id"`
	TagID     uint64    `json:"tag_id"`
	CreatedAt time.Time `json:"created_at"`
}

type LoggerFiled struct {
	Key   string
	Value interface{}
}
