package domain

import "errors"

// Sentinel errors classify failures across the service. Transport layers map
// them to status codes in a single place.
var (
	ErrNotFound         = errors.New("not found")
	ErrAlreadyExists    = errors.New("already exists")
	ErrPermissionDenied = errors.New("permission denied")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrValidation       = errors.New("validation failed")
	ErrConflict         = errors.New("conflict")
)

// SafeError carries a message that is safe to expose to API clients.
// It wraps one of the sentinel errors above so callers can classify it
// with errors.Is while transports surface Msg verbatim.
type SafeError struct {
	Kind error
	Msg  string
}

func (e *SafeError) Error() string { return e.Msg }

func (e *SafeError) Unwrap() error { return e.Kind }

// NewValidationError builds a client-visible validation error.
func NewValidationError(msg string) error {
	return &SafeError{Kind: ErrValidation, Msg: msg}
}

// NewNotFoundError builds a client-visible not-found error.
func NewNotFoundError(msg string) error {
	return &SafeError{Kind: ErrNotFound, Msg: msg}
}

// NewConflictError builds a client-visible conflict error.
func NewConflictError(msg string) error {
	return &SafeError{Kind: ErrConflict, Msg: msg}
}

// NewPermissionDeniedError builds a client-visible permission error.
func NewPermissionDeniedError(msg string) error {
	return &SafeError{Kind: ErrPermissionDenied, Msg: msg}
}
