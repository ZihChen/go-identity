package entity

import (
	"testing"
	"time"
)

func TestNewAgent(t *testing.T) {
	t.Run("Valid agent creation", func(t *testing.T) {
		merchantID := uint64(1)
		globalAgentID := "TEST-AGENT-1"
		account := "agent001"
		ancestry := "PARENT-AGENT-1/PARENT-AGENT-2"

		agent := NewAgent(merchantID, globalAgentID, account, ancestry)

		if agent.GetMerchantID() != merchantID {
			t.Errorf("Expected merchant ID %d, got %d", merchantID, agent.GetMerchantID())
		}
		if agent.GetGlobalAgentID() != globalAgentID {
			t.Errorf("Expected global agent ID %s, got %s", globalAgentID, agent.GetGlobalAgentID())
		}
		if agent.GetAccount() != account {
			t.Errorf("Expected account %s, got %s", account, agent.GetAccount())
		}
		if agent.GetAncestry() != ancestry {
			t.Errorf("Expected ancestry %s, got %s", ancestry, agent.GetAncestry())
		}
		if agent.GetCurrentSignInAt() != nil {
			t.Errorf("Expected nil current sign in time, got %v", agent.GetCurrentSignInAt())
		}
		if agent.GetID() != 0 {
			t.Errorf("Expected ID 0, got %d", agent.GetID())
		}
	})
}

func TestNewAgentWithTimes(t *testing.T) {
	merchantID := uint64(1)
	globalAgentID := "TEST-AGENT-1"
	account := "agent001"
	ancestry := "PARENT-AGENT-1"
	signInTime := time.Now()
	createdAt := time.Now().Add(-time.Hour)
	updatedAt := time.Now()

	agent := NewAgentWithTimes(
		merchantID,
		globalAgentID,
		account,
		ancestry,
		&signInTime,
		createdAt,
		updatedAt,
	)

	if agent.GetMerchantID() != merchantID {
		t.Errorf("Expected merchant ID %d, got %d", merchantID, agent.GetMerchantID())
	}
	if agent.GetCurrentSignInAt() == nil || !agent.GetCurrentSignInAt().Equal(signInTime) {
		t.Errorf("Expected sign in time %v, got %v", signInTime, agent.GetCurrentSignInAt())
	}
	if !agent.GetCreatedAt().Equal(createdAt) {
		t.Errorf("Expected created at %v, got %v", createdAt, agent.GetCreatedAt())
	}
	if !agent.GetUpdatedAt().Equal(updatedAt) {
		t.Errorf("Expected updated at %v, got %v", updatedAt, agent.GetUpdatedAt())
	}
}

func TestAgent_IsValid(t *testing.T) {
	tests := []struct {
		name          string
		merchantID    uint64
		globalAgentID string
		account       string
		wantErr       bool
	}{
		{
			name:          "Valid agent",
			merchantID:    1,
			globalAgentID: "TEST-AGENT-1",
			account:       "agent001",
			wantErr:       false,
		},
		{
			name:          "Zero merchant ID",
			merchantID:    0,
			globalAgentID: "TEST-AGENT-1",
			account:       "agent001",
			wantErr:       true,
		},
		{
			name:          "Empty global agent ID",
			merchantID:    1,
			globalAgentID: "",
			account:       "agent001",
			wantErr:       true,
		},
		{
			name:          "Empty account",
			merchantID:    1,
			globalAgentID: "TEST-AGENT-1",
			account:       "",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := NewAgent(tt.merchantID, tt.globalAgentID, tt.account, "ancestry")
			err := agent.IsValid()
			if (err != nil) != tt.wantErr {
				t.Errorf("Agent.IsValid() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAgent_UpdateAccount(t *testing.T) {
	agent := NewAgent(1, "TEST-AGENT-1", "agent001", "ancestry")
	oldUpdatedAt := agent.GetUpdatedAt()

	// Wait a bit to ensure time difference
	time.Sleep(1 * time.Millisecond)

	err := agent.UpdateAccount("new_account")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if agent.GetAccount() != "new_account" {
		t.Errorf("Expected account 'new_account', got '%s'", agent.GetAccount())
	}

	if !agent.GetUpdatedAt().After(oldUpdatedAt) {
		t.Errorf("Expected updated time to be updated")
	}

	// Test empty account
	err = agent.UpdateAccount("")
	if err == nil {
		t.Errorf("Expected error for empty account")
	}
}

func TestAgent_IsDeleted(t *testing.T) {
	agent := NewAgent(1, "TEST-AGENT-1", "agent001", "ancestry")

	if agent.IsDeleted() {
		t.Errorf("New agent should not be deleted")
	}

	now := time.Now()
	agent.SetDeletedAt(&now)

	if !agent.IsDeleted() {
		t.Errorf("Agent should be deleted after setting deleted_at")
	}
}
