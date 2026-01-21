package errors

import (
	"errors"
	"testing"
)

func TestNew(t *testing.T) {
	err := New(PermissionError, "test error")
	if err == nil {
		t.Fatal("New() returned nil")
	}
	if err.Type != PermissionError {
		t.Errorf("Expected PermissionError, got %v", err.Type)
	}
	if err.Message != "test error" {
		t.Errorf("Expected 'test error', got %s", err.Message)
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("original error")
	wrapped := Wrap(cause, DiskSpaceError, "wrapped message")

	if wrapped.Type != DiskSpaceError {
		t.Errorf("Expected DiskSpaceError, got %v", wrapped.Type)
	}
	if wrapped.Message != "wrapped message" {
		t.Errorf("Expected 'wrapped message', got %s", wrapped.Message)
	}
	if wrapped.Cause != cause {
		t.Error("Cause should be preserved")
	}
}

func TestStashErrorError(t *testing.T) {
	// Without cause
	err := New(PermissionError, "test message")
	if err.Error() != "test message" {
		t.Errorf("Expected 'test message', got %s", err.Error())
	}

	// With cause
	cause := errors.New("underlying error")
	errWithCause := Wrap(cause, PermissionError, "test message")
	expected := "test message: underlying error"
	if errWithCause.Error() != expected {
		t.Errorf("Expected '%s', got %s", expected, errWithCause.Error())
	}
}

func TestStashErrorUnwrap(t *testing.T) {
	cause := errors.New("original")
	wrapped := Wrap(cause, PermissionError, "wrapped")

	unwrapped := wrapped.Unwrap()
	if unwrapped != cause {
		t.Error("Unwrap should return the original cause")
	}

	// Without cause
	noCause := New(PermissionError, "no cause")
	if noCause.Unwrap() != nil {
		t.Error("Unwrap should return nil when no cause")
	}
}

func TestWithSuggestion(t *testing.T) {
	err := New(PermissionError, "test").WithSuggestion("try this")
	if err.Suggestion != "try this" {
		t.Errorf("Expected 'try this', got %s", err.Suggestion)
	}
}

func TestWithAlternative(t *testing.T) {
	err := New(PermissionError, "test").WithAlternative("or do this")
	if err.Alternative != "or do this" {
		t.Errorf("Expected 'or do this', got %s", err.Alternative)
	}
}

func TestWithFilePath(t *testing.T) {
	err := New(PermissionError, "test").WithFilePath("/test/path")
	if err.FilePath != "/test/path" {
		t.Errorf("Expected '/test/path', got %s", err.FilePath)
	}
}

func TestChainedMethods(t *testing.T) {
	err := New(PermissionError, "test").
		WithSuggestion("suggestion").
		WithAlternative("alternative").
		WithFilePath("/path")

	if err.Suggestion != "suggestion" {
		t.Error("Suggestion not set")
	}
	if err.Alternative != "alternative" {
		t.Error("Alternative not set")
	}
	if err.FilePath != "/path" {
		t.Error("FilePath not set")
	}
}

func TestDetectErrorType(t *testing.T) {
	tests := []struct {
		errMsg   string
		expected ErrorType
	}{
		{"permission denied", PermissionError},
		{"access denied", PermissionError},
		{"no space left on device", DiskSpaceError},
		{"disk full", DiskSpaceError},
		{"no such file or directory", NotFoundError},
		{"does not exist", NotFoundError},
		{"encryption failed", EncryptionError},
		{"decrypt error", EncryptionError},
		{"network timeout", NetworkError},
		{"connection refused", NetworkError},
		{"config error", ConfigError},
		{"yaml parse error", ConfigError},
		{"some random error", UnknownError},
	}

	for _, tt := range tests {
		err := errors.New(tt.errMsg)
		result := DetectErrorType(err)
		if result != tt.expected {
			t.Errorf("DetectErrorType(%q) = %v, want %v", tt.errMsg, result, tt.expected)
		}
	}

	// Test nil error
	if DetectErrorType(nil) != UnknownError {
		t.Error("nil error should return UnknownError")
	}
}

func TestWrapWithDetection(t *testing.T) {
	tests := []struct {
		cause    error
		expected ErrorType
	}{
		{errors.New("permission denied"), PermissionError},
		{errors.New("no space left"), DiskSpaceError},
		{errors.New("random error"), UnknownError},
	}

	for _, tt := range tests {
		wrapped := WrapWithDetection(tt.cause, "wrapped")
		if wrapped.Type != tt.expected {
			t.Errorf("WrapWithDetection detected %v, want %v", wrapped.Type, tt.expected)
		}
	}
}

func TestWrapWithDetectionAddsSuggestions(t *testing.T) {
	err := WrapWithDetection(errors.New("permission denied"), "test")
	if err.Suggestion == "" {
		t.Error("Permission error should have suggestion")
	}

	err = WrapWithDetection(errors.New("no space left"), "test")
	if err.Suggestion == "" {
		t.Error("Disk space error should have suggestion")
	}

	err = WrapWithDetection(errors.New("encryption failed"), "test")
	if err.Suggestion == "" {
		t.Error("Encryption error should have suggestion")
	}

	err = WrapWithDetection(errors.New("no such file"), "test")
	if err.Suggestion == "" {
		t.Error("Not found error should have suggestion")
	}

	err = WrapWithDetection(errors.New("config invalid"), "test")
	if err.Suggestion == "" {
		t.Error("Config error should have suggestion")
	}
}

func TestNewPermissionError(t *testing.T) {
	cause := errors.New("underlying")
	err := NewPermissionError("/test/file.txt", cause)

	if err.Type != PermissionError {
		t.Errorf("Expected PermissionError, got %v", err.Type)
	}
	if err.FilePath != "/test/file.txt" {
		t.Errorf("Expected '/test/file.txt', got %s", err.FilePath)
	}
	if err.Cause != cause {
		t.Error("Cause should be set")
	}
	if err.Suggestion == "" {
		t.Error("Suggestion should be set")
	}
}

func TestNewDiskSpaceError(t *testing.T) {
	cause := errors.New("underlying")
	err := NewDiskSpaceError(1000, 500, cause)

	if err.Type != DiskSpaceError {
		t.Errorf("Expected DiskSpaceError, got %v", err.Type)
	}
	if err.Cause != cause {
		t.Error("Cause should be set")
	}
	if err.Suggestion == "" {
		t.Error("Suggestion should be set")
	}
}

func TestNewEncryptionError(t *testing.T) {
	cause := errors.New("underlying")
	err := NewEncryptionError("/path/to/key", cause)

	if err.Type != EncryptionError {
		t.Errorf("Expected EncryptionError, got %v", err.Type)
	}
	if err.FilePath != "/path/to/key" {
		t.Errorf("Expected '/path/to/key', got %s", err.FilePath)
	}
}

func TestNewNotFoundError(t *testing.T) {
	cause := errors.New("underlying")
	err := NewNotFoundError("/missing/file", cause)

	if err.Type != NotFoundError {
		t.Errorf("Expected NotFoundError, got %v", err.Type)
	}
	if err.FilePath != "/missing/file" {
		t.Errorf("Expected '/missing/file', got %s", err.FilePath)
	}
}

func TestNewConfigError(t *testing.T) {
	cause := errors.New("underlying")
	err := NewConfigError("search_paths", cause)

	if err.Type != ConfigError {
		t.Errorf("Expected ConfigError, got %v", err.Type)
	}
}

func TestNewNetworkError(t *testing.T) {
	cause := errors.New("underlying")
	err := NewNetworkError("download", cause)

	if err.Type != NetworkError {
		t.Errorf("Expected NetworkError, got %v", err.Type)
	}
}

func TestIsPermissionError(t *testing.T) {
	permErr := New(PermissionError, "test")
	if !IsPermissionError(permErr) {
		t.Error("Should detect permission error")
	}

	otherErr := New(DiskSpaceError, "test")
	if IsPermissionError(otherErr) {
		t.Error("Should not detect as permission error")
	}

	regularErr := errors.New("regular")
	if IsPermissionError(regularErr) {
		t.Error("Regular error should not be permission error")
	}
}

func TestIsDiskSpaceError(t *testing.T) {
	diskErr := New(DiskSpaceError, "test")
	if !IsDiskSpaceError(diskErr) {
		t.Error("Should detect disk space error")
	}

	otherErr := New(PermissionError, "test")
	if IsDiskSpaceError(otherErr) {
		t.Error("Should not detect as disk space error")
	}
}

func TestIsEncryptionError(t *testing.T) {
	encErr := New(EncryptionError, "test")
	if !IsEncryptionError(encErr) {
		t.Error("Should detect encryption error")
	}

	otherErr := New(PermissionError, "test")
	if IsEncryptionError(otherErr) {
		t.Error("Should not detect as encryption error")
	}
}

func TestIsRecoverable(t *testing.T) {
	// Permission errors are recoverable
	permErr := New(PermissionError, "test")
	if !IsRecoverable(permErr) {
		t.Error("Permission error should be recoverable")
	}

	// Not found errors are recoverable
	notFoundErr := New(NotFoundError, "test")
	if !IsRecoverable(notFoundErr) {
		t.Error("Not found error should be recoverable")
	}

	// Disk space errors are not recoverable
	diskErr := New(DiskSpaceError, "test")
	if IsRecoverable(diskErr) {
		t.Error("Disk space error should not be recoverable")
	}

	// Regular errors are not recoverable
	regularErr := errors.New("regular")
	if IsRecoverable(regularErr) {
		t.Error("Regular error should not be recoverable")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		result := formatBytes(tt.bytes)
		if result != tt.expected {
			t.Errorf("formatBytes(%d) = %s, want %s", tt.bytes, result, tt.expected)
		}
	}
}

func TestErrorTypesConstants(t *testing.T) {
	// Verify error types are distinct
	types := []ErrorType{
		PermissionError,
		DiskSpaceError,
		EncryptionError,
		NotFoundError,
		NetworkError,
		ConfigError,
		UnknownError,
	}

	seen := make(map[ErrorType]bool)
	for _, et := range types {
		if seen[et] {
			t.Errorf("Duplicate error type: %v", et)
		}
		seen[et] = true
	}
}
