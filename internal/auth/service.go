package auth

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/auth/jwt"
	"github.com/swm-launchpad/web-console-backend/internal/auth/password"
	"github.com/swm-launchpad/web-console-backend/internal/users/domain/service"
)

type AuthService struct {
	jwtService      *jwt.Service
	passwordService *password.Service
}

func NewAuthService(jwtService *jwt.Service, passwordService *password.Service) service.AuthService {
	return &AuthService{
		jwtService:      jwtService,
		passwordService: passwordService,
	}
}

func (s *AuthService) GenerateToken(ctx context.Context, userID string) (string, error) {
	return s.jwtService.GenerateToken(ctx, userID)
}

func (s *AuthService) ValidateToken(ctx context.Context, token string) (string, error) {
	return s.jwtService.ValidateToken(ctx, token)
}

func (s *AuthService) HashPassword(password string) (string, error) {
	return s.passwordService.HashPassword(password)
}

func (s *AuthService) VerifyPassword(hashedPassword, password string) error {
	return s.passwordService.VerifyPassword(hashedPassword, password)
}