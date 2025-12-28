package gittracker

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestDeduplication(t *testing.T) {

	tmpDir, err := os.MkdirTemp("", "stash-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	repo1 := filepath.Join(tmpDir, "repo1")
	subdir := filepath.Join(tmpDir, "subdir")
	repo2 := filepath.Join(subdir, "repo2")

	for _, p := range []string{repo1, repo2} {
		if err := os.MkdirAll(filepath.Join(p, ".git"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	gt := NewGitTracker(tmpDir)

	searchDirs := []string{tmpDir, subdir}

	if err := gt.ScanDirectories(searchDirs); err != nil {
		t.Fatal(err)
	}

	repos := gt.GetRepos()
	if len(repos) != 2 {
		t.Errorf("Expected 2 repos, got %d", len(repos))
		for _, r := range repos {
			t.Logf("Found: %s", r.Path)
		}
	}

	seen := make(map[string]bool)
	for _, r := range repos {
		if seen[r.Path] {
			t.Errorf("Duplicate repo found: %s", r.Path)
		}
		seen[r.Path] = true
	}
}

func TestDirtyRepo(t *testing.T) {

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found")
	}

	tmpDir, err := os.MkdirTemp("", "stash-test-dirty-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	repoPath := filepath.Join(tmpDir, "dirtyrepo")
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		t.Fatal(err)
	}

	runGit := func(args ...string) error {
		cmd := exec.Command("git", append([]string{"-C", repoPath}, args...)...)
		return cmd.Run()
	}

	if err := runGit("init"); err != nil {
		t.Skip("git init failed")
	}

	runGit("config", "user.email", "test@example.com")
	runGit("config", "user.name", "Test User")

	if err := os.WriteFile(filepath.Join(repoPath, "file.txt"), []byte("initial"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit("add", ".")
	runGit("commit", "-m", "initial")

	if err := os.WriteFile(filepath.Join(repoPath, "file.txt"), []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}

	gt := NewGitTracker(tmpDir)
	if err := gt.ScanDirectories([]string{tmpDir}); err != nil {
		t.Fatal(err)
	}

	repos := gt.GetRepos()
	if len(repos) != 1 {
		t.Fatalf("Expected 1 repo, got %d", len(repos))
	}

	if !repos[0].Dirty {
		t.Error("Expected repo to be dirty")
	}
}
