package jwt

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewJWTUtil(t *testing.T) {
	t.Run("성공: JWT 유틸 생성", func(t *testing.T) {
		secret := "test-secret-key"

		util := NewJWTUtil(secret)

		assert.NotNil(t, util)
	})

	t.Run("빈 시크릿 키로도 생성 가능", func(t *testing.T) {
		secret := ""

		util := NewJWTUtil(secret)

		assert.NotNil(t, util)
	})
}

func TestJWTUtil_GenerateToken(t *testing.T) {
	ctx := context.Background()
	secret := "test-secret-key-for-testing"
	util := NewJWTUtil(secret)

	t.Run("성공: 토큰 생성", func(t *testing.T) {
		userID := uint(123)

		token, err := util.GenerateToken(ctx, userID)

		require.NoError(t, err)
		assert.NotEmpty(t, token)
		assert.True(t, strings.Count(token, ".") == 2) // JWT는 3 부분으로 구성
	})

	t.Run("성공: 다른 userID로 토큰 생성", func(t *testing.T) {
		userID1 := uint(123)
		userID2 := uint(456)

		token1, err1 := util.GenerateToken(ctx, userID1)
		token2, err2 := util.GenerateToken(ctx, userID2)

		require.NoError(t, err1)
		require.NoError(t, err2)
		assert.NotEqual(t, token1, token2)
	})

	t.Run("성공: userID 0으로 토큰 생성", func(t *testing.T) {
		userID := uint(0)

		token, err := util.GenerateToken(ctx, userID)

		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("성공: 최대 userID로 토큰 생성", func(t *testing.T) {
		userID := uint(^uint(0))

		token, err := util.GenerateToken(ctx, userID)

		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})
}

func TestJWTUtil_ValidateToken(t *testing.T) {
	ctx := context.Background()
	secret := "test-secret-key-for-validation"
	util := NewJWTUtil(secret)

	t.Run("성공: 유효한 토큰 검증", func(t *testing.T) {
		userID := uint(123)
		token, err := util.GenerateToken(ctx, userID)
		require.NoError(t, err)

		validatedUserID, err := util.ValidateToken(ctx, token)

		require.NoError(t, err)
		assert.Equal(t, userID, validatedUserID)
	})

	t.Run("성공: userID 0 토큰 검증", func(t *testing.T) {
		userID := uint(0)
		token, err := util.GenerateToken(ctx, userID)
		require.NoError(t, err)

		validatedUserID, err := util.ValidateToken(ctx, token)

		require.NoError(t, err)
		assert.Equal(t, userID, validatedUserID)
	})

	t.Run("실패: 잘못된 시크릿 키로 서명된 토큰", func(t *testing.T) {
		userID := uint(123)

		// 다른 시크릿 키로 토큰 생성
		wrongUtil := NewJWTUtil("wrong-secret")
		token, err := wrongUtil.GenerateToken(ctx, userID)
		require.NoError(t, err)

		// 원래 서비스로 검증 시도
		validatedUserID, err := util.ValidateToken(ctx, token)

		assert.Error(t, err)
		assert.Equal(t, uint(0), validatedUserID)
		assert.Contains(t, err.Error(), "invalid token")
	})

	t.Run("실패: 만료된 토큰", func(t *testing.T) {
		// 만료된 토큰을 직접 생성 (24시간 전으로 설정)
		userIDStr := "123"
		claims := jwt.MapClaims{
			"user_id": userIDStr,
			"exp":     time.Now().Add(-25 * time.Hour).Unix(), // 25시간 전 만료
		}
		jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		token, err := jwtToken.SignedString([]byte(secret))
		require.NoError(t, err)

		validatedUserID, err := util.ValidateToken(ctx, token)

		assert.Error(t, err)
		assert.Equal(t, uint(0), validatedUserID)
		assert.Contains(t, err.Error(), "invalid token")
	})

	t.Run("실패: 잘못된 형식의 토큰", func(t *testing.T) {
		invalidTokens := []string{
			"",
			"invalid",
			"invalid.token",
			"invalid.token.format.with.extra",
			"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.invalid",
		}

		for _, token := range invalidTokens {
			validatedUserID, err := util.ValidateToken(ctx, token)

			assert.Error(t, err, "Token: %s", token)
			assert.Equal(t, uint(0), validatedUserID)
			assert.Contains(t, err.Error(), "invalid token")
		}
	})

	t.Run("실패: 변조된 토큰", func(t *testing.T) {
		userID := uint(123)
		token, err := util.GenerateToken(ctx, userID)
		require.NoError(t, err)

		// JWT 토큰의 서명 부분을 완전히 다른 값으로 교체
		parts := strings.Split(token, ".")
		require.Equal(t, 3, len(parts), "JWT should have 3 parts")

		// 서명과 같은 길이의 잘못된 서명 생성
		signature := parts[2]
		tamperedSignature := strings.Repeat("X", len(signature))
		tamperedToken := parts[0] + "." + parts[1] + "." + tamperedSignature

		validatedUserID, err := util.ValidateToken(ctx, tamperedToken)

		assert.Error(t, err, "Tampered token should fail validation")
		assert.Equal(t, uint(0), validatedUserID)
		assert.Contains(t, err.Error(), "invalid token")
	})

	t.Run("실패: UserID claim이 없는 토큰", func(t *testing.T) {
		// UserID 없이 직접 토큰 생성
		claims := jwt.MapClaims{
			"exp": time.Now().Add(24 * time.Hour).Unix(),
		}
		jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		token, err := jwtToken.SignedString([]byte(secret))
		require.NoError(t, err)

		validatedUserID, err := util.ValidateToken(ctx, token)

		assert.Error(t, err)
		assert.Equal(t, uint(0), validatedUserID)
		assert.Contains(t, err.Error(), "invalid token")
	})

	t.Run("실패: UserID가 문자열이 아닌 토큰", func(t *testing.T) {
		// UserID를 숫자로 직접 설정
		claims := jwt.MapClaims{
			"user_id": 123,
			"exp":     time.Now().Add(24 * time.Hour).Unix(),
		}
		jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		token, err := jwtToken.SignedString([]byte(secret))
		require.NoError(t, err)

		validatedUserID, err := util.ValidateToken(ctx, token)

		assert.Error(t, err)
		assert.Equal(t, uint(0), validatedUserID)
		assert.Contains(t, err.Error(), "invalid token")
	})

	t.Run("실패: UserID가 숫자로 변환할 수 없는 문자열", func(t *testing.T) {
		claims := jwt.MapClaims{
			"user_id": "not-a-number",
			"exp":     time.Now().Add(24 * time.Hour).Unix(),
		}
		jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		token, err := jwtToken.SignedString([]byte(secret))
		require.NoError(t, err)

		validatedUserID, err := util.ValidateToken(ctx, token)

		assert.Error(t, err)
		assert.Equal(t, uint(0), validatedUserID)
		assert.Contains(t, err.Error(), "invalid token")
	})
}
