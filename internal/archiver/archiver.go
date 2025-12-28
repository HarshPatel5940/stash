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

type Archiver struct {
	CompressionLevel int
}

func NewArchiver() *Archiver {
	return &Archiver{
		CompressionLevel: gzip.BestCompression,
	}
}

func (a *Archiver) Create(sourceDir, outputPath string) error {
	exclusions := getConfigExclusions()

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %w", err)
	}
	defer outFile.Close()

	gzipWriter, err := gzip.NewWriterLevel(outFile, a.CompressionLevel)
	if err != nil {
		return fmt.Errorf("failed to create gzip writer: %w", err)
	}
	defer gzipWriter.Close()

	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if shouldExcludeConfigPath(info.Name(), exclusions) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("failed to create tar header: %w", err)
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tarWriter.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header: %w", err)
		}

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

func (a *Archiver) Extract(archivePath, destDir string) error {

	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		target := filepath.Join(destDir, header.Name)

		cleanDest := filepath.Clean(destDir)
		cleanTarget := filepath.Clean(target)

		if header.Name == "." || header.Name == "./" {
			continue
		}

		if !strings.HasPrefix(cleanTarget, cleanDest) {
			return fmt.Errorf("illegal file path in archive: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:

			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}

		case tar.TypeReg:

			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("failed to write file content: %w", err)
			}
			outFile.Close()

		default:

			continue
		}
	}

	return nil
}

func (a *Archiver) CopyFile(src, dest string) error {

	destDir := filepath.Dir(dest)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	if err := os.Chmod(dest, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	return nil
}

func (a *Archiver) CopyDir(src, dest string) error {
	return a.copyDirWithExclusions(src, dest, getConfigExclusions())
}

func (a *Archiver) copyDirWithExclusions(src, dest string, exclusions []string) error {

	srcInfo, err := os.Lstat(src)
	if err != nil {
		return fmt.Errorf("failed to stat source directory: %w", err)
	}

	if srcInfo.Mode()&os.ModeSymlink != 0 {

		return nil
	}

	if err := os.MkdirAll(dest, srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		entryName := entry.Name()

		if shouldExcludeConfigPath(entryName, exclusions) {
			continue
		}

		srcPath := filepath.Join(src, entryName)
		destPath := filepath.Join(dest, entryName)

		info, err := os.Lstat(srcPath)
		if err != nil {

			continue
		}

		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}

		if entry.IsDir() {

			if err := a.copyDirWithExclusions(srcPath, destPath, exclusions); err != nil {

				continue
			}
		} else {

			if err := a.CopyFile(srcPath, destPath); err != nil {

				continue
			}
		}
	}

	return nil
}

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

func shouldExcludeConfigPath(name string, exclusions []string) bool {
	for _, pattern := range exclusions {

		if name == pattern {
			return true
		}

		if strings.HasPrefix(pattern, "*") {
			suffix := strings.TrimPrefix(pattern, "*")
			if strings.HasSuffix(name, suffix) {
				return true
			}
		}
	}
	return false
}
