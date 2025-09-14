package jwt

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	authErrors "github.com/swm-launchpad/web-console-backend/internal/auth/errors"
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

func (s *Service) GenerateToken(ctx context.Context, userID string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID,
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

func (s *Service) ValidateToken(ctx context.Context, tokenString string) (string, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, authErrors.ErrInvalidToken
		}
		return []byte(s.secret), nil
	})

	if err != nil {
		if err == jwt.ErrTokenExpired {
			return "", authErrors.ErrTokenExpired
		}
		return "", authErrors.ErrInvalidToken
	}

	if !token.Valid {
		return "", authErrors.ErrInvalidToken
	}

	return claims.UserID, nil
}

func (s *Service) SetTokenDuration(duration time.Duration) {
	s.tokenDuration = duration
}