package service

import "context"

type AuthService interface {
	GenerateToken(ctx context.Context, userID uint) (string, error)
	ValidateToken(ctx context.Context, token string) (uint, error) // returns userID
	HashPassword(password string) (string, error)
	VerifyPassword(hashedPassword, password string) error
}