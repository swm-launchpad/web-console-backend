package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewResourceLimits(t *testing.T) {
	t.Run("성공: 유효한 리소스 제한", func(t *testing.T) {
		cpu := uint32(1000)
		memoryRequest := uint32(1024)
		memoryLimit := uint32(2048)
		disk := uint32(1024)
		traffic := uint64(256)

		limits, err := NewResourceLimits(&cpu, &memoryRequest, &memoryLimit, &disk, &traffic)

		require.NoError(t, err)
		assert.NotNil(t, limits)
		assert.Equal(t, &cpu, limits.GetCPULimit())
		assert.Equal(t, &memoryRequest, limits.GetMemoryRequest())
		assert.Equal(t, &memoryLimit, limits.GetMemoryLimit())
		assert.Equal(t, &disk, limits.GetDiskLimit())
		assert.Equal(t, &traffic, limits.GetTrafficLimit())
	})

	t.Run("성공: nil 값 허용 (무제한)", func(t *testing.T) {
		limits, err := NewResourceLimits(nil, nil, nil, nil, nil)

		require.NoError(t, err)
		assert.NotNil(t, limits)
		assert.Nil(t, limits.GetCPULimit())
		assert.Nil(t, limits.GetMemoryRequest())
		assert.Nil(t, limits.GetMemoryLimit())
		assert.Nil(t, limits.GetDiskLimit())
		assert.Nil(t, limits.GetTrafficLimit())
		assert.True(t, limits.IsUnlimited())
	})

	t.Run("성공: 일부만 nil", func(t *testing.T) {
		cpu := uint32(1000)
		memoryRequest := uint32(1024)
		memoryLimit := uint32(2048)

		limits, err := NewResourceLimits(&cpu, &memoryRequest, &memoryLimit, nil, nil)

		require.NoError(t, err)
		assert.NotNil(t, limits)
		assert.Equal(t, &cpu, limits.GetCPULimit())
		assert.Equal(t, &memoryRequest, limits.GetMemoryRequest())
		assert.Equal(t, &memoryLimit, limits.GetMemoryLimit())
		assert.Nil(t, limits.GetDiskLimit())
		assert.Nil(t, limits.GetTrafficLimit())
	})

	t.Run("성공: CPU 제한이 0", func(t *testing.T) {
		cpu := uint32(0)
		memoryLimit := uint32(2048)

		limits, err := NewResourceLimits(&cpu, nil, &memoryLimit, nil, nil)

		assert.NoError(t, err)
		assert.NotNil(t, limits)
	})

	t.Run("실패: CPU 제한이 최대값 초과", func(t *testing.T) {
		cpu := uint32(5000) // MaxCPULimit = 4000
		memoryLimit := uint32(2048)

		limits, err := NewResourceLimits(&cpu, nil, &memoryLimit, nil, nil)

		assert.Error(t, err)
		assert.Nil(t, limits)
	})

	t.Run("실패: Memory Request가 최소값 미만", func(t *testing.T) {
		memoryRequest := uint32(100) // MinMemoryRequest = 128
		memoryLimit := uint32(2048)

		limits, err := NewResourceLimits(nil, &memoryRequest, &memoryLimit, nil, nil)

		assert.Error(t, err)
		assert.Nil(t, limits)
	})

	t.Run("실패: Memory Limit이 최소값 미만", func(t *testing.T) {
		memoryLimit := uint32(100) // MinMemoryLimit = 128

		limits, err := NewResourceLimits(nil, nil, &memoryLimit, nil, nil)

		assert.Error(t, err)
		assert.Nil(t, limits)
	})

	t.Run("실패: Memory Limit이 최대값 초과", func(t *testing.T) {
		memoryLimit := uint32(10000) // MaxMemoryLimit = 8192

		limits, err := NewResourceLimits(nil, nil, &memoryLimit, nil, nil)

		assert.Error(t, err)
		assert.Nil(t, limits)
	})

	t.Run("실패: Memory Request가 Memory Limit 초과", func(t *testing.T) {
		memoryRequest := uint32(2048)
		memoryLimit := uint32(1024)

		limits, err := NewResourceLimits(nil, &memoryRequest, &memoryLimit, nil, nil)

		assert.Error(t, err)
		assert.Nil(t, limits)
	})

	t.Run("실패: Disk 제한이 최소값 미만", func(t *testing.T) {
		disk := uint32(100) // MinDiskLimit = 128

		limits, err := NewResourceLimits(nil, nil, nil, &disk, nil)

		assert.Error(t, err)
		assert.Nil(t, limits)
	})

	t.Run("실패: Disk 제한이 최대값 초과", func(t *testing.T) {
		disk := uint32(15000) // MaxDiskLimit = 10240

		limits, err := NewResourceLimits(nil, nil, nil, &disk, nil)

		assert.Error(t, err)
		assert.Nil(t, limits)
	})

	t.Run("실패: Traffic 제한이 최소값 미만", func(t *testing.T) {
		traffic := uint64(100) // MinTrafficLimit = 128

		limits, err := NewResourceLimits(nil, nil, nil, nil, &traffic)

		assert.Error(t, err)
		assert.Nil(t, limits)
	})
}

