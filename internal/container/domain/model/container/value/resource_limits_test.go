package value

import (
	"testing"

	"github.com/stretchr/testify/assert"
	containererrors "github.com/swm-launchpad/web-console-backend/internal/container/domain/errors"
)

func TestNewResourceLimits_ValidLimits(t *testing.T) {
	tests := []struct {
		name        string
		cpuLimit    *uint32
		memoryLimit *uint32
	}{
		{
			name:        "Both limits set",
			cpuLimit:    uint32Ptr(1000),
			memoryLimit: uint32Ptr(2048),
		},
		{
			name:        "Only CPU limit",
			cpuLimit:    uint32Ptr(500),
			memoryLimit: nil,
		},
		{
			name:        "Only memory limit",
			cpuLimit:    nil,
			memoryLimit: uint32Ptr(1024),
		},
		{
			name:        "No limits (both nil)",
			cpuLimit:    nil,
			memoryLimit: nil,
		},
		{
			name:        "Minimum limits",
			cpuLimit:    uint32Ptr(MinCPULimit),
			memoryLimit: uint32Ptr(MinMemoryLimit),
		},
		{
			name:        "Maximum limits",
			cpuLimit:    uint32Ptr(MaxCPULimit),
			memoryLimit: uint32Ptr(MaxMemoryLimit),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limits, err := NewResourceLimits(tt.cpuLimit, tt.memoryLimit)
			assert.NoError(t, err)
			assert.Equal(t, tt.cpuLimit, limits.CPULimit())
			assert.Equal(t, tt.memoryLimit, limits.MemoryLimit())
		})
	}
}

func TestNewResourceLimits_CPUOutOfRange(t *testing.T) {
	tests := []struct {
		name     string
		cpuLimit uint32
	}{
		{name: "Below minimum", cpuLimit: MinCPULimit - 1},
		{name: "Above maximum", cpuLimit: MaxCPULimit + 1},
		{name: "Zero", cpuLimit: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewResourceLimits(&tt.cpuLimit, nil)
			assert.ErrorIs(t, err, containererrors.ErrCPULimitOutOfRange)
		})
	}
}

func TestNewResourceLimits_MemoryOutOfRange(t *testing.T) {
	tests := []struct {
		name        string
		memoryLimit uint32
	}{
		{name: "Below minimum", memoryLimit: MinMemoryLimit - 1},
		{name: "Above maximum", memoryLimit: MaxMemoryLimit + 1},
		{name: "Zero", memoryLimit: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewResourceLimits(nil, &tt.memoryLimit)
			assert.ErrorIs(t, err, containererrors.ErrMemoryLimitOutOfRange)
		})
	}
}

func TestResourceLimits_OrDefault(t *testing.T) {
	tests := []struct {
		name           string
		cpuLimit       *uint32
		memoryLimit    *uint32
		cpuDefault     uint32
		memoryDefault  uint32
		expectedCPU    uint32
		expectedMemory uint32
	}{
		{
			name:           "Both set, defaults not used",
			cpuLimit:       uint32Ptr(1000),
			memoryLimit:    uint32Ptr(2048),
			cpuDefault:     500,
			memoryDefault:  1024,
			expectedCPU:    1000,
			expectedMemory: 2048,
		},
		{
			name:           "Both nil, use defaults",
			cpuLimit:       nil,
			memoryLimit:    nil,
			cpuDefault:     500,
			memoryDefault:  1024,
			expectedCPU:    500,
			expectedMemory: 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limits, _ := NewResourceLimits(tt.cpuLimit, tt.memoryLimit)
			assert.Equal(t, tt.expectedCPU, limits.CPULimitOrDefault(tt.cpuDefault))
			assert.Equal(t, tt.expectedMemory, limits.MemoryLimitOrDefault(tt.memoryDefault))
		})
	}
}

func TestResourceLimits_IsEmpty(t *testing.T) {
	emptyLimits, _ := NewResourceLimits(nil, nil)
	assert.True(t, emptyLimits.IsEmpty())

	withCPU, _ := NewResourceLimits(uint32Ptr(1000), nil)
	assert.False(t, withCPU.IsEmpty())

	withMemory, _ := NewResourceLimits(nil, uint32Ptr(2048))
	assert.False(t, withMemory.IsEmpty())

	withBoth, _ := NewResourceLimits(uint32Ptr(1000), uint32Ptr(2048))
	assert.False(t, withBoth.IsEmpty())
}

func TestResourceLimits_Equals(t *testing.T) {
	limits1, _ := NewResourceLimits(uint32Ptr(1000), uint32Ptr(2048))
	limits2, _ := NewResourceLimits(uint32Ptr(1000), uint32Ptr(2048))
	limits3, _ := NewResourceLimits(uint32Ptr(2000), uint32Ptr(2048))

	assert.True(t, limits1.Equals(limits2))
	assert.False(t, limits1.Equals(limits3))
}

func uint32Ptr(v uint32) *uint32 {
	return &v
}
