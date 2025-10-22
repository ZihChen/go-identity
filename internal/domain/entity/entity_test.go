package entity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewPlayer(t *testing.T) {
	tests := []struct {
		name           string
		merchantID     uint64
		globalPlayerID string
		account        string
		levelID        uint64
		email          *string
		expectError    bool
	}{
		{
			name:           "Valid player creation",
			merchantID:     1,
			globalPlayerID: "global123",
			account:        "test@example.com",
			levelID:        2,
			email:          nil,
			expectError:    false,
		},
		{
			name:           "Valid player with email",
			merchantID:     1,
			globalPlayerID: "global456",
			account:        "test2@example.com",
			levelID:        3,
			email:          stringPtr("user@example.com"),
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			player := NewPlayer(tt.merchantID, tt.globalPlayerID, tt.account, tt.levelID, tt.email)

			// Test getter methods
			assert.Equal(t, tt.merchantID, player.GetMerchantID())
			assert.Equal(t, tt.globalPlayerID, player.GetGlobalPlayerID())
			assert.Equal(t, tt.account, player.GetAccount())
			assert.Equal(t, tt.levelID, player.GetLevelID())
			assert.Equal(t, tt.email, player.GetEmail())

			// Test that API key is generated
			assert.NotEmpty(t, player.GetAPIKey())

			// Test timestamps
			assert.False(t, player.GetCreatedAt().IsZero())
			assert.False(t, player.GetUpdatedAt().IsZero())

			// Test entity fields through getter methods
			assert.Equal(t, tt.merchantID, player.GetMerchantID())
			assert.Equal(t, tt.globalPlayerID, player.GetGlobalPlayerID())
			assert.Equal(t, tt.account, player.GetAccount())

			// Test validation
			err := player.IsValid()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewPlayerWithTimes(t *testing.T) {
	createdAt := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)

	player := NewPlayerWithTimes(1, "global123", "test@example.com", 2, nil, createdAt, updatedAt)

	assert.Equal(t, createdAt, player.GetCreatedAt())
	assert.Equal(t, updatedAt, player.GetUpdatedAt())
	assert.Equal(t, uint64(1), player.GetMerchantID())
	assert.Equal(t, "global123", player.GetGlobalPlayerID())
}

func TestPlayer_UpdateLastActive(t *testing.T) {
	player := NewPlayer(1, "global123", "test@example.com", 2, nil)
	oldUpdatedAt := player.GetUpdatedAt()

	// Wait a bit to ensure time difference
	time.Sleep(1 * time.Millisecond)
	player.UpdateLastActive()

	assert.NotNil(t, player.GetLastActiveAt())
	assert.True(t, player.GetUpdatedAt().After(oldUpdatedAt))

	// Test that fields are properly updated
	assert.NotNil(t, player.GetLastActiveAt())
	assert.False(t, player.GetUpdatedAt().IsZero())
}

func TestPlayer_ChangeLevel(t *testing.T) {
	player := NewPlayer(1, "global123", "test@example.com", 2, nil)
	oldUpdatedAt := player.GetUpdatedAt()

	tests := []struct {
		name        string
		newLevelID  uint64
		expectError bool
	}{
		{
			name:        "Valid level change",
			newLevelID:  5,
			expectError: false,
		},
		{
			name:        "Invalid level ID (zero)",
			newLevelID:  0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			time.Sleep(1 * time.Millisecond) // Ensure time difference
			err := player.ChangeLevel(tt.newLevelID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "invalid level ID")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.newLevelID, player.GetLevelID())
				assert.True(t, player.GetUpdatedAt().After(oldUpdatedAt))

				// Test that level ID is properly set
				assert.Equal(t, tt.newLevelID, player.GetLevelID())
			}
		})
	}
}

func TestPlayer_SetEmail(t *testing.T) {
	player := NewPlayer(1, "global123", "test@example.com", 2, nil)
	oldUpdatedAt := player.GetUpdatedAt()

	newEmail := stringPtr("newemail@example.com")
	time.Sleep(1 * time.Millisecond)
	player.SetEmail(newEmail)

	assert.Equal(t, newEmail, player.GetEmail())
	assert.True(t, player.GetUpdatedAt().After(oldUpdatedAt))

	// Test that email is properly set
	assert.Equal(t, newEmail, player.GetEmail())
}

