package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/users/application/usecase"
	"github.com/swm-launchpad/web-console-backend/internal/users/domain/repository"
)

type UserHandler struct {
	getUserUseCase *usecase.GetUserUseCase
}

func NewUserHandler(getUserUseCase *usecase.GetUserUseCase) *UserHandler {
	return &UserHandler{
		getUserUseCase: getUserUseCase,
	}
}

// UserResponse represents the response for user profile
type UserResponse struct {
	UserID          string `json:"user_id"`
	Username        string `json:"username"`
	Email           string `json:"email,omitempty"`
	Name            string `json:"name,omitempty"`
	ProfileImageURL string `json:"profile_image_url,omitempty"`
	Department      string `json:"department,omitempty"`
	Role            string `json:"role,omitempty"`
	Status          string `json:"status"`
	LastLoginAt     string `json:"last_login_at,omitempty"`
	CreatedAt       string `json:"created_at"`
}

// GetCurrentUser handles fetching the current authenticated user's profile
func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	input := usecase.GetUserInput{
		UserID: userID.(string),
	}

	output, err := h.getUserUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		if err == repository.ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "User not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch user profile",
		})
		return
	}

	response := UserResponse{
		UserID:          output.UserID,
		Username:        output.Username,
		Email:           output.Email,
		Name:            output.Name,
		ProfileImageURL: output.ProfileImageURL,
		Department:      output.Department,
		Role:            output.Role,
		Status:          output.Status,
		CreatedAt:       output.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if output.LastLoginAt != nil {
		response.LastLoginAt = output.LastLoginAt.Format("2006-01-02T15:04:05Z")
	}

	c.JSON(http.StatusOK, response)
}

// GetUserByID handles fetching a user profile by ID
func (h *UserHandler) GetUserByID(c *gin.Context) {
	userID := c.Param("id")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "User ID is required",
		})
		return
	}

	input := usecase.GetUserInput{
		UserID: userID,
	}

	output, err := h.getUserUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		if err == repository.ErrUserNotFound {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "User not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch user profile",
		})
		return
	}

	response := UserResponse{
		UserID:          output.UserID,
		Username:        output.Username,
		Email:           output.Email,
		Name:            output.Name,
		ProfileImageURL: output.ProfileImageURL,
		Department:      output.Department,
		Role:            output.Role,
		Status:          output.Status,
		CreatedAt:       output.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	if output.LastLoginAt != nil {
		response.LastLoginAt = output.LastLoginAt.Format("2006-01-02T15:04:05Z")
	}

	c.JSON(http.StatusOK, response)
}