package archiver

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Archiver handles creating and extracting tar.gz archives
type Archiver struct{}

// NewArchiver creates a new archiver
func NewArchiver() *Archiver {
	return &Archiver{}
}

// Create creates a tar.gz archive from a source directory
func (a *Archiver) Create(sourceDir, outputPath string) error {
	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %w", err)
	}
	defer outFile.Close()

	// Create gzip writer
	gzipWriter := gzip.NewWriter(outFile)
	defer gzipWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Walk through source directory
	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("failed to create tar header: %w", err)
		}

		// Update header name to be relative to source directory
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		header.Name = relPath

		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header: %w", err)
		}

		// If not a directory, write file content
		if !info.IsDir() {
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}
			defer file.Close()

			if _, err := io.Copy(tarWriter, file); err != nil {
				return fmt.Errorf("failed to write file to archive: %w", err)
			}
		}

		return nil
	})
}

// Extract extracts a tar.gz archive to a destination directory
func (a *Archiver) Extract(archivePath, destDir string) error {
	// Open archive file
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	// Create gzip reader
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	// Create tar reader
	tarReader := tar.NewReader(gzipReader)

	// Extract all files
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Construct target path
		target := filepath.Join(destDir, header.Name)

		// Prevent path traversal attacks
		cleanDest := filepath.Clean(destDir)
		cleanTarget := filepath.Clean(target)

		// Skip the root directory entry itself
		if header.Name == "." || header.Name == "./" {
			continue
		}

		// Ensure target is within destDir
		if !strings.HasPrefix(cleanTarget, cleanDest) {
			return fmt.Errorf("illegal file path in archive: %s", header.Name)
		}

		// Handle based on type
		switch header.Typeflag {
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}

		case tar.TypeReg:
			// Create parent directory
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			// Create file
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}

			// Copy content
			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file content: %w", err)
			}
			outFile.Close()

		default:
			// Skip other types
			continue
		}
	}

	return nil
}

// CopyFile copies a single file to an archive-ready structure
func (a *Archiver) CopyFile(src, dest string) error {
	// Create destination directory
	destDir := filepath.Dir(dest)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Get source file info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	// Create destination file
	destFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy content
	if _, err := io.Copy(destFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Preserve permissions
	if err := os.Chmod(dest, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	return nil
}

// CopyDir copies a directory recursively with smart exclusions
func (a *Archiver) CopyDir(src, dest string) error {
	return a.copyDirWithExclusions(src, dest, getConfigExclusions())
}

// copyDirWithExclusions copies a directory recursively, excluding certain patterns
func (a *Archiver) copyDirWithExclusions(src, dest string, exclusions []string) error {
	// Get source directory info
	srcInfo, err := os.Lstat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source directory: %w", err)
	}

	// Handle symlinks - don't follow them in .config
	if srcInfo.Mode()&os.ModeSymlink != 0 {
		// Skip symlinks in .config directory to avoid issues
		return nil
	}

	// Create destination directory
	if err := os.MkdirAll(dest, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Read directory entries
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	// Copy each entry
	for _, entry := range entries {
		entryName := entry.Name()

		// Skip if matches exclusion pattern
		if shouldExcludeConfigPath(entryName, exclusions) {
			continue
		}

		srcPath := filepath.Join(src, entryName)
		destPath := filepath.Join(dest, entryName)

		// Check if it's a symlink
		info, err := os.Lstat(srcPath)
		if err != nil {
			// Skip entries we can't read
			continue
		}

		// Skip symlinks
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}

		if entry.IsDir() {
			// Check for circular references or deep nesting
			if err := a.copyDirWithExclusions(srcPath, destPath, exclusions); err != nil {
				// Log error but continue with other files
				continue
			}
		} else {
			// Regular file
			if err := a.CopyFile(srcPath, destPath); err != nil {
				// Skip files that can't be copied
				continue
			}
		}
	}

	return nil
}

// getConfigExclusions returns patterns to exclude from .config backup
func getConfigExclusions() []string {
	return []string{
		"node_modules",
		"cache",
		"Cache",
		"tmp",
		"temp",
		"logs",
		"log",
		".git",
		"venv",
		".venv",
		"__pycache__",
		"*.pyc",
		".DS_Store",
		"Trash",
		"downloads",
		"Downloads",
	}
}

// shouldExcludeConfigPath checks if a path should be excluded
func shouldExcludeConfigPath(name string, exclusions []string) bool {
	for _, pattern := range exclusions {
		// Exact match
		if name == pattern {
			return true
		}
		// Prefix match for patterns like *.pyc
		if strings.HasPrefix(pattern, "*") {
			suffix := strings.TrimPrefix(pattern, "*")
			if strings.HasSuffix(name, suffix) {
				return true
			}
		}
	}
	return false
}
