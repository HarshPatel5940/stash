package finder

import (
	"os"
	"path/filepath"
	"strings"
)

type DotfilesFinder struct {
	homeDir string
}

func NewDotfilesFinder() (*DotfilesFinder, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	return &DotfilesFinder{
		homeDir: homeDir,
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

		if name == ".config" || name == ".ssh" || name == ".gnupg" || name == ".aws" {
			continue
		}

		if isIgnoredDir(name) {
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
