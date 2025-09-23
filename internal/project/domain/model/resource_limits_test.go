package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewResourceLimits(t *testing.T) {
	t.Run("성공: 유효한 리소스 제한", func(t *testing.T) {
		cpu := uint32(1000)
		memory := uint32(2048)
		disk := uint32(10)
		traffic := uint64(100)

		limits, err := NewResourceLimits(&cpu, &memory, &disk, &traffic)

		require.NoError(t, err)
		assert.NotNil(t, limits)
		assert.Equal(t, &cpu, limits.GetCPULimit())
		assert.Equal(t, &memory, limits.GetMemoryLimit())
		assert.Equal(t, &disk, limits.GetDiskLimit())
		assert.Equal(t, &traffic, limits.GetTrafficLimit())
	})

	t.Run("성공: nil 값 허용 (무제한)", func(t *testing.T) {
		limits, err := NewResourceLimits(nil, nil, nil, nil)

		require.NoError(t, err)
		assert.NotNil(t, limits)
		assert.Nil(t, limits.GetCPULimit())
		assert.Nil(t, limits.GetMemoryLimit())
		assert.Nil(t, limits.GetDiskLimit())
		assert.Nil(t, limits.GetTrafficLimit())
		assert.True(t, limits.IsUnlimited())
	})

	t.Run("성공: 일부만 nil", func(t *testing.T) {
		cpu := uint32(1000)
		memory := uint32(2048)

		limits, err := NewResourceLimits(&cpu, &memory, nil, nil)

		require.NoError(t, err)
		assert.NotNil(t, limits)
		assert.Equal(t, &cpu, limits.GetCPULimit())
		assert.Equal(t, &memory, limits.GetMemoryLimit())
		assert.Nil(t, limits.GetDiskLimit())
		assert.Nil(t, limits.GetTrafficLimit())
	})

	t.Run("실패: CPU 제한이 0", func(t *testing.T) {
		cpu := uint32(0)
		memory := uint32(2048)

		limits, err := NewResourceLimits(&cpu, &memory, nil, nil)

		assert.Error(t, err)
		assert.Nil(t, limits)
	})

	t.Run("실패: Memory 제한이 0", func(t *testing.T) {
		cpu := uint32(1000)
		memory := uint32(0)

		limits, err := NewResourceLimits(&cpu, &memory, nil, nil)

		assert.Error(t, err)
		assert.Nil(t, limits)
	})

	t.Run("실패: Disk 제한이 0", func(t *testing.T) {
		disk := uint32(0)

		limits, err := NewResourceLimits(nil, nil, &disk, nil)

		assert.Error(t, err)
		assert.Nil(t, limits)
	})

	t.Run("실패: Traffic 제한이 0", func(t *testing.T) {
		traffic := uint64(0)

		limits, err := NewResourceLimits(nil, nil, nil, &traffic)

		assert.Error(t, err)
		assert.Nil(t, limits)
	})
}

func TestNewResourceLimitsForPlan(t *testing.T) {
	t.Run("성공: 유효한 플랜", func(t *testing.T) {
		plans := []string{"free", "starter", "pro", "enterprise"}

		for _, plan := range plans {
			t.Run(plan, func(t *testing.T) {
				limits, err := NewResourceLimitsForPlan(plan)

				require.NoError(t, err)
				assert.NotNil(t, limits)
				assert.NotNil(t, limits.GetCPULimit())
				assert.NotNil(t, limits.GetMemoryLimit())
				assert.NotNil(t, limits.GetDiskLimit())
				assert.NotNil(t, limits.GetTrafficLimit())
			})
		}
	})

	t.Run("실패: 존재하지 않는 플랜", func(t *testing.T) {
		limits, err := NewResourceLimitsForPlan("invalid-plan")

		assert.Error(t, err)
		assert.Nil(t, limits)
	})
}

