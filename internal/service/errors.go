package service

import (
	"errors"
	"fmt"

	"github.com/dcm-project/placement-manager/internal/policy"
)

// Error codes returned by service operations.
const (
	ErrCodeNotFound       = "NOT_FOUND"
	ErrCodeConflict       = "CONFLICT"
	ErrCodeValidation     = "VALIDATION"
	ErrCodeProviderError  = "PROVIDER_ERROR"
	ErrCodeInternal       = "INTERNAL_ERROR"
	ErrCodePolicyError    = "POLICY_ERROR"
	ErrCodePolicyRejected = "POLICY_REJECTED"
	ErrCodePolicyConflict = "POLICY_CONFLICT"
)

// ServiceError represents a business logic error with a code for HTTP mapping.
type ServiceError struct {
	Code    string
	Message string
}

func (e *ServiceError) Error() string {
	return e.Message
}

// Helper functions for creating ServiceErrors

func NewNotFoundError(message string) *ServiceError {
	return &ServiceError{
		Code:    ErrCodeNotFound,
		Message: message,
	}
}

func NewValidationError(message string) *ServiceError {
	return &ServiceError{
		Code:    ErrCodeValidation,
		Message: message,
	}
}

func NewInternalError(message string) *ServiceError {
	return &ServiceError{
		Code:    ErrCodeInternal,
		Message: message,
	}
}

func NewConflictError(message string) *ServiceError {
	return &ServiceError{
		Code:    ErrCodeConflict,
		Message: message,
	}
}

func NewPolicyError(message string) *ServiceError {
	return &ServiceError{
		Code:    ErrCodePolicyError,
		Message: message,
	}
}

func NewPolicyRejectedError(message string) *ServiceError {
	return &ServiceError{
		Code:    ErrCodePolicyRejected,
		Message: message,
	}
}

func NewPolicyConflictError(message string) *ServiceError {
	return &ServiceError{
		Code:    ErrCodePolicyConflict,
		Message: message,
	}
}

// HandlePolicyError maps policy client errors to service errors by checking
// the error type and extracting the HTTP status code.
func HandlePolicyError(err error) *ServiceError {
	// Try to unwrap and get the actual error
	var httpErr *policy.HTTPError
	if errors.As(err, &httpErr) {
		// We have an HTTPError with status code
		switch httpErr.StatusCode {
		case 400:
			return NewValidationError("invalid request format for policy evaluation")
		case 406:
			return NewPolicyRejectedError("request explicitly rejected by policy")
		case 409:
			return NewPolicyConflictError("policy conflict detected")
		case 500:
			return NewPolicyError("policy engine internal error")
		default:
			return NewPolicyError(fmt.Sprintf("policy evaluation failed with status %d: %s", httpErr.StatusCode, httpErr.Body))
		}
	}

	// Default to policy error for any other error
	return NewPolicyError("policy evaluation failed: " + err.Error())
}
