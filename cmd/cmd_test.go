package cmd

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCmd(t *testing.T) {

	tmpHome, err := os.MkdirTemp("", "stash-test-home-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpHome)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", oldHome)

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.SetArgs([]string{"init"})
	err = rootCmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("Init command failed: %v", err)
	}

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Created config") {
		t.Error("Expected 'Created config' in output")
	}
	if !strings.Contains(output, "Generated encryption key") {
		t.Error("Expected 'Generated encryption key' in output")
	}

	configPath := filepath.Join(tmpHome, ".stash.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error(".stash.yaml was not created")
	}

	keyPath := filepath.Join(tmpHome, ".stash.key")
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		t.Error(".stash.key was not created")
	}

	r, w, _ = os.Pipe()
	os.Stdout = w
	rootCmd.SetArgs([]string{"init"})
	err = rootCmd.Execute()
	w.Close()
	os.Stdout = oldStdout

	var buf2 bytes.Buffer
	io.Copy(&buf2, r)
	output2 := buf2.String()

	if !strings.Contains(output2, "Config already exists") {
		t.Error("Expected 'Config already exists' in second run")
	}
}

func TestBackupCmd(t *testing.T) {

	tmpHome, err := os.MkdirTemp("", "stash-test-backup-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpHome)

	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", oldHome)

	rootCmd.SetArgs([]string{"init"})
	rootCmd.SetOut(io.Discard)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	os.WriteFile(filepath.Join(tmpHome, ".zshrc"), []byte("alias ls='ls -G'"), 0644)
	os.Mkdir(filepath.Join(tmpHome, ".ssh"), 0700)
	os.WriteFile(filepath.Join(tmpHome, ".ssh", "id_rsa"), []byte("fake-key"), 0600)

	backupDir := filepath.Join(tmpHome, "stash-backups")

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	rootCmd.SetArgs([]string{"backup", "--no-encrypt", "--verbose", "--output", backupDir})
	err = rootCmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("Backup command failed: %v", err)
	}

	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Backup completed successfully") {
		t.Errorf("Backup failed, output:\n%s", output)
	}

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("Failed to read backup dir: %v", err)
	}

	found := false
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".tar.gz") {
			found = true
			break
		}
	}

	if !found {
		t.Error("No .tar.gz backup file found")
	}
}