// TestNewResourceLimitsForPlan - Plan functionality disabled per user request
// func TestNewResourceLimitsForPlan(t *testing.T) { ... }

// TestResourceLimits_ValidateForPlan - Plan functionality disabled per user request
// func TestResourceLimits_ValidateForPlan(t *testing.T) { ... }

func TestResourceLimits_Exceeds(t *testing.T) {
	t.Run("초과하지 않음", func(t *testing.T) {
		cpu1 := uint32(500)
		memoryReq1 := uint32(256)
		memoryLimit1 := uint32(512)
		limits1, _ := NewResourceLimits(&cpu1, &memoryReq1, &memoryLimit1, nil, nil)

		cpu2 := uint32(1000)
		memoryReq2 := uint32(512)
		memoryLimit2 := uint32(1024)
		limits2, _ := NewResourceLimits(&cpu2, &memoryReq2, &memoryLimit2, nil, nil)

		assert.False(t, limits1.Exceeds(*limits2))
	})

	t.Run("CPU 초과", func(t *testing.T) {
		cpu1 := uint32(2000)
		limits1, _ := NewResourceLimits(&cpu1, nil, nil, nil, nil)

		cpu2 := uint32(1000)
		limits2, _ := NewResourceLimits(&cpu2, nil, nil, nil, nil)

		assert.True(t, limits1.Exceeds(*limits2))
	})

	t.Run("Memory Request 초과", func(t *testing.T) {
		memoryReq1 := uint32(1024)
		memoryLimit1 := uint32(2048)
		limits1, _ := NewResourceLimits(nil, &memoryReq1, &memoryLimit1, nil, nil)

		memoryReq2 := uint32(512)
		memoryLimit2 := uint32(2048)
		limits2, _ := NewResourceLimits(nil, &memoryReq2, &memoryLimit2, nil, nil)

		assert.True(t, limits1.Exceeds(*limits2))
	})

	t.Run("Memory Limit 초과", func(t *testing.T) {
		memoryLimit1 := uint32(2048)
		limits1, _ := NewResourceLimits(nil, nil, &memoryLimit1, nil, nil)

		memoryLimit2 := uint32(1024)
		limits2, _ := NewResourceLimits(nil, nil, &memoryLimit2, nil, nil)

		assert.True(t, limits1.Exceeds(*limits2))
	})

	t.Run("nil 비교", func(t *testing.T) {
		cpu := uint32(1000)
		limits1, _ := NewResourceLimits(&cpu, nil, nil, nil, nil)
		limits2, _ := NewResourceLimits(nil, nil, nil, nil, nil)

		// limits2가 무제한이므로 limits1은 초과하지 않음
		assert.False(t, limits1.Exceeds(*limits2))
	})
}

