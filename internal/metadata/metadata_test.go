package metadata

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	meta := New()
	if meta == nil {
		t.Fatal("New() returned nil")
	}
	if meta.Version != "1.1.0" {
		t.Errorf("Expected version 1.1.0, got %s", meta.Version)
	}
	if meta.Files == nil {
		t.Error("Files should be initialized")
	}
	if meta.PackageCounts == nil {
		t.Error("PackageCounts should be initialized")
	}
	if meta.Categories == nil {
		t.Error("Categories should be initialized")
	}
	if meta.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

func TestAddFile(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	content := []byte("test content")
	os.WriteFile(testFile, content, 0644)

	meta := New()
	err := meta.AddFile(testFile, "backup/test.txt")
	if err != nil {
		t.Fatalf("AddFile failed: %v", err)
	}

	if len(meta.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(meta.Files))
	}
	if meta.Files[0].OriginalPath != testFile {
		t.Errorf("Expected original path %s, got %s", testFile, meta.Files[0].OriginalPath)
	}
	if meta.Files[0].BackupPath != "backup/test.txt" {
		t.Errorf("Expected backup path 'backup/test.txt', got %s", meta.Files[0].BackupPath)
	}
	if meta.Files[0].Size != int64(len(content)) {
		t.Errorf("Expected size %d, got %d", len(content), meta.Files[0].Size)
	}
	if meta.Files[0].Checksum == "" {
		t.Error("Checksum should be set")
	}
	if meta.BackupSize != int64(len(content)) {
		t.Errorf("Expected backup size %d, got %d", len(content), meta.BackupSize)
	}
}

func TestAddFileNonexistent(t *testing.T) {
	meta := New()
	err := meta.AddFile("/nonexistent/file.txt", "backup/file.txt")
	if err == nil {
		t.Error("AddFile should fail for nonexistent file")
	}
}

func TestAddFileInfo(t *testing.T) {
	meta := New()
	fileInfo := FileInfo{
		OriginalPath: "/test/file.txt",
		BackupPath:   "backup/file.txt",
		Size:         1024,
		IsDir:        false,
	}

	meta.AddFileInfo(fileInfo)

	if len(meta.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(meta.Files))
	}
	if meta.BackupSize != 1024 {
		t.Errorf("Expected backup size 1024, got %d", meta.BackupSize)
	}
}

func TestSetPackageCount(t *testing.T) {
	meta := New()
	meta.SetPackageCount("homebrew", 150)
	meta.SetPackageCount("npm", 25)

	if meta.PackageCounts["homebrew"] != 150 {
		t.Errorf("Expected 150 homebrew packages, got %d", meta.PackageCounts["homebrew"])
	}
	if meta.PackageCounts["npm"] != 25 {
		t.Errorf("Expected 25 npm packages, got %d", meta.PackageCounts["npm"])
	}
}

func TestSaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	metaPath := filepath.Join(tempDir, "metadata.json")

	meta := New()
	meta.SetPackageCount("test", 10)
	meta.BackupSize = 1000

	if err := meta.Save(metaPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load(metaPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Version != meta.Version {
		t.Errorf("Version mismatch: expected %s, got %s", meta.Version, loaded.Version)
	}
	if loaded.PackageCounts["test"] != 10 {
		t.Error("PackageCounts not preserved")
	}
	if loaded.BackupSize != 1000 {
		t.Error("BackupSize not preserved")
	}
}

func TestLoadNonexistent(t *testing.T) {
	_, err := Load("/nonexistent/metadata.json")
	if err == nil {
		t.Error("Load should fail for nonexistent file")
	}
}

func TestSummary(t *testing.T) {
	meta := New()
	meta.AddFileInfo(FileInfo{
		OriginalPath: "/test/file.txt",
		BackupPath:   "dotfiles/file.txt",
		Size:         1024,
		IsDir:        false,
	})
	meta.AddFileInfo(FileInfo{
		OriginalPath: "/test/dir",
		BackupPath:   "dotfiles/dir",
		Size:         0,
		IsDir:        true,
	})
	meta.SetPackageCount("homebrew", 100)

	summary := meta.Summary()
	if summary == "" {
		t.Error("Summary should not be empty")
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
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		result := FormatSize(tt.bytes)
		if result != tt.expected {
			t.Errorf("FormatSize(%d) = %s, want %s", tt.bytes, result, tt.expected)
		}
	}
}

func TestAddCategoryTiming(t *testing.T) {
	meta := New()
	meta.AddCategoryTiming("dotfiles", 50, 10240, 500*time.Millisecond)

	cat, exists := meta.Categories["dotfiles"]
	if !exists {
		t.Fatal("Category should exist")
	}
	if cat.FileCount != 50 {
		t.Errorf("Expected 50 files, got %d", cat.FileCount)
	}
	if cat.TotalSize != 10240 {
		t.Errorf("Expected size 10240, got %d", cat.TotalSize)
	}
	if cat.Duration != 500*time.Millisecond {
		t.Errorf("Expected duration 500ms, got %v", cat.Duration)
	}
}

func TestSetCompressedSize(t *testing.T) {
	meta := New()
	meta.SetCompressedSize(5000)
	if meta.CompressedSize != 5000 {
		t.Errorf("Expected 5000, got %d", meta.CompressedSize)
	}
}

func TestSetEncryptedSize(t *testing.T) {
	meta := New()
	meta.SetEncryptedSize(5100)
	if meta.EncryptedSize != 5100 {
		t.Errorf("Expected 5100, got %d", meta.EncryptedSize)
	}
}

func TestSetTotalDuration(t *testing.T) {
	meta := New()
	meta.SetTotalDuration(5 * time.Second)
	if meta.TotalDuration != 5*time.Second {
		t.Errorf("Expected 5s, got %v", meta.TotalDuration)
	}
}

func TestGetLargestFiles(t *testing.T) {
	meta := New()
	meta.AddFileInfo(FileInfo{OriginalPath: "/a", Size: 100})
	meta.AddFileInfo(FileInfo{OriginalPath: "/b", Size: 500})
	meta.AddFileInfo(FileInfo{OriginalPath: "/c", Size: 200})

	largest := meta.GetLargestFiles(2)
	if len(largest) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(largest))
	}
	if largest[0].Size != 500 {
		t.Errorf("First file should have size 500, got %d", largest[0].Size)
	}
	if largest[1].Size != 200 {
		t.Errorf("Second file should have size 200, got %d", largest[1].Size)
	}

	// Request more than available
	all := meta.GetLargestFiles(10)
	if len(all) != 3 {
		t.Errorf("Expected 3 files, got %d", len(all))
	}
}

