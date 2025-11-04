package entity

import (
	"errors"
	"time"
)

// Agent 代理模型
type Agent struct {
	id              uint64
	merchantID      uint64
	globalAgentID   string
	account         string
	ancestry        string
	currentSignInAt *time.Time
	createdAt       time.Time
	updatedAt       time.Time
	deletedAt       *time.Time
}

// NewAgent 建立新的Agent實體
func NewAgent(merchantID uint64, globalAgentID, account, ancestry string) *Agent {
	now := time.Now()
	agent := &Agent{
		id:              0, // 將在資料庫中自動分配
		merchantID:      merchantID,
		globalAgentID:   globalAgentID,
		account:         account,
		ancestry:        ancestry,
		currentSignInAt: nil,
		createdAt:       now,
		updatedAt:       now,
	}

	return agent
}

// NewAgentWithTimes 建立新的Agent實體（包含指定時間）
func NewAgentWithTimes(
	merchantID uint64,
	globalAgentID, account, ancestry string,
	currentSignInAt *time.Time,
	createdAt, updatedAt time.Time,
) *Agent {
	agent := &Agent{
		id:              0,
		merchantID:      merchantID,
		globalAgentID:   globalAgentID,
		account:         account,
		ancestry:        ancestry,
		currentSignInAt: currentSignInAt,
		createdAt:       createdAt,
		updatedAt:       updatedAt,
	}

	return agent
}

// Getter methods for Agent
func (a *Agent) GetID() uint64                 { return a.id }
func (a *Agent) GetMerchantID() uint64         { return a.merchantID }
func (a *Agent) GetGlobalAgentID() string      { return a.globalAgentID }
func (a *Agent) GetAccount() string            { return a.account }
func (a *Agent) GetAncestry() string           { return a.ancestry }
func (a *Agent) GetCurrentSignInAt() *time.Time { return a.currentSignInAt }
func (a *Agent) GetCreatedAt() time.Time       { return a.createdAt }
func (a *Agent) GetUpdatedAt() time.Time       { return a.updatedAt }
func (a *Agent) GetDeletedAt() *time.Time      { return a.deletedAt }

// 業務方法
func (a *Agent) UpdateAccount(newAccount string) error {
	if newAccount == "" {
		return errors.New("agent account cannot be empty")
	}
	a.account = newAccount
	a.updatedAt = time.Now()
	return nil
}

func (a *Agent) UpdateAncestry(newAncestry string) {
	a.ancestry = newAncestry
	a.updatedAt = time.Now()
}

func (a *Agent) UpdateSignInTime(signInAt *time.Time) {
	a.currentSignInAt = signInAt
	a.updatedAt = time.Now()
}

func (a *Agent) SetDeletedAt(deletedAt *time.Time) {
	a.deletedAt = deletedAt
	a.updatedAt = time.Now()
}

// Setter methods for repository mapping
func (a *Agent) SetID(id uint64) {
	a.id = id
}

func (a *Agent) SetMerchantID(merchantID uint64) {
	a.merchantID = merchantID
}

func (a *Agent) SetGlobalAgentID(globalAgentID string) {
	a.globalAgentID = globalAgentID
}

func (a *Agent) SetAccount(account string) {
	a.account = account
}

func (a *Agent) SetAncestry(ancestry string) {
	a.ancestry = ancestry
}

func (a *Agent) SetCurrentSignInAt(currentSignInAt *time.Time) {
	a.currentSignInAt = currentSignInAt
}

func (a *Agent) SetCreatedAt(createdAt time.Time) {
	a.createdAt = createdAt
}

func (a *Agent) SetUpdatedAt(updatedAt time.Time) {
	a.updatedAt = updatedAt
}

// IsValid 驗證方法
func (a *Agent) IsValid() error {
	if a.globalAgentID == "" {
		return errors.New("agent global ID cannot be empty")
	}
	if a.account == "" {
		return errors.New("agent account cannot be empty")
	}
	if a.merchantID == 0 {
		return errors.New("agent merchant ID cannot be zero")
	}
	return nil
}

func (a *Agent) IsDeleted() bool {
	return a.deletedAt != nil
}