func TestResourceLimits_IsWithinQuota(t *testing.T) {
	cpu := uint32(1000)
	memoryRequest := uint32(1024)
	memoryLimit := uint32(2048)
	disk := uint32(512)    // Must be >= 128Mi
	traffic := uint64(256) // Must be >= 128Mi
	limits, err := NewResourceLimits(&cpu, &memoryRequest, &memoryLimit, &disk, &traffic)
	require.NoError(t, err)
	require.NotNil(t, limits)

	t.Run("할당량 내의 사용량", func(t *testing.T) {
		usage := ResourceUsage{
			CPUUsage:         500,
			MemoryReqUsage:   512,
			MemoryLimitUsage: 1024,
			DiskUsage:        256,
			TrafficUsage:     128,
		}

		assert.True(t, limits.IsWithinQuota(usage))
	})

	t.Run("할당량과 동일한 사용량", func(t *testing.T) {
		usage := ResourceUsage{
			CPUUsage:         1000,
			MemoryReqUsage:   1024,
			MemoryLimitUsage: 2048,
			DiskUsage:        512,
			TrafficUsage:     256,
		}

		assert.True(t, limits.IsWithinQuota(usage))
	})

	t.Run("CPU 할당량 초과", func(t *testing.T) {
		usage := ResourceUsage{
			CPUUsage:         1001,
			MemoryReqUsage:   512,
			MemoryLimitUsage: 1024,
			DiskUsage:        256,
			TrafficUsage:     128,
		}

		assert.False(t, limits.IsWithinQuota(usage))
	})

	t.Run("Memory Limit 할당량 초과", func(t *testing.T) {
		usage := ResourceUsage{
			CPUUsage:         500,
			MemoryReqUsage:   1024,
			MemoryLimitUsage: 2049,
			DiskUsage:        256,
			TrafficUsage:     128,
		}

		assert.False(t, limits.IsWithinQuota(usage))
	})

	t.Run("Memory Request 할당량 초과", func(t *testing.T) {
		usage := ResourceUsage{
			CPUUsage:         500,
			MemoryReqUsage:   1025,
			MemoryLimitUsage: 1500,
			DiskUsage:        256,
			TrafficUsage:     128,
		}

		assert.False(t, limits.IsWithinQuota(usage))
	})

	t.Run("Disk 할당량 초과", func(t *testing.T) {
		usage := ResourceUsage{
			CPUUsage:         500,
			MemoryReqUsage:   512,
			MemoryLimitUsage: 1024,
			DiskUsage:        513,
			TrafficUsage:     128,
		}

		assert.False(t, limits.IsWithinQuota(usage))
	})

	t.Run("Traffic 할당량 초과", func(t *testing.T) {
		usage := ResourceUsage{
			CPUUsage:         500,
			MemoryReqUsage:   512,
			MemoryLimitUsage: 1024,
			DiskUsage:        256,
			TrafficUsage:     257,
		}

		assert.False(t, limits.IsWithinQuota(usage))
	})

	t.Run("무제한 리소스", func(t *testing.T) {
		unlimitedLimits, err := NewResourceLimits(nil, nil, nil, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, unlimitedLimits)

		usage := ResourceUsage{
			CPUUsage:         999999,
			MemoryReqUsage:   999999,
			MemoryLimitUsage: 999999,
			DiskUsage:        999999,
			TrafficUsage:     999999,
		}

		assert.True(t, unlimitedLimits.IsWithinQuota(usage))
	})
}

func TestResourceLimits_Getters(t *testing.T) {
	cpu := uint32(1000)
	memoryRequest := uint32(1024)
	memoryLimit := uint32(2048)
	disk := uint32(512)
	traffic := uint64(256)
	limits, err := NewResourceLimits(&cpu, &memoryRequest, &memoryLimit, &disk, &traffic)
	require.NoError(t, err)
	require.NotNil(t, limits)

	t.Run("GetCPULimit", func(t *testing.T) {
		cpuLimit := limits.GetCPULimit()
		assert.NotNil(t, cpuLimit)
		assert.Equal(t, cpu, *cpuLimit)

		// 원본 수정이 복사본에 영향을 주지 않는지 확인
		*cpuLimit = 2000
		assert.Equal(t, uint32(1000), *limits.GetCPULimit())
	})

	t.Run("GetMemoryRequest", func(t *testing.T) {
		memReq := limits.GetMemoryRequest()
		assert.NotNil(t, memReq)
		assert.Equal(t, memoryRequest, *memReq)
	})

	t.Run("GetMemoryLimit", func(t *testing.T) {
		memLimit := limits.GetMemoryLimit()
		assert.NotNil(t, memLimit)
		assert.Equal(t, memoryLimit, *memLimit)
	})

	t.Run("GetDiskLimit", func(t *testing.T) {
		diskLimit := limits.GetDiskLimit()
		assert.NotNil(t, diskLimit)
		assert.Equal(t, disk, *diskLimit)
	})

	t.Run("GetTrafficLimit", func(t *testing.T) {
		trafficLimit := limits.GetTrafficLimit()
		assert.NotNil(t, trafficLimit)
		assert.Equal(t, traffic, *trafficLimit)
	})

	t.Run("nil 값 getter", func(t *testing.T) {
		nilLimits, _ := NewResourceLimits(nil, nil, nil, nil, nil)
		assert.Nil(t, nilLimits.GetCPULimit())
		assert.Nil(t, nilLimits.GetMemoryRequest())
		assert.Nil(t, nilLimits.GetMemoryLimit())
		assert.Nil(t, nilLimits.GetDiskLimit())
		assert.Nil(t, nilLimits.GetTrafficLimit())
	})
}

