package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
	"github.com/swm-launchpad/web-console-backend/internal/common/response"
	"github.com/swm-launchpad/web-console-backend/internal/user/application"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
)

type UserHandler struct {
	getUserUseCase        *application.GetUserUseCase
	updateUserUseCase     *application.UpdateUserUseCase
	changePasswordUseCase *application.ChangePasswordUseCase
}

func NewUserHandler(
	getUserUseCase *application.GetUserUseCase,
	updateUserUseCase *application.UpdateUserUseCase,
	changePasswordUseCase *application.ChangePasswordUseCase,
) *UserHandler {
	return &UserHandler{
		getUserUseCase:        getUserUseCase,
		updateUserUseCase:     updateUserUseCase,
		changePasswordUseCase: changePasswordUseCase,
	}
}

// UserResponse represents the response for user profile
type UserResponse struct {
	UserID       uint   `json:"user_id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	Name         string `json:"name,omitempty"`
	Phone        string `json:"phone,omitempty"`
	Organization string `json:"organization,omitempty"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
}

// GetCurrentUser handles fetching the current authenticated user's profile
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapUserError)
		return
	}

	input := application.GetUserInput{
		UserID: userID.(uint),
	}

	output, err := h.getUserUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapUserError)
		return
	}

	resp := UserResponse{
		UserID:       output.UserID,
		Username:     output.Username,
		Email:        output.Email,
		Name:         output.Name,
		Phone:        output.Phone,
		Organization: output.Organization,
		Status:       output.Status,
		CreatedAt:    output.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	response.OK(c, resp)
}

// GetUserByID handles fetching a user profile by ID
func (h *UserHandler) GetUserByID(c *gin.Context) {
	userIDStr := c.Param("id")
	if userIDStr == "" {
		response.Error(c, usererrors.ErrMissingField, mapUserError)
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		response.Error(c, usererrors.ErrInvalidFormat, mapUserError)
		return
	}

	input := application.GetUserInput{
		UserID: uint(userID),
	}

	output, err := h.getUserUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapUserError)
		return
	}

	resp := UserResponse{
		UserID:       output.UserID,
		Username:     output.Username,
		Email:        output.Email,
		Name:         output.Name,
		Phone:        output.Phone,
		Organization: output.Organization,
		Status:       output.Status,
		CreatedAt:    output.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	response.OK(c, resp)
}

// UpdateUserRequest represents the request for updating user profile
type UpdateUserRequest struct {
	Name         *string `json:"name"`
	Phone        *string `json:"phone"`
	Organization *string `json:"organization"`
}

// UpdateCurrentUser handles updating the current authenticated user's profile
func (h *UserHandler) UpdateCurrentUser(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapUserError)
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, usererrors.ErrInvalidFormat, mapUserError)
		return
	}

	input := application.UpdateUserInput{
		UserID:       userID.(uint),
		Name:         req.Name,
		Phone:        req.Phone,
		Organization: req.Organization,
	}

	output, err := h.updateUserUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapUserError)
		return
	}

	resp := UserResponse{
		UserID:       output.UserID,
		Username:     output.Username,
		Email:        output.Email,
		Name:         output.Name,
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
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get(auth.ContextKeyUserID)
	if !exists {
		response.Error(c, auth.ErrUnauthorized, mapUserError)
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, usererrors.ErrInvalidFormat, mapUserError)
		return
	}

	input := application.ChangePasswordInput{
		UserID:          userID.(uint),
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	}

	_, err := h.changePasswordUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		response.Error(c, err, mapUserError)
		return
	}

	resp := ChangePasswordResponse{
		Success: true,
		Message: "Password changed successfully",
	}

	response.OK(c, resp)
}
