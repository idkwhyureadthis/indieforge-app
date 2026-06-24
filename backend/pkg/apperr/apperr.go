package apperr

import "fmt"

// Error is an application error with an associated HTTP status code.
type Error struct {
	Status  int
	Message string
}

// Error implements the error interface, returning the user-facing message.
func (e *Error) Error() string { return e.Message }

// New builds an Error with the given HTTP status and message.
func New(status int, msg string) *Error { return &Error{Status: status, Message: msg} }

// Newf builds an Error with a formatted message.
func Newf(status int, format string, a ...any) *Error {
	return &Error{Status: status, Message: fmt.Sprintf(format, a...)}
}

// Common helpers (messages in the UI language — English).

// BadRequest builds a 400 error.
func BadRequest(msg string) *Error { return New(400, msg) }

// Unauthorized builds a 401 error.
func Unauthorized(msg string) *Error { return New(401, msg) }

// Forbidden builds a 403 error.
func Forbidden(msg string) *Error { return New(403, msg) }

// NotFound builds a 404 error.
func NotFound(msg string) *Error { return New(404, msg) }

// Conflict builds a 409 error.
func Conflict(msg string) *Error { return New(409, msg) }

// Unprocessable builds a 422 error.
func Unprocessable(msg string) *Error { return New(422, msg) }

// Internal builds a 500 error.
func Internal(msg string) *Error { return New(500, msg) }