func TestPlayer_SetPlayerLevel(t *testing.T) {
	player := NewPlayer(1, "global123", "test@example.com", 2, nil)
	oldUpdatedAt := player.GetUpdatedAt()

	newLevel := PlayerLevel{
		GlobalPlayerLevelID: "level456",
		Name:                "Premium",
	}

	time.Sleep(1 * time.Millisecond)
	player.SetPlayerLevel(newLevel)

	assert.Equal(t, newLevel, player.GetPlayerLevel())
	assert.True(t, player.GetUpdatedAt().After(oldUpdatedAt))

	// Test that player level is properly set
	assert.Equal(t, newLevel, player.GetPlayerLevel())
}

func TestPlayer_RegenerateAPIKey(t *testing.T) {
	player := NewPlayer(1, "global123", "test@example.com", 2, nil)
	oldAPIKey := player.GetAPIKey()
	oldUpdatedAt := player.GetUpdatedAt()

	time.Sleep(1 * time.Millisecond)
	player.RegenerateAPIKey()

	assert.NotEqual(t, oldAPIKey, player.GetAPIKey())
	assert.NotEmpty(t, player.GetAPIKey())
	assert.True(t, player.GetUpdatedAt().After(oldUpdatedAt))

	// Test that API key is properly regenerated
	assert.NotEqual(t, oldAPIKey, player.GetAPIKey())
	assert.NotEmpty(t, player.GetAPIKey())
}

