package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// OK sends a 200 OK response with data
func OK(c *gin.Context, data interface{}, opt ...func(*response)) {
	c.JSON(http.StatusOK, newResponse(data, opt...))
}

// Created sends a 201 Created response with data
func Created(c *gin.Context, data interface{}, opt ...func(*response)) {
	c.JSON(http.StatusCreated, newResponse(data, opt...))
}

// Accepted sends a 202 Accepted response with data
func Accepted(c *gin.Context, data interface{}, opt ...func(*response)) {
	c.JSON(http.StatusAccepted, newResponse(data, opt...))
}

// NoContent sends a 204 No Content response
func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// Error handles errors with domain-specific mapping, falling back to common errors.
// domainMapper(optional) is a function that maps errors to ErrorMapping.
// opt(optional) is a function that adds additional information to the error response.
func Error(c *gin.Context, err error, domainMapper ErrorMapper, opt ...func(*errorInfo)) {
	if err == nil {
		return
	}

	// Try domain-specific mapping first
	if domainMapper != nil {
		if mapping, found := domainMapper(err); found {
			sendError(c, mapping.StatusCode, mapping.Code, mapping.Message, opt...)
			return
		}
	}

	// Fall back to handleCommonError
	handleCommonError(c, err, opt...)
}

// handleCommonError handles errors with common error mapping
func handleCommonError(c *gin.Context, err error, opt ...func(*errorInfo)) {
	if err == nil {
		return
	}

	mapping, found := mapCommonError(err)

	// Fallback to generic internal error
	if !found {
		mapping = ErrorMapping{
			StatusCode: http.StatusInternalServerError,
			Code:       "INTERNAL_ERROR",
			Message:    err.Error(),
		}
	}

	sendError(c, mapping.StatusCode, mapping.Code, mapping.Message, opt...)
}

// sendError sends a standard error response
func sendError(c *gin.Context, statusCode int, code string, message string, opt ...func(*errorInfo)) {
	c.JSON(statusCode, newErrorResponse(code, message, opt...))
	c.Abort()
}
