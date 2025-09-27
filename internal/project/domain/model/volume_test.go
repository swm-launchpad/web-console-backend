package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	projecterrors "github.com/swm-launchpad/web-console-backend/internal/project/domain/errors"
)

func TestNewVolume(t *testing.T) {
	t.Run("성공: 유효한 볼륨 생성", func(t *testing.T) {
		volume, err := NewVolume(1, "data-volume", 512)

		require.NoError(t, err)
		assert.Equal(t, uint(0), volume.GetVolumeID())
		assert.Equal(t, uint(1), volume.GetProjectID())
		assert.Equal(t, "data-volume", volume.GetName())
		assert.Equal(t, uint32(512), volume.GetCapacity())
		assert.NotZero(t, volume.GetCreatedAt())
		assert.NotNil(t, volume.GetUpdatedAt())
	})

	t.Run("실패: 잘못된 프로젝트 ID", func(t *testing.T) {
		volume, err := NewVolume(0, "data-volume", 512)

		assert.Nil(t, volume)
		assert.Equal(t, projecterrors.ErrInvalidProjectID, err)
	})

	t.Run("실패: 빈 볼륨 이름", func(t *testing.T) {
		volume, err := NewVolume(1, "", 512)

		assert.Nil(t, volume)
		assert.Equal(t, projecterrors.ErrVolumeNameRequired, err)
	})

	t.Run("실패: 용량이 최소값 미만", func(t *testing.T) {
		volume, err := NewVolume(1, "data-volume", 100)

		assert.Nil(t, volume)
		assert.Equal(t, projecterrors.ErrVolumeCapacityTooSmall, err)
	})

	t.Run("실패: 용량이 최대값 초과", func(t *testing.T) {
		volume, err := NewVolume(1, "data-volume", 15000)

		assert.Nil(t, volume)
		assert.Equal(t, projecterrors.ErrVolumeCapacityExceeded, err)
	})
}

func TestVolume_SetVolumeID(t *testing.T) {
	volume, _ := NewVolume(1, "data-volume", 512)

	volume.SetVolumeID(999)

	assert.Equal(t, uint(999), volume.GetVolumeID())
}

func TestVolume_UpdateName(t *testing.T) {
	t.Run("성공: 이름 변경", func(t *testing.T) {
		volume, _ := NewVolume(1, "old-name", 512)
		originalUpdatedAt := volume.GetUpdatedAt()

		time.Sleep(time.Millisecond) // 타임스탬프 변경 확인용
		err := volume.UpdateName("new-name")

		require.NoError(t, err)
		assert.Equal(t, "new-name", volume.GetName())
		assert.True(t, volume.GetUpdatedAt().After(*originalUpdatedAt))
	})

	t.Run("실패: 빈 이름", func(t *testing.T) {
		volume, _ := NewVolume(1, "data-volume", 512)

		err := volume.UpdateName("")

		assert.Equal(t, projecterrors.ErrVolumeNameRequired, err)
		assert.Equal(t, "data-volume", volume.GetName())
	})
}

func TestVolume_UpdateCapacity(t *testing.T) {
	t.Run("성공: 용량 변경", func(t *testing.T) {
		volume, _ := NewVolume(1, "data-volume", 512)
		originalUpdatedAt := volume.GetUpdatedAt()

		time.Sleep(time.Millisecond)
		err := volume.UpdateCapacity(1024)

		require.NoError(t, err)
		assert.Equal(t, uint32(1024), volume.GetCapacity())
		assert.True(t, volume.GetUpdatedAt().After(*originalUpdatedAt))
	})

	t.Run("실패: 용량이 최소값 미만", func(t *testing.T) {
		volume, _ := NewVolume(1, "data-volume", 512)

		err := volume.UpdateCapacity(100)

		assert.Equal(t, projecterrors.ErrVolumeCapacityTooSmall, err)
		assert.Equal(t, uint32(512), volume.GetCapacity())
	})

	t.Run("실패: 용량이 최대값 초과", func(t *testing.T) {
		volume, _ := NewVolume(1, "data-volume", 512)

		err := volume.UpdateCapacity(15000)

		assert.Equal(t, projecterrors.ErrVolumeCapacityExceeded, err)
		assert.Equal(t, uint32(512), volume.GetCapacity())
	})
}

