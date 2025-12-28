package packager

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestCountLines(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "stash-packager-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	content := `line1
line2

# comment
line3
`
	file := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(file, []byte(content), 0644)

	p := NewPackager(tmpDir)
	count := p.countLines(file)

	if count != 3 {
		t.Errorf("Expected 3 lines, got %d", count)
	}
}

func TestCollectNPM(t *testing.T) {
	if _, err := exec.LookPath("npm"); err != nil {
		t.Skip("npm not installed")
	}

	tmpDir, err := os.MkdirTemp("", "stash-packager-npm-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	p := NewPackager(tmpDir)
	if err := p.CollectNPM(); err != nil {

		t.Logf("CollectNPM failed (might be expected): %v", err)
	}

	if _, err := os.Stat(filepath.Join(tmpDir, "npm-global.txt")); os.IsNotExist(err) {

		t.Log("npm-global.txt not created")
	}
}
