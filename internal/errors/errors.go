// Package errors provides structured error types with helpful suggestions.
// Each error includes a type classification, message, suggestion for fixing
// the issue, and optional alternative solutions. This enables better user
// experience by providing actionable error messages.
//
// Error types include permission, disk space, encryption, not found,
// network, and configuration errors.
package errors

import (
	"fmt"
	"strings"
)

// ErrorType categorizes the type of error
type ErrorType int

const (
	// PermissionError indicates a file permission issue
	PermissionError ErrorType = iota
	// DiskSpaceError indicates insufficient disk space
	DiskSpaceError
	// EncryptionError indicates an encryption-related issue
	EncryptionError
	// NotFoundError indicates a file or directory was not found
	NotFoundError
	// NetworkError indicates a network-related issue
	NetworkError
	// ConfigError indicates a configuration issue
	ConfigError
	// UnknownError is a catch-all for unexpected errors
	UnknownError
)

// StashError represents an error with context and suggestions
type StashError struct {
	Type       ErrorType
	Message    string
	Suggestion string
	Alternative string
	Cause      error
	FilePath   string
}

// Error implements the error interface
func (e *StashError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *StashError) Unwrap() error {
	return e.Cause
}

// New creates a new StashError
func New(errType ErrorType, message string) *StashError {
	return &StashError{
		Type:    errType,
		Message: message,
	}
}

// Wrap wraps an existing error with additional context
func Wrap(err error, errType ErrorType, message string) *StashError {
	return &StashError{
		Type:    errType,
		Message: message,
		Cause:   err,
	}
}

// WithSuggestion adds a suggestion to the error
func (e *StashError) WithSuggestion(suggestion string) *StashError {
	e.Suggestion = suggestion
	return e
}

// WithAlternative adds an alternative solution to the error
func (e *StashError) WithAlternative(alternative string) *StashError {
	e.Alternative = alternative
	return e
}

// WithFilePath adds a file path to the error
func (e *StashError) WithFilePath(path string) *StashError {
	e.FilePath = path
	return e
}

// DetectErrorType attempts to detect the error type from a generic error
func DetectErrorType(err error) ErrorType {
	if err == nil {
		return UnknownError
	}

	errStr := strings.ToLower(err.Error())

	if strings.Contains(errStr, "permission denied") || strings.Contains(errStr, "access denied") {
		return PermissionError
	}
	if strings.Contains(errStr, "no space left") || strings.Contains(errStr, "disk full") {
		return DiskSpaceError
	}
	if strings.Contains(errStr, "no such file") || strings.Contains(errStr, "does not exist") {
		return NotFoundError
	}
	if strings.Contains(errStr, "encrypt") || strings.Contains(errStr, "decrypt") {
		return EncryptionError
	}
	if strings.Contains(errStr, "network") || strings.Contains(errStr, "timeout") || strings.Contains(errStr, "connection refused") {
		return NetworkError
	}
	if strings.Contains(errStr, "config") || strings.Contains(errStr, "yaml") {
		return ConfigError
	}

	return UnknownError
}

// WrapWithDetection wraps an error and attempts to detect its type
func WrapWithDetection(err error, message string) *StashError {
	errType := DetectErrorType(err)
	stashErr := Wrap(err, errType, message)

	// Add type-specific suggestions
	switch errType {
	case PermissionError:
		stashErr.WithSuggestion("Check file permissions using 'ls -la'").
			WithAlternative("Run with appropriate permissions or skip this file")
	case DiskSpaceError:
		stashErr.WithSuggestion("Free up disk space by removing old backups").
			WithAlternative("Use 'stash cleanup' to remove old backups")
	case EncryptionError:
		stashErr.WithSuggestion("Ensure the encryption key exists and is readable").
			WithAlternative("Run 'stash init' to generate a new key")
	case NotFoundError:
		stashErr.WithSuggestion("Verify the file or directory exists").
			WithAlternative("Update the configuration to remove missing paths")
	case ConfigError:
		stashErr.WithSuggestion("Check your ~/.stash.yaml configuration file").
			WithAlternative("Run 'stash init' to regenerate the default configuration")
	}

	return stashErr
}

