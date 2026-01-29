package finder

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/harshpatel5940/stash/internal/config"
)

type DotfilesFinder struct {
	homeDir    string
	cfg        *config.Config
	ignoredMap map[string]bool
}

func NewDotfilesFinder() (*DotfilesFinder, error) {
	return NewDotfilesFinderWithConfig(nil)
}

func NewDotfilesFinderWithConfig(cfg *config.Config) (*DotfilesFinder, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	if cfg == nil {
		cfg, _ = config.Load()
		if cfg == nil {
			cfg = config.DefaultConfig()
		}
	}

	// Build ignored dirs map
	ignoredDirs := cfg.GetDotfilesIgnoredDirs()
	ignoredMap := make(map[string]bool, len(ignoredDirs))
	for _, dir := range ignoredDirs {
		ignoredMap[dir] = true
	}

	return &DotfilesFinder{
		homeDir:    homeDir,
		cfg:        cfg,
		ignoredMap: ignoredMap,
	}, nil
}

func (df *DotfilesFinder) Find(additional []string) ([]string, error) {
	var dotfiles []string

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

	for _, dotfile := range commonDotfiles {
		path := filepath.Join(df.homeDir, dotfile)
		if fileExists(path) {
			dotfiles = append(dotfiles, path)
		}
	}

	for _, dotfile := range additional {
		path := filepath.Join(df.homeDir, dotfile)
		if fileExists(path) {

			if !contains(dotfiles, path) {
				dotfiles = append(dotfiles, path)
			}
		}
	}

	entries, err := os.ReadDir(df.homeDir)
	if err != nil {
		return dotfiles, nil
	}

	for _, entry := range entries {
		name := entry.Name()

		if !strings.HasPrefix(name, ".") {
			continue
		}

		// Skip .config and secret directories (they're handled separately)
		if name == ".config" {
			continue
		}

		// Skip secret directories
		secretDirs := df.cfg.GetSecretDirs()
		isSecret := false
		for _, secretDir := range secretDirs {
			if name == secretDir {
				isSecret = true
				break
			}
		}
		if isSecret {
			continue
		}

		if df.isIgnoredDir(name) {
			continue
		}

		path := filepath.Join(df.homeDir, name)

		if entry.Type().IsRegular() {
			if !contains(dotfiles, path) {
				dotfiles = append(dotfiles, path)
			}
		}
	}

	return dotfiles, nil
}

func (df *DotfilesFinder) FindConfigDir() (string, bool) {
	configDir := filepath.Join(df.homeDir, ".config")
	if dirExists(configDir) {
		return configDir, true
	}
	return "", false
}

func (df *DotfilesFinder) FindSecretDirs() map[string]string {
	secrets := make(map[string]string)

	secretDirs := df.cfg.GetSecretDirs()

	for _, dirName := range secretDirs {
		dirPath := filepath.Join(df.homeDir, dirName)
		if dirExists(dirPath) {
			// Use the directory name without the leading dot as the key
			key := strings.TrimPrefix(dirName, ".")
			// Maintain backward compatibility: .gnupg -> gpg
			if key == "gnupg" {
				key = "gpg"
			}
			secrets[key] = dirPath
		}
	}

	return secrets
}

func (df *DotfilesFinder) isIgnoredDir(name string) bool {
	return df.ignoredMap[name]
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