func TestVolume_Update(t *testing.T) {
	t.Run("성공: 이름과 용량 동시 변경", func(t *testing.T) {
		volume, _ := NewVolume(1, "old-name", 512)
		originalUpdatedAt := volume.GetUpdatedAt()

		time.Sleep(time.Millisecond)
		err := volume.Update("new-name", 1024)

		require.NoError(t, err)
		assert.Equal(t, "new-name", volume.GetName())
		assert.Equal(t, uint32(1024), volume.GetCapacity())
		assert.True(t, volume.GetUpdatedAt().After(*originalUpdatedAt))
	})

	t.Run("실패: 빈 이름", func(t *testing.T) {
		volume, _ := NewVolume(1, "data-volume", 512)

		err := volume.Update("", 1024)

		assert.Equal(t, projecterrors.ErrVolumeNameRequired, err)
		assert.Equal(t, "data-volume", volume.GetName())
		assert.Equal(t, uint32(512), volume.GetCapacity())
	})

	t.Run("실패: 용량이 최소값 미만", func(t *testing.T) {
		volume, _ := NewVolume(1, "data-volume", 512)

		err := volume.Update("new-name", 100)

		assert.Equal(t, projecterrors.ErrVolumeCapacityTooSmall, err)
		assert.Equal(t, "data-volume", volume.GetName())
		assert.Equal(t, uint32(512), volume.GetCapacity())
	})

	t.Run("실패: 용량이 최대값 초과", func(t *testing.T) {
		volume, _ := NewVolume(1, "data-volume", 512)

		err := volume.Update("new-name", 15000)

		assert.Equal(t, projecterrors.ErrVolumeCapacityExceeded, err)
		assert.Equal(t, "data-volume", volume.GetName())
		assert.Equal(t, uint32(512), volume.GetCapacity())
	})
}

func TestVolume_Equals(t *testing.T) {
	t.Run("동일한 볼륨", func(t *testing.T) {
		volume1, _ := NewVolume(1, "data-volume", 512)
		volume1.SetVolumeID(999)

		volume2, _ := NewVolume(1, "data-volume", 512)
		volume2.SetVolumeID(999)

		assert.True(t, volume1.Equals(volume2))
	})

	t.Run("다른 VolumeID", func(t *testing.T) {
		volume1, _ := NewVolume(1, "data-volume", 512)
		volume1.SetVolumeID(999)

		volume2, _ := NewVolume(1, "data-volume", 512)
		volume2.SetVolumeID(888)

		assert.False(t, volume1.Equals(volume2))
	})

	t.Run("다른 이름", func(t *testing.T) {
		volume1, _ := NewVolume(1, "volume-1", 512)
		volume2, _ := NewVolume(1, "volume-2", 512)

		assert.False(t, volume1.Equals(volume2))
	})

	t.Run("다른 용량", func(t *testing.T) {
		volume1, _ := NewVolume(1, "data-volume", 512)
		volume2, _ := NewVolume(1, "data-volume", 1024)

		assert.False(t, volume1.Equals(volume2))
	})

	t.Run("nil 비교", func(t *testing.T) {
		volume, _ := NewVolume(1, "data-volume", 512)

		assert.False(t, volume.Equals(nil))
	})
}

func TestVolume_Getters(t *testing.T) {
	now := time.Now()
	volume := &Volume{
		volumeID:  999,
		projectID: 1,
		name:      "data-volume",
		capacity:  512,
		createdAt: now,
		updatedAt: &now,
	}

	assert.Equal(t, uint(999), volume.GetVolumeID())
	assert.Equal(t, uint(1), volume.GetProjectID())
	assert.Equal(t, "data-volume", volume.GetName())
	assert.Equal(t, uint32(512), volume.GetCapacity())
	assert.Equal(t, now, volume.GetCreatedAt())
	assert.Equal(t, &now, volume.GetUpdatedAt())

	// nil updatedAt 테스트
	volumeWithNilUpdatedAt := &Volume{
		volumeID:  999,
		projectID: 1,
		name:      "data-volume",
		capacity:  512,
		createdAt: now,
		updatedAt: nil,
	}
	assert.Nil(t, volumeWithNilUpdatedAt.GetUpdatedAt())
}
