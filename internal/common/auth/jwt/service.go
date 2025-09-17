package jwt

import (
	"context"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	authErrors "github.com/swm-launchpad/web-console-backend/internal/common/auth/errors"
)

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

type Service struct {
	secret        string
	issuer        string
	tokenDuration time.Duration
}

func NewService(secret string) *Service {
	return &Service{
		secret:        secret,
		issuer:        "web-console",
		tokenDuration: 24 * time.Hour, // 24 hours
	}
}

func (s *Service) GenerateToken(ctx context.Context, userID uint) (string, error) {
	now := time.Now()
	userIDStr := strconv.FormatUint(uint64(userID), 10)
	claims := Claims{
		UserID: userIDStr,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userIDStr,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.tokenDuration)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.secret))
	if err != nil {
		return "", authErrors.ErrTokenGenerationFailed
	}

	return tokenString, nil
}

func (s *Service) ValidateToken(ctx context.Context, tokenString string) (uint, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, authErrors.ErrInvalidToken
		}
		return []byte(s.secret), nil
	})

	if err != nil {
		if err == jwt.ErrTokenExpired {
			return 0, authErrors.ErrTokenExpired
		}
		return 0, authErrors.ErrInvalidToken
	}

	if !token.Valid {
		return 0, authErrors.ErrInvalidToken
	}

	// Convert string UserID back to uint
	userID, err := strconv.ParseUint(claims.UserID, 10, 32)
	if err != nil {
		return 0, authErrors.ErrInvalidToken
	}

	return uint(userID), nil
}

func (s *Service) SetTokenDuration(duration time.Duration) {
	s.tokenDuration = duration
}