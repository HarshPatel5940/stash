package gittracker

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/harshpatel5940/stash/internal/security"
)

type GitRepo struct {
	Path          string   `json:"path"`
	RemoteURL     string   `json:"remote_url"`
	Branch        string   `json:"branch"`
	Dirty         bool     `json:"dirty"`
	Remotes       []string `json:"remotes"`
	Ahead         int      `json:"ahead"`          // Commits ahead of remote
	Behind        int      `json:"behind"`         // Commits behind remote
	HasUpstream   bool     `json:"has_upstream"`   // Has tracking branch configured
	UnpushedCount int      `json:"unpushed_count"` // Number of unpushed commits (alias for Ahead)
}

type GitTracker struct {
	outputDir string
	repos     []GitRepo
	seenPaths map[string]bool
	maxDepth  int
	skipDirs  map[string]bool
}

func NewGitTracker(outputDir string) *GitTracker {
	return &GitTracker{
		outputDir: outputDir,
		repos:     []GitRepo{},
		seenPaths: make(map[string]bool),
		maxDepth:  5,
		skipDirs:  defaultSkipDirs(),
	}
}

// NewGitTrackerWithConfig creates a GitTracker with custom config
func NewGitTrackerWithConfig(outputDir string, maxDepth int, skipDirs []string) *GitTracker {
	gt := &GitTracker{
		outputDir: outputDir,
		repos:     []GitRepo{},
		seenPaths: make(map[string]bool),
		maxDepth:  maxDepth,
		skipDirs:  make(map[string]bool),
	}
	for _, dir := range skipDirs {
		gt.skipDirs[dir] = true
	}
	return gt
}

func defaultSkipDirs() map[string]bool {
	return map[string]bool{
		"node_modules": true,
		".npm":         true,
		".cache":       true,
		"vendor":       true,
		"venv":         true,
		".venv":        true,
		"dist":         true,
		"build":        true,
		"Library":      true,
		"Applications": true,
	}
}

func (gt *GitTracker) ScanDirectories(searchDirs []string) error {
	for _, dir := range searchDirs {
		if err := gt.scanDir(dir, 0, gt.maxDepth); err != nil {
			continue
		}
	}
	return nil
}

func (gt *GitTracker) scanDir(dir string, depth, maxDepth int) error {
	if depth > maxDepth {
		return nil
	}

	if strings.HasPrefix(dir, "~") {
		homeDir, _ := os.UserHomeDir()
		dir = filepath.Join(homeDir, dir[1:])
	}

	absPath, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	if gt.seenPaths[absPath] {
		return nil
	}
	gt.seenPaths[absPath] = true

	entries, err := os.ReadDir(security.CleanPath(dir))
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		if gt.skipDirs[entry.Name()] {
			continue
		}

		fullPath := security.CleanPath(filepath.Join(dir, entry.Name()))

		gitPath := filepath.Join(fullPath, ".git")
		if _, err := os.Stat(gitPath); err == nil {

			repo, err := gt.extractRepoInfo(fullPath)
			if err == nil {
				gt.repos = append(gt.repos, repo)
			}

			continue
		}

		gt.scanDir(fullPath, depth+1, maxDepth)
	}

	return nil
}

func (gt *GitTracker) extractRepoInfo(repoPath string) (GitRepo, error) {
	repo := GitRepo{
		Path:    repoPath,
		Remotes: []string{},
	}

	cmd := exec.Command("git", "-C", repoPath, "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err == nil {
		repo.RemoteURL = strings.TrimSpace(string(output))
	}

	cmd = exec.Command("git", "-C", repoPath, "remote", "-v")
	output, err = cmd.Output()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		seen := make(map[string]bool)
		for _, line := range lines {
			if line == "" {
				continue
			}
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				remote := parts[0] + " " + parts[1]
				if !seen[remote] {
					repo.Remotes = append(repo.Remotes, remote)
					seen[remote] = true
				}
			}
		}
	}

	cmd = exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	output, err = cmd.Output()
	if err == nil {
		repo.Branch = strings.TrimSpace(string(output))
	}

	cmd = exec.Command("git", "-C", repoPath, "status", "--porcelain")
	output, err = cmd.Output()
	if err == nil {
		repo.Dirty = len(strings.TrimSpace(string(output))) > 0
	}

	// Check for upstream tracking and ahead/behind status
	repo.Ahead, repo.Behind, repo.HasUpstream = gt.getAheadBehind(repoPath)
	repo.UnpushedCount = repo.Ahead // Alias for convenience

	return repo, nil
}

// getAheadBehind returns the number of commits ahead and behind the upstream
func (gt *GitTracker) getAheadBehind(repoPath string) (ahead, behind int, hasUpstream bool) {
	// Check if there's an upstream branch configured
	cmd := exec.Command("git", "-C", repoPath, "rev-parse", "--abbrev-ref", "@{upstream}")
	if err := cmd.Run(); err != nil {
		// No upstream configured
		return 0, 0, false
	}

	// Get ahead/behind counts
	cmd = exec.Command("git", "-C", repoPath, "rev-list", "--left-right", "--count", "@{upstream}...HEAD")
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, true // Upstream exists but couldn't get counts
	}

	// Parse output: "behind\tahead"
	parts := strings.Fields(strings.TrimSpace(string(output)))
	if len(parts) >= 2 {
		fmt.Sscanf(parts[0], "%d", &behind)
		fmt.Sscanf(parts[1], "%d", &ahead)
	}

	return ahead, behind, true
}

