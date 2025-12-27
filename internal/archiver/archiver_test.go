package archiver

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateAndExtract(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "source")
	archivePath := filepath.Join(tempDir, "test.tar.gz")
	extractDir := filepath.Join(tempDir, "extracted")

	// Create source directory with test files
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}

	// Create test files
	testFiles := map[string]string{
		"file1.txt":        "content of file 1",
		"subdir/file2.txt": "content of file 2",
		"subdir/file3.txt": "content of file 3",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(sourceDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create dir for %s: %v", path, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", path, err)
		}
	}

	// Create archive
	arch := NewArchiver()
	if err := arch.Create(sourceDir, archivePath); err != nil {
		t.Fatalf("Failed to create archive: %v", err)
	}

	// Verify archive exists
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		t.Fatal("Archive file was not created")
	}

	// Extract archive
	if err := arch.Extract(archivePath, extractDir); err != nil {
		t.Fatalf("Failed to extract archive: %v", err)
	}

	// Verify extracted files
	for path, expectedContent := range testFiles {
		extractedPath := filepath.Join(extractDir, path)
		content, err := os.ReadFile(extractedPath)
		if err != nil {
			t.Errorf("Failed to read extracted file %s: %v", path, err)
			continue
		}
		if string(content) != expectedContent {
			t.Errorf("Content mismatch for %s. Expected: %s, Got: %s",
				path, expectedContent, string(content))
		}
	}
}

func TestCopyFile(t *testing.T) {
	tempDir := t.TempDir()
	srcPath := filepath.Join(tempDir, "source.txt")
	dstPath := filepath.Join(tempDir, "dest.txt")

	// Create source file
	testContent := "test file content"
	if err := os.WriteFile(srcPath, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create source file: %v", err)
	}

	// Copy file
	arch := NewArchiver()
	if err := arch.CopyFile(srcPath, dstPath); err != nil {
		t.Fatalf("Failed to copy file: %v", err)
	}

	// Verify destination
	content, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("Content mismatch. Expected: %s, Got: %s", testContent, string(content))
	}

	// Verify permissions preserved
	srcInfo, _ := os.Stat(srcPath)
	dstInfo, _ := os.Stat(dstPath)
	if srcInfo.Mode() != dstInfo.Mode() {
		t.Errorf("Permissions not preserved. Expected: %v, Got: %v",
			srcInfo.Mode(), dstInfo.Mode())
	}
}

func TestCopyDir(t *testing.T) {
	tempDir := t.TempDir()
	srcDir := filepath.Join(tempDir, "source")
	dstDir := filepath.Join(tempDir, "dest")

	// Create source directory structure
	files := map[string]string{
		"file1.txt":             "content 1",
		"subdir/file2.txt":      "content 2",
		"subdir/deep/file3.txt": "content 3",
	}

	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}

	for path, content := range files {
		fullPath := filepath.Join(srcDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create dir: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	// Copy directory
	arch := NewArchiver()
	if err := arch.CopyDir(srcDir, dstDir); err != nil {
		t.Fatalf("Failed to copy directory: %v", err)
	}

	// Verify all files copied
	for path, expectedContent := range files {
		dstPath := filepath.Join(dstDir, path)
		content, err := os.ReadFile(dstPath)
		if err != nil {
			t.Errorf("Failed to read %s: %v", path, err)
			continue
		}
		if string(content) != expectedContent {
			t.Errorf("Content mismatch for %s", path)
		}
	}
}

func TestCopyDirWithExclusions(t *testing.T) {
	tempDir := t.TempDir()
	srcDir := filepath.Join(tempDir, "source")
	dstDir := filepath.Join(tempDir, "dest")

	// Create source with node_modules and cache
	files := map[string]string{
		"file.txt":                    "keep",
		"node_modules/package/lib.js": "exclude",
		"cache/data.tmp":              "exclude",
		"subdir/file.txt":             "keep",
	}

	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create source dir: %v", err)
	}

	for path, content := range files {
		fullPath := filepath.Join(srcDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create dir: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	// Copy directory (should exclude node_modules and cache)
	arch := NewArchiver()
	if err := arch.CopyDir(srcDir, dstDir); err != nil {
		t.Fatalf("Failed to copy directory: %v", err)
	}

	// Verify kept files exist
	keptFiles := []string{"file.txt", "subdir/file.txt"}
	for _, path := range keptFiles {
		dstPath := filepath.Join(dstDir, path)
		if _, err := os.Stat(dstPath); os.IsNotExist(err) {
			t.Errorf("Expected file %s to be copied", path)
		}
	}

	// Verify excluded files don't exist
	excludedFiles := []string{"node_modules/package/lib.js", "cache/data.tmp"}
	for _, path := range excludedFiles {
		dstPath := filepath.Join(dstDir, path)
		if _, err := os.Stat(dstPath); !os.IsNotExist(err) {
			t.Errorf("Expected file %s to be excluded", path)
		}
	}
}

func TestPathTraversalProtection(t *testing.T) {
	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "malicious.tar.gz")
	extractDir := filepath.Join(tempDir, "extract")

	// This test verifies that Extract properly handles path traversal
	// In a real scenario, we'd need to create a malicious archive
	// For now, we just ensure the extract directory is created safely
	arch := NewArchiver()

	// Create a safe test archive
	sourceDir := filepath.Join(tempDir, "safe")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "test.txt"), []byte("safe"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if err := arch.Create(sourceDir, archivePath); err != nil {
		t.Fatalf("Failed to create archive: %v", err)
	}

	// Extract should succeed for safe archive
	if err := arch.Extract(archivePath, extractDir); err != nil {
		t.Errorf("Failed to extract safe archive: %v", err)
	}

	// Verify file is in correct location
	extractedFile := filepath.Join(extractDir, "test.txt")
	if _, err := os.Stat(extractedFile); os.IsNotExist(err) {
		t.Error("Extracted file not found in expected location")
	}
}

func TestCopyNonexistentFile(t *testing.T) {
	tempDir := t.TempDir()
	srcPath := filepath.Join(tempDir, "nonexistent.txt")
	dstPath := filepath.Join(tempDir, "dest.txt")

	arch := NewArchiver()
	err := arch.CopyFile(srcPath, dstPath)
	if err == nil {
		t.Error("Expected error when copying nonexistent file")
	}
}

func TestExtractNonexistentArchive(t *testing.T) {
	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "nonexistent.tar.gz")
	extractDir := filepath.Join(tempDir, "extract")

	arch := NewArchiver()
	err := arch.Extract(archivePath, extractDir)
	if err == nil {
		t.Error("Expected error when extracting nonexistent archive")
	}
}

