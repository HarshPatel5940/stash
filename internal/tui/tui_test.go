package tui

import (
	"testing"

	"github.com/harshpatel5940/stash/internal/metadata"
)

func TestGetCategoryFromPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"~/.ssh/id_rsa", "SSH"},
		{".ssh/config", "SSH"},
		{"~/.gnupg/pubring.kbx", "GPG"},
		{".gpg/keys", "GPG"},
		{"~/.aws/credentials", "AWS"},
		{".aws/config", "AWS"},
		{"~/.config/nvim/init.lua", "Config"},
		{".config/fish/config.fish", "Config"},
		{"~/.zshrc", "Dotfiles"},
		{".bashrc", "Dotfiles"},
		{".gitconfig", "Dotfiles"},
		{"~/projects/.env", "Environment"},
		{"/some/path/.env.local", "Environment"},
		{"~/Documents/file.txt", "Other"},
		{"/usr/local/bin/app", "Other"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := getCategoryFromPath(tt.path)
			if result != tt.expected {
				t.Errorf("getCategoryFromPath(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestFormatFileLabel(t *testing.T) {
	tests := []struct {
		name     string
		file     metadata.FileInfo
		contains string
	}{
		{
			name: "regular file",
			file: metadata.FileInfo{
				OriginalPath: "~/.zshrc",
				Size:         1024,
				IsDir:        false,
			},
			contains: "[F]",
		},
		{
			name: "directory",
			file: metadata.FileInfo{
				OriginalPath: "~/.config",
				Size:         4096,
				IsDir:        true,
			},
			contains: "[D]",
		},
		{
			name: "long path gets truncated",
			file: metadata.FileInfo{
				OriginalPath: "/very/long/path/that/should/be/truncated/because/it/is/too/long/for/display",
				Size:         512,
				IsDir:        false,
			},
			contains: "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatFileLabel(tt.file)
			if len(result) == 0 {
				t.Error("formatFileLabel returned empty string")
			}
			// Check that the result contains expected elements
			if tt.contains != "" && !containsString(result, tt.contains) {
				t.Errorf("formatFileLabel() = %q, expected to contain %q", result, tt.contains)
			}
		})
	}
}

// Helper function
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
