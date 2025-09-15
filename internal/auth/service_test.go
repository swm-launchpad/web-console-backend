package auth

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swm-launchpad/web-console-backend/internal/auth/jwt"
	"github.com/swm-launchpad/web-console-backend/internal/auth/password"
)

func TestAuthService_GenerateToken(t *testing.T) {
	t.Run("토큰 생성 및 검증 통합 테스트", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		jwtSvc := jwt.NewService("test-secret-key")
		pwdSvc := password.NewService()
		service := NewAuthService(jwtSvc, pwdSvc)

		userID := uint(123)

		// Act - Generate token
		token, err := service.GenerateToken(ctx, userID)

		// Assert
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		// Verify the generated token
		validatedUserID, err := service.ValidateToken(ctx, token)
		require.NoError(t, err)
		assert.Equal(t, userID, validatedUserID)
	})

	t.Run("서로 다른 사용자 ID로 토큰 생성", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		jwtSvc := jwt.NewService("test-secret-key")
		pwdSvc := password.NewService()
		service := NewAuthService(jwtSvc, pwdSvc)

		userID1 := uint(123)
		userID2 := uint(456)

		// Act
		token1, err1 := service.GenerateToken(ctx, userID1)
		token2, err2 := service.GenerateToken(ctx, userID2)

		// Assert
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEmpty(t, token1)
		assert.NotEmpty(t, token2)
		assert.NotEqual(t, token1, token2) // Different tokens for different users

		// Verify each token
		validatedID1, _ := service.ValidateToken(ctx, token1)
		validatedID2, _ := service.ValidateToken(ctx, token2)
		assert.Equal(t, userID1, validatedID1)
		assert.Equal(t, userID2, validatedID2)
	})
}

func TestAuthService_ValidateToken(t *testing.T) {
	t.Run("잘못된 형식의 토큰 검증", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		jwtSvc := jwt.NewService("test-secret-key")
		pwdSvc := password.NewService()
		service := NewAuthService(jwtSvc, pwdSvc)

		invalidToken := "invalid.token.format"

		// Act
		userID, err := service.ValidateToken(ctx, invalidToken)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, uint(0), userID)
	})

	t.Run("다른 시크릿으로 서명된 토큰 검증", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		jwtSvc1 := jwt.NewService("secret-key-1")
		jwtSvc2 := jwt.NewService("secret-key-2")
		pwdSvc := password.NewService()

		service1 := NewAuthService(jwtSvc1, pwdSvc)
		service2 := NewAuthService(jwtSvc2, pwdSvc)

		// Generate token with service1
		token, err := service1.GenerateToken(ctx, 123)
		require.NoError(t, err)

		// Act - Try to validate with service2 (different secret)
		userID, err := service2.ValidateToken(ctx, token)

		// Assert
		assert.Error(t, err)
		assert.Equal(t, uint(0), userID)
	})

	t.Run("빈 토큰 검증", func(t *testing.T) {
		// Arrange
		ctx := context.Background()
		jwtSvc := jwt.NewService("test-secret-key")
		pwdSvc := password.NewService()
		service := NewAuthService(jwtSvc, pwdSvc)

		// Act
		userID, err := service.ValidateToken(ctx, "")

		// Assert
		assert.Error(t, err)
		assert.Equal(t, uint(0), userID)
	})
}

