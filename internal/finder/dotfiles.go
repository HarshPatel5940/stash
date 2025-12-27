package finder

import (
	"os"
	"path/filepath"
	"strings"
)

// DotfilesFinder handles finding dotfiles in the home directory
type DotfilesFinder struct {
	homeDir string
}

// NewDotfilesFinder creates a new dotfiles finder
func NewDotfilesFinder() (*DotfilesFinder, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	return &DotfilesFinder{
		homeDir: homeDir,
	}, nil
}

// Find discovers all dotfiles in the home directory
func (df *DotfilesFinder) Find(additional []string) ([]string, error) {
	var dotfiles []string

	// Common dotfiles to look for
	commonDotfiles := []string{
		".zshrc",
		".bashrc",
		".bash_profile",
		".gitconfig",
		".gitignore_global",
		".vimrc",
		".tmux.conf",
		".profile",
		".zprofile",
		".inputrc",
		".curlrc",
		".wgetrc",
	}

	// Check for common dotfiles
	for _, dotfile := range commonDotfiles {
		path := filepath.Join(df.homeDir, dotfile)
		if fileExists(path) {
			dotfiles = append(dotfiles, path)
		}
	}

	// Check for additional dotfiles specified in config
	for _, dotfile := range additional {
		path := filepath.Join(df.homeDir, dotfile)
		if fileExists(path) {
			// Avoid duplicates
			if !contains(dotfiles, path) {
				dotfiles = append(dotfiles, path)
			}
		}
	}

	// Find all other dotfiles in home directory (non-recursive)
	entries, err := os.ReadDir(df.homeDir)
	if err != nil {
		return dotfiles, nil // Return what we found so far
	}

	for _, entry := range entries {
		name := entry.Name()

		// Skip if not a dotfile
		if !strings.HasPrefix(name, ".") {
			continue
		}

		// Skip special directories we handle separately
		if name == ".config" || name == ".ssh" || name == ".gnupg" || name == ".aws" {
			continue
		}

		// Skip common cache/state directories
		if isIgnoredDir(name) {
			continue
		}

		path := filepath.Join(df.homeDir, name)

		// Only add regular files, not directories
		if entry.Type().IsRegular() {
			if !contains(dotfiles, path) {
				dotfiles = append(dotfiles, path)
			}
		}
	}

	return dotfiles, nil
}

// FindConfigDir finds the ~/.config directory if it exists
func (df *DotfilesFinder) FindConfigDir() (string, bool) {
	configDir := filepath.Join(df.homeDir, ".config")
	if dirExists(configDir) {
		return configDir, true
	}
	return "", false
}

// FindSecretDirs finds SSH, GPG, and AWS directories
func (df *DotfilesFinder) FindSecretDirs() map[string]string {
	secrets := make(map[string]string)

	sshDir := filepath.Join(df.homeDir, ".ssh")
	if dirExists(sshDir) {
		secrets["ssh"] = sshDir
	}

	gpgDir := filepath.Join(df.homeDir, ".gnupg")
	if dirExists(gpgDir) {
		secrets["gpg"] = gpgDir
	}

	awsDir := filepath.Join(df.homeDir, ".aws")
	if dirExists(awsDir) {
		secrets["aws"] = awsDir
	}

	return secrets
}

func isIgnoredDir(name string) bool {
	ignored := []string{
		".cache",
		".local",
		".npm",
		".node_modules",
		".vscode",
		".Trash",
		".DS_Store",
		".docker",
		".gem",
		".cargo",
		".rustup",
		".gradle",
		".m2",
		".android",
		".minecraft",
	}

	for _, ig := range ignored {
		if name == ig {
			return true
		}
	}
	return false
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