func TestGetCompressionRatio(t *testing.T) {
	meta := New()

	// Zero backup size
	if meta.GetCompressionRatio() != 0 {
		t.Error("Zero backup size should return 0 ratio")
	}

	meta.BackupSize = 1000
	meta.CompressedSize = 500
	ratio := meta.GetCompressionRatio()
	if ratio != 50.0 {
		t.Errorf("Expected 50%% compression, got %.1f%%", ratio)
	}
}

func TestGetFileCount(t *testing.T) {
	meta := New()
	meta.AddFileInfo(FileInfo{IsDir: false})
	meta.AddFileInfo(FileInfo{IsDir: false})
	meta.AddFileInfo(FileInfo{IsDir: true})

	count := meta.GetFileCount()
	if count != 2 {
		t.Errorf("Expected 2 files (excluding dirs), got %d", count)
	}
}

func TestGetCategoryStats(t *testing.T) {
	meta := New()
	meta.AddCategoryTiming("test", 10, 1024, time.Second)

	stats := meta.GetCategoryStats()
	if _, exists := stats["test"]; !exists {
		t.Error("Category 'test' should exist in stats")
	}
}

func TestSetBackupType(t *testing.T) {
	meta := New()
	meta.SetBackupType("incremental")
	if meta.BackupType != "incremental" {
		t.Errorf("Expected 'incremental', got %s", meta.BackupType)
	}
}

func TestSetBaseBackup(t *testing.T) {
	meta := New()
	meta.SetBaseBackup("backup-2024-01-01")
	if meta.BaseBackup != "backup-2024-01-01" {
		t.Errorf("Expected 'backup-2024-01-01', got %s", meta.BaseBackup)
	}
}

func TestSetChangedFilesOnly(t *testing.T) {
	meta := New()
	meta.SetChangedFilesOnly(true)
	if !meta.ChangedFilesOnly {
		t.Error("ChangedFilesOnly should be true")
	}
}

func TestIsIncremental(t *testing.T) {
	meta := New()
	if meta.IsIncremental() {
		t.Error("New metadata should not be incremental")
	}

	meta.BackupType = "incremental"
	if !meta.IsIncremental() {
		t.Error("Should be incremental when BackupType is 'incremental'")
	}
}

func TestIsFull(t *testing.T) {
	meta := New()
	if !meta.IsFull() {
		t.Error("New metadata should be full (default)")
	}

	meta.BackupType = "full"
	if !meta.IsFull() {
		t.Error("Should be full when BackupType is 'full'")
	}

	meta.BackupType = "incremental"
	if meta.IsFull() {
		t.Error("Should not be full when BackupType is 'incremental'")
	}
}

func TestConcurrentAccess(t *testing.T) {
	meta := New()
	done := make(chan bool)

	// Concurrent writes
	go func() {
		for i := 0; i < 100; i++ {
			meta.AddFileInfo(FileInfo{OriginalPath: "/test"})
		}
		done <- true
	}()

	// Concurrent reads
	go func() {
		for i := 0; i < 100; i++ {
			meta.GetFileCount()
		}
		done <- true
	}()

	<-done
	<-done
}
