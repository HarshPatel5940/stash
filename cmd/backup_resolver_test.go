package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResolveBackupInputByIndex(t *testing.T) {
	dir := t.TempDir()

	newer := filepath.Join(dir, "backup-2026-04-07.tar.gz.age")
	older := filepath.Join(dir, "backup-2026-04-06.tar.gz.age")
	if err := os.WriteFile(newer, []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(older, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	if err := os.Chtimes(newer, now, now); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(older, now.Add(-time.Hour), now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}

	got, err := resolveBackupInput("1", dir)
	if err != nil {
		t.Fatalf("resolveBackupInput returned error: %v", err)
	}
	if got.Name != filepath.Base(newer) {
		t.Fatalf("expected newest backup for index 1, got %s", got.Name)
	}
}

func TestResolveBackupInputByNameWithoutExtension(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backup-2026-04-06.tar.gz.age")
	if err := os.WriteFile(path, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := resolveBackupInput("backup-2026-04-06", dir)
	if err != nil {
		t.Fatalf("resolveBackupInput returned error: %v", err)
	}
	if got.Path != path {
		t.Fatalf("expected %s, got %s", path, got.Path)
	}
}
