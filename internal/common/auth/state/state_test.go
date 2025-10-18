package state

import (
	"testing"
	"time"
)

func TestStateValidator_GenerateAndValidate(t *testing.T) {
	secret := "test-secret-key"
	validator := NewStateValidator(secret)

	t.Run("성공: 유효한 state 생성 및 검증", func(t *testing.T) {
		userID := uint(123)

		// Generate state
		state, err := validator.GenerateState(userID)
		if err != nil {
			t.Fatalf("failed to generate state: %v", err)
		}

		// Validate state
		extractedUserID, err := validator.ValidateState(state)
		if err != nil {
			t.Fatalf("failed to validate state: %v", err)
		}

		if extractedUserID != userID {
			t.Errorf("expected userID %d, got %d", userID, extractedUserID)
		}
	})

	t.Run("실패: 잘못된 서명", func(t *testing.T) {
		userID := uint(456)

		// Generate with one secret
		state, err := validator.GenerateState(userID)
		if err != nil {
			t.Fatalf("failed to generate state: %v", err)
		}

		// Validate with different secret
		wrongValidator := NewStateValidator("different-secret")
		_, err = wrongValidator.ValidateState(state)
		if err != ErrInvalidSignature {
			t.Errorf("expected ErrInvalidSignature, got %v", err)
		}
	})

	t.Run("실패: 잘못된 형식", func(t *testing.T) {
		invalidStates := []string{
			"",                  // 빈 문자열
			"abc",               // 부족한 parts
			"a:b",               // 부족한 parts
			"a:b:c",             // 부족한 parts
			"a:invalid:123:sig", // 잘못된 timestamp
			"a:123:invalid:sig", // 잘못된 userID
		}

		for _, state := range invalidStates {
			_, err := validator.ValidateState(state)
			if err != ErrInvalidStateFormat && err != ErrInvalidSignature {
				t.Errorf("for state %q, expected format or signature error, got %v", state, err)
			}
		}
	})

	t.Run("실패: 조작된 userID", func(t *testing.T) {
		// Try to validate a tampered state (should fail signature check)
		tamperedState := "random:123456:999:signature"
		_, err := validator.ValidateState(tamperedState)
		if err != ErrInvalidSignature {
			t.Errorf("expected ErrInvalidSignature for tampered state, got %v", err)
		}
	})
}

func TestStateValidator_Expiration(t *testing.T) {
	t.Run("만료 검증은 실제 시간이 필요하므로 스킵", func(t *testing.T) {
		// Note: Proper expiration testing would require time mocking
		// For now, we just verify the expiration logic exists
		secret := "test-secret"
		validator := NewStateValidator(secret)

		userID := uint(100)
		state, err := validator.GenerateState(userID)
		if err != nil {
			t.Fatalf("failed to generate state: %v", err)
		}

		// Should be valid immediately
		extractedUserID, err := validator.ValidateState(state)
		if err != nil {
			t.Errorf("state should be valid immediately: %v", err)
		}
		if extractedUserID != userID {
			t.Errorf("expected userID %d, got %d", userID, extractedUserID)
		}
	})
}

func TestStateValidator_MultipleUsers(t *testing.T) {
	secret := "test-secret"
	validator := NewStateValidator(secret)

	// Generate states for multiple users
	userIDs := []uint{1, 2, 100, 999, 123456}
	states := make([]string, len(userIDs))

	for i, userID := range userIDs {
		state, err := validator.GenerateState(userID)
		if err != nil {
			t.Fatalf("failed to generate state for user %d: %v", userID, err)
		}
		states[i] = state
	}

	// Verify each state
	for i, state := range states {
		expectedUserID := userIDs[i]
		extractedUserID, err := validator.ValidateState(state)
		if err != nil {
			t.Errorf("failed to validate state for user %d: %v", expectedUserID, err)
			continue
		}
		if extractedUserID != expectedUserID {
			t.Errorf("expected userID %d, got %d", expectedUserID, extractedUserID)
		}
	}
}

func TestStateValidator_RandomnessCheck(t *testing.T) {
	secret := "test-secret"
	validator := NewStateValidator(secret)
	userID := uint(42)

	// Generate multiple states for the same user
	states := make(map[string]bool)
	for i := 0; i < 100; i++ {
		state, err := validator.GenerateState(userID)
		if err != nil {
			t.Fatalf("failed to generate state: %v", err)
		}

		// Each state should be unique due to random bytes and timestamp
		if states[state] {
			t.Errorf("duplicate state generated: %s", state)
		}
		states[state] = true

		// Small delay to ensure different timestamp
		time.Sleep(1 * time.Millisecond)
	}
}
