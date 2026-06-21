package apperrors

import "fmt"

type ValidationError struct {
	Fields  map[string]string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "validation error"
}

type NotFoundError struct {
	Resource string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found", e.Resource)
}

type UnauthorizedError struct {
	Message string
}

func (e *UnauthorizedError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "unauthorized"
}
