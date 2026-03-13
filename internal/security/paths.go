// Package security provides security utilities for the stash application.
package security

import (
	"fmt"
	"path/filepath"
	"strings"
)

// SanitizePath cleans and validates a file path to prevent path traversal attacks.
// It ensures the path doesn't escape the base directory.
func SanitizePath(basePath, userPath string) (string, error) {
	// Clean both paths
	cleanBase := filepath.Clean(basePath)
	cleanUser := filepath.Clean(userPath)

	// Join them
	fullPath := filepath.Join(cleanBase, cleanUser)
	cleanFull := filepath.Clean(fullPath)

	// Use filepath.Rel to properly enforce path-segment boundaries
	rel, err := filepath.Rel(cleanBase, cleanFull)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path traversal detected: %s escapes base %s", userPath, basePath)
	}

	return cleanFull, nil
}

// ValidatePath checks if a path is within the allowed base directory.
// Returns an error if the path attempts to traverse outside the base.
func ValidatePath(basePath, targetPath string) error {
	cleanBase := filepath.Clean(basePath)
	cleanTarget := filepath.Clean(targetPath)

	// Allow equal paths (target can be the base itself)
	if cleanBase == cleanTarget {
		return nil
	}

	// Add trailing separator to base to avoid partial directory matches
	if !strings.HasSuffix(cleanBase, string(filepath.Separator)) {
		cleanBase += string(filepath.Separator)
	}

	if !strings.HasPrefix(cleanTarget, cleanBase) {
		return fmt.Errorf("path traversal detected: %s is outside %s", targetPath, basePath)
	}

	return nil
}

// CleanPath returns a cleaned path (relative or absolute).
// For absolute path cleaning, use filepath.Abs before calling CleanPath.
func CleanPath(path string) string {
	return filepath.Clean(path)
}

// IsPathSafe checks if a filename contains any suspicious path traversal characters.
func IsPathSafe(filename string) bool {
	// Check for dangerous patterns
	dangerous := []string{
		"..",
		"./",
		"../",
		":\\", // Windows drive letters
	}

	for _, pattern := range dangerous {
		if strings.Contains(filename, pattern) {
			return false
		}
	}

	return true
}
