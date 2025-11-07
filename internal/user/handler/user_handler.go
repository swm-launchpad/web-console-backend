package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/common/response"
	"github.com/swm-launchpad/web-console-backend/internal/user/application"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"go.uber.org/zap"
)

type UserHandler struct {
	getUserUseCase        *application.GetUserUseCase
	updateUserUseCase     *application.UpdateUserUseCase
	changePasswordUseCase *application.ChangePasswordUseCase
	logger                logger.Logger
}

func NewUserHandler(
	getUserUseCase *application.GetUserUseCase,
	updateUserUseCase *application.UpdateUserUseCase,
	changePasswordUseCase *application.ChangePasswordUseCase,
	log logger.Logger,
) *UserHandler {
	return &UserHandler{
		getUserUseCase:        getUserUseCase,
		updateUserUseCase:     updateUserUseCase,
		changePasswordUseCase: changePasswordUseCase,
		logger:                log,
	}
}

// UserResponse represents the response for user profile
type UserResponse struct {
	UserID       uint   `json:"user_id"`
	Email        string `json:"email"`
	Nickname     string `json:"nickname"`
	Phone        string `json:"phone,omitempty"`
	Organization string `json:"organization,omitempty"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
}

// GetCurrentUser handles fetching the current authenticated user's profile
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	ctx := c.Request.Context()
	h.logger.Info(ctx, "get current user handler started",
		zap.String("handler", "GetCurrentUser"),
	)

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "GetCurrentUser"),
		)
		response.Error(c, auth.ErrUnauthorized, mapUserError)
		return
	}

	input := application.GetUserInput{
		UserID: userID.(uint),
	}

	output, err := h.getUserUseCase.Execute(ctx, input)
	if err != nil {
		h.logger.Error(ctx, "get user use case failed",
			zap.Error(err),
			zap.String("handler", "GetCurrentUser"),
		)
		response.Error(c, err, mapUserError)
		return
	}

	h.logger.Info(ctx, "get current user handler completed",
		zap.String("handler", "GetCurrentUser"),
		zap.Uint("user_id", output.UserID),
	)

	resp := UserResponse{
		UserID:       output.UserID,
		Email:        output.Email,
		Nickname:     output.Nickname,
		Phone:        output.Phone,
		Organization: output.Organization,
		Status:       output.Status,
		CreatedAt:    output.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	response.OK(c, resp)
}

// GetUserByID handles fetching a user profile by ID
func (h *UserHandler) GetUserByID(c *gin.Context) {
	ctx := c.Request.Context()
	h.logger.Info(ctx, "get user by id handler started",
		zap.String("handler", "GetUserByID"),
		zap.String("param_id", c.Param("id")),
	)

	userIDStr := c.Param("id")
	if userIDStr == "" {
		h.logger.Warn(ctx, "missing user id parameter",
			zap.String("handler", "GetUserByID"),
		)
		response.Error(c, usererrors.ErrMissingField, mapUserError)
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		h.logger.Warn(ctx, "invalid user id format",
			zap.Error(err),
			zap.String("handler", "GetUserByID"),
			zap.String("user_id_str", userIDStr),
		)
		response.Error(c, usererrors.ErrInvalidFormat, mapUserError)
		return
	}

	input := application.GetUserInput{
		UserID: uint(userID),
	}

	output, err := h.getUserUseCase.Execute(ctx, input)
	if err != nil {
		h.logger.Error(ctx, "get user use case failed",
			zap.Error(err),
			zap.String("handler", "GetUserByID"),
			zap.Uint64("requested_user_id", userID),
		)
		response.Error(c, err, mapUserError)
		return
	}

	h.logger.Info(ctx, "get user by id handler completed",
		zap.String("handler", "GetUserByID"),
		zap.Uint("user_id", output.UserID),
	)

	resp := UserResponse{
		UserID:       output.UserID,
		Email:        output.Email,
		Nickname:     output.Nickname,
		Phone:        output.Phone,
		Organization: output.Organization,
		Status:       output.Status,
		CreatedAt:    output.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	response.OK(c, resp)
}

// UpdateUserRequest represents the request for updating user profile
type UpdateUserRequest struct {
	Nickname     *string `json:"nickname"`
	Phone        *string `json:"phone"`
	Organization *string `json:"organization"`
}

// UpdateCurrentUser handles updating the current authenticated user's profile
func (h *UserHandler) UpdateCurrentUser(c *gin.Context) {
	ctx := c.Request.Context()
	h.logger.Info(ctx, "update current user handler started",
		zap.String("handler", "UpdateCurrentUser"),
	)

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "UpdateCurrentUser"),
		)
		response.Error(c, auth.ErrUnauthorized, mapUserError)
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "request validation failed",
			zap.Error(err),
			zap.String("handler", "UpdateCurrentUser"),
		)
		response.Error(c, usererrors.ErrInvalidFormat, mapUserError)
		return
	}

	input := application.UpdateUserInput{
		UserID:       userID.(uint),
		Nickname:     req.Nickname,
		Phone:        req.Phone,
		Organization: req.Organization,
	}

	output, err := h.updateUserUseCase.Execute(ctx, input)
	if err != nil {
		h.logger.Error(ctx, "update user use case failed",
			zap.Error(err),
			zap.String("handler", "UpdateCurrentUser"),
		)
		response.Error(c, err, mapUserError)
		return
	}

	h.logger.Info(ctx, "update current user handler completed",
		zap.String("handler", "UpdateCurrentUser"),
		zap.Uint("user_id", output.UserID),
	)

	resp := UserResponse{
		UserID:       output.UserID,
		Email:        output.Email,
		Nickname:     output.Nickname,
		Phone:        output.Phone,
		Organization: output.Organization,
		Status:       output.Status,
	}

	response.OK(c, resp)
}

// ChangePasswordRequest represents the request for changing password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

// ChangePasswordResponse represents the response for password change
type ChangePasswordResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ChangePassword handles changing the current authenticated user's password
func (h *UserHandler) ChangePassword(c *gin.Context) {
	ctx := c.Request.Context()
	h.logger.Info(ctx, "change password handler started",
		zap.String("handler", "ChangePassword"),
	)

	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		h.logger.Warn(ctx, "user not authenticated",
			zap.String("handler", "ChangePassword"),
		)
		response.Error(c, auth.ErrUnauthorized, mapUserError)
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "request validation failed",
			zap.Error(err),
			zap.String("handler", "ChangePassword"),
		)
		response.Error(c, usererrors.ErrInvalidFormat, mapUserError)
		return
	}

	input := application.ChangePasswordInput{
		UserID:          userID.(uint),
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	}

	_, err := h.changePasswordUseCase.Execute(ctx, input)
	if err != nil {
		h.logger.Error(ctx, "change password use case failed",
			zap.Error(err),
			zap.String("handler", "ChangePassword"),
		)
		response.Error(c, err, mapUserError)
		return
	}

	h.logger.Info(ctx, "change password handler completed",
		zap.String("handler", "ChangePassword"),
	)

	resp := ChangePasswordResponse{
		Success: true,
		Message: "Password changed successfully",
	}

	response.OK(c, resp)
}
