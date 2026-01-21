// Package kubernetes provides Kubernetes configuration backup functionality.
// It backs up kubeconfig files, context information, namespace lists,
// and Helm release information to enable quick K8s environment recovery.
package kubernetes

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// KubernetesManager handles Kubernetes configuration backups
type KubernetesManager struct {
	outputDir string
}

// NewKubernetesManager creates a new Kubernetes manager
func NewKubernetesManager(outputDir string) *KubernetesManager {
	return &KubernetesManager{
		outputDir: outputDir,
	}
}

// BackupAll backs up all Kubernetes-related configurations
func (km *KubernetesManager) BackupAll() (int, error) {
	if err := os.MkdirAll(km.outputDir, 0755); err != nil {
		return 0, err
	}

	fileCount := 0

	// 1. Backup kubeconfig file
	if count := km.backupKubeConfig(); count > 0 {
		fileCount += count
	}

	// 2. List contexts
	if count := km.listContexts(); count > 0 {
		fileCount += count
	}

	// 3. List namespaces
	if count := km.listNamespaces(); count > 0 {
		fileCount += count
	}

	// 4. Backup Helm configuration
	if count := km.backupHelmConfig(); count > 0 {
		fileCount += count
	}

	// 5. List Helm releases
	if count := km.listHelmReleases(); count > 0 {
		fileCount += count
	}

	// Create README
	km.createReadme()

	if fileCount == 0 {
		return 0, fmt.Errorf("no Kubernetes configuration found")
	}

	return fileCount, nil
}

func (km *KubernetesManager) backupKubeConfig() int {
	homeDir := os.Getenv("HOME")
	kubeConfigPath := filepath.Join(homeDir, ".kube/config")

	// Check for KUBECONFIG environment variable
	if envKubeConfig := os.Getenv("KUBECONFIG"); envKubeConfig != "" {
		kubeConfigPath = envKubeConfig
	}

	if _, err := os.Stat(kubeConfigPath); os.IsNotExist(err) {
		return 0
	}

	data, err := os.ReadFile(kubeConfigPath)
	if err != nil {
		return 0
	}

	destPath := filepath.Join(km.outputDir, "kubeconfig")
	if err := os.WriteFile(destPath, data, 0600); err != nil { // 0600 for security
		return 0
	}

	return 1
}

func (km *KubernetesManager) listContexts() int {
	if !commandExists("kubectl") {
		return 0
	}

	output, err := exec.Command("kubectl", "config", "get-contexts", "-o", "name").Output()
	if err != nil {
		return 0
	}

	if len(output) == 0 {
		return 0
	}

	// Get current context
	currentContext, _ := exec.Command("kubectl", "config", "current-context").Output()

	var formatted strings.Builder
	formatted.WriteString("# Kubernetes Contexts\n")
	if len(currentContext) > 0 {
		formatted.WriteString(fmt.Sprintf("# Current context: %s\n", strings.TrimSpace(string(currentContext))))
	}
	formatted.WriteString("\n# All contexts:\n")
	formatted.WriteString(string(output))

	contextPath := filepath.Join(km.outputDir, "contexts.txt")
	os.WriteFile(contextPath, []byte(formatted.String()), 0644)

	return 1
}

func (km *KubernetesManager) listNamespaces() int {
	if !commandExists("kubectl") {
		return 0
	}

	output, err := exec.Command("kubectl", "get", "namespaces", "-o", "name").Output()
	if err != nil {
		return 0
	}

	if len(output) == 0 {
		return 0
	}

	var formatted strings.Builder
	formatted.WriteString("# Kubernetes Namespaces\n\n")
	formatted.WriteString(string(output))

	namespacePath := filepath.Join(km.outputDir, "namespaces.txt")
	os.WriteFile(namespacePath, []byte(formatted.String()), 0644)

	return 1
}

func (km *KubernetesManager) backupHelmConfig() int {
	homeDir := os.Getenv("HOME")

	// Helm 3 stores config in ~/.config/helm or XDG_CONFIG_HOME
	helmConfigDir := filepath.Join(homeDir, ".config/helm")
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		helmConfigDir = filepath.Join(xdgConfig, "helm")
	}

	// Also check for Helm cache/data directories
	helmCacheDir := filepath.Join(homeDir, ".cache/helm")
	if xdgCache := os.Getenv("XDG_CACHE_HOME"); xdgCache != "" {
		helmCacheDir = filepath.Join(xdgCache, "helm")
	}

	fileCount := 0

	// Backup repositories.yaml if it exists
	repoFile := filepath.Join(helmConfigDir, "repositories.yaml")
	if _, err := os.Stat(repoFile); err == nil {
		data, err := os.ReadFile(repoFile)
		if err == nil {
			destPath := filepath.Join(km.outputDir, "helm-repositories.yaml")
			if os.WriteFile(destPath, data, 0644) == nil {
				fileCount++
			}
		}
	}

	// List repositories from cache
	reposDir := filepath.Join(helmCacheDir, "repository")
	if _, err := os.Stat(reposDir); err == nil {
		entries, err := os.ReadDir(reposDir)
		if err == nil && len(entries) > 0 {
			var repoList strings.Builder
			repoList.WriteString("# Helm Repository Cache\n\n")
			for _, entry := range entries {
				if strings.HasSuffix(entry.Name(), "-index.yaml") {
					repoName := strings.TrimSuffix(entry.Name(), "-index.yaml")
					repoList.WriteString(fmt.Sprintf("%s\n", repoName))
				}
			}

			repoListPath := filepath.Join(km.outputDir, "helm-repo-cache.txt")
			os.WriteFile(repoListPath, []byte(repoList.String()), 0644)
			fileCount++
		}
	}

	return fileCount
}

func (km *KubernetesManager) listHelmReleases() int {
	if !commandExists("helm") {
		return 0
	}

	output, err := exec.Command("helm", "list", "--all-namespaces", "--output", "table").Output()
	if err != nil {
		return 0
	}

	if len(output) == 0 {
		return 0
	}

	var formatted strings.Builder
	formatted.WriteString("# Helm Releases (all namespaces)\n\n")
	formatted.WriteString(string(output))

	releasePath := filepath.Join(km.outputDir, "helm-releases.txt")
	os.WriteFile(releasePath, []byte(formatted.String()), 0644)

	return 1
}

func (km *KubernetesManager) createReadme() {
	readme := `Kubernetes Configuration Backup

This directory contains Kubernetes-related configurations:

Files:
- kubeconfig: Kubernetes cluster configuration and credentials
- contexts.txt: List of kubectl contexts
- namespaces.txt: List of Kubernetes namespaces
- helm-repositories.yaml: Helm repository configuration
- helm-repo-cache.txt: List of cached Helm repositories
- helm-releases.txt: List of Helm releases across all namespaces

To Restore:
1. Copy kubeconfig to ~/.kube/config
   chmod 600 ~/.kube/config

2. Verify contexts:
   kubectl config get-contexts

3. Switch to desired context:
   kubectl config use-context <context-name>

4. Restore Helm repositories (if using Helm):
   helm repo add <repo-name> <repo-url>
   helm repo update

5. Review helm-releases.txt for installed Helm charts

Security Note:
- kubeconfig contains cluster credentials
- Store this backup securely
- Consider using encrypted storage
- Rotate credentials if backup is compromised
`

	readmePath := filepath.Join(km.outputDir, "README.txt")
	os.WriteFile(readmePath, []byte(readme), 0644)
}

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
