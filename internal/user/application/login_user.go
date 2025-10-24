package application

import (
	"context"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
	"go.uber.org/zap"
)

type LoginUserInput struct {
	Username string
	Password string
}

type LoginUserOutput struct {
	UserID   uint
	Token    string
	Username string
	Email    string
	Name     string
}

type LoginUserUseCase struct {
	authService service.AuthService
	logger      logger.Logger
}

func NewLoginUserUseCase(authService service.AuthService, log logger.Logger) *LoginUserUseCase {
	return &LoginUserUseCase{
		authService: authService,
		logger:      log,
	}
}

func (uc *LoginUserUseCase) Execute(ctx context.Context, input LoginUserInput) (*LoginUserOutput, error) {
	uc.logger.Info(ctx, "user login started",
		zap.String("username", input.Username),
	)

	// Authenticate user through AuthenticationService
	user, token, err := uc.authService.AuthenticateUser(ctx, input.Username, input.Password)
	if err != nil {
		uc.logger.Error(ctx, "authentication failed",
			zap.Error(err),
			zap.String("username", input.Username),
		)
		return nil, err
	}

	// Prepare output
	output := &LoginUserOutput{
		UserID:   user.UserID,
		Token:    token,
		Username: user.Username,
		Email:    user.Email,
	}

	if user.Name != nil {
		output.Name = *user.Name
	}

	uc.logger.Info(ctx, "user login completed",
		zap.Uint("user_id", user.UserID),
		zap.String("username", user.Username),
	)

	return output, nil
}
