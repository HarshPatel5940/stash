package index

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	idx := New()
	if idx == nil {
		t.Fatal("New() returned nil")
	}
	if idx.Version != "1.0" {
		t.Errorf("Expected version 1.0, got %s", idx.Version)
	}
	if idx.Files == nil {
		t.Error("Files map should be initialized")
	}
	if len(idx.Files) != 0 {
		t.Errorf("Expected empty files map, got %d files", len(idx.Files))
	}
}

func TestLoadNonExistent(t *testing.T) {
	idx, err := Load("/nonexistent/path/index.json")
	if err != nil {
		t.Fatalf("Load should not error for nonexistent file: %v", err)
	}
	if idx == nil {
		t.Fatal("Load should return new index for nonexistent file")
	}
	if len(idx.Files) != 0 {
		t.Error("New index should have empty files map")
	}
}

func TestSaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	indexPath := filepath.Join(tempDir, "test-index.json")

	// Create and populate an index
	idx := New()
	idx.AddFile("/test/file1.txt", &FileFingerprint{
		Path:       "/test/file1.txt",
		Size:       100,
		ModTime:    time.Now(),
		Checksum:   "abc123",
		BackupedIn: "backup-2024-01-01",
	})

	// Save
	if err := idx.Save(indexPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load
	loaded, err := Load(indexPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded.Files) != 1 {
		t.Errorf("Expected 1 file, got %d", len(loaded.Files))
	}

	fp, exists := loaded.GetFile("/test/file1.txt")
	if !exists {
		t.Fatal("File should exist in loaded index")
	}
	if fp.Checksum != "abc123" {
		t.Errorf("Expected checksum abc123, got %s", fp.Checksum)
	}
}

func TestAddAndGetFile(t *testing.T) {
	idx := New()

	fp := &FileFingerprint{
		Path:       "/test/path",
		Size:       1024,
		ModTime:    time.Now(),
		Checksum:   "checksum123",
		BackupedIn: "backup-1",
	}

	idx.AddFile("/test/path", fp)

	retrieved, exists := idx.GetFile("/test/path")
	if !exists {
		t.Fatal("Added file should exist")
	}
	if retrieved.Size != 1024 {
		t.Errorf("Expected size 1024, got %d", retrieved.Size)
	}

	_, exists = idx.GetFile("/nonexistent")
	if exists {
		t.Error("Nonexistent file should not exist")
	}
}

