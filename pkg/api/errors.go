package api

import (
	"fmt"
	"net/http"
)

// ErrorCode represents a specific error category.
type ErrorCode int

const (
	// General errors (1000-1099).
	ErrUnknown       ErrorCode = 1000
	ErrInternal      ErrorCode = 1001
	ErrInvalidInput  ErrorCode = 1002
	ErrNotFound      ErrorCode = 1003
	ErrConflict      ErrorCode = 1004
	ErrRateLimit     ErrorCode = 1005

	// Authentication errors (1100-1199).
	ErrUnauthorized  ErrorCode = 1100
	ErrForbidden     ErrorCode = 1101
	ErrTokenExpired  ErrorCode = 1102
	ErrInvalidToken  ErrorCode = 1103

	// Task errors (1200-1299).
	ErrTaskNotFound       ErrorCode = 1200
	ErrTaskAlreadyExists  ErrorCode = 1201
	ErrTaskTimeout        ErrorCode = 1202
	ErrTaskCancelled      ErrorCode = 1203
	ErrCapsuleTooLarge    ErrorCode = 1204
	ErrInvalidEntrypoint  ErrorCode = 1205
	ErrUnsupportedRuntime ErrorCode = 1206

	// Worker errors (1300-1399).
	ErrWorkerNotFound    ErrorCode = 1300
	ErrWorkerOffline     ErrorCode = 1301
	ErrNoAvailableWorker ErrorCode = 1302
	ErrWorkerBanned      ErrorCode = 1303

	// P2P errors (1400-1499).
	ErrSignalingFailed ErrorCode = 1400
	ErrICEFailed       ErrorCode = 1401
	ErrDataChannelFail ErrorCode = 1402
	ErrTransferFailed  ErrorCode = 1403

	// Integrity errors (1500-1599).
	ErrHashMismatch     ErrorCode = 1500
	ErrVerificationFail ErrorCode = 1501
	ErrSpotCheckFail    ErrorCode = 1502
)

// APIError is a structured error response returned by the UniCompute API.
type APIError struct {
	// Code is the machine-readable error code.
	Code ErrorCode `json:"code"`

	// HTTPStatus is the corresponding HTTP status code.
	HTTPStatus int `json:"http_status"`

	// Message is the human-readable error message.
	Message string `json:"message"`

	// Details provides additional context (stack traces, field-level errors).
	Details string `json:"details,omitempty"`

	// RequestID correlates this error with server logs.
	RequestID string `json:"request_id,omitempty"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("[%d] %s: %s", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// NewError creates a new APIError.
func NewError(code ErrorCode, message string) *APIError {
	return &APIError{
		Code:       code,
		HTTPStatus: codeToHTTPStatus(code),
		Message:    message,
	}
}

// NewErrorWithDetails creates a new APIError with additional details.
func NewErrorWithDetails(code ErrorCode, message, details string) *APIError {
	err := NewError(code, message)
	err.Details = details
	return err
}

// codeToHTTPStatus maps error codes to HTTP status codes.
func codeToHTTPStatus(code ErrorCode) int {
	switch {
	case code >= 1100 && code < 1200:
		if code == ErrForbidden {
			return http.StatusForbidden
		}
		return http.StatusUnauthorized
	case code == ErrNotFound || code == ErrTaskNotFound || code == ErrWorkerNotFound:
		return http.StatusNotFound
	case code == ErrConflict || code == ErrTaskAlreadyExists:
		return http.StatusConflict
	case code == ErrRateLimit:
		return http.StatusTooManyRequests
	case code == ErrInvalidInput || code == ErrCapsuleTooLarge || code == ErrInvalidEntrypoint:
		return http.StatusBadRequest
	case code == ErrNoAvailableWorker:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
