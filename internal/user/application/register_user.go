package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/db"
	"github.com/swm-launchpad/web-console-backend/internal/common/email"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
	"go.uber.org/zap"
)

type RegisterUserInput struct {
	Email    string
	Password string
	Nickname string
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
	logger       logger.Logger
}

func NewRegisterUserUseCase(
	authService service.AuthService,
	tokenService service.TokenService,
	emailService email.Service,
	txManager db.TxManager,
	log logger.Logger,
) *RegisterUserUseCase {
	return &RegisterUserUseCase{
		authService:  authService,
		tokenService: tokenService,
		emailService: emailService,
		txManager:    txManager,
		logger:       log,
	}
}

func (uc *RegisterUserUseCase) Execute(ctx context.Context, input RegisterUserInput) (*RegisterUserOutput, error) {
	uc.logger.Info(ctx, "user registration started",
		zap.String("email", input.Email),
		zap.String("nickname", input.Nickname),
	)

	var output *RegisterUserOutput
	var userEmail string
	var nickname string
	var verificationTokenStr string

	err := uc.txManager.RunInTx(ctx, func(txCtx context.Context) error {
		// Register user through AuthenticationService (user will be in 'pending' status)
		user, token, err := uc.authService.RegisterUser(txCtx, input.Email, input.Password, input.Nickname)
		if err != nil {
			uc.logger.Error(ctx, "failed to register user",
				zap.Error(err),
				zap.String("email", input.Email),
				zap.String("nickname", input.Nickname),
			)
			return err
		}

		// Create email verification token
		verificationToken, err := uc.tokenService.CreateEmailVerificationToken(txCtx, user.UserID)
		if err != nil {
			uc.logger.Error(ctx, "failed to create email verification token",
				zap.Error(err),
				zap.Uint("user_id", user.UserID),
			)
			return err
		}

		output = &RegisterUserOutput{
			UserID: user.UserID,
			Token:  token,
		}

		// Store for email sending (outside transaction)
		userEmail = user.Email
		nickname = user.Nickname
		verificationTokenStr = verificationToken.Token

		uc.logger.Info(ctx, "user registered successfully",
			zap.Uint("user_id", user.UserID),
			zap.String("nickname", nickname),
		)

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Send verification email (outside transaction to avoid holding DB connection)
	// If email sending fails, we log it but don't fail the registration
	if err := uc.emailService.SendVerificationEmail(ctx, userEmail, nickname, verificationTokenStr); err != nil {
		uc.logger.Error(ctx, "failed to send verification email",
			zap.Error(err),
			zap.Uint("user_id", output.UserID),
			zap.String("email", userEmail),
		)
		// Don't return error - user is registered, they can resend verification email later
	} else {
		uc.logger.Info(ctx, "verification email sent successfully",
			zap.Uint("user_id", output.UserID),
			zap.String("email", userEmail),
		)
	}

	return output, nil
}
