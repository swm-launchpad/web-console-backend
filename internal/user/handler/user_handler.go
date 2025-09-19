package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/auth"
	"github.com/swm-launchpad/web-console-backend/internal/user/application"
	"github.com/swm-launchpad/web-console-backend/internal/user/domain/repository"
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
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	input := application.GetUserInput{
		UserID: userID.(uint),
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
		UserID:       output.UserID,
		Username:     output.Username,
		Email:        output.Email,
		Name:         output.Name,
		Phone:        output.Phone,
		Organization: output.Organization,
		Status:       output.Status,
		CreatedAt:    output.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	c.JSON(http.StatusOK, response)
}

// GetUserByID handles fetching a user profile by ID
func (h *UserHandler) GetUserByID(c *gin.Context) {
	userIDStr := c.Param("id")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "User ID is required",
		})
		return
	}

	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid user ID format",
		})
		return
	}

	input := application.GetUserInput{
		UserID: uint(userID),
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
		UserID:       output.UserID,
		Username:     output.Username,
		Email:        output.Email,
		Name:         output.Name,
		Phone:        output.Phone,
		Organization: output.Organization,
		Status:       output.Status,
		CreatedAt:    output.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}

	c.JSON(http.StatusOK, response)
}
