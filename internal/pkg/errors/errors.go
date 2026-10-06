package errors

import "fmt"

// Error codes (aligned with tech.md §5.1).
const (
	CodeOK              = 0
	CodeInvalidParam    = 40001
	CodeUnauthorized    = 40100
	CodeForbidden       = 40300
	CodeNotFound        = 40400
	CodeRateLimited     = 42900
	CodeInternal        = 50000
	CodeExternalFailure = 50200
)

// AppError is a typed error carrying an API code.
type AppError struct {
	Code    int
	Message string
	Cause   error
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error { return e.Cause }

func New(code int, msg string) *AppError {
	return &AppError{Code: code, Message: msg}
}

func Wrap(code int, msg string, err error) *AppError {
	return &AppError{Code: code, Message: msg, Cause: err}
}

func InvalidParam(msg string) *AppError    { return New(CodeInvalidParam, msg) }
func Unauthorized(msg string) *AppError    { return New(CodeUnauthorized, msg) }
func Forbidden(msg string) *AppError       { return New(CodeForbidden, msg) }
func NotFound(msg string) *AppError        { return New(CodeNotFound, msg) }
func RateLimited(msg string) *AppError     { return New(CodeRateLimited, msg) }
func Internal(msg string, err error) error { return Wrap(CodeInternal, msg, err) }
