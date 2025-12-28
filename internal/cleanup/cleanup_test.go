package cleanup

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetBackups(t *testing.T) {

	tmpDir, err := os.MkdirTemp("", "stash-cleanup-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	files := []struct {
		name    string
		modTime time.Time
	}{
		{"backup-old.tar.gz", time.Now().Add(-24 * time.Hour)},
		{"backup-new.tar.gz.age", time.Now()},
		{"backup-mid.tar.gz", time.Now().Add(-1 * time.Hour)},
		{"not-a-backup.txt", time.Now()},
		{"folder", time.Now()},
	}

	for _, f := range files {
		path := filepath.Join(tmpDir, f.name)
		if f.name == "folder" {
			os.Mkdir(path, 0755)
			continue
		}
		os.WriteFile(path, []byte("dummy"), 0644)
		os.Chtimes(path, f.modTime, f.modTime)
	}

	cm := NewCleanupManager(tmpDir)
	backups, err := cm.GetBackups()
	if err != nil {
		t.Fatalf("GetBackups failed: %v", err)
	}

	if len(backups) != 3 {
		t.Errorf("Expected 3 backups, got %d", len(backups))
	}

	if backups[0].Path != filepath.Join(tmpDir, "backup-new.tar.gz.age") {
		t.Error("Expected backup-new.tar.gz.age to be first")
	}
	if backups[2].Path != filepath.Join(tmpDir, "backup-old.tar.gz") {
		t.Error("Expected backup-old.tar.gz to be last")
	}
}

func TestRotateByCount(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "stash-cleanup-count-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("backup-%d.tar.gz", i)
		path := filepath.Join(tmpDir, name)
		os.WriteFile(path, []byte("dummy"), 0644)

		ts := time.Now().Add(time.Duration(i) * time.Minute)
		os.Chtimes(path, ts, ts)
	}

	cm := NewCleanupManager(tmpDir)

	deleted, err := cm.RotateByCount(3)
	if err != nil {
		t.Fatal(err)
	}

	if deleted != 2 {
		t.Errorf("Expected 2 deleted files, got %d", deleted)
	}

	backups, _ := cm.GetBackups()
	if len(backups) != 3 {
		t.Errorf("Expected 3 files remaining, got %d", len(backups))
	}
}

func TestRotateByAge(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "stash-cleanup-age-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	files := []struct {
		name string
		age  time.Duration
	}{
		{"backup-recent.tar.gz", 1 * time.Hour},
		{"backup-old.tar.gz", 25 * time.Hour},
		{"backup-older.tar.gz", 48 * time.Hour},
	}

	for _, f := range files {
		path := filepath.Join(tmpDir, f.name)
		os.WriteFile(path, []byte("dummy"), 0644)
		ts := time.Now().Add(-f.age)
		os.Chtimes(path, ts, ts)
	}

	cm := NewCleanupManager(tmpDir)

	deleted, err := cm.RotateByAge(24 * time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	if deleted != 2 {
		t.Errorf("Expected 2 deleted files, got %d", deleted)
	}

	backups, _ := cm.GetBackups()
	if len(backups) != 1 {
		t.Errorf("Expected 1 file remaining, got %d", len(backups))
	}
	if filepath.Base(backups[0].Path) != "backup-recent.tar.gz" {
		t.Error("Wrong file remaining")
	}
}

func TestRotateBySize(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "stash-cleanup-size-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	content := []byte("1234567890")
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("backup-%d.tar.gz", i)
		path := filepath.Join(tmpDir, name)
		os.WriteFile(path, content, 0644)
		ts := time.Now().Add(time.Duration(i) * time.Minute)
		os.Chtimes(path, ts, ts)
	}

	cm := NewCleanupManager(tmpDir)

	deleted, err := cm.RotateBySize(25)
	if err != nil {
		t.Fatal(err)
	}

	if deleted != 1 {
		t.Errorf("Expected 1 deleted file, got %d", deleted)
	}

	backups, _ := cm.GetBackups()
	if len(backups) != 2 {
		t.Errorf("Expected 2 files remaining, got %d", len(backups))
	}
}

func TestStats(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "stash-cleanup-stats-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	os.WriteFile(filepath.Join(tmpDir, "backup1.tar.gz"), []byte("12345"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "backup2.tar.gz"), []byte("12345"), 0644)

	cm := NewCleanupManager(tmpDir)

	total, err := cm.GetTotalSize()
	if err != nil {
		t.Fatal(err)
	}
	if total != 10 {
		t.Errorf("Expected total size 10, got %d", total)
	}

	stats, err := cm.GetStats()
	if err != nil {
		t.Fatal(err)
	}
	if stats["count"].(int) != 2 {
		t.Errorf("Expected count 2, got %v", stats["count"])
	}
}

func TestFormatting(t *testing.T) {
	if formatBytes(500) != "500 B" {
		t.Errorf("formatBytes(500) failed")
	}
	if formatBytes(2048) != "2.0 KB" {
		t.Errorf("formatBytes(2048) failed")
	}

	if formatDuration(30*time.Second) != "just now" {
		t.Errorf("formatDuration(30s) failed")
	}
	if formatDuration(2*time.Hour) != "2 hours" {
		t.Errorf("formatDuration(2h) failed")
	}
}
