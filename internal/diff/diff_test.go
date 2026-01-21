package diff

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/harshpatel5940/stash/internal/metadata"
)

func TestFileChange(t *testing.T) {
	fc := FileChange{
		Path:        "/test/file.txt",
		OldSize:     100,
		NewSize:     200,
		SizeDelta:   100,
		OldChecksum: "abc",
		NewChecksum: "def",
	}

	if fc.Path != "/test/file.txt" {
		t.Errorf("Expected path '/test/file.txt', got %s", fc.Path)
	}
	if fc.SizeDelta != 100 {
		t.Errorf("Expected delta 100, got %d", fc.SizeDelta)
	}
}

func TestBackupDiffGetAddedFilesCount(t *testing.T) {
	diff := &BackupDiff{
		AddedFiles: []metadata.FileInfo{
			{OriginalPath: "/file1", IsDir: false},
			{OriginalPath: "/file2", IsDir: false},
			{OriginalPath: "/dir1", IsDir: true},
		},
	}

	count := diff.GetAddedFilesCount()
	if count != 2 {
		t.Errorf("Expected 2 (excluding dirs), got %d", count)
	}
}

func TestBackupDiffGetRemovedFilesCount(t *testing.T) {
	diff := &BackupDiff{
		RemovedFiles: []metadata.FileInfo{
			{OriginalPath: "/file1", IsDir: false},
			{OriginalPath: "/dir1", IsDir: true},
		},
	}

	count := diff.GetRemovedFilesCount()
	if count != 1 {
		t.Errorf("Expected 1 (excluding dirs), got %d", count)
	}
}

func TestBackupDiffGetModifiedFilesCount(t *testing.T) {
	diff := &BackupDiff{
		ModifiedFiles: []FileChange{
			{Path: "/file1"},
			{Path: "/file2"},
		},
	}

	count := diff.GetModifiedFilesCount()
	if count != 2 {
		t.Errorf("Expected 2, got %d", count)
	}
}

func TestBackupDiffHasChanges(t *testing.T) {
	// No changes
	diff := &BackupDiff{
		AddedFiles:     []metadata.FileInfo{},
		RemovedFiles:   []metadata.FileInfo{},
		ModifiedFiles:  []FileChange{},
		PackageChanges: make(map[string]PackageChange),
	}

	if diff.HasChanges() {
		t.Error("Empty diff should have no changes")
	}

	// With added files
	diff.AddedFiles = append(diff.AddedFiles, metadata.FileInfo{})
	if !diff.HasChanges() {
		t.Error("Diff with added files should have changes")
	}

	// With removed files
	diff2 := &BackupDiff{
		AddedFiles:     []metadata.FileInfo{},
		RemovedFiles:   []metadata.FileInfo{{}},
		ModifiedFiles:  []FileChange{},
		PackageChanges: make(map[string]PackageChange),
	}
	if !diff2.HasChanges() {
		t.Error("Diff with removed files should have changes")
	}

	// With modified files
	diff3 := &BackupDiff{
		AddedFiles:     []metadata.FileInfo{},
		RemovedFiles:   []metadata.FileInfo{},
		ModifiedFiles:  []FileChange{{}},
		PackageChanges: make(map[string]PackageChange),
	}
	if !diff3.HasChanges() {
		t.Error("Diff with modified files should have changes")
	}

	// With package changes
	diff4 := &BackupDiff{
		AddedFiles:    []metadata.FileInfo{},
		RemovedFiles:  []metadata.FileInfo{},
		ModifiedFiles: []FileChange{},
		PackageChanges: map[string]PackageChange{
			"homebrew": {Delta: 5},
		},
	}
	if !diff4.HasChanges() {
		t.Error("Diff with package changes should have changes")
	}
}

func TestBackupDiffGetTotalFileChanges(t *testing.T) {
	diff := &BackupDiff{
		AddedFiles: []metadata.FileInfo{
			{IsDir: false},
			{IsDir: false},
		},
		RemovedFiles: []metadata.FileInfo{
			{IsDir: false},
		},
		ModifiedFiles: []FileChange{
			{Path: "/mod1"},
			{Path: "/mod2"},
			{Path: "/mod3"},
		},
	}

	total := diff.GetTotalFileChanges()
	if total != 6 {
		t.Errorf("Expected 6 total changes, got %d", total)
	}
}

func TestBackupDiffGetTopAddedFiles(t *testing.T) {
	diff := &BackupDiff{
		AddedFiles: []metadata.FileInfo{
			{OriginalPath: "/small", Size: 100},
			{OriginalPath: "/large", Size: 1000},
			{OriginalPath: "/medium", Size: 500},
		},
	}

	top := diff.GetTopAddedFiles(2)
	if len(top) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(top))
	}
	if top[0].Size != 1000 {
		t.Errorf("First should be largest (1000), got %d", top[0].Size)
	}
	if top[1].Size != 500 {
		t.Errorf("Second should be medium (500), got %d", top[1].Size)
	}

	// Request more than available
	all := diff.GetTopAddedFiles(10)
	if len(all) != 3 {
		t.Errorf("Should return all available: expected 3, got %d", len(all))
	}
}

func TestBackupDiffGetTopModifiedFiles(t *testing.T) {
	diff := &BackupDiff{
		ModifiedFiles: []FileChange{
			{Path: "/a", SizeDelta: 10},
			{Path: "/b", SizeDelta: -50}, // Negative delta, but larger absolute
			{Path: "/c", SizeDelta: 30},
		},
	}

	top := diff.GetTopModifiedFiles(2)
	if len(top) != 2 {
		t.Fatalf("Expected 2 files, got %d", len(top))
	}
	// Should be sorted by absolute delta
	if top[0].SizeDelta != -50 {
		t.Errorf("First should have largest absolute delta (-50), got %d", top[0].SizeDelta)
	}
}

