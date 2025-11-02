package jwt

import (
	"github.com/golang-jwt/jwt/v5"
)

// BuildLogTokenClaims represents the claims for a build log access token
// This token is specifically for streaming build logs via WebSocket
// and has a shorter expiration time (30 minutes) than regular JWT tokens
type BuildLogTokenClaims struct {
	UserID      uint `json:"user_id"`
	ContainerID uint `json:"container_id"`
	jwt.RegisteredClaims
}
