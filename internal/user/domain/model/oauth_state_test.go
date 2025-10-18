package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestOAuthState_IsExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{
			name:      "Not expired - future expiry",
			expiresAt: now.Add(5 * time.Minute),
			want:      false,
		},
		{
			name:      "Expired - past expiry",
			expiresAt: now.Add(-5 * time.Minute),
			want:      true,
		},
		{
			name:      "Edge case - just expired",
			expiresAt: now.Add(-1 * time.Millisecond),
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &OAuthState{
				State:     "test-state",
				UserID:    1,
				ExpiresAt: tt.expiresAt,
				CreatedAt: now,
			}
			assert.Equal(t, tt.want, state.IsExpired())
		})
	}
}

func TestOAuthState_IsConsumed(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		consumedAt *time.Time
		want       bool
	}{
		{
			name:       "Not consumed - nil consumedAt",
			consumedAt: nil,
			want:       false,
		},
		{
			name:       "Consumed - has consumedAt",
			consumedAt: &now,
			want:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &OAuthState{
				State:      "test-state",
				UserID:     1,
				ExpiresAt:  now.Add(10 * time.Minute),
				CreatedAt:  now,
				ConsumedAt: tt.consumedAt,
			}
			assert.Equal(t, tt.want, state.IsConsumed())
		})
	}
}

func TestOAuthState_CanBeUsed(t *testing.T) {
	now := time.Now()
	consumedTime := now

	tests := []struct {
		name       string
		expiresAt  time.Time
		consumedAt *time.Time
		want       bool
	}{
		{
			name:       "Can be used - valid and not consumed",
			expiresAt:  now.Add(10 * time.Minute),
			consumedAt: nil,
			want:       true,
		},
		{
			name:       "Cannot be used - expired",
			expiresAt:  now.Add(-5 * time.Minute),
			consumedAt: nil,
			want:       false,
		},
		{
			name:       "Cannot be used - already consumed",
			expiresAt:  now.Add(10 * time.Minute),
			consumedAt: &consumedTime,
			want:       false,
		},
		{
			name:       "Cannot be used - both expired and consumed",
			expiresAt:  now.Add(-5 * time.Minute),
			consumedAt: &consumedTime,
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &OAuthState{
				State:      "test-state",
				UserID:     1,
				ExpiresAt:  tt.expiresAt,
				CreatedAt:  now,
				ConsumedAt: tt.consumedAt,
			}
			assert.Equal(t, tt.want, state.CanBeUsed())
		})
	}
}

func TestOAuthState_MatchesInstallation(t *testing.T) {
	now := time.Now()
	installationID1 := int64(12345)
	installationID2 := int64(67890)

	tests := []struct {
		name           string
		stateInstallID *int64
		checkInstallID int64
		want           bool
	}{
		{
			name:           "Matches - same installation ID",
			stateInstallID: &installationID1,
			checkInstallID: 12345,
			want:           true,
		},
		{
			name:           "Does not match - different installation ID",
			stateInstallID: &installationID1,
			checkInstallID: 67890,
			want:           false,
		},
		{
			name:           "Does not match - installation ID not set",
			stateInstallID: nil,
			checkInstallID: 12345,
			want:           false,
		},
		{
			name:           "Matches - large installation ID",
			stateInstallID: &installationID2,
			checkInstallID: 67890,
			want:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := &OAuthState{
				State:          "test-state",
				UserID:         1,
				InstallationID: tt.stateInstallID,
				ExpiresAt:      now.Add(10 * time.Minute),
				CreatedAt:      now,
			}
			assert.Equal(t, tt.want, state.MatchesInstallation(tt.checkInstallID))
		})
	}
}

func TestOAuthState_CompleteLifecycle(t *testing.T) {
	now := time.Now()
	installationID := int64(12345)

	// Step 1: Create fresh state
	state := &OAuthState{
		State:          "random-state-token",
		UserID:         1,
		InstallationID: nil, // Not yet bound
		ExpiresAt:      now.Add(10 * time.Minute),
		CreatedAt:      now,
		ConsumedAt:     nil,
	}

	// Initially: not expired, not consumed, can be used
	assert.False(t, state.IsExpired(), "Fresh state should not be expired")
	assert.False(t, state.IsConsumed(), "Fresh state should not be consumed")
	assert.True(t, state.CanBeUsed(), "Fresh state should be usable")

	// Step 2: Bind to installation (happens during callback)
	state.InstallationID = &installationID
	assert.True(t, state.MatchesInstallation(12345), "Should match bound installation")
	assert.False(t, state.MatchesInstallation(99999), "Should not match different installation")

	// Step 3: Consume the state
	consumedTime := now.Add(1 * time.Second)
	state.ConsumedAt = &consumedTime

	// After consumption: cannot be used
	assert.True(t, state.IsConsumed(), "State should be consumed")
	assert.False(t, state.CanBeUsed(), "Consumed state should not be usable")

	// Step 4: Simulate expiry
	state.ExpiresAt = now.Add(-1 * time.Minute)
	assert.True(t, state.IsExpired(), "State should be expired")
	assert.False(t, state.CanBeUsed(), "Expired and consumed state should not be usable")
}
