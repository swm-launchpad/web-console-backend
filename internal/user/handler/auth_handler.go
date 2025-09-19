package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/user/application"
)

type AuthHandler struct {
	registerUseCase *application.RegisterUserUseCase
	loginUseCase    *application.LoginUserUseCase
}

func NewAuthHandler(
	registerUseCase *application.RegisterUserUseCase,
	loginUseCase *application.LoginUserUseCase,
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

	input := application.RegisterUserInput{
		Username: req.Username,
		Password: req.Password,
		Email:    req.Email,
		Name:     req.Name,
	}

	output, err := h.registerUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		// Determine appropriate status code based on error
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "already exists") {
			statusCode = http.StatusConflict
		} else if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "invalid") || strings.Contains(err.Error(), "weak") {
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

	input := application.LoginUserInput{
		Username: req.Username,
		Password: req.Password,
	}

	output, err := h.loginUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		// Determine appropriate status code based on error
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "invalid credentials") || strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusUnauthorized
		} else if strings.Contains(err.Error(), "not active") {
			statusCode = http.StatusForbidden
		} else if strings.Contains(err.Error(), "required") {
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
