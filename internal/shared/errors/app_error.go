package errors

import "net/http"

type AppError struct {
	Status  int
	Code    string
	Message string
	Details any
	Err     error
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}

	if e.Err != nil {
		return e.Err.Error()
	}

	return e.Message
}

func (e *AppError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Err
}

func (e *AppError) StatusCode() int {
	if e == nil || e.Status <= 0 {
		return http.StatusInternalServerError
	}

	return e.Status
}

func New(status int, code string, message string, details any) *AppError {
	return &AppError{
		Status:  status,
		Code:    code,
		Message: message,
		Details: details,
	}
}

func Wrap(status int, code string, message string, err error, details any) *AppError {
	return &AppError{
		Status:  status,
		Code:    code,
		Message: message,
		Details: details,
		Err:     err,
	}
}

func Validation(message string, details any) *AppError {
	return New(http.StatusBadRequest, "VALIDATION_ERROR", message, details)
}

func Unauthorized(message string, details any) *AppError {
	return New(http.StatusUnauthorized, "UNAUTHORIZED", message, details)
}

func Forbidden(message string, details any) *AppError {
	return New(http.StatusForbidden, "FORBIDDEN", message, details)
}

func NotFound(message string, details any) *AppError {
	return New(http.StatusNotFound, "NOT_FOUND", message, details)
}

func Conflict(message string, details any) *AppError {
	return New(http.StatusConflict, "CONFLICT", message, details)
}

func NotImplemented(message string, details any) *AppError {
	return New(http.StatusNotImplemented, "NOT_IMPLEMENTED", message, details)
}

func ServiceUnavailable(message string, details any) *AppError {
	return New(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", message, details)
}

func Internal(message string, details any) *AppError {
	return New(http.StatusInternalServerError, "INTERNAL_ERROR", message, details)
}
