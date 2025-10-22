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
		assert.Equal(t, uint(0), volume.VolumeID())
		assert.Equal(t, uint(1), volume.ProjectID())
		assert.Equal(t, "data-volume", volume.Name())
		assert.Equal(t, uint32(512), volume.Capacity())
		assert.NotZero(t, volume.CreatedAt())
		assert.NotNil(t, volume.UpdatedAt())
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

	t.Run("성공: 단일 영소문자 이름", func(t *testing.T) {
		volume, err := NewVolume(1, "a", 512)

		require.NoError(t, err)
		assert.Equal(t, "a", volume.Name())
	})

	t.Run("성공: 하이픈 포함 이름", func(t *testing.T) {
		volume, err := NewVolume(1, "data-volume-1", 512)

		require.NoError(t, err)
		assert.Equal(t, "data-volume-1", volume.Name())
	})

	t.Run("성공: 숫자로 끝나는 이름", func(t *testing.T) {
		volume, err := NewVolume(1, "volume1", 512)

		require.NoError(t, err)
		assert.Equal(t, "volume1", volume.Name())
	})

	t.Run("성공: 대문자로 시작하는 이름 (디스플레이용)", func(t *testing.T) {
		volume, err := NewVolume(1, "Data-volume", 512)

		assert.NotNil(t, volume)
		assert.NoError(t, err)
	})

	t.Run("성공: 숫자로 시작하는 이름 (디스플레이용)", func(t *testing.T) {
		volume, err := NewVolume(1, "1-volume", 512)

		assert.NotNil(t, volume)
		assert.NoError(t, err)
	})

	t.Run("성공: 하이픈으로 시작하는 이름 (디스플레이용)", func(t *testing.T) {
		volume, err := NewVolume(1, "-volume", 512)

		assert.NotNil(t, volume)
		assert.NoError(t, err)
	})

	t.Run("성공: 하이픈으로 끝나는 이름 (디스플레이용)", func(t *testing.T) {
		volume, err := NewVolume(1, "volume-", 512)

		assert.NotNil(t, volume)
		assert.NoError(t, err)
	})

	t.Run("성공: 특수문자 포함 이름 (디스플레이용)", func(t *testing.T) {
		volume, err := NewVolume(1, "data_volume", 512)

		assert.NotNil(t, volume)
		assert.NoError(t, err)
	})

	t.Run("성공: 대문자 포함 이름 (디스플레이용)", func(t *testing.T) {
		volume, err := NewVolume(1, "dataVolume", 512)

		assert.NotNil(t, volume)
		assert.NoError(t, err)
	})

	t.Run("실패: 이름이 255자 초과", func(t *testing.T) {
		longName := string(make([]byte, 256))
		for i := range longName {
			longName = longName[:i] + "a" + longName[i+1:]
		}
		volume, err := NewVolume(1, longName, 512)

		assert.Nil(t, volume)
		assert.Equal(t, projecterrors.ErrInvalidVolumeName, err)
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

	assert.Equal(t, uint(999), volume.VolumeID())
}

func TestVolume_Getters(t *testing.T) {
	now := time.Now()
	volume := &Volume{
		volumeID:  999,
		projectID: 1,
		name:      "data-volume",
		capacity:  512,
		createdAt: now,
		updatedAt: now,
	}

	assert.Equal(t, uint(999), volume.VolumeID())
	assert.Equal(t, uint(1), volume.ProjectID())
	assert.Equal(t, "data-volume", volume.Name())
	assert.Equal(t, uint32(512), volume.Capacity())
	assert.Equal(t, now, volume.CreatedAt())
	assert.Equal(t, now, volume.UpdatedAt())
}
