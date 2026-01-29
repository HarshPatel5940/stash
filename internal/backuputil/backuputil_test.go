package backuputil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsEncrypted(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"backup.tar.gz.age", true},
		{"backup.tar.gz", false},
		{"/path/to/backup.tar.gz.age", true},
		{"/path/to/backup.tar.gz", false},
		{"backup.age", true},
		{"backup.txt", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := IsEncrypted(tt.path)
			if result != tt.expected {
				t.Errorf("IsEncrypted(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestGetBackupBaseName(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"backup.tar.gz.age", "backup.tar.gz"},
		{"backup.tar.gz", "backup.tar.gz"},
		{"/path/to/backup-2024-01-15.tar.gz.age", "backup-2024-01-15.tar.gz"},
		{"/path/to/backup-2024-01-15.tar.gz", "backup-2024-01-15.tar.gz"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := GetBackupBaseName(tt.path)
			if result != tt.expected {
				t.Errorf("GetBackupBaseName(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestExtractMetadata_FileNotFound(t *testing.T) {
	_, err := ExtractMetadata("/nonexistent/path/backup.tar.gz", "")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

func TestExtractMetadata_NoKey(t *testing.T) {
	// Create a temporary encrypted backup file (just to test key checking)
	tempDir, err := os.MkdirTemp("", "backuputil-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Create a fake .age file
	fakePath := filepath.Join(tempDir, "backup.tar.gz.age")
	if err := os.WriteFile(fakePath, []byte("fake encrypted data"), 0644); err != nil {
		t.Fatal(err)
	}

	// Try to extract with non-existent key
	_, err = ExtractMetadata(fakePath, "/nonexistent/key")
	if err == nil {
		t.Error("Expected error for missing key, got nil")
	}
}
