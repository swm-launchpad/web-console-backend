package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
)

func TestNewEnvVar_Success(t *testing.T) {
	containerID := uint(1)
	key := "APP_ENV"
	value := "production"

	envVar, err := NewEnvVar(containerID, key, value)

	require.NoError(t, err)
	assert.NotNil(t, envVar)
	assert.Equal(t, containerID, envVar.ContainerID())
	assert.Equal(t, key, envVar.Key())
	assert.Equal(t, value, envVar.Value())
	assert.NotZero(t, envVar.CreatedAt())
	assert.True(t, envVar.UpdatedAt().IsZero())
}

func TestNewEnvVar_InvalidContainerID(t *testing.T) {
	envVar, err := NewEnvVar(0, "APP_ENV", "production")

	assert.ErrorIs(t, err, containererrors.ErrInvalidContainerID)
	assert.Nil(t, envVar)
}

func TestNewEnvVar_EmptyKey(t *testing.T) {
	envVar, err := NewEnvVar(1, "", "production")

	assert.ErrorIs(t, err, containererrors.ErrEnvVarKeyRequired)
	assert.Nil(t, envVar)
}

func TestNewEnvVar_InvalidKeyFormat(t *testing.T) {
	invalidKeys := []string{
		"123KEY",  // starts with number
		"APP-ENV", // contains hyphen
		"APP.ENV", // contains dot
		"APP ENV", // contains space
		"APP@ENV", // contains special char
		"app_env", // lowercase (should start with uppercase or underscore)
	}

	for _, key := range invalidKeys {
		t.Run(key, func(t *testing.T) {
			envVar, err := NewEnvVar(1, key, "value")
			assert.ErrorIs(t, err, containererrors.ErrInvalidEnvVarKey)
			assert.Nil(t, envVar)
		})
	}
}

func TestNewEnvVar_ReservedKey(t *testing.T) {
	reservedKeys := []string{
		"PATH",
		"HOME",
		"USER",
		"SHELL",
		"HOSTNAME",
	}

	for _, key := range reservedKeys {
		t.Run(key, func(t *testing.T) {
			envVar, err := NewEnvVar(1, key, "value")
			assert.ErrorIs(t, err, containererrors.ErrReservedEnvVarKey)
			assert.Nil(t, envVar)
		})
	}
}

func TestNewEnvVar_KeyTooLong(t *testing.T) {
	// Create a 256-character key (all 'A's)
	longKey := ""
	for i := 0; i < 256; i++ {
		longKey += "A"
	}

	envVar, err := NewEnvVar(1, longKey, "value")

	assert.ErrorIs(t, err, containererrors.ErrEnvVarKeyTooLong)
	assert.Nil(t, envVar)
}

func TestNewEnvVar_ValueTooLong(t *testing.T) {
	longValue := string(make([]byte, 5001)) // 5001 characters

	envVar, err := NewEnvVar(1, "APP_ENV", longValue)

	assert.ErrorIs(t, err, containererrors.ErrEnvVarValueTooLong)
	assert.Nil(t, envVar)
}

func TestEnvVar_UpdateValue(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		envVar, _ := NewEnvVar(1, "APP_ENV", "development")
		newValue := "production"

		err := envVar.UpdateValue(newValue)

		require.NoError(t, err)
		assert.Equal(t, newValue, envVar.Value())
		assert.False(t, envVar.UpdatedAt().IsZero())
	})

	t.Run("Same value - no update", func(t *testing.T) {
		envVar, _ := NewEnvVar(1, "APP_ENV", "production")
		originalUpdatedAt := envVar.UpdatedAt()

		err := envVar.UpdateValue("production")

		require.NoError(t, err)
		assert.Equal(t, "production", envVar.Value())
		assert.Equal(t, originalUpdatedAt, envVar.UpdatedAt())
	})

	t.Run("Value too long", func(t *testing.T) {
		envVar, _ := NewEnvVar(1, "APP_ENV", "production")
		longValue := string(make([]byte, 5001))

		err := envVar.UpdateValue(longValue)

		assert.ErrorIs(t, err, containererrors.ErrEnvVarValueTooLong)
		assert.Equal(t, "production", envVar.Value())
	})
}

func TestEnvVar_SetEnvVarID(t *testing.T) {
	envVar, _ := NewEnvVar(1, "APP_ENV", "production")

	assert.Equal(t, uint(0), envVar.EnvVarID())

	envVar.SetEnvVarID(999)
	assert.Equal(t, uint(999), envVar.EnvVarID())
}

func TestReconstructEnvVar(t *testing.T) {
	containerID := uint(1)
	envVarID := uint(100)
	key := "APP_ENV"
	value := "production"

	envVar := ReconstructEnvVar(envVarID, containerID, key, value, time.Now(), time.Now())

	assert.NotNil(t, envVar)
	assert.Equal(t, envVarID, envVar.EnvVarID())
	assert.Equal(t, containerID, envVar.ContainerID())
	assert.Equal(t, key, envVar.Key())
	assert.Equal(t, value, envVar.Value())
}
