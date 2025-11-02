package jwt

import (
	"context"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
)

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

type JWTUtil struct {
	secret        string
	issuer        string
	tokenDuration time.Duration
}

func NewJWTUtil(secret string) *JWTUtil {
	return &JWTUtil{
		secret:        secret,
		issuer:        "web-console",
		tokenDuration: 24 * time.Hour, // 24 hours
	}
}

func (u *JWTUtil) GenerateToken(ctx context.Context, userID uint) (string, error) {
	now := time.Now()
	userIDStr := strconv.FormatUint(uint64(userID), 10)
	claims := Claims{
		UserID: userIDStr,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    u.issuer,
			Subject:   userIDStr,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(u.tokenDuration)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(u.secret))
	if err != nil {
		return "", auth.ErrTokenGenerationFailed
	}

	return tokenString, nil
}

func (u *JWTUtil) ValidateToken(ctx context.Context, tokenString string) (uint, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, auth.ErrInvalidToken
		}
		return []byte(u.secret), nil
	})

	if err != nil {
		if err == jwt.ErrTokenExpired {
			return 0, auth.ErrTokenExpired
		}
		return 0, auth.ErrInvalidToken
	}

	if !token.Valid {
		return 0, auth.ErrInvalidToken
	}

	// Convert string UserID back to uint
	userID, err := strconv.ParseUint(claims.UserID, 10, 32)
	if err != nil {
		return 0, auth.ErrInvalidToken
	}

	return uint(userID), nil
}

func (u *JWTUtil) SetTokenDuration(duration time.Duration) {
	u.tokenDuration = duration
}

// GenerateBuildLogToken generates a short-lived token (30 minutes) for build log access
// This token is scoped to a specific container and has minimal permissions
func (u *JWTUtil) GenerateBuildLogToken(ctx context.Context, userID uint, containerID uint) (string, error) {
	now := time.Now()
	buildLogTokenDuration := 30 * time.Minute // 30 minutes for build log access

	claims := BuildLogTokenClaims{
		UserID:      userID,
		ContainerID: containerID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    u.issuer,
			Subject:   strconv.FormatUint(uint64(userID), 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(buildLogTokenDuration)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(u.secret))
	if err != nil {
		return "", auth.ErrTokenGenerationFailed
	}

	return tokenString, nil
}

// ValidateBuildLogToken validates a build log token and returns the claims
func (u *JWTUtil) ValidateBuildLogToken(ctx context.Context, tokenString string) (*BuildLogTokenClaims, error) {
	claims := &BuildLogTokenClaims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, auth.ErrInvalidToken
		}
		return []byte(u.secret), nil
	})

	if err != nil {
		if err == jwt.ErrTokenExpired {
			return nil, auth.ErrTokenExpired
		}
		return nil, auth.ErrInvalidToken
	}

	if !token.Valid {
		return nil, auth.ErrInvalidToken
	}

	return claims, nil
}
