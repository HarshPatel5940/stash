package finder

import (
	"os"
	"path/filepath"
	"strings"
)

type EnvFilesFinder struct {
	searchPaths []string
	exclude     []string
}

func NewEnvFilesFinder(searchPaths, exclude []string) *EnvFilesFinder {
	return &EnvFilesFinder{
		searchPaths: searchPaths,
		exclude:     exclude,
	}
}

func (ef *EnvFilesFinder) FindEnvFiles() ([]string, error) {
	var envFiles []string

	for _, searchPath := range ef.searchPaths {
		if !dirExists(searchPath) {
			continue
		}

		err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if ef.shouldExclude(path) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

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

func (ef *EnvFilesFinder) FindPemFiles() ([]string, error) {
	var pemFiles []string

	for _, searchPath := range ef.searchPaths {
		if !dirExists(searchPath) {
			continue
		}

		err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			if ef.shouldExclude(path) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

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

func (ef *EnvFilesFinder) shouldExclude(path string) bool {
	for _, pattern := range ef.exclude {

		matched, err := filepath.Match(pattern, path)
		if err == nil && matched {
			return true
		}

		if strings.Contains(pattern, "*") {
			cleanPattern := strings.ReplaceAll(pattern, "*", "")
			if strings.Contains(path, cleanPattern) {
				return true
			}
		}
	}
	return false
}

func isEnvFile(name string) bool {

	if name == ".env" {
		return true
	}
	if strings.HasPrefix(name, ".env.") {
		return true
	}

	if strings.HasSuffix(name, ".env") {
		return true
	}
	return false
}
