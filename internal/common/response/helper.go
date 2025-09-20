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

// HandleError translates a domain error and sends appropriate response
func HandleError(c *gin.Context, err error) {
	status, code, message := TranslateError(err, nil)
	Error(c, status, code, message)
}

// HandleErrorWithMapper translates a domain error using a custom error mapper
func HandleErrorWithMapper(c *gin.Context, err error, errorMapper func(error) (string, bool)) {
	status, code, message := TranslateError(err, errorMapper)
	Error(c, status, code, message)
}

// HandleErrorWithMessage translates a domain error but uses a custom message
func HandleErrorWithMessage(c *gin.Context, err error, customMessage string) {
	status, code, _ := TranslateError(err, nil)
	Error(c, status, code, customMessage)
}
