package errors

import "fmt"

// ValidationError represents an error during input validation
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed for %s: %s", e.Field, e.Message)
}

// NewValidationError creates a new ValidationError
func NewValidationError(field, message string) error {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// APIError represents an error from external API calls
type APIError struct {
	StatusCode int
	Message    string
	Operation  string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s failed with status %d: %s", e.Operation, e.StatusCode, e.Message)
}

// NewAPIError creates a new APIError
func NewAPIError(operation string, statusCode int, message string) error {
	return &APIError{
		Operation:  operation,
		StatusCode: statusCode,
		Message:    message,
	}
}

// ConfigError represents an error in configuration
type ConfigError struct {
	Component string
	Message   string
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("configuration error in %s: %s", e.Component, e.Message)
}

// NewConfigError creates a new ConfigError
func NewConfigError(component, message string) error {
	return &ConfigError{
		Component: component,
		Message:   message,
	}
} 