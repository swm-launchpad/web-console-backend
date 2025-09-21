package response

// response represents a standard success response structure
type response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Meta    *meta       `json:"meta,omitempty"`
}

// errorResponse represents a standard error response structure
type errorResponse struct {
	Success bool       `json:"success"`
	Error   *errorInfo `json:"error"`
}

// errorInfo contains error details
type errorInfo struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// meta contains metadata for paginated responses
type meta struct {
	Page       int `json:"page,omitempty"`
	Limit      int `json:"limit,omitempty"`
	TotalCount int `json:"total_count,omitempty"`
	TotalPages int `json:"total_pages,omitempty"`
}

// newResponse creates a new success response
func newResponse(data interface{}, opt ...func(*response)) *response {
	resp := &response{
		Success: true,
		Data:    data,
	}
	for _, opt := range opt {
		opt(resp)
	}
	return resp
}

func WithMeta(page int, limit int, totalCount int, totalPages int) func(*response) {
	return func(resp *response) {
		resp.Meta = &meta{
			Page:       page,
			Limit:      limit,
			TotalCount: totalCount,
			TotalPages: totalPages,
		}
	}
}

// newErrorResponse creates a new error response
func newErrorResponse(code string, message string, opt ...func(*errorInfo)) *errorResponse {
	info := &errorInfo{
		Code:    code,
		Message: message,
	}
	for _, opt := range opt {
		opt(info)
	}
	return &errorResponse{
		Success: false,
		Error:   info,
	}
}

func WithDetails(details map[string]interface{}) func(*errorInfo) {
	return func(info *errorInfo) {
		info.Details = details
	}
}
