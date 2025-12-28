package metadata

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	meta := New()

	if meta.Version == "" {
		t.Error("Version should not be empty")
	}

	if meta.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}

	if meta.Files == nil {
		t.Error("Files slice should be initialized")
	}

	if meta.PackageCounts == nil {
		t.Error("PackageCounts map should be initialized")
	}

	if meta.Hostname == "" {
		t.Error("Hostname should be set")
	}
}

func TestAddFile(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := []byte("test content for checksum")

	if err := os.WriteFile(testFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	meta := New()
	err := meta.AddFile(testFile, "backup/test.txt")
	if err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	if len(meta.Files) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(meta.Files))
	}

	fileInfo := meta.Files[0]
	if fileInfo.OriginalPath != testFile {
		t.Errorf("Expected original path %s, got %s", testFile, fileInfo.OriginalPath)
	}

	if fileInfo.BackupPath != "backup/test.txt" {
		t.Errorf("Expected backup path 'backup/test.txt', got %s", fileInfo.BackupPath)
	}

	if fileInfo.Size != int64(len(testContent)) {
		t.Errorf("Expected size %d, got %d", len(testContent), fileInfo.Size)
	}

	if fileInfo.Checksum == "" {
		t.Error("Checksum should not be empty")
	}

	if fileInfo.IsDir {
		t.Error("File should not be marked as directory")
	}

	if meta.BackupSize != int64(len(testContent)) {
		t.Errorf("Expected backup size %d, got %d", len(testContent), meta.BackupSize)
	}
}

func TestAddDirectory(t *testing.T) {
	tempDir := t.TempDir()
	testDir := filepath.Join(tempDir, "testdir")

	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	meta := New()
	err := meta.AddFile(testDir, "backup/testdir")
	if err != nil {
		t.Fatalf("Failed to add directory: %v", err)
	}

	if len(meta.Files) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(meta.Files))
	}

	fileInfo := meta.Files[0]
	if !fileInfo.IsDir {
		t.Error("Entry should be marked as directory")
	}

	if fileInfo.Checksum != "" {
		t.Error("Directory should not have checksum")
	}

	if meta.BackupSize != 0 {
		t.Error("Adding directory should not increase BackupSize")
	}
}

func TestAddNonexistentFile(t *testing.T) {
	tempDir := t.TempDir()
	nonexistent := filepath.Join(tempDir, "nonexistent.txt")

	meta := New()
	err := meta.AddFile(nonexistent, "backup/nonexistent.txt")
	if err == nil {
		t.Error("Expected error when adding nonexistent file")
	}
}

func TestSetPackageCount(t *testing.T) {
	meta := New()

	meta.SetPackageCount("Homebrew", 42)
	meta.SetPackageCount("VSCode", 10)

	if meta.PackageCounts["Homebrew"] != 42 {
		t.Errorf("Expected Homebrew count 42, got %d", meta.PackageCounts["Homebrew"])
	}

	if meta.PackageCounts["VSCode"] != 10 {
		t.Errorf("Expected VSCode count 10, got %d", meta.PackageCounts["VSCode"])
	}
}

func TestSaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	metaPath := filepath.Join(tempDir, "metadata.json")

	meta := New()
	meta.SetPackageCount("Homebrew", 5)
	meta.SetPackageCount("NPM", 3)

	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := meta.AddFile(testFile, "backup/test.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	if err := meta.Save(metaPath); err != nil {
		t.Fatalf("Failed to save metadata: %v", err)
	}

	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		t.Fatal("Metadata file was not created")
	}

	loadedMeta, err := Load(metaPath)
	if err != nil {
		t.Fatalf("Failed to load metadata: %v", err)
	}

	if loadedMeta.Version != meta.Version {
		t.Errorf("Version mismatch. Expected %s, got %s", meta.Version, loadedMeta.Version)
	}

	if loadedMeta.Hostname != meta.Hostname {
		t.Errorf("Hostname mismatch")
	}

	if len(loadedMeta.Files) != len(meta.Files) {
		t.Errorf("Files count mismatch. Expected %d, got %d", len(meta.Files), len(loadedMeta.Files))
	}

	if loadedMeta.PackageCounts["Homebrew"] != 5 {
		t.Errorf("Package count mismatch for Homebrew")
	}

	if loadedMeta.BackupSize != meta.BackupSize {
		t.Errorf("BackupSize mismatch. Expected %d, got %d", meta.BackupSize, loadedMeta.BackupSize)
	}
}

