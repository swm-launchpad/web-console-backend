package infrastructure

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/model"
	"github.com/swm-launchpad/web-console-backend/internal/user/infrastructure/sqlc"
)

func TestToDomainUser(t *testing.T) {
	t.Run("모든 필드가 채워진 경우", func(t *testing.T) {
		// Arrange
		now := time.Now()
		name := "Test User"
		phone := "010-1234-5678"
		org := "Test Org"

		// Act
		result := toDomainUser(
			uint(123),
			"testuser",
			"$2a$10$hashedpassword",
			sql.NullTime{Time: now, Valid: true},
			sql.NullString{String: name, Valid: true},
			"test@example.com",
			sql.NullString{String: phone, Valid: true},
			sql.NullString{String: org, Valid: true},
			sqlc.UsersStatusActive,
			false,
			sql.NullTime{},
			now,
			sql.NullTime{Time: now, Valid: true},
		)

		// Assert
		assert.Equal(t, uint(123), result.UserID)
		assert.Equal(t, "testuser", result.Username)
		assert.Equal(t, "$2a$10$hashedpassword", result.PasswordHash)
		assert.NotNil(t, result.PasswordUpdatedAt)
		assert.Equal(t, now, *result.PasswordUpdatedAt)
		assert.NotNil(t, result.Name)
		assert.Equal(t, name, *result.Name)
		assert.Equal(t, "test@example.com", result.Email)
		assert.NotNil(t, result.Phone)
		assert.Equal(t, phone, *result.Phone)
		assert.NotNil(t, result.Organization)
		assert.Equal(t, org, *result.Organization)
		assert.Equal(t, model.UserStatusActive, result.Status)
		assert.False(t, result.IsDeleted)
		assert.Nil(t, result.DeletedAt)
		assert.Equal(t, now, result.CreatedAt)
		assert.NotNil(t, result.UpdatedAt)
		assert.Equal(t, now, *result.UpdatedAt)
	})

	t.Run("NULL 가능 필드가 NULL인 경우", func(t *testing.T) {
		// Arrange
		now := time.Now()

		// Act
		result := toDomainUser(
			uint(456),
			"nulluser",
			"hashedpwd",
			sql.NullTime{},   // NULL
			sql.NullString{}, // NULL
			"",               // empty email becomes nil
			sql.NullString{}, // NULL
			sql.NullString{}, // NULL
			sqlc.UsersStatusInactive,
			true,
			sql.NullTime{Time: now, Valid: true},
			now,
			sql.NullTime{}, // NULL
		)

		// Assert
		assert.Equal(t, uint(456), result.UserID)
		assert.Equal(t, "nulluser", result.Username)
		assert.Nil(t, result.PasswordUpdatedAt)
		assert.Nil(t, result.Name)
		assert.Equal(t, "", result.Email)
		assert.Nil(t, result.Phone)
		assert.Nil(t, result.Organization)
		assert.Equal(t, model.UserStatusInactive, result.Status)
		assert.True(t, result.IsDeleted)
		assert.NotNil(t, result.DeletedAt)
		assert.Equal(t, now, *result.DeletedAt)
		assert.Nil(t, result.UpdatedAt)
	})
}

func TestToNullString(t *testing.T) {
	t.Run("nil 포인터인 경우", func(t *testing.T) {
		result := toNullString(nil)
		assert.False(t, result.Valid)
		assert.Empty(t, result.String)
	})

	t.Run("값이 있는 포인터인 경우", func(t *testing.T) {
		str := "test value"
		result := toNullString(&str)
		assert.True(t, result.Valid)
		assert.Equal(t, "test value", result.String)
	})

	t.Run("빈 문자열 포인터인 경우", func(t *testing.T) {
		str := ""
		result := toNullString(&str)
		assert.True(t, result.Valid)
		assert.Equal(t, "", result.String)
	})
}

func TestFromNullString(t *testing.T) {
	t.Run("Valid가 false인 경우", func(t *testing.T) {
		ns := sql.NullString{String: "ignored", Valid: false}
		result := fromNullString(ns)
		assert.Nil(t, result)
	})

	t.Run("Valid가 true인 경우", func(t *testing.T) {
		ns := sql.NullString{String: "test value", Valid: true}
		result := fromNullString(ns)
		require.NotNil(t, result)
		assert.Equal(t, "test value", *result)
	})

	t.Run("빈 문자열이지만 Valid가 true인 경우", func(t *testing.T) {
		ns := sql.NullString{String: "", Valid: true}
		result := fromNullString(ns)
		require.NotNil(t, result)
		assert.Equal(t, "", *result)
	})
}

