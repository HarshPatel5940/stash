package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	if len(cfg.SearchPaths) == 0 {
		t.Error("SearchPaths should not be empty")
	}

	if len(cfg.Exclude) == 0 {
		t.Error("Exclude patterns should not be empty")
	}

	if cfg.BackupDir == "" {
		t.Error("BackupDir should not be empty")
	}

	if cfg.EncryptionKey == "" {
		t.Error("EncryptionKey should not be empty")
	}
}

func TestConfigSave(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test-config.yaml")

	// Create and save config
	cfg := DefaultConfig()
	cfg.BackupDir = "/tmp/test-backups"

	err := cfg.Save(configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not created")
	}

	// Verify file is readable
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read saved config: %v", err)
	}

	if len(data) == 0 {
		t.Error("Saved config file is empty")
	}
}

func TestExpandPaths(t *testing.T) {
	cfg := &Config{
		SearchPaths:   []string{"~/test/path"},
		BackupDir:     "~/backups",
		EncryptionKey: "~/.test.key",
	}

	cfg.ExpandPaths()

	homeDir, _ := os.UserHomeDir()

	// Check that tilde was expanded
	if cfg.BackupDir == "~/backups" {
		t.Error("BackupDir tilde was not expanded")
	}

	expectedBackupDir := filepath.Join(homeDir, "backups")
	if cfg.BackupDir != expectedBackupDir {
		t.Errorf("BackupDir = %s, want %s", cfg.BackupDir, expectedBackupDir)
	}

	expectedKeyPath := filepath.Join(homeDir, ".test.key")
	if cfg.EncryptionKey != expectedKeyPath {
		t.Errorf("EncryptionKey = %s, want %s", cfg.EncryptionKey, expectedKeyPath)
	}

	if len(cfg.SearchPaths) > 0 && cfg.SearchPaths[0] == "~/test/path" {
		t.Error("SearchPaths tilde was not expanded")
	}
}

func TestConfigExcludePatterns(t *testing.T) {
	cfg := DefaultConfig()

	// Verify common exclude patterns are present
	expectedPatterns := []string{
		"*/node_modules/*",
		"*/vendor/*",
		"*/.git/*",
	}

	for _, pattern := range expectedPatterns {
		found := false
		for _, exclude := range cfg.Exclude {
			if exclude == pattern {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected exclude pattern %s not found", pattern)
		}
	}
}
