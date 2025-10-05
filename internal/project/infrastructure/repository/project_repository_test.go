package repository

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/project/domain/model/project/value"
	"github.com/swm-launchpad/web-console-backend/internal/project/infrastructure/repository/sqlc"
)

func TestToNullTime(t *testing.T) {
	t.Run("nil 포인터인 경우", func(t *testing.T) {
		result := toNullTime(nil)
		assert.False(t, result.Valid)
		assert.True(t, result.Time.IsZero())
	})

	t.Run("값이 있는 포인터인 경우", func(t *testing.T) {
		value := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)
		result := toNullTime(&value)
		assert.True(t, result.Valid)
		assert.Equal(t, value, result.Time)
	})
}

func TestFromNullTime(t *testing.T) {
	t.Run("Valid가 false인 경우", func(t *testing.T) {
		nullTime := sql.NullTime{Valid: false, Time: time.Now()}
		result := fromNullTime(nullTime)
		assert.Nil(t, result)
	})

	t.Run("Valid가 true인 경우", func(t *testing.T) {
		value := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)
		nullTime := sql.NullTime{Valid: true, Time: value}
		result := fromNullTime(nullTime)
		assert.NotNil(t, result)
		assert.Equal(t, value, *result)
	})
}

func TestToDomainProject(t *testing.T) {
	t.Run("모든 필드가 채워진 경우", func(t *testing.T) {
		// Arrange
		repo := &projectRepository{}
		now := time.Now()
		fqdn := "test.example.com"
		plan := "premium"

		sqlcProject := sqlc.Project{
			ProjectID:    123,
			Name:         "테스트 프로젝트",
			Slug:         "test-project",
			Fqdn:         sql.NullString{String: fqdn, Valid: true},
			Status:       sqlc.ProjectsStatusActive,
			Plan:         sql.NullString{String: plan, Valid: true},
			CpuLimit:     sql.NullInt32{Int32: 1000, Valid: true},
			MemoryLimit:  sql.NullInt32{Int32: 2048, Valid: true},
			DiskLimit:    sql.NullInt32{Int32: 2048, Valid: true},
			TrafficLimit: sql.NullInt64{Int64: 10485760, Valid: true},
			CreatedAt:    now,
			UpdatedAt:    sql.NullTime{Time: now, Valid: true},
		}

		// Act
		result, err := repo.toDomainProject(sqlcProject)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, uint(123), result.ProjectID())
		assert.Equal(t, "테스트 프로젝트", result.Name())
		assert.Equal(t, "test-project", result.Slug().String())
		fqdnResult, fqdnOk := result.FQDN()
		assert.True(t, fqdnOk)
		assert.Equal(t, fqdn, fqdnResult)
		assert.Equal(t, value.ProjectStatusActive, result.Status())
		planResult, planOk := result.Plan()
		assert.True(t, planOk)
		assert.Equal(t, plan, planResult)

		limits := result.Limits()
		assert.Equal(t, uint32(1000), limits.CPULimit())
		assert.Equal(t, uint32(2048), limits.MemoryLimit())
		assert.Equal(t, uint32(2048), limits.DiskLimit())
		assert.Equal(t, uint32(10485760), limits.TrafficLimit())

		// Check times are close enough (within 1 second)
		assert.WithinDuration(t, now, result.CreatedAt(), time.Second)
		assert.NotZero(t, result.UpdatedAt())
		assert.WithinDuration(t, now, result.UpdatedAt(), time.Second)
	})

	t.Run("NULL 가능 필드가 NULL인 경우", func(t *testing.T) {
		// Arrange
		repo := &projectRepository{}
		now := time.Now()

		sqlcProject := sqlc.Project{
			ProjectID:    456,
			Name:         "최소 프로젝트",
			Slug:         "minimal-project",
			Fqdn:         sql.NullString{Valid: false},
			Status:       sqlc.ProjectsStatusActive, // Use active status since only active is supported in domain
			Plan:         sql.NullString{Valid: false},
			CpuLimit:     sql.NullInt32{Int32: 100, Valid: true}, // Minimum value
			MemoryLimit:  sql.NullInt32{Int32: 128, Valid: true}, // Minimum value
			DiskLimit:    sql.NullInt32{Int32: 128, Valid: true}, // Minimum value
			TrafficLimit: sql.NullInt64{Int64: 128, Valid: true}, // Minimum value
			CreatedAt:    now,
			UpdatedAt:    sql.NullTime{Valid: false},
		}

		// Act
		result, err := repo.toDomainProject(sqlcProject)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, uint(456), result.ProjectID())
		assert.Equal(t, "최소 프로젝트", result.Name())
		assert.Equal(t, "minimal-project", result.Slug().String())
		_, fqdnOk := result.FQDN()
		assert.False(t, fqdnOk)
		assert.Equal(t, value.ProjectStatusActive, result.Status()) // Only active status is supported
		_, planOk := result.Plan()
		assert.False(t, planOk)

		limits := result.Limits()
		assert.Equal(t, uint32(100), limits.CPULimit())
		assert.Equal(t, uint32(128), limits.MemoryLimit())
		assert.Equal(t, uint32(128), limits.DiskLimit())
		assert.Equal(t, uint32(128), limits.TrafficLimit())

		// Check time is close enough
		assert.WithinDuration(t, now, result.CreatedAt(), time.Second)
		// UpdatedAt is always set by the domain model
		assert.NotZero(t, result.UpdatedAt())
		assert.WithinDuration(t, now, result.UpdatedAt(), time.Second)
	})
}

