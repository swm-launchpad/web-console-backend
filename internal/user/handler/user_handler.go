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
	getUserUseCase *application.GetUserUseCase
}

func NewUserHandler(getUserUseCase *application.GetUserUseCase) *UserHandler {
	return &UserHandler{
		getUserUseCase: getUserUseCase,
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
		RespondWithError(c, auth.ErrUnauthorized)
		return
	}

	input := application.GetUserInput{
		UserID: userID.(uint),
	}

	output, err := h.getUserUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		RespondWithError(c, err)
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
		RespondWithError(c, usererrors.ErrMissingField)
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		RespondWithError(c, usererrors.ErrInvalidFormat)
		return
	}

	input := application.GetUserInput{
		UserID: uint(userID),
	}

	output, err := h.getUserUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		RespondWithError(c, err)
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
