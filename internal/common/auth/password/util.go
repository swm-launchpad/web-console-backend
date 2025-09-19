package password

import (
	"golang.org/x/crypto/bcrypt"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
)

type PasswordUtil struct {
	cost int
}

func NewPasswordUtil() *PasswordUtil {
	return &PasswordUtil{
		cost: bcrypt.DefaultCost,
	}
}

func (u *PasswordUtil) HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", auth.ErrPasswordTooWeak
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), u.cost)
	if err != nil {
		return "", err
	}

	return string(hashedBytes), nil
}

func (u *PasswordUtil) VerifyPassword(hashedPassword, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return auth.ErrPasswordMismatch
		}
		return err
	}

	return nil
}

func (u *PasswordUtil) ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return auth.ErrPasswordTooWeak
	}

	// Additional validation rules can be added here
	// For example: checking for uppercase, lowercase, numbers, special characters

	return nil
}