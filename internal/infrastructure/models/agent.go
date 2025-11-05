package models

import (
	"time"

	"gorm.io/gorm"
)

// Agent 代理數據模型
type Agent struct {
	ID              uint64         `gorm:"primaryKey;autoIncrement"                                                    json:"id"`
	MerchantID      uint64         `gorm:"index;not null"                                                              json:"merchant_id"`
	GlobalAgentID   string         `gorm:"uniqueIndex:unique_global_agent_id;size:255;not null;column:global_agent_id" json:"global_agent_id"`
	Account         string         `gorm:"size:255;not null;index"                                                     json:"account"`
	Ancestry        string         `gorm:"type:text"                                                                   json:"ancestry"`           // 父代理層級路徑
	CurrentSignInAt *time.Time     `gorm:"type:datetime;index;column:current_sign_in_at"                               json:"current_sign_in_at"` // 當前登入時間，添加索引
	CreatedAt       time.Time      `gorm:"type:datetime;default:CURRENT_TIMESTAMP"                                     json:"created_at"`
	UpdatedAt       time.Time      `gorm:"type:datetime;default:CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP;index"   json:"updated_at"` // 添加索引用於時間戳比較
	DeletedAt       gorm.DeletedAt `gorm:"index"                                                                       json:"deleted_at,omitempty"`
}

func (*Agent) TableName() string {
	return "agents"
}