func TestLoadNonexistentFile(t *testing.T) {
	tempDir := t.TempDir()
	metaPath := filepath.Join(tempDir, "nonexistent.json")

	_, err := Load(metaPath)
	if err == nil {
		t.Error("Expected error when loading nonexistent file")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	metaPath := filepath.Join(tempDir, "invalid.json")

	if err := os.WriteFile(metaPath, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("Failed to create invalid JSON file: %v", err)
	}

	_, err := Load(metaPath)
	if err == nil {
		t.Error("Expected error when loading invalid JSON")
	}
}

func TestSummary(t *testing.T) {
	tempDir := t.TempDir()
	meta := New()

	file1 := filepath.Join(tempDir, "file1.txt")
	if err := os.WriteFile(file1, []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	meta.AddFile(file1, "backup/file1.txt")

	dir1 := filepath.Join(tempDir, "dir1")
	if err := os.Mkdir(dir1, 0755); err != nil {
		t.Fatalf("Failed to create dir1: %v", err)
	}
	meta.AddFile(dir1, "backup/dir1")

	meta.SetPackageCount("Homebrew", 10)
	meta.SetPackageCount("VSCode", 5)

	summary := meta.Summary()

	if summary == "" {
		t.Error("Summary should not be empty")
	}

	if !contains(summary, meta.Hostname) {
		t.Error("Summary should contain hostname")
	}

	if !contains(summary, "Files: 1, Directories: 1") {
		t.Error("Summary should contain file and directory counts")
	}

	if !contains(summary, "Homebrew") {
		t.Error("Summary should contain package information")
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{1536 * 1024 * 1024, "1.5 GB"},
	}

	for _, tt := range tests {
		result := FormatSize(tt.bytes)
		if result != tt.expected {
			t.Errorf("FormatSize(%d) = %s, expected %s", tt.bytes, result, tt.expected)
		}
	}
}

func TestChecksumConsistency(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	content := []byte("consistent content")

	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	checksum1, err := calculateChecksum(testFile)
	if err != nil {
		t.Fatalf("Failed to calculate checksum: %v", err)
	}

	checksum2, err := calculateChecksum(testFile)
	if err != nil {
		t.Fatalf("Failed to calculate checksum: %v", err)
	}

	if checksum1 != checksum2 {
		t.Error("Checksums should be consistent for same file")
	}

	if err := os.WriteFile(testFile, []byte("modified content"), 0644); err != nil {
		t.Fatalf("Failed to modify file: %v", err)
	}

	checksum3, err := calculateChecksum(testFile)
	if err != nil {
		t.Fatalf("Failed to calculate checksum: %v", err)
	}

	if checksum1 == checksum3 {
		t.Error("Checksums should differ for modified file")
	}
}

func TestFileInfoTimestamp(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	meta := New()
	if err := meta.AddFile(testFile, "backup/test.txt"); err != nil {
		t.Fatalf("Failed to add file: %v", err)
	}

	fileInfo := meta.Files[0]
	if fileInfo.ModTime.IsZero() {
		t.Error("ModTime should be set")
	}

	if time.Since(fileInfo.ModTime) > time.Minute {
		t.Error("ModTime should be recent")
	}
}

func TestMultipleFiles(t *testing.T) {
	tempDir := t.TempDir()
	meta := New()

	for i := 0; i < 5; i++ {
		filename := filepath.Join(tempDir, "file"+string(rune('0'+i))+".txt")
		content := []byte("content " + string(rune('0'+i)))
		if err := os.WriteFile(filename, content, 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
		if err := meta.AddFile(filename, "backup/file"+string(rune('0'+i))+".txt"); err != nil {
			t.Fatalf("Failed to add file: %v", err)
		}
	}

	if len(meta.Files) != 5 {
		t.Errorf("Expected 5 files, got %d", len(meta.Files))
	}

	checksums := make(map[string]bool)
	for _, f := range meta.Files {
		if checksums[f.Checksum] {
			t.Error("Duplicate checksum found")
		}
		checksums[f.Checksum] = true
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) >= len(substr) &&
			findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
