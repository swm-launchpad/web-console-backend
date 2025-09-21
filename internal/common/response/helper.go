package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Success sends a standard success response
func Success(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, NewResponse(data))
}

// SuccessWithMeta sends a success response with metadata
func SuccessWithMeta(c *gin.Context, statusCode int, data interface{}, meta *Meta) {
	c.JSON(statusCode, NewResponseWithMeta(data, meta))
}

// OK sends a 200 OK response with data
func OK(c *gin.Context, data interface{}) {
	Success(c, http.StatusOK, data)
}

// Created sends a 201 Created response with data
func Created(c *gin.Context, data interface{}) {
	Success(c, http.StatusCreated, data)
}

// NoContent sends a 204 No Content response
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
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

// BadRequest sends a 400 Bad Request error response
func BadRequest(c *gin.Context, code string, message string) {
	Error(c, http.StatusBadRequest, code, message)
}

// Unauthorized sends a 401 Unauthorized error response
func Unauthorized(c *gin.Context, code string, message string) {
	Error(c, http.StatusUnauthorized, code, message)
}

// Forbidden sends a 403 Forbidden error response
func Forbidden(c *gin.Context, code string, message string) {
	Error(c, http.StatusForbidden, code, message)
}

// NotFound sends a 404 Not Found error response
func NotFound(c *gin.Context, code string, message string) {
	Error(c, http.StatusNotFound, code, message)
}

// Conflict sends a 409 Conflict error response
func Conflict(c *gin.Context, code string, message string) {
	Error(c, http.StatusConflict, code, message)
}

// InternalServerError sends a 500 Internal Server Error response
func InternalServerError(c *gin.Context, code string, message string) {
	Error(c, http.StatusInternalServerError, code, message)
}

// ValidationError sends a validation error response with field details
func ValidationError(c *gin.Context, fields map[string]interface{}) {
	ErrorWithDetails(
		c,
		http.StatusBadRequest,
		"VALIDATION_FAILED",
		"Validation failed",
		fields,
	)
}

// HandleDomainError handles errors with domain-specific mapping, falling back to common errors
func HandleDomainError(c *gin.Context, err error, domainMapper ErrorMapper) {
	if err == nil {
		return
	}

	var mapping ErrorMapping
	var found bool

	// Try domain-specific mapping first
	if domainMapper != nil {
		mapping, found = domainMapper(err)
	}

	// Fall back to common error mapping
	if !found {
		mapping, found = MapCommonError(err)
	}

	// Final fallback to generic internal error
	if !found {
		mapping = ErrorMapping{
			StatusCode: http.StatusInternalServerError,
			Code:       "INTERNAL_ERROR",
			Message:    err.Error(),
		}
	}

	Error(c, mapping.StatusCode, mapping.Code, mapping.Message)
}
