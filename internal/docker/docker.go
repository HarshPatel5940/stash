// Package docker provides Docker configuration backup functionality.
// It backs up Docker daemon configurations, running container info,
// images list, Docker Compose files, and Docker contexts.
//
// This enables quick recovery of Docker environments on new machines.
package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DockerManager handles Docker configuration backups
type DockerManager struct {
	outputDir   string
	searchPaths []string
}

// NewDockerManager creates a new Docker manager
func NewDockerManager(outputDir string, searchPaths []string) *DockerManager {
	return &DockerManager{
		outputDir:   outputDir,
		searchPaths: searchPaths,
	}
}

// BackupAll backs up all Docker-related configurations
func (dm *DockerManager) BackupAll() (int, error) {
	if err := os.MkdirAll(dm.outputDir, 0755); err != nil {
		return 0, err
	}

	fileCount := 0

	// 1. Backup Docker daemon.json if it exists
	if count := dm.backupDaemonConfig(); count > 0 {
		fileCount += count
	}

	// 2. Backup Docker config.json (credentials, etc.)
	if count := dm.backupDockerConfig(); count > 0 {
		fileCount += count
	}

	// 3. Find and list all docker-compose files
	if count := dm.findComposeFiles(); count > 0 {
		fileCount += count
	}

	// 4. List running containers
	if count := dm.listContainers(); count > 0 {
		fileCount += count
	}

	// 5. List Docker images
	if count := dm.listImages(); count > 0 {
		fileCount += count
	}

	// 6. Export Docker contexts
	if count := dm.exportContexts(); count > 0 {
		fileCount += count
	}

	// Create README
	dm.createReadme()

	if fileCount == 0 {
		return 0, fmt.Errorf("no Docker configuration found")
	}

	return fileCount, nil
}

func (dm *DockerManager) backupDaemonConfig() int {
	// macOS locations for daemon.json
	possiblePaths := []string{
		filepath.Join(os.Getenv("HOME"), ".docker/daemon.json"),
		"/etc/docker/daemon.json",
	}

	for _, daemonPath := range possiblePaths {
		if _, err := os.Stat(daemonPath); err == nil {
			data, err := os.ReadFile(daemonPath)
			if err != nil {
				continue
			}

			destPath := filepath.Join(dm.outputDir, "daemon.json")
			if err := os.WriteFile(destPath, data, 0644); err != nil {
				continue
			}
			return 1
		}
	}

	return 0
}

func (dm *DockerManager) backupDockerConfig() int {
	homeDir := os.Getenv("HOME")
	configPath := filepath.Join(homeDir, ".docker/config.json")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return 0
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return 0
	}

	destPath := filepath.Join(dm.outputDir, "config.json")
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return 0
	}

	return 1
}

func (dm *DockerManager) findComposeFiles() int {
	var composePaths []string

	// Common docker-compose file names
	composeFileNames := []string{
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
	}

	// Search in configured paths
	for _, searchPath := range dm.searchPaths {
		// Expand home directory
		if strings.HasPrefix(searchPath, "~") {
			homeDir := os.Getenv("HOME")
			searchPath = filepath.Join(homeDir, searchPath[1:])
		}

		filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}

			// Skip hidden directories and common exclusions
			if info.IsDir() {
				name := info.Name()
				if strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" {
					return filepath.SkipDir
				}
				return nil
			}

			// Check if file matches docker-compose naming
			for _, composeName := range composeFileNames {
				if info.Name() == composeName {
					composePaths = append(composePaths, path)
					break
				}
			}

			return nil
		})
	}

	if len(composePaths) == 0 {
		return 0
	}

	// Write list of compose files
	var output strings.Builder
	output.WriteString("# Docker Compose Files Found\n")
	output.WriteString(fmt.Sprintf("# Total: %d files\n\n", len(composePaths)))

	for _, path := range composePaths {
		output.WriteString(fmt.Sprintf("%s\n", path))
	}

	listPath := filepath.Join(dm.outputDir, "docker-compose-files.txt")
	os.WriteFile(listPath, []byte(output.String()), 0644)

	return 1
}

func (dm *DockerManager) listContainers() int {
	if !commandExists("docker") {
		return 0
	}

	// List all containers (running and stopped)
	output, err := exec.Command("docker", "ps", "-a", "--format", "{{.Names}}\t{{.Image}}\t{{.Status}}").Output()
	if err != nil {
		return 0
	}

	if len(output) == 0 {
		return 0
	}

	var formatted strings.Builder
	formatted.WriteString("# Docker Containers\n")
	formatted.WriteString("# Format: Name\tImage\tStatus\n\n")
	formatted.WriteString(string(output))

	containerPath := filepath.Join(dm.outputDir, "containers.txt")
	os.WriteFile(containerPath, []byte(formatted.String()), 0644)

	return 1
}

func (dm *DockerManager) listImages() int {
	if !commandExists("docker") {
		return 0
	}

	output, err := exec.Command("docker", "images", "--format", "{{.Repository}}:{{.Tag}}\t{{.Size}}").Output()
	if err != nil {
		return 0
	}

	if len(output) == 0 {
		return 0
	}

	var formatted strings.Builder
	formatted.WriteString("# Docker Images\n")
	formatted.WriteString("# Format: Image:Tag\tSize\n\n")
	formatted.WriteString(string(output))

	imagePath := filepath.Join(dm.outputDir, "images.txt")
	os.WriteFile(imagePath, []byte(formatted.String()), 0644)

	return 1
}

func (dm *DockerManager) exportContexts() int {
	if !commandExists("docker") {
		return 0
	}

	output, err := exec.Command("docker", "context", "ls", "--format", "{{.Name}}\t{{.DockerEndpoint}}").Output()
	if err != nil {
		return 0
	}

	if len(output) == 0 {
		return 0
	}

	var formatted strings.Builder
	formatted.WriteString("# Docker Contexts\n")
	formatted.WriteString("# Format: Name\tEndpoint\n\n")
	formatted.WriteString(string(output))

	contextPath := filepath.Join(dm.outputDir, "contexts.txt")
	os.WriteFile(contextPath, []byte(formatted.String()), 0644)

	return 1
}

func (dm *DockerManager) createReadme() {
	readme := `Docker Configuration Backup

This directory contains Docker-related configurations:

Files:
- daemon.json: Docker daemon configuration
- config.json: Docker CLI configuration (may contain registry credentials)
- docker-compose-files.txt: List of all docker-compose files found
- containers.txt: List of Docker containers (running and stopped)
- images.txt: List of Docker images
- contexts.txt: Docker contexts configuration

To Restore:
1. Copy daemon.json to ~/.docker/daemon.json (if needed)
2. Copy config.json to ~/.docker/config.json (if needed)
3. Review docker-compose-files.txt to see where your compose files were located
4. Review containers.txt and images.txt for reference
5. Pull Docker images: docker pull <image>
6. Recreate containers from docker-compose files

Note: This backup includes configuration and references only.
Container data and volumes are not included.
Consider backing up Docker volumes separately if needed.
`

	readmePath := filepath.Join(dm.outputDir, "README.txt")
	os.WriteFile(readmePath, []byte(readme), 0644)
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
