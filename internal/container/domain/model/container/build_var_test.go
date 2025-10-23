package model

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
)

func TestNewBuildVar_Success(t *testing.T) {
	containerID := uint(1)
	key := "NODE_ENV"
	value := "production"

	buildVar, err := NewBuildVar(containerID, key, value)

	require.NoError(t, err)
	assert.NotNil(t, buildVar)
	assert.Equal(t, containerID, buildVar.ContainerID())
	assert.Equal(t, key, buildVar.Key())
	assert.Equal(t, value, buildVar.Value())
	assert.NotZero(t, buildVar.CreatedAt())
	assert.True(t, buildVar.UpdatedAt().IsZero())
}

func TestNewBuildVar_InvalidContainerID(t *testing.T) {
	buildVar, err := NewBuildVar(0, "NODE_ENV", "production")

	assert.ErrorIs(t, err, containererrors.ErrInvalidContainerID)
	assert.Nil(t, buildVar)
}

func TestNewBuildVar_EmptyKey(t *testing.T) {
	buildVar, err := NewBuildVar(1, "", "production")

	assert.ErrorIs(t, err, containererrors.ErrBuildVarKeyRequired)
	assert.Nil(t, buildVar)
}

func TestNewBuildVar_InvalidKeyFormat(t *testing.T) {
	invalidKeys := []string{
		"123KEY",    // starts with number
		"BUILD-VAR", // contains hyphen
		"BUILD.VAR", // contains dot
		"BUILD VAR", // contains space
		"BUILD@VAR", // contains special char
		"build_var", // lowercase (should start with uppercase or underscore)
	}

	for _, key := range invalidKeys {
		t.Run(key, func(t *testing.T) {
			buildVar, err := NewBuildVar(1, key, "value")
			assert.ErrorIs(t, err, containererrors.ErrInvalidBuildVarKey)
			assert.Nil(t, buildVar)
		})
	}
}

func TestNewBuildVar_ValidKeyFormats(t *testing.T) {
	validKeys := []string{
		"BUILD_VAR",
		"_BUILD_VAR",
		"BUILD_123",
		"_123",
		"API_KEY_V2",
	}

	for _, key := range validKeys {
		t.Run(key, func(t *testing.T) {
			buildVar, err := NewBuildVar(1, key, "value")
			assert.NoError(t, err)
			assert.NotNil(t, buildVar)
			assert.Equal(t, key, buildVar.Key())
		})
	}
}

func TestNewBuildVar_ReservedKey(t *testing.T) {
	reservedKeys := []string{
		"PATH",
		"HOME",
		"USER",
		"SHELL",
		"PWD",
		"LANG",
		"TERM",
		"HOSTNAME",
	}

	for _, key := range reservedKeys {
		t.Run(key, func(t *testing.T) {
			buildVar, err := NewBuildVar(1, key, "value")
			assert.ErrorIs(t, err, containererrors.ErrReservedBuildVarKey)
			assert.Nil(t, buildVar)
		})
	}
}

func TestNewBuildVar_KeyTooLong(t *testing.T) {
	// Create a 256-character key (all 'A's)
	key := strings.Repeat("A", MaxBuildVarKeyLength+1)

	buildVar, err := NewBuildVar(1, key, "value")

	assert.ErrorIs(t, err, containererrors.ErrBuildVarKeyTooLong)
	assert.Nil(t, buildVar)
}

func TestNewBuildVar_EmptyValue(t *testing.T) {
	buildVar, err := NewBuildVar(1, "BUILD_VAR", "")

	assert.ErrorIs(t, err, containererrors.ErrBuildVarValueRequired)
	assert.Nil(t, buildVar)
}

func TestNewBuildVar_ValueTooLong(t *testing.T) {
	// Create a value that exceeds the maximum length
	value := strings.Repeat("a", MaxBuildVarValueLength+1)

	buildVar, err := NewBuildVar(1, "BUILD_VAR", value)

	assert.ErrorIs(t, err, containererrors.ErrBuildVarValueTooLong)
	assert.Nil(t, buildVar)
}