func TestBackupDiffGetSizeDelta(t *testing.T) {
	diff := &BackupDiff{
		AddedSize:    1000,
		RemovedSize:  300,
		ModifiedSize: 200,
	}

	delta := diff.GetSizeDelta()
	expected := int64(1000 - 300 + 200)
	if delta != expected {
		t.Errorf("Expected %d, got %d", expected, delta)
	}
}

func TestBackupDiffSummaryNoChanges(t *testing.T) {
	diff := &BackupDiff{
		AddedFiles:     []metadata.FileInfo{},
		RemovedFiles:   []metadata.FileInfo{},
		ModifiedFiles:  []FileChange{},
		PackageChanges: make(map[string]PackageChange),
	}

	summary := diff.Summary()
	if summary != "No changes detected between backups" {
		t.Errorf("Unexpected summary: %s", summary)
	}
}

func TestBackupDiffSummaryWithChanges(t *testing.T) {
	diff := &BackupDiff{
		AddedFiles: []metadata.FileInfo{
			{IsDir: false},
		},
		RemovedFiles:  []metadata.FileInfo{},
		ModifiedFiles: []FileChange{},
		PackageChanges: map[string]PackageChange{
			"homebrew": {Delta: 5},
		},
	}

	summary := diff.Summary()
	if summary == "" {
		t.Error("Summary should not be empty")
	}
	if summary == "No changes detected between backups" {
		t.Error("Should show changes when there are some")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{-500, "-500 B"},
		{1024, "1.0 KB"},
		{-1024, "-1.0 KB"},
		{1048576, "1.0 MB"},
		{-1048576, "-1.0 MB"},
	}

	for _, tt := range tests {
		result := formatBytes(tt.bytes)
		if result != tt.expected {
			t.Errorf("formatBytes(%d) = %s, want %s", tt.bytes, result, tt.expected)
		}
	}
}

func TestPackageChange(t *testing.T) {
	pc := PackageChange{
		Name:     "homebrew",
		OldCount: 100,
		NewCount: 110,
		Delta:    10,
	}

	if pc.Name != "homebrew" {
		t.Errorf("Expected 'homebrew', got %s", pc.Name)
	}
	if pc.Delta != 10 {
		t.Errorf("Expected delta 10, got %d", pc.Delta)
	}
}

func TestCompareWithMetadataFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create old backup metadata
	oldMeta := metadata.New()
	oldMeta.AddFileInfo(metadata.FileInfo{
		OriginalPath: "/common/file.txt",
		Size:         100,
		Checksum:     "abc",
	})
	oldMeta.AddFileInfo(metadata.FileInfo{
		OriginalPath: "/removed/file.txt",
		Size:         200,
		Checksum:     "def",
	})
	oldMeta.SetPackageCount("homebrew", 100)

	oldBackupPath := filepath.Join(tempDir, "old-backup.tar.gz.age")
	oldMetaPath := oldBackupPath + ".metadata.json"
	os.WriteFile(oldBackupPath, []byte("dummy"), 0644)
	oldMeta.Save(oldMetaPath)

	// Create new backup metadata
	newMeta := metadata.New()
	newMeta.AddFileInfo(metadata.FileInfo{
		OriginalPath: "/common/file.txt",
		Size:         150, // Modified
		Checksum:     "xyz",
	})
	newMeta.AddFileInfo(metadata.FileInfo{
		OriginalPath: "/added/file.txt",
		Size:         300,
		Checksum:     "ghi",
	})
	newMeta.SetPackageCount("homebrew", 110)

	newBackupPath := filepath.Join(tempDir, "new-backup.tar.gz.age")
	newMetaPath := newBackupPath + ".metadata.json"
	os.WriteFile(newBackupPath, []byte("dummy2"), 0644)
	newMeta.Save(newMetaPath)

	// Compare
	diff, err := Compare(oldBackupPath, newBackupPath)
	if err != nil {
		t.Fatalf("Compare failed: %v", err)
	}

	// Verify diff
	if diff.GetAddedFilesCount() != 1 {
		t.Errorf("Expected 1 added file, got %d", diff.GetAddedFilesCount())
	}
	if diff.GetRemovedFilesCount() != 1 {
		t.Errorf("Expected 1 removed file, got %d", diff.GetRemovedFilesCount())
	}
	if diff.GetModifiedFilesCount() != 1 {
		t.Errorf("Expected 1 modified file, got %d", diff.GetModifiedFilesCount())
	}
	if len(diff.PackageChanges) != 1 {
		t.Errorf("Expected 1 package change, got %d", len(diff.PackageChanges))
	}
	if diff.PackageChanges["homebrew"].Delta != 10 {
		t.Errorf("Expected homebrew delta 10, got %d", diff.PackageChanges["homebrew"].Delta)
	}
}

func TestCompareNoMetadata(t *testing.T) {
	tempDir := t.TempDir()

	oldBackup := filepath.Join(tempDir, "old.tar.gz.age")
	newBackup := filepath.Join(tempDir, "new.tar.gz.age")

	os.WriteFile(oldBackup, []byte("dummy"), 0644)
	os.WriteFile(newBackup, []byte("dummy"), 0644)

	_, err := Compare(oldBackup, newBackup)
	if err == nil {
		t.Error("Compare should fail when metadata is missing")
	}
}
