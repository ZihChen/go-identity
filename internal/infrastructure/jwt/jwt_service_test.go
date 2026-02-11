package jwt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJWTService_GenerateToken(t *testing.T) {
	tests := []struct {
		name           string
		secret         string
		issuer         string
		account        string
		playerGlobalID string
		expectError    bool
		errorMessage   string
	}{
		{
			name:           "Valid token generation",
			secret:         "test-secret-key",
			issuer:         "test-issuer",
			account:        "testuser",
			playerGlobalID: "FATCAT-PLAYER-123",
			expectError:    false,
		},
		{
			name:           "Empty account",
			secret:         "test-secret-key",
			issuer:         "test-issuer",
			account:        "",
			playerGlobalID: "FATCAT-PLAYER-123",
			expectError:    true,
			errorMessage:   "account cannot be empty",
		},
		{
			name:           "Empty player global ID",
			secret:         "test-secret-key",
			issuer:         "test-issuer",
			account:        "testuser",
			playerGlobalID: "",
			expectError:    true,
			errorMessage:   "player_global_id cannot be empty",
		},
		{
			name:           "Production configuration values",
			secret:         "REDACTED_JWT_SECRET",
			issuer:         "REDACTED_JWT_ISSUER",
			account:        "winston883",
			playerGlobalID: "FATCAT-PLAYER-883",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jwtService := NewJWTService(tt.secret, tt.issuer)

			token, err := jwtService.GenerateToken(tt.account, tt.playerGlobalID)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.errorMessage, err.Error())
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, token)

				// Token should be a valid JWT format (header.payload.signature)
				assert.Regexp(t, `^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$`, token)
			}
		})
	}
}

func TestNewJWTService(t *testing.T) {
	secret := "test-secret"
	issuer := "test-issuer"

	service := NewJWTService(secret, issuer)

	assert.NotNil(t, service)
	assert.IsType(t, &jwtService{}, service)
}

func TestNewJWTService_WithEmptySecret(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected fatal error for empty secret")
		}
	}()

	// 這應該會導致fatal
	NewJWTService("", "test-issuer")
}

func TestNewJWTService_WithEmptyIssuer(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("Expected fatal error for empty issuer")
		}
	}()

	// 這應該會導致fatal
	NewJWTService("test-secret", "")
}

func TestJWTService_GenerateTokenWithMetadata(t *testing.T) {
	tests := []struct {
		name             string
		secret           string
		issuer           string
		globalMerchantID string
		metadata         map[string]interface{}
		expectError      bool
		errorMessage     string
	}{
		{
			name:             "Valid token with metadata",
			secret:           "test-secret-key",
			issuer:           "test-issuer",
			globalMerchantID: "FATCAT-MERCHANT-001",
			metadata: map[string]interface{}{
				"player_id": "883",
				"username":  "winston",
			},
			expectError: false,
		},
		{
			name:             "Empty global merchant ID",
			secret:           "test-secret-key",
			issuer:           "test-issuer",
			globalMerchantID: "",
			metadata: map[string]interface{}{
				"player_id": "883",
			},
			expectError:  true,
			errorMessage: "global_merchant_id cannot be empty",
		},
		{
			name:             "Empty metadata",
			secret:           "test-secret-key",
			issuer:           "test-issuer",
			globalMerchantID: "FATCAT-MERCHANT-001",
			metadata:         map[string]interface{}{},
			expectError:      true,
			errorMessage:     "metadata cannot be empty",
		},
		{
			name:             "Nil metadata",
			secret:           "test-secret-key",
			issuer:           "test-issuer",
			globalMerchantID: "FATCAT-MERCHANT-001",
			metadata:         nil,
			expectError:      true,
			errorMessage:     "metadata cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jwtService := NewJWTService(tt.secret, tt.issuer)

			token, err := jwtService.GenerateTokenWithMetadata(tt.globalMerchantID, tt.metadata)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.errorMessage, err.Error())
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, token)

				// Token should be a valid JWT format (header.payload.signature)
				assert.Regexp(t, `^[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+$`, token)
			}
		})
	}
}