func TestHasChanged(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	// Create a test file
	if err := os.WriteFile(testFile, []byte("hello"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	info, _ := os.Stat(testFile)

	idx := New()

	// New file should be detected as changed
	changed, err := idx.HasChanged(testFile)
	if err != nil {
		t.Fatalf("HasChanged failed: %v", err)
	}
	if !changed {
		t.Error("New file should be detected as changed")
	}

	// Add file to index
	idx.AddFile(testFile, &FileFingerprint{
		Path:    testFile,
		Size:    info.Size(),
		ModTime: info.ModTime(),
	})

	// Same file should not be changed
	changed, err = idx.HasChanged(testFile)
	if err != nil {
		t.Fatalf("HasChanged failed: %v", err)
	}
	if changed {
		t.Error("Unchanged file should not be detected as changed")
	}

	// Modify the file
	time.Sleep(10 * time.Millisecond) // Ensure mtime changes
	if err := os.WriteFile(testFile, []byte("hello world"), 0644); err != nil {
		t.Fatalf("Failed to modify test file: %v", err)
	}

	// Modified file should be detected as changed
	changed, err = idx.HasChanged(testFile)
	if err != nil {
		t.Fatalf("HasChanged failed: %v", err)
	}
	if !changed {
		t.Error("Modified file should be detected as changed")
	}
}

func TestHasChangedDeletedFile(t *testing.T) {
	idx := New()
	idx.AddFile("/deleted/file.txt", &FileFingerprint{
		Path: "/deleted/file.txt",
		Size: 100,
	})

	changed, err := idx.HasChanged("/deleted/file.txt")
	if err != nil {
		t.Fatalf("HasChanged failed: %v", err)
	}
	if !changed {
		t.Error("Deleted file should be detected as changed")
	}
}

func TestGetChangedFiles(t *testing.T) {
	tempDir := t.TempDir()
	file1 := filepath.Join(tempDir, "file1.txt")
	file2 := filepath.Join(tempDir, "file2.txt")

	os.WriteFile(file1, []byte("content1"), 0644)
	os.WriteFile(file2, []byte("content2"), 0644)

	info1, _ := os.Stat(file1)

	idx := New()
	idx.AddFile(file1, &FileFingerprint{
		Path:    file1,
		Size:    info1.Size(),
		ModTime: info1.ModTime(),
	})
	// file2 not in index

	changed, err := idx.GetChangedFiles([]string{file1, file2})
	if err != nil {
		t.Fatalf("GetChangedFiles failed: %v", err)
	}

	// file2 should be detected as changed (new)
	if len(changed) != 1 {
		t.Errorf("Expected 1 changed file, got %d", len(changed))
	}
	if len(changed) > 0 && changed[0] != file2 {
		t.Errorf("Expected %s to be changed, got %s", file2, changed[0])
	}
}

func TestCreateFingerprint(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")

	content := []byte("test content for fingerprint")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	fp, err := CreateFingerprint(testFile, "backup-test")
	if err != nil {
		t.Fatalf("CreateFingerprint failed: %v", err)
	}

	if fp.Path != testFile {
		t.Errorf("Expected path %s, got %s", testFile, fp.Path)
	}
	if fp.Size != int64(len(content)) {
		t.Errorf("Expected size %d, got %d", len(content), fp.Size)
	}
	if fp.Checksum == "" {
		t.Error("Checksum should not be empty")
	}
	if fp.BackupedIn != "backup-test" {
		t.Errorf("Expected BackupedIn 'backup-test', got %s", fp.BackupedIn)
	}
}

func TestMarkFullBackup(t *testing.T) {
	idx := New()
	now := time.Now()

	idx.MarkFullBackup(now, "full-backup-1")

	if !idx.LastFullBackup.Equal(now) {
		t.Error("LastFullBackup should be set")
	}
	if !idx.LastBackup.Equal(now) {
		t.Error("LastBackup should be set")
	}
	if idx.GetLastFullBackupName() != "full-backup-1" {
		t.Errorf("Expected backup name 'full-backup-1', got %s", idx.GetLastFullBackupName())
	}
}

func TestMarkIncrementalBackup(t *testing.T) {
	idx := New()
	fullTime := time.Now()
	idx.MarkFullBackup(fullTime, "full-backup")

	incrTime := fullTime.Add(time.Hour)
	idx.MarkIncrementalBackup(incrTime)

	if !idx.LastBackup.Equal(incrTime) {
		t.Error("LastBackup should be updated")
	}
	if !idx.LastFullBackup.Equal(fullTime) {
		t.Error("LastFullBackup should not change")
	}
}

func TestNeedFullBackup(t *testing.T) {
	idx := New()

	// New index needs full backup
	if !idx.NeedFullBackup(time.Hour) {
		t.Error("New index should need full backup")
	}

	// After marking full backup, should not need another
	idx.MarkFullBackup(time.Now(), "backup")
	if idx.NeedFullBackup(time.Hour) {
		t.Error("Should not need full backup immediately after one")
	}

	// After interval passes, should need full backup
	idx.LastFullBackup = time.Now().Add(-2 * time.Hour)
	if !idx.NeedFullBackup(time.Hour) {
		t.Error("Should need full backup after interval passes")
	}
}

func TestGetFileCount(t *testing.T) {
	idx := New()

	if idx.GetFileCount() != 0 {
		t.Error("New index should have 0 files")
	}

	idx.AddFile("/file1", &FileFingerprint{Path: "/file1", Size: 100})
	idx.AddFile("/file2", &FileFingerprint{Path: "/file2", Size: 200})

	if idx.GetFileCount() != 2 {
		t.Errorf("Expected 2 files, got %d", idx.GetFileCount())
	}
}

func TestGetTotalSize(t *testing.T) {
	idx := New()

	if idx.GetTotalSize() != 0 {
		t.Error("New index should have 0 total size")
	}

	idx.AddFile("/file1", &FileFingerprint{Path: "/file1", Size: 100})
	idx.AddFile("/file2", &FileFingerprint{Path: "/file2", Size: 200})

	if idx.GetTotalSize() != 300 {
		t.Errorf("Expected total size 300, got %d", idx.GetTotalSize())
	}
}

func TestRemoveFile(t *testing.T) {
	idx := New()
	idx.AddFile("/file1", &FileFingerprint{Path: "/file1"})

	if idx.GetFileCount() != 1 {
		t.Fatal("File should be added")
	}

	idx.RemoveFile("/file1")

	if idx.GetFileCount() != 0 {
		t.Error("File should be removed")
	}

	_, exists := idx.GetFile("/file1")
	if exists {
		t.Error("File should not exist after removal")
	}
}

func TestGetBackupedFiles(t *testing.T) {
	idx := New()
	idx.AddFile("/file1", &FileFingerprint{Path: "/file1", BackupedIn: "backup-1"})
	idx.AddFile("/file2", &FileFingerprint{Path: "/file2", BackupedIn: "backup-1"})
	idx.AddFile("/file3", &FileFingerprint{Path: "/file3", BackupedIn: "backup-2"})

	files := idx.GetBackupedFiles("backup-1")
	if len(files) != 2 {
		t.Errorf("Expected 2 files in backup-1, got %d", len(files))
	}

	files = idx.GetBackupedFiles("backup-2")
	if len(files) != 1 {
		t.Errorf("Expected 1 file in backup-2, got %d", len(files))
	}

	files = idx.GetBackupedFiles("nonexistent")
	if len(files) != 0 {
		t.Error("Nonexistent backup should have 0 files")
	}
}

func TestGetDefaultIndexPath(t *testing.T) {
	path := GetDefaultIndexPath()
	if path == "" {
		t.Error("Default index path should not be empty")
	}
	if !filepath.IsAbs(path) {
		t.Error("Default index path should be absolute")
	}
}

func TestConcurrentAccess(t *testing.T) {
	idx := New()
	done := make(chan bool)

	// Concurrent writes
	go func() {
		for i := 0; i < 100; i++ {
			idx.AddFile("/file"+string(rune(i)), &FileFingerprint{Path: "/file" + string(rune(i))})
		}
		done <- true
	}()

	// Concurrent reads
	go func() {
		for i := 0; i < 100; i++ {
			idx.GetFileCount()
			idx.GetTotalSize()
		}
		done <- true
	}()

	<-done
	<-done
}
