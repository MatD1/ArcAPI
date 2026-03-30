package handlers

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error string `json:"error" example:"Description of the error"`
}

// MessageResponse represents a standard message response
type MessageResponse struct {
	Message string `json:"message" example:"Success message"`
}

// PaginationDetails represents standard pagination metadata
type PaginationDetails struct {
	Page  int   `json:"page" example:"1"`
	Limit int   `json:"limit" example:"20"`
	Total int64 `json:"total" example:"100"`
}

// PaginatedResponse is a generic wrapper for paginated data in Swagger
// Note: In real responses, "data" will be a specific slice of models
type PaginatedResponse struct {
	Data       interface{}       `json:"data"`
	Pagination PaginationDetails `json:"pagination"`
}