func (gt *GitTracker) Save() error {
	if len(gt.repos) == 0 {
		return fmt.Errorf("no repositories found")
	}

	if err := os.MkdirAll(gt.outputDir, 0755); err != nil {
		return err
	}

	jsonFile := filepath.Join(gt.outputDir, "git-repos.json")
	data, err := json.MarshalIndent(gt.repos, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(jsonFile, data, 0644); err != nil {
		return err
	}

	textFile := filepath.Join(gt.outputDir, "git-repos.txt")
	var text strings.Builder
	text.WriteString("# Git Repositories\n")
	text.WriteString("# This is a list of git repositories found on your system\n")
	text.WriteString("# Use this to re-clone repos on a new machine\n\n")

	for _, repo := range gt.repos {
		text.WriteString(fmt.Sprintf("# Path: %s\n", repo.Path))
		text.WriteString(fmt.Sprintf("# Branch: %s", repo.Branch))

		// Add status indicators
		var statusNotes []string
		if repo.Dirty {
			statusNotes = append(statusNotes, "uncommitted changes")
		}
		if repo.UnpushedCount > 0 {
			statusNotes = append(statusNotes, fmt.Sprintf("%d unpushed commit(s)", repo.UnpushedCount))
		}
		if repo.Behind > 0 {
			statusNotes = append(statusNotes, fmt.Sprintf("%d commit(s) behind", repo.Behind))
		}
		if len(statusNotes) > 0 {
			text.WriteString(fmt.Sprintf(" (%s)", strings.Join(statusNotes, ", ")))
		}
		text.WriteString("\n")

		if repo.RemoteURL != "" {
			text.WriteString(fmt.Sprintf("git clone %s\n", repo.RemoteURL))
		} else {
			text.WriteString("# No remote URL found\n")
		}
		text.WriteString("\n")
	}

	if err := os.WriteFile(textFile, []byte(text.String()), 0644); err != nil {
		return err
	}

	scriptFile := filepath.Join(gt.outputDir, "clone-repos.sh")
	var script strings.Builder
	script.WriteString("#!/bin/bash\n")
	script.WriteString("# Git Repository Clone Script\n")
	script.WriteString("# Generated by stash\n\n")
	script.WriteString("set -e\n\n")
	script.WriteString("echo \"📁 Cloning repositories...\"\n\n")

	for _, repo := range gt.repos {
		if repo.RemoteURL != "" {

			repoName := filepath.Base(repo.Path)
			script.WriteString(fmt.Sprintf("# Original path: %s\n", repo.Path))
			script.WriteString(fmt.Sprintf("if [ ! -d \"%s\" ]; then\n", repoName))
			script.WriteString(fmt.Sprintf("  echo \"Cloning %s...\"\n", repoName))
			script.WriteString(fmt.Sprintf("  git clone %s\n", repo.RemoteURL))
			if repo.Branch != "main" && repo.Branch != "master" {
				script.WriteString(fmt.Sprintf("  cd %s && git checkout %s && cd ..\n", repoName, repo.Branch))
			}
			script.WriteString("else\n")
			script.WriteString(fmt.Sprintf("  echo \"Skipping %s (already exists)\"\n", repoName))
			script.WriteString("fi\n\n")
		}
	}

	script.WriteString("echo \"✓ Done!\"\n")

	return os.WriteFile(scriptFile, []byte(script.String()), 0755)
}

func (gt *GitTracker) GetCount() int {
	return len(gt.repos)
}

func (gt *GitTracker) GetRepos() []GitRepo {
	return gt.repos
}

// GetReposNeedingAttention returns repos with uncommitted or unpushed changes
func (gt *GitTracker) GetReposNeedingAttention() []GitRepo {
	var needsAttention []GitRepo
	for _, repo := range gt.repos {
		if repo.Dirty || repo.UnpushedCount > 0 {
			needsAttention = append(needsAttention, repo)
		}
	}
	return needsAttention
}

// NeedsAttention returns true if the repo has uncommitted or unpushed changes
func (r *GitRepo) NeedsAttention() bool {
	return r.Dirty || r.UnpushedCount > 0
}

// GetStatusSummary returns a human-readable status summary
func (r *GitRepo) GetStatusSummary() string {
	var parts []string
	if r.Dirty {
		parts = append(parts, "uncommitted changes")
	}
	if r.UnpushedCount > 0 {
		parts = append(parts, fmt.Sprintf("%d unpushed", r.UnpushedCount))
	}
	if r.Behind > 0 {
		parts = append(parts, fmt.Sprintf("%d behind", r.Behind))
	}
	if len(parts) == 0 {
		return "clean"
	}
	return strings.Join(parts, ", ")
}
