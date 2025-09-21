package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// OK sends a 200 OK response with data
func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, NewResponse(data))
}

// Created sends a 201 Created response with data
func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, NewResponse(data))
}

// Error sends a standard error response
func Error(c *gin.Context, statusCode int, code string, message string) {
	c.JSON(statusCode, NewErrorResponse(code, message))
	c.Abort()
}

// ErrorWithDetails sends an error response with additional details
func ErrorWithDetails(c *gin.Context, statusCode int, code string, message string, details map[string]interface{}) {
	c.JSON(statusCode, NewErrorResponseWithDetails(code, message, details))
	c.Abort()
}

// HandleCommonError handles errors with common error mapping
func HandleCommonError(c *gin.Context, err error) {
	if err == nil {
		return
	}

	mapping, found := MapCommonError(err)

	// Fallback to generic internal error
	if !found {
		mapping = ErrorMapping{
			StatusCode: http.StatusInternalServerError,
			Code:       "INTERNAL_ERROR",
			Message:    err.Error(),
		}
	}

	Error(c, mapping.StatusCode, mapping.Code, mapping.Message)
}

// HandleDomainError handles errors with domain-specific mapping, falling back to common errors
func HandleDomainError(c *gin.Context, err error, domainMapper ErrorMapper) {
	if err == nil {
		return
	}

	// Try domain-specific mapping first
	if domainMapper != nil {
		if mapping, found := domainMapper(err); found {
			Error(c, mapping.StatusCode, mapping.Code, mapping.Message)
			return
		}
	}

	// Fall back to HandleCommonError
	HandleCommonError(c, err)
}