func TestAuthService_HashPassword(t *testing.T) {
	t.Run("패스워드 해싱 및 검증 통합 테스트", func(t *testing.T) {
		// Arrange
		jwtSvc := jwt.NewService("test-secret-key")
		pwdSvc := password.NewService()
		service := NewAuthService(jwtSvc, pwdSvc)

		password := "SecurePassword123!"

		// Act - Hash password
		hash, err := service.HashPassword(password)

		// Assert
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.NotEqual(t, password, hash) // Hash should be different from original

		// Verify the password
		err = service.VerifyPassword(hash, password)
		assert.NoError(t, err)
	})

	t.Run("같은 패스워드로 다른 해시 생성", func(t *testing.T) {
		// Arrange
		jwtSvc := jwt.NewService("test-secret-key")
		pwdSvc := password.NewService()
		service := NewAuthService(jwtSvc, pwdSvc)

		password := "TestPassword456"

		// Act
		hash1, err1 := service.HashPassword(password)
		hash2, err2 := service.HashPassword(password)

		// Assert
		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEmpty(t, hash1)
		assert.NotEmpty(t, hash2)
		assert.NotEqual(t, hash1, hash2) // Same password, different hashes (due to salt)

		// Both hashes should verify correctly
		assert.NoError(t, service.VerifyPassword(hash1, password))
		assert.NoError(t, service.VerifyPassword(hash2, password))
	})

	t.Run("빈 패스워드 해싱", func(t *testing.T) {
		// Arrange
		jwtSvc := jwt.NewService("test-secret-key")
		pwdSvc := password.NewService()
		service := NewAuthService(jwtSvc, pwdSvc)

		// Act
		hash, err := service.HashPassword("")

		// Assert
		assert.Error(t, err)
		assert.Empty(t, hash)
	})

	t.Run("매우 긴 패스워드 해싱", func(t *testing.T) {
		// Arrange
		jwtSvc := jwt.NewService("test-secret-key")
		pwdSvc := password.NewService()
		service := NewAuthService(jwtSvc, pwdSvc)

		// bcrypt has a 72 byte limit
		longPassword := ""
		for i := 0; i < 100; i++ {
			longPassword += "a"
		}

		// Act
		hash, err := service.HashPassword(longPassword)

		// Assert - bcrypt will return an error for passwords exceeding 72 bytes
		assert.Error(t, err)
		assert.Empty(t, hash)
	})
}

func TestAuthService_VerifyPassword(t *testing.T) {
	t.Run("올바른 패스워드 검증", func(t *testing.T) {
		// Arrange
		jwtSvc := jwt.NewService("test-secret-key")
		pwdSvc := password.NewService()
		service := NewAuthService(jwtSvc, pwdSvc)

		password := "CorrectPassword123!"
		hash, _ := service.HashPassword(password)

		// Act
		err := service.VerifyPassword(hash, password)

		// Assert
		assert.NoError(t, err)
	})

	t.Run("잘못된 패스워드 검증", func(t *testing.T) {
		// Arrange
		jwtSvc := jwt.NewService("test-secret-key")
		pwdSvc := password.NewService()
		service := NewAuthService(jwtSvc, pwdSvc)

		correctPassword := "CorrectPassword123!"
		wrongPassword := "WrongPassword456!"
		hash, _ := service.HashPassword(correctPassword)

		// Act
		err := service.VerifyPassword(hash, wrongPassword)

		// Assert
		assert.Error(t, err)
	})

	t.Run("빈 해시로 검증", func(t *testing.T) {
		// Arrange
		jwtSvc := jwt.NewService("test-secret-key")
		pwdSvc := password.NewService()
		service := NewAuthService(jwtSvc, pwdSvc)

		// Act
		err := service.VerifyPassword("", "password")

		// Assert
		assert.Error(t, err)
	})

	t.Run("잘못된 형식의 해시로 검증", func(t *testing.T) {
		// Arrange
		jwtSvc := jwt.NewService("test-secret-key")
		pwdSvc := password.NewService()
		service := NewAuthService(jwtSvc, pwdSvc)

		// Act
		err := service.VerifyPassword("not-a-bcrypt-hash", "password")

		// Assert
		assert.Error(t, err)
	})

	t.Run("대소문자 구분 검증", func(t *testing.T) {
		// Arrange
		jwtSvc := jwt.NewService("test-secret-key")
		pwdSvc := password.NewService()
		service := NewAuthService(jwtSvc, pwdSvc)

		password := "Password123"
		hash, _ := service.HashPassword(password)

		// Act
		err1 := service.VerifyPassword(hash, "password123") // lowercase
		err2 := service.VerifyPassword(hash, "PASSWORD123") // uppercase

		// Assert
		assert.Error(t, err1)
		assert.Error(t, err2)
	})
}

func TestNewAuthService(t *testing.T) {
	t.Run("AuthService 생성", func(t *testing.T) {
		// Arrange
		jwtSvc := jwt.NewService("test-secret-key")
		pwdSvc := password.NewService()

		// Act
		service := NewAuthService(jwtSvc, pwdSvc)

		// Assert
		assert.NotNil(t, service)
		authSvc, ok := service.(*AuthService)
		require.True(t, ok)
		assert.NotNil(t, authSvc.jwtService)
		assert.NotNil(t, authSvc.passwordService)
	})
}