func TestBuildVar_UpdateValue_Success(t *testing.T) {
	buildVar, err := NewBuildVar(1, "BUILD_VAR", "old_value")
	require.NoError(t, err)

	oldUpdatedAt := buildVar.UpdatedAt()
	time.Sleep(time.Millisecond) // Ensure time difference

	err = buildVar.UpdateValue("new_value")

	require.NoError(t, err)
	assert.Equal(t, "new_value", buildVar.Value())
	assert.NotEqual(t, oldUpdatedAt, buildVar.UpdatedAt())
	assert.False(t, buildVar.UpdatedAt().IsZero())
}

func TestBuildVar_UpdateValue_EmptyValue(t *testing.T) {
	buildVar, err := NewBuildVar(1, "BUILD_VAR", "old_value")
	require.NoError(t, err)

	err = buildVar.UpdateValue("")

	assert.ErrorIs(t, err, containererrors.ErrBuildVarValueRequired)
	assert.Equal(t, "old_value", buildVar.Value()) // Value unchanged
}

func TestBuildVar_UpdateValue_ValueTooLong(t *testing.T) {
	buildVar, err := NewBuildVar(1, "BUILD_VAR", "old_value")
	require.NoError(t, err)

	value := strings.Repeat("a", MaxBuildVarValueLength+1)
	err = buildVar.UpdateValue(value)

	assert.ErrorIs(t, err, containererrors.ErrBuildVarValueTooLong)
	assert.Equal(t, "old_value", buildVar.Value()) // Value unchanged
}

func TestBuildVar_UpdateValue_SameValue(t *testing.T) {
	buildVar, err := NewBuildVar(1, "BUILD_VAR", "same_value")
	require.NoError(t, err)

	oldUpdatedAt := buildVar.UpdatedAt()

	err = buildVar.UpdateValue("same_value")

	require.NoError(t, err)
	assert.Equal(t, "same_value", buildVar.Value())
	assert.Equal(t, oldUpdatedAt, buildVar.UpdatedAt()) // UpdatedAt should not change
}

func TestBuildVar_Equals(t *testing.T) {
	buildVar1, _ := NewBuildVar(1, "BUILD_VAR", "value1")
	buildVar2, _ := NewBuildVar(1, "BUILD_VAR", "value2")
	buildVar3, _ := NewBuildVar(1, "OTHER_VAR", "value1")

	assert.True(t, buildVar1.Equals(buildVar2), "BuildVars with same key should be equal")
	assert.False(t, buildVar1.Equals(buildVar3), "BuildVars with different keys should not be equal")
	assert.False(t, buildVar1.Equals(nil), "BuildVar should not equal nil")
}

func TestReconstructBuildVar(t *testing.T) {
	buildVarID := uint(10)
	containerID := uint(1)
	key := "BUILD_VAR"
	value := "test_value"
	createdAt := time.Now().Add(-24 * time.Hour)
	updatedAt := time.Now()

	buildVar := ReconstructBuildVar(buildVarID, containerID, key, value, createdAt, updatedAt)

	assert.NotNil(t, buildVar)
	assert.Equal(t, buildVarID, buildVar.BuildVarID())
	assert.Equal(t, containerID, buildVar.ContainerID())
	assert.Equal(t, key, buildVar.Key())
	assert.Equal(t, value, buildVar.Value())
	assert.Equal(t, createdAt, buildVar.CreatedAt())
	assert.Equal(t, updatedAt, buildVar.UpdatedAt())
}

func TestBuildVar_SetBuildVarID(t *testing.T) {
	buildVar, err := NewBuildVar(1, "BUILD_VAR", "value")
	require.NoError(t, err)

	assert.Zero(t, buildVar.BuildVarID())

	buildVar.SetBuildVarID(100)

	assert.Equal(t, uint(100), buildVar.BuildVarID())
}
