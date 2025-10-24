package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/swm-launchpad/web-console-backend/internal/common/logger"
	"github.com/swm-launchpad/web-console-backend/internal/common/response"
	"github.com/swm-launchpad/web-console-backend/internal/user/application"
	usererrors "github.com/swm-launchpad/web-console-backend/internal/user/domain/errors"
	"go.uber.org/zap"
)

type AuthHandler struct {
	registerUseCase *application.RegisterUserUseCase
	loginUseCase    *application.LoginUserUseCase
	logger          logger.Logger
}

func NewAuthHandler(
	registerUseCase *application.RegisterUserUseCase,
	loginUseCase *application.LoginUserUseCase,
	log logger.Logger,
) *AuthHandler {
	return &AuthHandler{
		registerUseCase: registerUseCase,
		loginUseCase:    loginUseCase,
		logger:          log,
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
	ctx := c.Request.Context()
	h.logger.Info(ctx, "register handler started",
		zap.String("handler", "Register"),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
	)

	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "request validation failed",
			zap.Error(err),
			zap.String("handler", "Register"),
		)
		response.Error(c, usererrors.ErrValidationFailed, mapUserError, response.WithDetails(map[string]interface{}{
			"message": "Invalid request format: " + err.Error(),
		}))
		return
	}

	input := application.RegisterUserInput{
		Username: req.Username,
		Password: req.Password,
		Email:    req.Email,
		Name:     req.Name,
	}

	output, err := h.registerUseCase.Execute(ctx, input)
	if err != nil {
		h.logger.Error(ctx, "register use case failed",
			zap.Error(err),
			zap.String("handler", "Register"),
			zap.String("username", req.Username),
		)
		response.Error(c, err, mapUserError)
		return
	}

	h.logger.Info(ctx, "register handler completed successfully",
		zap.String("handler", "Register"),
		zap.Uint("user_id", output.UserID),
	)

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
	ctx := c.Request.Context()
	h.logger.Info(ctx, "login handler started",
		zap.String("handler", "Login"),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
	)

	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn(ctx, "request validation failed",
			zap.Error(err),
			zap.String("handler", "Login"),
		)
		response.Error(c, usererrors.ErrValidationFailed, mapUserError, response.WithDetails(map[string]interface{}{
			"message": "Invalid request format: " + err.Error(),
		}))
		return
	}

	input := application.LoginUserInput{
		Username: req.Username,
		Password: req.Password,
	}

	output, err := h.loginUseCase.Execute(ctx, input)
	if err != nil {
		h.logger.Error(ctx, "login use case failed",
			zap.Error(err),
			zap.String("handler", "Login"),
			zap.String("username", req.Username),
		)
		response.Error(c, err, mapUserError)
		return
	}

	h.logger.Info(ctx, "login handler completed successfully",
		zap.String("handler", "Login"),
		zap.Uint("user_id", output.UserID),
		zap.String("username", output.Username),
	)

	response.OK(c, LoginResponse{
		UserID:   output.UserID,
		Token:    output.Token,
		Username: output.Username,
		Email:    output.Email,
		Name:     output.Name,
		Message:  "Login successful",
	})
}
