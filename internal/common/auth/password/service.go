package password

import (
	"golang.org/x/crypto/bcrypt"
	authErrors "github.com/swm-launchpad/web-console-backend/internal/common/auth/errors"
)

type Service struct {
	cost int
}

func NewService() *Service {
	return &Service{
		cost: bcrypt.DefaultCost,
	}
}

func (s *Service) HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", authErrors.ErrPasswordTooWeak
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), s.cost)
	if err != nil {
		return "", err
	}

	return string(hashedBytes), nil
}

func (s *Service) VerifyPassword(hashedPassword, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return authErrors.ErrPasswordMismatch
		}
		return err
	}

	return nil
}

func (s *Service) ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return authErrors.ErrPasswordTooWeak
	}

	// Additional validation rules can be added here
	// For example: checking for uppercase, lowercase, numbers, special characters

	return nil
}