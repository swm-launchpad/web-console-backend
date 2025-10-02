package value

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewResourceLimits(t *testing.T) {
	t.Run("성공: 유효한 리소스 제한", func(t *testing.T) {
		cpu := uint32(1000)
		memoryLimit := uint32(2048)
		disk := uint32(1024)
		traffic := uint32(256)

		limits, err := NewResourceLimits(cpu, memoryLimit, disk, traffic)

		require.NoError(t, err)
		assert.NotNil(t, limits)
		assert.Equal(t, cpu, limits.CPULimit())
		assert.Equal(t, memoryLimit, limits.MemoryLimit())
		assert.Equal(t, disk, limits.DiskLimit())
		assert.Equal(t, traffic, limits.TrafficLimit())
	})

	t.Run("성공: 최소 유효 값", func(t *testing.T) {
		cpu := uint32(100)
		memoryLimit := uint32(128)
		disk := uint32(128)
		traffic := uint32(128)

		limits, err := NewResourceLimits(cpu, memoryLimit, disk, traffic)

		require.NoError(t, err)
		assert.NotNil(t, limits)
		assert.Equal(t, cpu, limits.CPULimit())
		assert.Equal(t, memoryLimit, limits.MemoryLimit())
		assert.Equal(t, disk, limits.DiskLimit())
		assert.Equal(t, traffic, limits.TrafficLimit())
	})

	t.Run("성공: 최대 유효 값", func(t *testing.T) {
		cpu := uint32(4000)
		memoryLimit := uint32(8192)
		disk := uint32(10240)
		traffic := uint32(1073741824) // 1TB

		limits, err := NewResourceLimits(cpu, memoryLimit, disk, traffic)

		require.NoError(t, err)
		assert.NotNil(t, limits)
		assert.Equal(t, cpu, limits.CPULimit())
		assert.Equal(t, memoryLimit, limits.MemoryLimit())
		assert.Equal(t, disk, limits.DiskLimit())
		assert.Equal(t, traffic, limits.TrafficLimit())
	})

	t.Run("실패: CPU 제한이 최소값 미만", func(t *testing.T) {
		cpu := uint32(50) // MinCPULimit = 100
		memoryLimit := uint32(2048)

		limits, err := NewResourceLimits(cpu, memoryLimit, 128, 128)

		assert.Error(t, err)
		assert.Nil(t, limits)
	})

	t.Run("실패: CPU 제한이 최대값 초과", func(t *testing.T) {
		cpu := uint32(5000) // MaxCPULimit = 4000
		memoryLimit := uint32(2048)

		limits, err := NewResourceLimits(cpu, memoryLimit, 128, 128)

		assert.Error(t, err)
		assert.Nil(t, limits)
	})

	t.Run("실패: Memory Limit이 최소값 미만", func(t *testing.T) {
		memoryLimit := uint32(100) // MinMemoryLimit = 128

		limits, err := NewResourceLimits(100, memoryLimit, 128, 128)

		assert.Error(t, err)
		assert.Nil(t, limits)
	})

	t.Run("실패: Memory Limit이 최대값 초과", func(t *testing.T) {
		memoryLimit := uint32(10000) // MaxMemoryLimit = 8192

		limits, err := NewResourceLimits(100, memoryLimit, 128, 128)

		assert.Error(t, err)
		assert.Nil(t, limits)
	})

	t.Run("실패: Disk 제한이 최소값 미만", func(t *testing.T) {
		disk := uint32(100) // MinDiskLimit = 128

		limits, err := NewResourceLimits(100, 128, disk, 128)

		assert.Error(t, err)
		assert.Nil(t, limits)
	})

	t.Run("실패: Disk 제한이 최대값 초과", func(t *testing.T) {
		disk := uint32(15000) // MaxDiskLimit = 10240

		limits, err := NewResourceLimits(100, 128, disk, 128)

		assert.Error(t, err)
		assert.Nil(t, limits)
	})

	t.Run("실패: Traffic 제한이 최소값 미만", func(t *testing.T) {
		traffic := uint32(100) // MinTrafficLimit = 128

		limits, err := NewResourceLimits(100, 128, 128, traffic)

		assert.Error(t, err)
		assert.Nil(t, limits)
	})

	t.Run("실패: Traffic 제한이 최대값 초과", func(t *testing.T) {
		traffic := uint32(2000000000) // MaxTrafficLimit = 1073741824 (1TB)

		limits, err := NewResourceLimits(100, 128, 128, traffic)

		assert.Error(t, err)
		assert.Nil(t, limits)
	})
}

