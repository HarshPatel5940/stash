package incremental

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/harshpatel5940/stash/internal/config"
)

func TestParseIntervalString(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"7d", 7 * 24 * time.Hour},
		{"1d", 24 * time.Hour},
		{"30d", 30 * 24 * time.Hour},
		{"24h", 24 * time.Hour},
		{"48h", 48 * time.Hour},
		{"168h", 168 * time.Hour},
		{"", 0},
		{"invalid", 0},
		{"0d", 0},
		{"-1d", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseIntervalString(tt.input)
			if result != tt.expected {
				t.Errorf("parseIntervalString(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNewManager(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	cfg := config.DefaultConfig()
	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if mgr == nil {
		t.Fatal("Manager should not be nil")
	}
	if mgr.cfg != cfg {
		t.Error("Config should be set")
	}
}

func TestManagerShouldDoFullBackup(t *testing.T) {
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	cfg := config.DefaultConfig()
	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Empty index should require full backup
	if !mgr.ShouldDoFullBackup() {
		t.Error("Empty index should require full backup")
	}
}

func TestManagerShouldDoFullBackupWithConfig(t *testing.T) {
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	cfg := config.DefaultConfig()
	cfg.Incremental = &config.IncrementalConfig{
		Enabled:            true,
		FullBackupInterval: "1d",
		AutoMergeThreshold: 5,
	}

	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Should require full backup for empty index
	if !mgr.ShouldDoFullBackup() {
		t.Error("Empty index should require full backup")
	}
}

func TestManagerFindChangedFiles(t *testing.T) {
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	// Create test files
	file1 := filepath.Join(tempDir, "file1.txt")
	file2 := filepath.Join(tempDir, "file2.txt")
	os.WriteFile(file1, []byte("content1"), 0644)
	os.WriteFile(file2, []byte("content2"), 0644)

	cfg := config.DefaultConfig()
	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// All files should be changed initially
	changed, err := mgr.FindChangedFiles([]string{file1, file2})
	if err != nil {
		t.Fatalf("FindChangedFiles failed: %v", err)
	}
	if len(changed) != 2 {
		t.Errorf("Expected 2 changed files, got %d", len(changed))
	}
}

func TestManagerGetBaseBackup(t *testing.T) {
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	cfg := config.DefaultConfig()
	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// No base backup initially
	if mgr.GetBaseBackup() != "" {
		t.Error("Should have no base backup initially")
	}
}

func TestManagerUpdateIndex(t *testing.T) {
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	// Create test file
	testFile := filepath.Join(tempDir, "testfile.txt")
	os.WriteFile(testFile, []byte("test content"), 0644)

	cfg := config.DefaultConfig()
	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Update index with full backup
	err = mgr.UpdateIndex("backup-001", []string{testFile}, true)
	if err != nil {
		t.Fatalf("UpdateIndex failed: %v", err)
	}

	// Should have the base backup set
	if mgr.GetBaseBackup() != "backup-001" {
		t.Errorf("Expected base backup 'backup-001', got %s", mgr.GetBaseBackup())
	}
}

func TestManagerGetStats(t *testing.T) {
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	cfg := config.DefaultConfig()
	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	fileCount, totalSize, lastBackup := mgr.GetStats()
	if fileCount != 0 {
		t.Errorf("Expected 0 files, got %d", fileCount)
	}
	if totalSize != 0 {
		t.Errorf("Expected 0 size, got %d", totalSize)
	}
	if !lastBackup.IsZero() {
		t.Error("Expected zero last backup time")
	}
}

func TestManagerIsFirstBackup(t *testing.T) {
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	cfg := config.DefaultConfig()
	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if !mgr.IsFirstBackup() {
		t.Error("Should be first backup for empty index")
	}
}

func TestManagerEstimateSavings(t *testing.T) {
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	cfg := config.DefaultConfig()
	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Empty index should show no savings
	skipped, percent := mgr.EstimateSavings(100)
	if skipped != 0 {
		t.Errorf("Expected 0 skipped, got %d", skipped)
	}
	if percent != 0 {
		t.Errorf("Expected 0%% savings, got %.1f%%", percent)
	}

	// Zero total should return zero
	skipped, percent = mgr.EstimateSavings(0)
	if skipped != 0 || percent != 0 {
		t.Error("Zero total should return zero savings")
	}
}

func TestManagerGetBackupType(t *testing.T) {
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	cfg := config.DefaultConfig()
	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Should recommend full backup for empty index
	if mgr.GetBackupType() != "full" {
		t.Errorf("Expected 'full', got %s", mgr.GetBackupType())
	}
}

func TestManagerGetRecommendation(t *testing.T) {
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	cfg := config.DefaultConfig()
	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	rec := mgr.GetRecommendation()
	if rec == "" {
		t.Error("Recommendation should not be empty")
	}
}

func TestManagerCleanupOldIndex(t *testing.T) {
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	cfg := config.DefaultConfig()
	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Add a file that doesn't exist
	mgr.index.AddFile("/nonexistent/file.txt", nil)

	removed, err := mgr.CleanupOldIndex()
	if err != nil {
		t.Fatalf("CleanupOldIndex failed: %v", err)
	}
	if removed != 1 {
		t.Errorf("Expected 1 removed, got %d", removed)
	}
}

func TestManagerGetChangedFilesByPath(t *testing.T) {
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	// Create test directory with files
	testDir := filepath.Join(tempDir, "testdir")
	os.MkdirAll(testDir, 0755)
	os.WriteFile(filepath.Join(testDir, "file1.txt"), []byte("content"), 0644)

	cfg := config.DefaultConfig()
	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	changed, total, err := mgr.GetChangedFilesByPath([]string{testDir})
	if err != nil {
		t.Fatalf("GetChangedFilesByPath failed: %v", err)
	}
	if total != 1 {
		t.Errorf("Expected 1 total file, got %d", total)
	}
	if len(changed) != 1 {
		t.Errorf("Expected 1 changed file, got %d", len(changed))
	}
}

func TestManagerGetChangedFilesByPathNonexistent(t *testing.T) {
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	cfg := config.DefaultConfig()
	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// Non-existent path should be skipped
	changed, total, err := mgr.GetChangedFilesByPath([]string{"/nonexistent/path"})
	if err != nil {
		t.Fatalf("GetChangedFilesByPath failed: %v", err)
	}
	if total != 0 {
		t.Errorf("Expected 0 total files, got %d", total)
	}
	if len(changed) != 0 {
		t.Errorf("Expected 0 changed files, got %d", len(changed))
	}
}

func TestManagerGetChangedFilesByPathWithTilde(t *testing.T) {
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	// Create test file in "home"
	os.WriteFile(filepath.Join(tempDir, "testfile.txt"), []byte("content"), 0644)

	cfg := config.DefaultConfig()
	mgr, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	// This should expand ~ to tempDir
	changed, total, err := mgr.GetChangedFilesByPath([]string{"~/testfile.txt"})
	if err != nil {
		t.Fatalf("GetChangedFilesByPath failed: %v", err)
	}
	// Note: this tests the tilde expansion path
	_ = changed
	_ = total
}