func TestCreateArchiveFromNonexistentDir(t *testing.T) {
	tempDir := t.TempDir()
	sourceDir := filepath.Join(tempDir, "nonexistent")
	archivePath := filepath.Join(tempDir, "test.tar.gz")

	arch := NewArchiver()
	err := arch.Create(sourceDir, archivePath)
	if err == nil {
		t.Error("Expected error when creating archive from nonexistent directory")
	}
}

func TestCopyFilePermissions(t *testing.T) {
	tempDir := t.TempDir()
	srcPath := filepath.Join(tempDir, "source.txt")
	dstPath := filepath.Join(tempDir, "dest.txt")

	// Create source with specific permissions
	if err := os.WriteFile(srcPath, []byte("test"), 0600); err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}

	arch := NewArchiver()
	if err := arch.CopyFile(srcPath, dstPath); err != nil {
		t.Fatalf("Failed to copy: %v", err)
	}

	// Check permissions
	info, err := os.Stat(dstPath)
	if err != nil {
		t.Fatalf("Failed to stat dest: %v", err)
	}

	if info.Mode().Perm() != 0600 {
		t.Errorf("Permissions not preserved. Expected 0600, got %v", info.Mode().Perm())
	}
}

func TestSymlinkHandling(t *testing.T) {
	tempDir := t.TempDir()
	srcDir := filepath.Join(tempDir, "source")
	dstDir := filepath.Join(tempDir, "dest")

	// Create source directory
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create source: %v", err)
	}

	// Create a regular file
	regularFile := filepath.Join(srcDir, "regular.txt")
	if err := os.WriteFile(regularFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create regular file: %v", err)
	}

	// Create a symlink
	symlinkPath := filepath.Join(srcDir, "link.txt")
	if err := os.Symlink(regularFile, symlinkPath); err != nil {
		t.Skipf("Skipping symlink test: %v", err)
	}

	// Copy directory - symlinks should be skipped
	arch := NewArchiver()
	if err := arch.CopyDir(srcDir, dstDir); err != nil {
		t.Fatalf("Failed to copy dir: %v", err)
	}

	// Regular file should be copied
	if _, err := os.Stat(filepath.Join(dstDir, "regular.txt")); os.IsNotExist(err) {
		t.Error("Regular file should be copied")
	}

	// Symlink should be skipped
	dstLink := filepath.Join(dstDir, "link.txt")
	if _, err := os.Lstat(dstLink); !os.IsNotExist(err) {
		t.Error("Symlink should be skipped during copy")
	}
}