// TestNewResourceLimitsForPlan - Plan functionality disabled per user request
// func TestNewResourceLimitsForPlan(t *testing.T) { ... }

// TestResourceLimits_ValidateForPlan - Plan functionality disabled per user request
// func TestResourceLimits_ValidateForPlan(t *testing.T) { ... }

func TestResourceLimits_Getters(t *testing.T) {
	cpu := uint32(1000)
	memoryLimit := uint32(2048)
	disk := uint32(512)
	traffic := uint32(256)
	limits, err := NewResourceLimits(cpu, memoryLimit, disk, traffic)
	require.NoError(t, err)
	require.NotNil(t, limits)

	t.Run("GetCPULimit", func(t *testing.T) {
		cpuLimit := limits.CPULimit()
		assert.Equal(t, cpu, cpuLimit)
		assert.Equal(t, uint32(1000), cpuLimit)
	})

	t.Run("GetMemoryLimit", func(t *testing.T) {
		memLimit := limits.MemoryLimit()
		assert.Equal(t, memoryLimit, memLimit)
	})

	t.Run("GetDiskLimit", func(t *testing.T) {
		diskLimit := limits.DiskLimit()
		assert.Equal(t, disk, diskLimit)
	})

	t.Run("GetTrafficLimit", func(t *testing.T) {
		trafficLimit := limits.TrafficLimit()
		assert.Equal(t, traffic, trafficLimit)
	})

	t.Run("최소 값 getter", func(t *testing.T) {
		minLimits, _ := NewResourceLimits(100, 128, 128, 128)
		assert.Equal(t, uint32(100), minLimits.CPULimit())
		assert.Equal(t, uint32(128), minLimits.MemoryLimit())
		assert.Equal(t, uint32(128), minLimits.DiskLimit())
		assert.Equal(t, uint32(128), minLimits.TrafficLimit())
	})
}

func TestResourceLimits_Equals(t *testing.T) {
	t.Run("동일한 리소스 제한", func(t *testing.T) {
		cpu := uint32(1000)
		memoryLimit := uint32(2048)
		limits1, _ := NewResourceLimits(cpu, memoryLimit, 128, 128)
		limits2, _ := NewResourceLimits(cpu, memoryLimit, 128, 128)

		assert.True(t, limits1.Equals(*limits2))
	})

	t.Run("다른 CPU 제한", func(t *testing.T) {
		cpu1 := uint32(1000)
		cpu2 := uint32(2000)
		limits1, _ := NewResourceLimits(cpu1, 128, 128, 128)
		limits2, _ := NewResourceLimits(cpu2, 128, 128, 128)

		assert.False(t, limits1.Equals(*limits2))
	})

	t.Run("최소값 vs 일반값", func(t *testing.T) {
		limits1, _ := NewResourceLimits(1000, 2048, 2048, 256)
		limits2, _ := NewResourceLimits(100, 128, 128, 128)

		assert.False(t, limits1.Equals(*limits2))
	})

	t.Run("모두 최소값", func(t *testing.T) {
		limits1, _ := NewResourceLimits(100, 128, 128, 128)
		limits2, _ := NewResourceLimits(100, 128, 128, 128)

		assert.True(t, limits1.Equals(*limits2))
	})
}