func TestResourceLimits_ValidateForPlan(t *testing.T) {
	t.Run("성공: 플랜 제한 내의 리소스", func(t *testing.T) {
		cpu := uint32(500)
		memory := uint32(512)
		disk := uint32(1)
		traffic := uint64(10)

		limits, _ := NewResourceLimits(&cpu, &memory, &disk, &traffic)
		err := limits.ValidateForPlan("free")

		assert.NoError(t, err)
	})

	t.Run("성공: 플랜 제한과 동일한 리소스", func(t *testing.T) {
		freeLimits, _ := NewResourceLimitsForPlan("free")
		err := freeLimits.ValidateForPlan("free")

		assert.NoError(t, err)
	})

	t.Run("실패: 플랜 제한을 초과하는 CPU", func(t *testing.T) {
		cpu := uint32(1000) // free plan limit is 500
		memory := uint32(512)

		limits, _ := NewResourceLimits(&cpu, &memory, nil, nil)
		err := limits.ValidateForPlan("free")

		assert.Error(t, err)
	})

	t.Run("실패: 존재하지 않는 플랜", func(t *testing.T) {
		cpu := uint32(500)
		limits, _ := NewResourceLimits(&cpu, nil, nil, nil)
		err := limits.ValidateForPlan("invalid-plan")

		assert.Error(t, err)
	})
}

func TestResourceLimits_Exceeds(t *testing.T) {
	t.Run("초과하지 않음", func(t *testing.T) {
		cpu1 := uint32(500)
		memory1 := uint32(512)
		limits1, _ := NewResourceLimits(&cpu1, &memory1, nil, nil)

		cpu2 := uint32(1000)
		memory2 := uint32(1024)
		limits2, _ := NewResourceLimits(&cpu2, &memory2, nil, nil)

		assert.False(t, limits1.Exceeds(*limits2))
	})

	t.Run("CPU 초과", func(t *testing.T) {
		cpu1 := uint32(2000)
		limits1, _ := NewResourceLimits(&cpu1, nil, nil, nil)

		cpu2 := uint32(1000)
		limits2, _ := NewResourceLimits(&cpu2, nil, nil, nil)

		assert.True(t, limits1.Exceeds(*limits2))
	})

	t.Run("Memory 초과", func(t *testing.T) {
		memory1 := uint32(2048)
		limits1, _ := NewResourceLimits(nil, &memory1, nil, nil)

		memory2 := uint32(1024)
		limits2, _ := NewResourceLimits(nil, &memory2, nil, nil)

		assert.True(t, limits1.Exceeds(*limits2))
	})

	t.Run("nil 비교", func(t *testing.T) {
		cpu := uint32(1000)
		limits1, _ := NewResourceLimits(&cpu, nil, nil, nil)
		limits2, _ := NewResourceLimits(nil, nil, nil, nil)

		// limits2가 무제한이므로 limits1은 초과하지 않음
		assert.False(t, limits1.Exceeds(*limits2))
	})
}

func TestResourceLimits_IsWithinQuota(t *testing.T) {
	cpu := uint32(1000)
	memory := uint32(2048)
	disk := uint32(10)
	traffic := uint64(100)
	limits, _ := NewResourceLimits(&cpu, &memory, &disk, &traffic)

	t.Run("할당량 내의 사용량", func(t *testing.T) {
		usage := ResourceUsage{
			CPUUsage:     500,
			MemoryUsage:  1024,
			DiskUsage:    5,
			TrafficUsage: 50,
		}

		assert.True(t, limits.IsWithinQuota(usage))
	})

	t.Run("할당량과 동일한 사용량", func(t *testing.T) {
		usage := ResourceUsage{
			CPUUsage:     1000,
			MemoryUsage:  2048,
			DiskUsage:    10,
			TrafficUsage: 100,
		}

		assert.True(t, limits.IsWithinQuota(usage))
	})

	t.Run("CPU 할당량 초과", func(t *testing.T) {
		usage := ResourceUsage{
			CPUUsage:     1001,
			MemoryUsage:  1024,
			DiskUsage:    5,
			TrafficUsage: 50,
		}

		assert.False(t, limits.IsWithinQuota(usage))
	})

	t.Run("Memory 할당량 초과", func(t *testing.T) {
		usage := ResourceUsage{
			CPUUsage:     500,
			MemoryUsage:  2049,
			DiskUsage:    5,
			TrafficUsage: 50,
		}

		assert.False(t, limits.IsWithinQuota(usage))
	})

	t.Run("무제한 리소스", func(t *testing.T) {
		unlimitedLimits, _ := NewResourceLimits(nil, nil, nil, nil)
		usage := ResourceUsage{
			CPUUsage:     999999,
			MemoryUsage:  999999,
			DiskUsage:    999999,
			TrafficUsage: 999999,
		}

		assert.True(t, unlimitedLimits.IsWithinQuota(usage))
	})
}