// Common error constructors

// NewPermissionError creates a permission error with helpful suggestions
func NewPermissionError(path string, cause error) *StashError {
	return &StashError{
		Type:        PermissionError,
		Message:     fmt.Sprintf("Permission denied accessing: %s", path),
		Suggestion:  fmt.Sprintf("Run: chmod 600 %s", path),
		Alternative: "Or skip this file with appropriate --exclude flags",
		Cause:       cause,
		FilePath:    path,
	}
}

// NewDiskSpaceError creates a disk space error
func NewDiskSpaceError(requiredSpace, availableSpace int64, cause error) *StashError {
	return &StashError{
		Type:        DiskSpaceError,
		Message:     fmt.Sprintf("Insufficient disk space (need %s, have %s)", formatBytes(requiredSpace), formatBytes(availableSpace)),
		Suggestion:  "Free up disk space by removing old files or backups",
		Alternative: "Use 'stash cleanup --keep 3' to keep only recent backups",
		Cause:       cause,
	}
}

// NewEncryptionError creates an encryption error
func NewEncryptionError(keyPath string, cause error) *StashError {
	return &StashError{
		Type:        EncryptionError,
		Message:     fmt.Sprintf("Encryption failed using key: %s", keyPath),
		Suggestion:  "Ensure the encryption key exists and has correct permissions (chmod 600)",
		Alternative: "Run 'stash init' to generate a new encryption key",
		Cause:       cause,
		FilePath:    keyPath,
	}
}

// NewNotFoundError creates a not found error
func NewNotFoundError(path string, cause error) *StashError {
	return &StashError{
		Type:        NotFoundError,
		Message:     fmt.Sprintf("File or directory not found: %s", path),
		Suggestion:  "Verify the path exists",
		Alternative: "Update ~/.stash.yaml to remove this path from search_paths",
		Cause:       cause,
		FilePath:    path,
	}
}

// NewConfigError creates a configuration error
func NewConfigError(field string, cause error) *StashError {
	return &StashError{
		Type:        ConfigError,
		Message:     fmt.Sprintf("Configuration error in field: %s", field),
		Suggestion:  "Check your ~/.stash.yaml configuration file",
		Alternative: "Run 'stash init' to regenerate default configuration",
		Cause:       cause,
	}
}

// NewNetworkError creates a network error
func NewNetworkError(operation string, cause error) *StashError {
	return &StashError{
		Type:        NetworkError,
		Message:     fmt.Sprintf("Network error during %s", operation),
		Suggestion:  "Check your internet connection and try again",
		Alternative: "Operation will be retried automatically",
		Cause:       cause,
	}
}

// formatBytes formats bytes into human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// IsPermissionError checks if an error is a permission error
func IsPermissionError(err error) bool {
	if stashErr, ok := err.(*StashError); ok {
		return stashErr.Type == PermissionError
	}
	return false
}

// IsDiskSpaceError checks if an error is a disk space error
func IsDiskSpaceError(err error) bool {
	if stashErr, ok := err.(*StashError); ok {
		return stashErr.Type == DiskSpaceError
	}
	return false
}

// IsEncryptionError checks if an error is an encryption error
func IsEncryptionError(err error) bool {
	if stashErr, ok := err.(*StashError); ok {
		return stashErr.Type == EncryptionError
	}
	return false
}

// IsRecoverable checks if an error is recoverable (can continue with partial backup)
func IsRecoverable(err error) bool {
	if stashErr, ok := err.(*StashError); ok {
		// Permission errors and not found errors are recoverable
		return stashErr.Type == PermissionError || stashErr.Type == NotFoundError
	}
	return false
}
