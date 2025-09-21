package response

// Response represents a standard success response structure
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// ErrorResponse represents a standard error response structure
type ErrorResponse struct {
	Success bool       `json:"success"`
	Error   *ErrorInfo `json:"error"`
}

// ErrorInfo contains error details
type ErrorInfo struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Meta contains metadata for paginated responses
type Meta struct {
	Page       int `json:"page,omitempty"`
	Limit      int `json:"limit,omitempty"`
	TotalCount int `json:"total_count,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
}

// NewResponse creates a new success response
func NewResponse(data interface{}) *Response {
	return &Response{
		Success: true,
		Data:    data,
	}
}

// NewResponseWithMeta creates a new success response with metadata
func NewResponseWithMeta(data interface{}, meta *Meta) *Response {
	return &Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	}
}

// NewErrorResponse creates a new error response
func NewErrorResponse(code string, message string) *ErrorResponse {
	return &ErrorResponse{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	}
}

// NewErrorResponseWithDetails creates a new error response with additional details
func NewErrorResponseWithDetails(code string, message string, details map[string]interface{}) *ErrorResponse {
	return &ErrorResponse{
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
}

// ErrorMapping contains complete error mapping information
type ErrorMapping struct {
	StatusCode int
	Code       string
	Message    string
}

// ErrorMapper is a function type for domain-specific error mapping
type ErrorMapper func(error) (ErrorMapping, bool)