func TestResourceLimits_Getters(t *testing.T) {
	cpu := uint32(1000)
	memory := uint32(2048)
	disk := uint32(10)
	traffic := uint64(100)
	limits, _ := NewResourceLimits(&cpu, &memory, &disk, &traffic)

	t.Run("GetCPULimit", func(t *testing.T) {
		cpuLimit := limits.GetCPULimit()
		assert.NotNil(t, cpuLimit)
		assert.Equal(t, cpu, *cpuLimit)

		// 원본 수정이 복사본에 영향을 주지 않는지 확인
		*cpuLimit = 2000
		assert.Equal(t, uint32(1000), *limits.GetCPULimit())
	})

	t.Run("GetMemoryLimit", func(t *testing.T) {
		memLimit := limits.GetMemoryLimit()
		assert.NotNil(t, memLimit)
		assert.Equal(t, memory, *memLimit)
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
		nilLimits, _ := NewResourceLimits(nil, nil, nil, nil)
		assert.Nil(t, nilLimits.GetCPULimit())
		assert.Nil(t, nilLimits.GetMemoryLimit())
		assert.Nil(t, nilLimits.GetDiskLimit())
		assert.Nil(t, nilLimits.GetTrafficLimit())
	})
}

func TestResourceLimits_Equals(t *testing.T) {
	t.Run("동일한 리소스 제한", func(t *testing.T) {
		cpu := uint32(1000)
		memory := uint32(2048)
		limits1, _ := NewResourceLimits(&cpu, &memory, nil, nil)
		limits2, _ := NewResourceLimits(&cpu, &memory, nil, nil)

		assert.True(t, limits1.Equals(*limits2))
	})

	t.Run("다른 CPU 제한", func(t *testing.T) {
		cpu1 := uint32(1000)
		cpu2 := uint32(2000)
		limits1, _ := NewResourceLimits(&cpu1, nil, nil, nil)
		limits2, _ := NewResourceLimits(&cpu2, nil, nil, nil)

		assert.False(t, limits1.Equals(*limits2))
	})

	t.Run("nil vs non-nil", func(t *testing.T) {
		cpu := uint32(1000)
		limits1, _ := NewResourceLimits(&cpu, nil, nil, nil)
		limits2, _ := NewResourceLimits(nil, nil, nil, nil)

		assert.False(t, limits1.Equals(*limits2))
	})

	t.Run("모두 nil", func(t *testing.T) {
		limits1, _ := NewResourceLimits(nil, nil, nil, nil)
		limits2, _ := NewResourceLimits(nil, nil, nil, nil)

		assert.True(t, limits1.Equals(*limits2))
	})
}

func TestResourceLimits_HasLimitChecks(t *testing.T) {
	cpu := uint32(1000)
	memory := uint32(2048)
	limits, _ := NewResourceLimits(&cpu, &memory, nil, nil)

	t.Run("HasCPULimit", func(t *testing.T) {
		assert.True(t, limits.HasCPULimit())
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
		unlimited, _ := NewResourceLimits(nil, nil, nil, nil)
		assert.False(t, unlimited.HasCPULimit())
		assert.False(t, unlimited.HasMemoryLimit())
		assert.False(t, unlimited.HasDiskLimit())
		assert.False(t, unlimited.HasTrafficLimit())
	})
}

func TestResourceLimits_IsUnlimited(t *testing.T) {
	t.Run("완전히 무제한", func(t *testing.T) {
		limits, _ := NewResourceLimits(nil, nil, nil, nil)
		assert.True(t, limits.IsUnlimited())
	})

	t.Run("일부만 제한", func(t *testing.T) {
		cpu := uint32(1000)
		limits, _ := NewResourceLimits(&cpu, nil, nil, nil)
		assert.False(t, limits.IsUnlimited())
	})

	t.Run("모두 제한", func(t *testing.T) {
		cpu := uint32(1000)
		memory := uint32(2048)
		disk := uint32(10)
		traffic := uint64(100)
		limits, _ := NewResourceLimits(&cpu, &memory, &disk, &traffic)
		assert.False(t, limits.IsUnlimited())
	})
}