func TestResourceLimits_Equals(t *testing.T) {
	t.Run("동일한 리소스 제한", func(t *testing.T) {
		cpu := uint32(1000)
		memoryRequest := uint32(1024)
		memoryLimit := uint32(2048)
		limits1, _ := NewResourceLimits(&cpu, &memoryRequest, &memoryLimit, nil, nil)
		limits2, _ := NewResourceLimits(&cpu, &memoryRequest, &memoryLimit, nil, nil)

		assert.True(t, limits1.Equals(*limits2))
	})

	t.Run("다른 CPU 제한", func(t *testing.T) {
		cpu1 := uint32(1000)
		cpu2 := uint32(2000)
		limits1, _ := NewResourceLimits(&cpu1, nil, nil, nil, nil)
		limits2, _ := NewResourceLimits(&cpu2, nil, nil, nil, nil)

		assert.False(t, limits1.Equals(*limits2))
	})

	t.Run("nil vs non-nil", func(t *testing.T) {
		cpu := uint32(1000)
		limits1, _ := NewResourceLimits(&cpu, nil, nil, nil, nil)
		limits2, _ := NewResourceLimits(nil, nil, nil, nil, nil)

		assert.False(t, limits1.Equals(*limits2))
	})

	t.Run("모두 nil", func(t *testing.T) {
		limits1, _ := NewResourceLimits(nil, nil, nil, nil, nil)
		limits2, _ := NewResourceLimits(nil, nil, nil, nil, nil)

		assert.True(t, limits1.Equals(*limits2))
	})
}

func TestResourceLimits_HasLimitChecks(t *testing.T) {
	cpu := uint32(1000)
	memoryRequest := uint32(1024)
	memoryLimit := uint32(2048)
	limits, _ := NewResourceLimits(&cpu, &memoryRequest, &memoryLimit, nil, nil)

	t.Run("HasCPULimit", func(t *testing.T) {
		assert.True(t, limits.HasCPULimit())
	})

	t.Run("HasMemoryRequest", func(t *testing.T) {
		assert.True(t, limits.HasMemoryRequest())
	})

	t.Run("HasMemoryLimit", func(t *testing.T) {
		assert.True(t, limits.HasMemoryLimit())
	})

	t.Run("HasDiskLimit", func(t *testing.T) {
		assert.False(t, limits.HasDiskLimit())
	})

	t.Run("HasTrafficLimit", func(t *testing.T) {
		assert.False(t, limits.HasTrafficLimit())
	})

	t.Run("무제한 리소스", func(t *testing.T) {
		unlimited, _ := NewResourceLimits(nil, nil, nil, nil, nil)
		assert.False(t, unlimited.HasCPULimit())
		assert.False(t, unlimited.HasMemoryRequest())
		assert.False(t, unlimited.HasMemoryLimit())
		assert.False(t, unlimited.HasDiskLimit())
		assert.False(t, unlimited.HasTrafficLimit())
	})
}

func TestResourceLimits_IsUnlimited(t *testing.T) {
	t.Run("완전히 무제한", func(t *testing.T) {
		limits, err := NewResourceLimits(nil, nil, nil, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, limits)
		assert.True(t, limits.IsUnlimited())
	})

	t.Run("일부만 제한", func(t *testing.T) {
		cpu := uint32(1000)
		limits, err := NewResourceLimits(&cpu, nil, nil, nil, nil)
		require.NoError(t, err)
		require.NotNil(t, limits)
		assert.False(t, limits.IsUnlimited())
	})

	t.Run("모두 제한", func(t *testing.T) {
		cpu := uint32(1000)
		memoryRequest := uint32(1024)
		memoryLimit := uint32(2048)
		disk := uint32(512)
		traffic := uint64(256)
		limits, err := NewResourceLimits(&cpu, &memoryRequest, &memoryLimit, &disk, &traffic)
		require.NoError(t, err)
		require.NotNil(t, limits)
		assert.False(t, limits.IsUnlimited())
	})
}
