package finder

import (
	"os"
	"path/filepath"
	"strings"
)

// EnvFilesFinder handles finding .env and .pem files
type EnvFilesFinder struct {
	searchPaths []string
	exclude     []string
}

// NewEnvFilesFinder creates a new env files finder
func NewEnvFilesFinder(searchPaths, exclude []string) *EnvFilesFinder {
	return &EnvFilesFinder{
		searchPaths: searchPaths,
		exclude:     exclude,
	}
}

// FindEnvFiles finds all .env files in search paths
func (ef *EnvFilesFinder) FindEnvFiles() ([]string, error) {
	var envFiles []string

	for _, searchPath := range ef.searchPaths {
		if !dirExists(searchPath) {
			continue
		}

		err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors, continue walking
			}

			// Skip if matches exclude pattern
			if ef.shouldExclude(path) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			// Check for .env files
			if !info.IsDir() && isEnvFile(info.Name()) {
				envFiles = append(envFiles, path)
			}

			return nil
		})

		if err != nil {
			return envFiles, err
		}
	}

	return envFiles, nil
}

// FindPemFiles finds all .pem files in search paths
func (ef *EnvFilesFinder) FindPemFiles() ([]string, error) {
	var pemFiles []string

	for _, searchPath := range ef.searchPaths {
		if !dirExists(searchPath) {
			continue
		}

		err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil // Skip errors, continue walking
			}

			// Skip if matches exclude pattern
			if ef.shouldExclude(path) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			// Check for .pem files
			if !info.IsDir() && strings.HasSuffix(info.Name(), ".pem") {
				pemFiles = append(pemFiles, path)
			}

			return nil
		})

		if err != nil {
			return pemFiles, err
		}
	}

	return pemFiles, nil
}

// shouldExclude checks if a path matches any exclude pattern
func (ef *EnvFilesFinder) shouldExclude(path string) bool {
	for _, pattern := range ef.exclude {
		// Simple glob matching
		matched, err := filepath.Match(pattern, path)
		if err == nil && matched {
			return true
		}

		// Also check if pattern is contained in path (for patterns like */node_modules/*)
		if strings.Contains(pattern, "*") {
			cleanPattern := strings.ReplaceAll(pattern, "*", "")
			if strings.Contains(path, cleanPattern) {
				return true
			}
		}
	}
	return false
}

// isEnvFile checks if a filename is an env file
func isEnvFile(name string) bool {
	// Match .env, .env.local, .env.production, etc.
	if name == ".env" {
		return true
	}
	if strings.HasPrefix(name, ".env.") {
		return true
	}
	// Also match files ending with .env
	if strings.HasSuffix(name, ".env") {
		return true
	}
	return false
}
