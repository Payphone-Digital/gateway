package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// DomainError represents a domain-specific error with a code and message
type DomainError struct {
	Code    string
	Message string
	Err     error // underlying error for wrapping
}

// Error implements the error interface
func (e *DomainError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the underlying error for errors.Is and errors.As
func (e *DomainError) Unwrap() error {
	return e.Err
}

// NewDomainError creates a new domain error
func NewDomainError(code, message string) *DomainError {
	return &DomainError{
		Code:    code,
		Message: message,
	}
}

// WrapError wraps an existing error with domain error context
func WrapError(domainErr *DomainError, err error) *DomainError {
	return &DomainError{
		Code:    domainErr.Code,
		Message: domainErr.Message,
		Err:     err,
	}
}

// Predefined domain errors
var (
	// User errors
	ErrUserNotFound       = NewDomainError("USER_NOT_FOUND", "user not found")
	ErrEmailExists        = NewDomainError("EMAIL_EXISTS", "email already exists")
	ErrInvalidCredentials = NewDomainError("INVALID_CREDENTIALS", "invalid credentials")
	ErrSelfDeletion       = NewDomainError("SELF_DELETION", "users cannot delete themselves")

	// Authentication errors
	ErrUnauthorized        = NewDomainError("UNAUTHORIZED", "unauthorized")
	ErrInvalidToken        = NewDomainError("INVALID_TOKEN", "invalid or expired token")
	ErrTokenExpired        = NewDomainError("TOKEN_EXPIRED", "token has expired")
	ErrInvalidRefreshToken = NewDomainError("INVALID_REFRESH_TOKEN", "invalid refresh token")

	// Validation errors
	ErrInvalidInput      = NewDomainError("INVALID_INPUT", "invalid input")
	ErrPasswordMismatch  = NewDomainError("PASSWORD_MISMATCH", "new password and confirmation do not match")
	ErrIncorrectPassword = NewDomainError("INCORRECT_PASSWORD", "current password is incorrect")

	// System errors
	ErrInternal           = NewDomainError("INTERNAL_ERROR", "internal server error")
	ErrServiceUnavailable = NewDomainError("SERVICE_UNAVAILABLE", "service unavailable")
)

// IsDomainError checks if an error is a domain error
func IsDomainError(err error) bool {
	var domainErr *DomainError
	return errors.As(err, &domainErr)
}

// GetDomainError extracts the domain error from an error
func GetDomainError(err error) *DomainError {
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr
	}
	return nil
}

// ToHTTPStatus maps domain errors to HTTP status codes
// This should only be used in the handler/presentation layer
func ToHTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}

	// Check if it's a domain error
	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErrorToHTTPStatus(domainErr)
	}

	// Default to internal server error for unknown errors
	return http.StatusInternalServerError
}

// domainErrorToHTTPStatus maps specific domain errors to HTTP status codes
func domainErrorToHTTPStatus(err *DomainError) int {
	switch err.Code {
	// 400 Bad Request
	case "INVALID_INPUT", "PASSWORD_MISMATCH":
		return http.StatusBadRequest

	// 401 Unauthorized
	case "UNAUTHORIZED", "INVALID_CREDENTIALS", "INVALID_TOKEN",
		"TOKEN_EXPIRED", "INVALID_REFRESH_TOKEN", "INCORRECT_PASSWORD":
		return http.StatusUnauthorized

	// 403 Forbidden
	case "SELF_DELETION":
		return http.StatusForbidden

	// 404 Not Found
	case "USER_NOT_FOUND":
		return http.StatusNotFound

	// 409 Conflict
	case "EMAIL_EXISTS":
		return http.StatusConflict

	// 503 Service Unavailable
	case "SERVICE_UNAVAILABLE":
		return http.StatusServiceUnavailable

	// 500 Internal Server Error (default)
	default:
		return http.StatusInternalServerError
	}
}

// GetErrorMessage safely extracts error message
func GetErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	var domainErr *DomainError
	if errors.As(err, &domainErr) {
		return domainErr.Message
	}

	return err.Error()
}