func TestPlayer_IsValid(t *testing.T) {
	tests := []struct {
		name           string
		merchantID     uint64
		globalPlayerID string
		account        string
		expectError    bool
		errorContains  string
	}{
		{
			name:           "Valid player",
			merchantID:     1,
			globalPlayerID: "global123",
			account:        "test@example.com",
			expectError:    false,
		},
		{
			name:           "Empty account",
			merchantID:     1,
			globalPlayerID: "global123",
			account:        "",
			expectError:    true,
			errorContains:  "account cannot be empty",
		},
		{
			name:           "Zero merchant ID",
			merchantID:     0,
			globalPlayerID: "global123",
			account:        "test@example.com",
			expectError:    true,
			errorContains:  "must belong to a merchant",
		},
		{
			name:           "Empty global player ID",
			merchantID:     1,
			globalPlayerID: "",
			account:        "test@example.com",
			expectError:    true,
			errorContains:  "global ID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			player := NewPlayer(tt.merchantID, tt.globalPlayerID, tt.account, 1, nil)
			err := player.IsValid()

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPlayer_IsDeleted(t *testing.T) {
	player := NewPlayer(1, "global123", "test@example.com", 2, nil)

	// Initially not deleted
	assert.False(t, player.IsDeleted())

	// Set deleted
	now := time.Now()
	player.SetDeletedAt(&now)
	assert.True(t, player.IsDeleted())

	// Set back to not deleted
	player.SetDeletedAt(nil)
	assert.False(t, player.IsDeleted())
}

func TestPlayer_SetID(t *testing.T) {
	player := NewPlayer(1, "global123", "test@example.com", 2, nil)

	newID := uint64(42)
	player.SetID(newID)

	assert.Equal(t, newID, player.GetID())

	// Test that ID is properly set
	assert.Equal(t, newID, player.GetID())
}

func TestPlayer_SetLastActiveAt(t *testing.T) {
	player := NewPlayer(1, "global123", "test@example.com", 2, nil)
	oldUpdatedAt := player.GetUpdatedAt()

	lastActive := time.Now()
	time.Sleep(1 * time.Millisecond)
	player.SetLastActiveAt(&lastActive)

	assert.Equal(t, &lastActive, player.GetLastActiveAt())
	assert.True(t, player.GetUpdatedAt().After(oldUpdatedAt))

	// Test that last active time is properly set
	assert.Equal(t, &lastActive, player.GetLastActiveAt())
}

func TestPlayer_SetDeletedAt(t *testing.T) {
	player := NewPlayer(1, "global123", "test@example.com", 2, nil)
	oldUpdatedAt := player.GetUpdatedAt()

	deletedAt := time.Now()
	time.Sleep(1 * time.Millisecond)
	player.SetDeletedAt(&deletedAt)

	assert.Equal(t, &deletedAt, player.GetDeletedAt())
	assert.True(t, player.GetUpdatedAt().After(oldUpdatedAt))

	// Test that deleted at time is properly set
	assert.Equal(t, &deletedAt, player.GetDeletedAt())
}

// Helper function for creating string pointers
func stringPtr(s string) *string {
	return &s
}

// ===== Merchant Tests =====

func TestNewMerchant(t *testing.T) {
	tests := []struct {
		name             string
		globalMerchantID string
		merchantName     string
		expectError      bool
	}{
		{
			name:             "Valid merchant creation",
			globalMerchantID: "global123",
			merchantName:     "Test Merchant",
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			merchant := NewMerchant(tt.globalMerchantID, tt.merchantName)

			assert.Equal(t, tt.globalMerchantID, merchant.GetGlobalMerchantID())
			assert.Equal(t, tt.merchantName, merchant.GetName())
			assert.Equal(t, tt.merchantName, merchant.GetDisplayName()) // Default same as name
			assert.NotEmpty(t, merchant.GetAPIKey())
			assert.False(t, merchant.GetCreatedAt().IsZero())
			assert.False(t, merchant.GetUpdatedAt().IsZero())

			// Test backward compatibility - deprecated fields removed
			// assert.Equal(t, merchant.GetGlobalMerchantID(), merchant.GlobalMerchantID)
			// assert.Equal(t, merchant.GetName(), merchant.Name)

			// Test validation
			err := merchant.IsValid()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMerchant_UpdateName(t *testing.T) {
	merchant := NewMerchant("global123", "Original Name")
	oldUpdatedAt := merchant.GetUpdatedAt()

	tests := []struct {
		name        string
		newName     string
		expectError bool
	}{
		{
			name:        "Valid name update",
			newName:     "Updated Name",
			expectError: false,
		},
		{
			name:        "Empty name",
			newName:     "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			time.Sleep(1 * time.Millisecond)
			err := merchant.UpdateName(tt.newName)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "name cannot be empty")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.newName, merchant.GetName())
				assert.True(t, merchant.GetUpdatedAt().After(oldUpdatedAt))
			}
		})
	}
}

func TestMerchant_RegenerateAPIKey(t *testing.T) {
	merchant := NewMerchant("global123", "Test Merchant")
	oldAPIKey := merchant.GetAPIKey()
	oldUpdatedAt := merchant.GetUpdatedAt()

	time.Sleep(1 * time.Millisecond)
	merchant.RegenerateAPIKey()

	assert.NotEqual(t, oldAPIKey, merchant.GetAPIKey())
	assert.NotEmpty(t, merchant.GetAPIKey())
	assert.True(t, merchant.GetUpdatedAt().After(oldUpdatedAt))
}

// ===== Manager Tests =====

func TestNewManager(t *testing.T) {
	manager := NewManager(1, "global123", "manager@example.com", stringPtr("manager@test.com"))

	assert.Equal(t, uint64(1), manager.GetMerchantID())
	assert.Equal(t, "global123", manager.GetGlobalManagerID())
	assert.Equal(t, "manager@example.com", manager.GetAccount())
	assert.Equal(t, stringPtr("manager@test.com"), manager.GetEmail())
	assert.False(t, manager.GetCreatedAt().IsZero())
	assert.NoError(t, manager.IsValid())
}

func TestManager_IsValid(t *testing.T) {
	tests := []struct {
		name            string
		merchantID      uint64
		globalManagerID string
		account         string
		expectError     bool
		errorContains   string
	}{
		{
			name:            "Valid manager",
			merchantID:      1,
			globalManagerID: "global123",
			account:         "manager@example.com",
			expectError:     false,
		},
		{
			name:            "Zero merchant ID",
			merchantID:      0,
			globalManagerID: "global123",
			account:         "manager@example.com",
			expectError:     true,
			errorContains:   "must belong to a merchant",
		},
		{
			name:            "Empty global ID",
			merchantID:      1,
			globalManagerID: "",
			account:         "manager@example.com",
			expectError:     true,
			errorContains:   "global ID cannot be empty",
		},
		{
			name:            "Empty account",
			merchantID:      1,
			globalManagerID: "global123",
			account:         "",
			expectError:     true,
			errorContains:   "account cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := NewManager(tt.merchantID, tt.globalManagerID, tt.account, nil)
			err := manager.IsValid()

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ===== Tag Tests =====

func TestNewTag(t *testing.T) {
	tag := NewTag(1, "VIP", "global_tag_123")

	assert.Equal(t, uint64(1), tag.GetMerchantID())
	assert.Equal(t, "VIP", tag.GetName())
	assert.Equal(t, "global_tag_123", tag.GetGlobalTagID())
	assert.False(t, tag.GetCreatedAt().IsZero())
	assert.NoError(t, tag.IsValid())
}

func TestTag_UpdateName(t *testing.T) {
	tag := NewTag(1, "VIP", "global_tag_123")
	oldUpdatedAt := tag.GetUpdatedAt()

	time.Sleep(1 * time.Millisecond)
	err := tag.UpdateName("Premium")

	assert.NoError(t, err)
	assert.Equal(t, "Premium", tag.GetName())
	assert.True(t, tag.GetUpdatedAt().After(oldUpdatedAt))

	// Test empty name
	err = tag.UpdateName("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name cannot be empty")
}

func TestTag_IsValidForMerchant(t *testing.T) {
	tag := NewTag(1, "VIP", "global_tag_123")

	assert.True(t, tag.IsValidForMerchant(1))
	assert.False(t, tag.IsValidForMerchant(2))

	// Test with deleted tag
	now := time.Now()
	tag.SetDeletedAt(&now)
	assert.False(t, tag.IsValidForMerchant(1))
}

// ===== Level Tests =====

func TestNewLevel(t *testing.T) {
	level := NewLevel(1, "Bronze", "global_level_123", "global_merchant_456")

	assert.Equal(t, uint64(1), level.GetMerchantID())
	assert.Equal(t, "Bronze", level.GetName())
	assert.Equal(t, "global_level_123", level.GetGlobalPlayerLevelID())
	assert.Equal(t, "global_merchant_456", level.GetGlobalMerchantID())
	assert.False(t, level.GetCreatedAt().IsZero())
	assert.NoError(t, level.IsValid())
}

func TestLevel_UpdateName(t *testing.T) {
	level := NewLevel(1, "Bronze", "global_level_123", "global_merchant_456")
	oldUpdatedAt := level.GetUpdatedAt()

	time.Sleep(1 * time.Millisecond)
	err := level.UpdateName("Silver")

	assert.NoError(t, err)
	assert.Equal(t, "Silver", level.GetName())
	assert.True(t, level.GetUpdatedAt().After(oldUpdatedAt))

	// Test empty name
	err = level.UpdateName("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name cannot be empty")
}

func TestLevel_IsValid(t *testing.T) {
	tests := []struct {
		name                string
		merchantID          uint64
		levelName           string
		globalPlayerLevelID string
		globalMerchantID    string
		expectError         bool
		errorContains       string
	}{
		{
			name:                "Valid level",
			merchantID:          1,
			levelName:           "Bronze",
			globalPlayerLevelID: "global_level_123",
			globalMerchantID:    "global_merchant_456",
			expectError:         false,
		},
		{
			name:                "Zero merchant ID",
			merchantID:          0,
			levelName:           "Bronze",
			globalPlayerLevelID: "global_level_123",
			globalMerchantID:    "global_merchant_456",
			expectError:         true,
			errorContains:       "must belong to a merchant",
		},
		{
			name:                "Empty name",
			merchantID:          1,
			levelName:           "",
			globalPlayerLevelID: "global_level_123",
			globalMerchantID:    "global_merchant_456",
			expectError:         true,
			errorContains:       "name cannot be empty",
		},
		{
			name:                "Empty global player level ID",
			merchantID:          1,
			levelName:           "Bronze",
			globalPlayerLevelID: "",
			globalMerchantID:    "global_merchant_456",
			expectError:         true,
			errorContains:       "global player level ID cannot be empty",
		},
		{
			name:                "Empty global merchant ID",
			merchantID:          1,
			levelName:           "Bronze",
			globalPlayerLevelID: "global_level_123",
			globalMerchantID:    "",
			expectError:         true,
			errorContains:       "global merchant ID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level := NewLevel(
				tt.merchantID,
				tt.levelName,
				tt.globalPlayerLevelID,
				tt.globalMerchantID,
			)
			err := level.IsValid()

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
