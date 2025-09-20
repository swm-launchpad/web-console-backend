package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/response"
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
		response.ValidationError(c, map[string]interface{}{
			"message": "Invalid request format: " + err.Error(),
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
		RespondWithError(c, err)
		return
	}

	response.Created(c, RegisterResponse{
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
		response.ValidationError(c, map[string]interface{}{
			"message": "Invalid request format: " + err.Error(),
		})
		return
	}

	input := application.LoginUserInput{
		Username: req.Username,
		Password: req.Password,
	}

	output, err := h.loginUseCase.Execute(c.Request.Context(), input)
	if err != nil {
		RespondWithError(c, err)
		return
	}

	response.OK(c, LoginResponse{
		UserID:   output.UserID,
		Token:    output.Token,
		Username: output.Username,
		Email:    output.Email,
		Name:     output.Name,
		Message:  "Login successful",
	})
}
