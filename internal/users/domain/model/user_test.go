package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUser(t *testing.T) {
	t.Run("성공: 유효한 username과 email로 User 생성", func(t *testing.T) {
		username := "testuser"
		email := "test@example.com"

		user, err := NewUser(username, email)

		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, username, user.Username)
		assert.NotNil(t, user.Email)
		assert.Equal(t, email, *user.Email)
		assert.Equal(t, UserStatusPending, user.Status)
		assert.False(t, user.IsDeleted)
		assert.NotZero(t, user.CreatedAt)
		assert.NotNil(t, user.UpdatedAt)
	})

	t.Run("실패: username이 빈 문자열", func(t *testing.T) {
		username := ""
		email := "test@example.com"

		user, err := NewUser(username, email)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, "username is required", err.Error())
	})

	t.Run("실패: email이 빈 문자열", func(t *testing.T) {
		username := "testuser"
		email := ""

		user, err := NewUser(username, email)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, "email is required", err.Error())
	})

	t.Run("실패: username과 email 모두 빈 문자열", func(t *testing.T) {
		username := ""
		email := ""

		user, err := NewUser(username, email)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Equal(t, "username is required", err.Error())
	})
}

func TestUser_IsActive(t *testing.T) {
	t.Run("활성 사용자 확인", func(t *testing.T) {
		user := &User{
			Status:    UserStatusActive,
			IsDeleted: false,
		}

		assert.True(t, user.IsActive())
	})

	t.Run("비활성 상태 사용자", func(t *testing.T) {
		user := &User{
			Status:    UserStatusInactive,
			IsDeleted: false,
		}

		assert.False(t, user.IsActive())
	})

	t.Run("정지된 사용자", func(t *testing.T) {
		user := &User{
			Status:    UserStatusSuspended,
			IsDeleted: false,
		}

		assert.False(t, user.IsActive())
	})

	t.Run("대기 중인 사용자", func(t *testing.T) {
		user := &User{
			Status:    UserStatusPending,
			IsDeleted: false,
		}

		assert.False(t, user.IsActive())
	})

	t.Run("삭제된 활성 사용자", func(t *testing.T) {
		user := &User{
			Status:    UserStatusActive,
			IsDeleted: true,
		}

		assert.False(t, user.IsActive())
	})

	t.Run("삭제된 비활성 사용자", func(t *testing.T) {
		user := &User{
			Status:    UserStatusInactive,
			IsDeleted: true,
		}

		assert.False(t, user.IsActive())
	})
}

func TestUser_Activate(t *testing.T) {
	t.Run("성공: Pending 상태 사용자 활성화", func(t *testing.T) {
		user := &User{
			Status:    UserStatusPending,
			IsDeleted: false,
		}

		err := user.Activate()

		require.NoError(t, err)
		assert.Equal(t, UserStatusActive, user.Status)
		assert.NotNil(t, user.UpdatedAt)
		assert.True(t, user.UpdatedAt.After(time.Now().Add(-time.Second)))
	})

	t.Run("성공: Inactive 상태 사용자 활성화", func(t *testing.T) {
		user := &User{
			Status:    UserStatusInactive,
			IsDeleted: false,
		}

		err := user.Activate()

		require.NoError(t, err)
		assert.Equal(t, UserStatusActive, user.Status)
		assert.NotNil(t, user.UpdatedAt)
	})

	t.Run("성공: Suspended 상태 사용자 활성화", func(t *testing.T) {
		user := &User{
			Status:    UserStatusSuspended,
			IsDeleted: false,
		}

		err := user.Activate()

		require.NoError(t, err)
		assert.Equal(t, UserStatusActive, user.Status)
		assert.NotNil(t, user.UpdatedAt)
	})

	t.Run("성공: 이미 Active 상태인 사용자", func(t *testing.T) {
		user := &User{
			Status:    UserStatusActive,
			IsDeleted: false,
		}
		originalUpdatedAt := time.Now().Add(-time.Hour)
		user.UpdatedAt = &originalUpdatedAt

		err := user.Activate()

		require.NoError(t, err)
		assert.Equal(t, UserStatusActive, user.Status)
		assert.NotEqual(t, originalUpdatedAt, *user.UpdatedAt)
	})

	t.Run("실패: 삭제된 사용자 활성화 시도", func(t *testing.T) {
		user := &User{
			Status:    UserStatusPending,
			IsDeleted: true,
		}

		err := user.Activate()

		assert.Error(t, err)
		assert.Equal(t, "cannot activate deleted user", err.Error())
		assert.Equal(t, UserStatusPending, user.Status)
	})
}

func TestUser_UpdatePassword(t *testing.T) {
	t.Run("성공: 비밀번호 업데이트", func(t *testing.T) {
		user := &User{
			PasswordHash: "old_hash",
		}
		newHash := "new_hash_value"

		user.UpdatePassword(newHash)

		assert.Equal(t, newHash, user.PasswordHash)
		assert.NotNil(t, user.PasswordUpdatedAt)
		assert.NotNil(t, user.UpdatedAt)
		assert.True(t, user.PasswordUpdatedAt.After(time.Now().Add(-time.Second)))
		assert.True(t, user.UpdatedAt.After(time.Now().Add(-time.Second)))
	})

	t.Run("성공: 처음 비밀번호 설정", func(t *testing.T) {
		user := &User{
			PasswordHash: "",
		}
		newHash := "first_password_hash"

		user.UpdatePassword(newHash)

		assert.Equal(t, newHash, user.PasswordHash)
		assert.NotNil(t, user.PasswordUpdatedAt)
		assert.NotNil(t, user.UpdatedAt)
	})

	t.Run("성공: 연속 비밀번호 업데이트", func(t *testing.T) {
		user := &User{
			PasswordHash: "hash1",
		}

		// 첫 번째 업데이트
		user.UpdatePassword("hash2")
		firstUpdate := *user.PasswordUpdatedAt

		// 시간 차이를 두기 위한 대기
		time.Sleep(10 * time.Millisecond)

		// 두 번째 업데이트
		user.UpdatePassword("hash3")
		secondUpdate := *user.PasswordUpdatedAt

		assert.Equal(t, "hash3", user.PasswordHash)
		assert.True(t, secondUpdate.After(firstUpdate))
	})
}