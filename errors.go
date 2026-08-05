package rnld

import "fmt"

// APIError is returned for any non-2xx response from the RNLD API that
// isn't mapped to a more specific error type below.
type APIError struct {
	StatusCode int
	Message    string
	Body       any
}

func (e *APIError) Error() string {
	return fmt.Sprintf("rnld: %s (status %d)", e.Message, e.StatusCode)
}

// NotFoundError is returned when the API responds with 404.
type NotFoundError struct{ *APIError }

func (e *NotFoundError) Unwrap() error { return e.APIError }

func newNotFoundError(message string) *NotFoundError {
	if message == "" {
		message = "Jogador não encontrado"
	}
	return &NotFoundError{&APIError{StatusCode: 404, Message: message}}
}

// UnauthorizedError is returned when the API responds with 401 or 403.
type UnauthorizedError struct{ *APIError }

func (e *UnauthorizedError) Unwrap() error { return e.APIError }

func newUnauthorizedError(message string) *UnauthorizedError {
	if message == "" {
		message = "API Key inválida ou guild_id incorreto"
	}
	return &UnauthorizedError{&APIError{StatusCode: 403, Message: message}}
}
