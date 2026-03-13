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

func TestGitRepoNeedsAttention(t *testing.T) {
	tests := []struct {
		name     string
		repo     GitRepo
		expected bool
	}{
		{
			name:     "clean repo",
			repo:     GitRepo{Dirty: false, UnpushedCount: 0},
			expected: false,
		},
		{
			name:     "dirty repo",
			repo:     GitRepo{Dirty: true, UnpushedCount: 0},
			expected: true,
		},
		{
			name:     "unpushed commits",
			repo:     GitRepo{Dirty: false, UnpushedCount: 3},
			expected: true,
		},
		{
			name:     "dirty with unpushed",
			repo:     GitRepo{Dirty: true, UnpushedCount: 2},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.repo.NeedsAttention()
			if result != tt.expected {
				t.Errorf("NeedsAttention() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGitRepoGetStatusSummary(t *testing.T) {
	tests := []struct {
		name     string
		repo     GitRepo
		expected string
	}{
		{
			name:     "clean repo",
			repo:     GitRepo{Dirty: false, UnpushedCount: 0, Behind: 0},
			expected: "clean",
		},
		{
			name:     "dirty only",
			repo:     GitRepo{Dirty: true, UnpushedCount: 0, Behind: 0},
			expected: "uncommitted changes",
		},
		{
			name:     "unpushed only",
			repo:     GitRepo{Dirty: false, UnpushedCount: 3, Behind: 0},
			expected: "3 unpushed",
		},
		{
			name:     "behind only",
			repo:     GitRepo{Dirty: false, UnpushedCount: 0, Behind: 2},
			expected: "2 behind",
		},
		{
			name:     "all issues",
			repo:     GitRepo{Dirty: true, UnpushedCount: 3, Behind: 2},
			expected: "uncommitted changes, 3 unpushed, 2 behind",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.repo.GetStatusSummary()
			if result != tt.expected {
				t.Errorf("GetStatusSummary() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestGetReposNeedingAttention(t *testing.T) {
	gt := NewGitTracker("")
	gt.repos = []GitRepo{
		{Path: "/clean", Dirty: false, UnpushedCount: 0},
		{Path: "/dirty", Dirty: true, UnpushedCount: 0},
		{Path: "/unpushed", Dirty: false, UnpushedCount: 5},
		{Path: "/also-clean", Dirty: false, UnpushedCount: 0},
	}

	needsAttention := gt.GetReposNeedingAttention()

	if len(needsAttention) != 2 {
		t.Errorf("Expected 2 repos needing attention, got %d", len(needsAttention))
	}

	// Verify the correct repos are returned
	paths := make(map[string]bool)
	for _, r := range needsAttention {
		paths[r.Path] = true
	}

	if !paths["/dirty"] {
		t.Error("Expected /dirty in repos needing attention")
	}
	if !paths["/unpushed"] {
		t.Error("Expected /unpushed in repos needing attention")
	}
	if paths["/clean"] {
		t.Error("Did not expect /clean in repos needing attention")
	}
}
