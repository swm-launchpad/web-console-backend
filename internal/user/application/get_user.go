package application

import (
	"context"
	"time"

	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/service"
	"go.uber.org/zap"
)

type GetUserInput struct {
	UserID uint
}

type GetUserOutput struct {
	UserID       uint
	Email        string
	Nickname     string
	Organization *string
	Status       string
	CreatedAt    time.Time
}

type GetUserUseCase struct {
	userService service.UserService
	logger      logger.Logger
}

func NewGetUserUseCase(userService service.UserService, log logger.Logger) *GetUserUseCase {
	return &GetUserUseCase{
		userService: userService,
		logger:      log,
	}
}

func (uc *GetUserUseCase) Execute(ctx context.Context, input GetUserInput) (*GetUserOutput, error) {
	uc.logger.Info(ctx, "get user started",
		zap.Uint("user_id", input.UserID),
	)

	// Get user through UserService
	user, err := uc.userService.GetUserByID(ctx, input.UserID)
	if err != nil {
		uc.logger.Error(ctx, "failed to get user",
			zap.Error(err),
			zap.Uint("user_id", input.UserID),
		)
		return nil, err
	}

	// Map domain model to output
	output := &GetUserOutput{
		UserID:       user.UserID,
		Email:        user.Email,
		Nickname:     user.Nickname,
		Organization: user.Organization,
		Status:       string(user.Status),
		CreatedAt:    user.CreatedAt,
	}

	uc.logger.Info(ctx, "get user completed",
		zap.Uint("user_id", user.UserID),
		zap.String("email", user.Email),
	)

	return output, nil
}
