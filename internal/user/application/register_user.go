package application

import (
	"context"
	"log"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/email"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
)

type RegisterUserInput struct {
	Username string
	Password string
	Email    string
	Name     string
}

type RegisterUserOutput struct {
	UserID uint
	Token  string
}

type RegisterUserUseCase struct {
	authService  service.AuthService
	tokenService service.TokenService
	emailService email.Service
	txManager    db.TxManager
}

func NewRegisterUserUseCase(
	authService service.AuthService,
	tokenService service.TokenService,
	emailService email.Service,
	txManager db.TxManager,
) *RegisterUserUseCase {
	return &RegisterUserUseCase{
		authService:  authService,
		tokenService: tokenService,
		emailService: emailService,
		txManager:    txManager,
	}
}

func (uc *RegisterUserUseCase) Execute(ctx context.Context, input RegisterUserInput) (*RegisterUserOutput, error) {
	var output *RegisterUserOutput
	var userEmail string
	var username string
	var verificationTokenStr string

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		var name *string
		if input.Name != "" {
			name = &input.Name
		}

		// Register user through AuthenticationService (user will be in 'pending' status)
		user, token, err := uc.authService.RegisterUser(txCtx, input.Username, input.Password, input.Email, name)
		if err != nil {
			return err
		}

		// Create email verification token
		verificationToken, err := uc.tokenService.CreateEmailVerificationToken(txCtx, user.UserID)
		if err != nil {
			return err
		}

		output = &RegisterUserOutput{
			UserID: user.UserID,
			Token:  token,
		}

		// Store for email sending (outside transaction)
		userEmail = user.Email
		username = user.Username
		verificationTokenStr = verificationToken.Token

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Send verification email (outside transaction to avoid holding DB connection)
	// If email sending fails, we log it but don't fail the registration
	if err := uc.emailService.SendVerificationEmail(userEmail, username, verificationTokenStr); err != nil {
		log.Printf("[EMAIL_ERROR] Failed to send verification email | user_id=%d | email=%s | error=%v",
			output.UserID, userEmail, err)
		// Don't return error - user is registered, they can resend verification email later
		// TODO: Consider adding metric/alert for email sending failures
	} else {
		log.Printf("[EMAIL_SUCCESS] Verification email sent successfully | user_id=%d | email=%s",
			output.UserID, userEmail)
	}

	return output, nil
}
