package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth/jwt"
)

type AuthMiddleware struct {
	jwtUtil *jwt.JWTUtil
}

func NewAuthMiddleware(jwtUtil *jwt.JWTUtil) *AuthMiddleware {
	return &AuthMiddleware{
		jwtUtil: jwtUtil,
	}
}

// RequireAuth is a middleware that validates JWT tokens
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header is required",
			})
			c.Abort()
			return
		}

		// Check if the header starts with "Bearer "
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		// Extract the token
		token := authHeader[len(bearerPrefix):]
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Token is required",
			})
			c.Abort()
			return
		}

		// Validate the token
		userID, err := m.jwtUtil.ValidateToken(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired token",
			})
			c.Abort()
			return
		}

		// Set user ID in context for use in handlers
		c.Set(auth.ContextKeyUserID, userID)
		c.Set(auth.ContextKeyAuth, true)

		// Continue to the next handler
		c.Next()
	}
}

// OptionalAuth is a middleware that validates JWT tokens if present but doesn't require them
func (m *AuthMiddleware) OptionalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// No token provided, continue without authentication
			c.Set(auth.ContextKeyAuth, false)
			c.Next()
			return
		}

		// Check if the header starts with "Bearer "
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			// Invalid format, continue without authentication
			c.Set(auth.ContextKeyAuth, false)
			c.Next()
			return
		}

		// Extract the token
		token := authHeader[len(bearerPrefix):]
		if token == "" {
			// No token after Bearer, continue without authentication
			c.Set(auth.ContextKeyAuth, false)
			c.Next()
			return
		}

		// Validate the token
		userID, err := m.jwtUtil.ValidateToken(c.Request.Context(), token)
		if err != nil {
			// Invalid token, continue without authentication
			c.Set(auth.ContextKeyAuth, false)
			c.Next()
			return
		}

		// Set user ID in context for use in handlers
		c.Set(auth.ContextKeyUserID, userID)
		c.Set(auth.ContextKeyAuth, true)

		// Continue to the next handler
		c.Next()
	}
}