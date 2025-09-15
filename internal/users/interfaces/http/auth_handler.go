package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/users/application/usecase"
)

type AuthHandler struct {
	registerUseCase *usecase.RegisterUserUseCase
	loginUseCase    *usecase.LoginUserUseCase
}

func NewAuthHandler(
	registerUseCase *usecase.RegisterUserUseCase,
	loginUseCase *usecase.LoginUserUseCase,
) *AuthHandler {
	return &AuthHandler{
		registerUseCase: registerUseCase,
		loginUseCase:    loginUseCase,
	}
}

// RegisterRequest represents the request body for user registration
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3"`
	Password string `json:"password" binding:"required,min=8"`
	Email    string `json:"email" binding:"required,email"`
	Name     string `json:"name"`
}

// RegisterResponse represents the response for user registration
type RegisterResponse struct {
	UserID  uint   `json:"user_id"`
	Token   string `json:"token"`
	Message string `json:"message"`
}

// Register handles user registration
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	input := usecase.RegisterUserInput{
		Username: req.Username,
		Password: req.Password,
		Email:    req.Email,
		Name:     req.Name,
	}

	output, err := h.registerUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		// Determine appropriate status code based on error
		statusCode := http.StatusInternalServerError
		if contains(err.Error(), "already exists") {
			statusCode = http.StatusConflict
		} else if contains(err.Error(), "required") || contains(err.Error(), "invalid") || contains(err.Error(), "weak") {
			statusCode = http.StatusBadRequest
		}

		c.JSON(statusCode, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, RegisterResponse{
		UserID:  output.UserID,
		Token:   output.Token,
		Message: "User registered successfully",
	})
}

// LoginRequest represents the request body for user login
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents the response for user login
type LoginResponse struct {
	UserID   uint   `json:"user_id"`
	Token    string `json:"token"`
	Username string `json:"username"`
	Email    string `json:"email,omitempty"`
	Name     string `json:"name,omitempty"`
	Message  string `json:"message"`
}

// Login handles user login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format: " + err.Error(),
		})
		return
	}

	input := usecase.LoginUserInput{
		Username: req.Username,
		Password: req.Password,
	}

	output, err := h.loginUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		// Determine appropriate status code based on error
		statusCode := http.StatusInternalServerError
		if contains(err.Error(), "invalid credentials") || contains(err.Error(), "not found") {
			statusCode = http.StatusUnauthorized
		} else if contains(err.Error(), "not active") {
			statusCode = http.StatusForbidden
		} else if contains(err.Error(), "required") {
			statusCode = http.StatusBadRequest
		}

		c.JSON(statusCode, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		UserID:   output.UserID,
		Token:    output.Token,
		Username: output.Username,
		Email:    output.Email,
		Name:     output.Name,
		Message:  "Login successful",
	})
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}