func TestToDomainProjectUser(t *testing.T) {
	t.Run("프로젝트 사용자 변환", func(t *testing.T) {
		// Arrange
		repo := &projectRepository{}
		now := time.Now()

		sqlcUser := sqlc.ProjectUser{
			ProjectID: 123,
			UserID:    456,
			Role:      sqlc.ProjectUserRoleOwner,
			CreatedAt: now,
			UpdatedAt: sql.NullTime{Time: now, Valid: true},
		}

		// Act
		result, err := repo.toDomainProjectUser(sqlcUser)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, uint(456), result.UserID())
		assert.Equal(t, value.ProjectUserRoleOwner, result.Role())
		assert.WithinDuration(t, now, result.CreatedAt(), time.Second)
		assert.NotZero(t, result.UpdatedAt())
		assert.WithinDuration(t, now, result.UpdatedAt(), time.Second)
		assert.True(t, result.IsActive())
	})

	t.Run("업데이트 시간이 NULL인 경우", func(t *testing.T) {
		// Arrange
		repo := &projectRepository{}
		now := time.Now()

		sqlcUser := sqlc.ProjectUser{
			ProjectID: 123,
			UserID:    789,
			Role:      sqlc.ProjectUserRoleOwner,
			CreatedAt: now,
			UpdatedAt: sql.NullTime{Valid: false},
		}

		// Act
		result, err := repo.toDomainProjectUser(sqlcUser)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, uint(789), result.UserID())
		assert.Equal(t, value.ProjectUserRoleOwner, result.Role())
		assert.WithinDuration(t, now, result.CreatedAt(), time.Second)
		// UpdatedAt is always set by domain model
		assert.NotZero(t, result.UpdatedAt())
		assert.WithinDuration(t, now, result.UpdatedAt(), time.Second)
		assert.True(t, result.IsActive())
	})
}

func TestIsDuplicateError(t *testing.T) {
	t.Run("중복 에러인 경우", func(t *testing.T) {
		err := assert.AnError // This is just a placeholder; real duplicate errors would have specific messages
		// Note: 실제 구현에서는 MySQL의 duplicate key error message를 확인해야 함
		// 예: "Error 1062: Duplicate entry"
		result := isDuplicateError(err)
		// 이 테스트는 실제 구현에 따라 조정이 필요함
		assert.False(t, result) // 현재는 placeholder error이므로 false
	})

	t.Run("중복이 아닌 에러인 경우", func(t *testing.T) {
		err := assert.AnError
		result := isDuplicateError(err)
		assert.False(t, result)
	})

	t.Run("nil 에러인 경우", func(t *testing.T) {
		result := isDuplicateError(nil)
		assert.False(t, result)
	})
}