func TestToNullTime(t *testing.T) {
	t.Run("nil 포인터인 경우", func(t *testing.T) {
		result := toNullTime(nil)
		assert.False(t, result.Valid)
		assert.Equal(t, time.Time{}, result.Time)
	})

	t.Run("값이 있는 포인터인 경우", func(t *testing.T) {
		now := time.Now()
		result := toNullTime(&now)
		assert.True(t, result.Valid)
		assert.Equal(t, now, result.Time)
	})
}

func TestFromNullTime(t *testing.T) {
	t.Run("Valid가 false인 경우", func(t *testing.T) {
		nt := sql.NullTime{Time: time.Now(), Valid: false}
		result := fromNullTime(nt)
		assert.Nil(t, result)
	})

	t.Run("Valid가 true인 경우", func(t *testing.T) {
		now := time.Now()
		nt := sql.NullTime{Time: now, Valid: true}
		result := fromNullTime(nt)
		require.NotNil(t, result)
		assert.Equal(t, now, *result)
	})
}

func TestPtrToString(t *testing.T) {
	t.Run("nil 포인터인 경우", func(t *testing.T) {
		result := ptrToString(nil)
		assert.Empty(t, result)
	})

	t.Run("값이 있는 포인터인 경우", func(t *testing.T) {
		str := "test value"
		result := ptrToString(&str)
		assert.Equal(t, "test value", result)
	})

	t.Run("빈 문자열 포인터인 경우", func(t *testing.T) {
		str := ""
		result := ptrToString(&str)
		assert.Equal(t, "", result)
	})
}

func TestStringToPtr(t *testing.T) {
	t.Run("빈 문자열인 경우", func(t *testing.T) {
		result := stringToPtr("")
		assert.Nil(t, result)
	})

	t.Run("값이 있는 문자열인 경우", func(t *testing.T) {
		result := stringToPtr("test value")
		require.NotNil(t, result)
		assert.Equal(t, "test value", *result)
	})
}

func TestIsDuplicateError(t *testing.T) {
	t.Run("nil 에러인 경우", func(t *testing.T) {
		result := isDuplicateError(nil)
		assert.False(t, result)
	})

	t.Run("Duplicate entry를 포함하는 에러", func(t *testing.T) {
		err := errors.New("Error 1062: Duplicate entry 'testuser' for key 'username'")
		result := isDuplicateError(err)
		assert.True(t, result)
	})

	t.Run("1062 에러 코드를 포함하는 에러", func(t *testing.T) {
		err := errors.New("MySQL Error 1062")
		result := isDuplicateError(err)
		assert.True(t, result)
	})

	t.Run("일반 에러", func(t *testing.T) {
		err := errors.New("some other error")
		result := isDuplicateError(err)
		assert.False(t, result)
	})

	t.Run("빈 에러 메시지", func(t *testing.T) {
		err := errors.New("")
		result := isDuplicateError(err)
		assert.False(t, result)
	})
}

func TestHelperFunctionsIntegration(t *testing.T) {
	t.Run("toNullString과 fromNullString 왕복 변환", func(t *testing.T) {
		original := "test string"
		nullStr := toNullString(&original)
		result := fromNullString(nullStr)
		require.NotNil(t, result)
		assert.Equal(t, original, *result)
	})

	t.Run("toNullTime과 fromNullTime 왕복 변환", func(t *testing.T) {
		original := time.Now()
		nullTime := toNullTime(&original)
		result := fromNullTime(nullTime)
		require.NotNil(t, result)
		assert.Equal(t, original, *result)
	})

	t.Run("ptrToString과 stringToPtr 상호 작용", func(t *testing.T) {
		// 비어있지 않은 문자열
		str := "test"
		ptr := stringToPtr(str)
		require.NotNil(t, ptr)
		result := ptrToString(ptr)
		assert.Equal(t, str, result)

		// 빈 문자열 -> nil 포인터 -> 빈 문자열
		emptyPtr := stringToPtr("")
		assert.Nil(t, emptyPtr)
		emptyResult := ptrToString(emptyPtr)
		assert.Equal(t, "", emptyResult)
	})
}
