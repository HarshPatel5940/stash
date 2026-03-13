package packager

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCountBrewfilePackages(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "installer-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{
			name: "standard brewfile",
			content: `tap "homebrew/cask"
brew "git"
brew "vim"
cask "visual-studio-code"
mas "Xcode", id: 497799835
`,
			expected: 5,
		},
		{
			name: "with comments",
			content: `# Development tools
brew "git"
# Editor
brew "vim"
# IDE
cask "visual-studio-code"
`,
			expected: 3,
		},
		{
			name:     "empty file",
			content:  "",
			expected: 0,
		},
		{
			name: "only comments",
			content: `# This is a comment
# Another comment
`,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			brewfilePath := filepath.Join(tempDir, "Brewfile")
			if err := os.WriteFile(brewfilePath, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			result := countBrewfilePackages(brewfilePath)
			if result != tt.expected {
				t.Errorf("countBrewfilePackages() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestReadNonEmptyLines(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "installer-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{
			name: "normal lines",
			content: `line1
line2
line3
`,
			expected: 3,
		},
		{
			name: "with empty lines",
			content: `line1

line2

line3
`,
			expected: 3,
		},
		{
			name: "with comments",
			content: `# comment
line1
# another comment
line2
`,
			expected: 2,
		},
		{
			name: "with whitespace",
			content: `  line1
	line2

line3
`,
			expected: 3,
		},
		{
			name:     "empty file",
			content:  "",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, "test.txt")
			if err := os.WriteFile(filePath, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			lines, err := readNonEmptyLines(filePath)
			if err != nil {
				t.Fatal(err)
			}

			if len(lines) != tt.expected {
				t.Errorf("readNonEmptyLines() returned %d lines, want %d", len(lines), tt.expected)
			}
		})
	}
}

func TestReadNonEmptyLines_FileNotFound(t *testing.T) {
	_, err := readNonEmptyLines("/nonexistent/file.txt")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}
}

func TestNewInstaller(t *testing.T) {
	installer := NewInstaller(false)
	if installer == nil {
		t.Error("NewInstaller returned nil")
	}

	installerVerbose := NewInstaller(true)
	if installerVerbose == nil {
		t.Error("NewInstaller(true) returned nil")
	}
}
