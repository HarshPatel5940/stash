package cloud

import (
	"testing"
)

func TestNewProvider_S3(t *testing.T) {
	cfg := Config{
		Provider: "s3",
		Bucket:   "test-bucket",
		Region:   "us-east-1",
	}

	// This will fail without AWS credentials, but tests the provider creation logic
	provider, err := NewProvider(cfg)
	if err != nil {
		// Expected to fail without credentials in CI, but the type should be correct
		t.Skipf("Skipping S3 provider test (likely no AWS credentials): %v", err)
	}

	if provider.GetName() != "AWS S3" {
		t.Errorf("Expected provider name 'AWS S3', got %q", provider.GetName())
	}
}

func TestNewProvider_S3Compatible(t *testing.T) {
	cfg := Config{
		Provider: "s3",
		Bucket:   "test-bucket",
		Region:   "us-east-1",
		Endpoint: "https://s3.example.com",
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Skipf("Skipping S3-compatible provider test: %v", err)
	}

	if provider.GetName() != "S3-compatible" {
		t.Errorf("Expected provider name 'S3-compatible', got %q", provider.GetName())
	}
}

func TestNewProvider_UnsupportedProvider(t *testing.T) {
	cfg := Config{
		Provider: "unsupported",
		Bucket:   "test-bucket",
		Region:   "us-east-1",
	}

	_, err := NewProvider(cfg)
	if err == nil {
		t.Error("Expected error for unsupported provider, got nil")
	}
}

func TestNewProvider_EmptyProviderDefaultsToS3(t *testing.T) {
	cfg := Config{
		Provider: "", // Empty should default to S3
		Bucket:   "test-bucket",
		Region:   "us-east-1",
	}

	provider, err := NewProvider(cfg)
	if err != nil {
		t.Skipf("Skipping empty provider test: %v", err)
	}

	// Empty provider should default to S3
	if provider.GetName() != "AWS S3" {
		t.Errorf("Expected provider name 'AWS S3' for empty provider, got %q", provider.GetName())
	}
}

func TestBackupEntry(t *testing.T) {
	entry := BackupEntry{
		Name: "backup-2024-01-15.tar.gz.age",
		Key:  "stash/backup-2024-01-15.tar.gz.age",
		Size: 1024 * 1024 * 100, // 100MB
	}

	if entry.Name != "backup-2024-01-15.tar.gz.age" {
		t.Errorf("Unexpected name: %s", entry.Name)
	}

	if entry.Size != 104857600 {
		t.Errorf("Unexpected size: %d", entry.Size)
	}
}
