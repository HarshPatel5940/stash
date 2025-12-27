package finder

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewDotfilesFinder(t *testing.T) {
	finder, err := NewDotfilesFinder()
	if err != nil {
		t.Fatalf("Failed to create dotfiles finder: %v", err)
	}

	if finder == nil {
		t.Fatal("Finder should not be nil")
	}
}

func TestFindDotfiles(t *testing.T) {
	// Create temporary home directory
	tempHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	// Create test dotfiles
	testDotfiles := []string{
		".bashrc",
		".zshrc",
		".gitconfig",
		".vimrc",
	}

	for _, dotfile := range testDotfiles {
		path := filepath.Join(tempHome, dotfile)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", dotfile, err)
		}
	}

	// Create non-dotfile (should be ignored)
	if err := os.WriteFile(filepath.Join(tempHome, "regular.txt"), []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create regular file: %v", err)
	}

	finder, err := NewDotfilesFinder()
	if err != nil {
		t.Fatalf("Failed to create finder: %v", err)
	}

	dotfiles, err := finder.Find(nil)
	if err != nil {
		t.Fatalf("Failed to find dotfiles: %v", err)
	}

	// Should find at least the files we created
	if len(dotfiles) < len(testDotfiles) {
		t.Errorf("Expected at least %d dotfiles, found %d", len(testDotfiles), len(dotfiles))
	}

	// Verify our test files are in the results
	foundMap := make(map[string]bool)
	for _, f := range dotfiles {
		foundMap[filepath.Base(f)] = true
	}

	for _, expected := range testDotfiles {
		if !foundMap[expected] {
			t.Errorf("Expected to find %s but didn't", expected)
		}
	}
}

func TestFindAdditionalDotfiles(t *testing.T) {
	tempHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	// Create additional dotfiles
	additional := []string{".custom_rc", ".my_config"}
	for _, dotfile := range additional {
		path := filepath.Join(tempHome, dotfile)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", dotfile, err)
		}
	}

	finder, err := NewDotfilesFinder()
	if err != nil {
		t.Fatalf("Failed to create finder: %v", err)
	}

	dotfiles, err := finder.Find(additional)
	if err != nil {
		t.Fatalf("Failed to find dotfiles: %v", err)
	}

	// Verify additional files are included
	foundMap := make(map[string]bool)
	for _, f := range dotfiles {
		foundMap[filepath.Base(f)] = true
	}

	for _, expected := range additional {
		if !foundMap[expected] {
			t.Errorf("Expected to find additional dotfile %s", expected)
		}
	}
}

func TestFindConfigDir(t *testing.T) {
	tempHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	// Create .config directory
	configDir := filepath.Join(tempHome, ".config")
	if err := os.Mkdir(configDir, 0755); err != nil {
		t.Fatalf("Failed to create .config: %v", err)
	}

	finder, err := NewDotfilesFinder()
	if err != nil {
		t.Fatalf("Failed to create finder: %v", err)
	}

	path, found := finder.FindConfigDir()
	if !found {
		t.Error("Should find .config directory")
	}

	if path != configDir {
		t.Errorf("Expected path %s, got %s", configDir, path)
	}
}

func TestFindConfigDirNotExists(t *testing.T) {
	tempHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	finder, err := NewDotfilesFinder()
	if err != nil {
		t.Fatalf("Failed to create finder: %v", err)
	}

	_, found := finder.FindConfigDir()
	if found {
		t.Error("Should not find .config when it doesn't exist")
	}
}

func TestFindSecretDirs(t *testing.T) {
	tempHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	// Create secret directories
	secretDirs := []string{".ssh", ".gnupg", ".aws"}
	for _, dir := range secretDirs {
		path := filepath.Join(tempHome, dir)
		if err := os.Mkdir(path, 0700); err != nil {
			t.Fatalf("Failed to create %s: %v", dir, err)
		}
	}

	finder, err := NewDotfilesFinder()
	if err != nil {
		t.Fatalf("Failed to create finder: %v", err)
	}

	found := finder.FindSecretDirs()

	expectedKeys := []string{"ssh", "gpg", "aws"}
	for _, key := range expectedKeys {
		if _, exists := found[key]; !exists {
			t.Errorf("Expected to find %s directory", key)
		}
	}
}

func TestFindSecretDirsNotExists(t *testing.T) {
	tempHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	finder, err := NewDotfilesFinder()
	if err != nil {
		t.Fatalf("Failed to create finder: %v", err)
	}

	found := finder.FindSecretDirs()

	// Should return empty map when no secret dirs exist
	if len(found) != 0 {
		t.Error("Should return empty map when no secret directories exist")
	}
}

func TestEnvFilesFinder(t *testing.T) {
	tempDir := t.TempDir()

	// Create test directory structure
	projectDir := filepath.Join(tempDir, "projects", "myapp")
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	// Create .env files
	envFiles := []string{
		filepath.Join(projectDir, ".env"),
		filepath.Join(projectDir, ".env.local"),
		filepath.Join(projectDir, "subdir", ".env.production"),
	}

	for _, envFile := range envFiles {
		if err := os.MkdirAll(filepath.Dir(envFile), 0755); err != nil {
			t.Fatalf("Failed to create dir: %v", err)
		}
		if err := os.WriteFile(envFile, []byte("KEY=value"), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", envFile, err)
		}
	}

	searchPaths := []string{tempDir}
	exclude := []string{}

	finder := NewEnvFilesFinder(searchPaths, exclude)
	found, err := finder.FindEnvFiles()
	if err != nil {
		t.Fatalf("Failed to find env files: %v", err)
	}

	if len(found) != len(envFiles) {
		t.Errorf("Expected %d env files, found %d", len(envFiles), len(found))
	}
}

func TestEnvFilesFinderWithExclusions(t *testing.T) {
	tempDir := t.TempDir()

	// Create test structure with node_modules
	projectDir := filepath.Join(tempDir, "project")
	nodeModules := filepath.Join(projectDir, "node_modules", "package")

	if err := os.MkdirAll(nodeModules, 0755); err != nil {
		t.Fatalf("Failed to create dirs: %v", err)
	}

	// Create .env in root (should be found)
	rootEnv := filepath.Join(projectDir, ".env")
	if err := os.WriteFile(rootEnv, []byte("ROOT=1"), 0644); err != nil {
		t.Fatalf("Failed to create root .env: %v", err)
	}

	// Create .env in node_modules (should be excluded)
	nmEnv := filepath.Join(nodeModules, ".env")
	if err := os.WriteFile(nmEnv, []byte("NM=1"), 0644); err != nil {
		t.Fatalf("Failed to create node_modules .env: %v", err)
	}

	searchPaths := []string{tempDir}
	exclude := []string{"*/node_modules/*"}

	finder := NewEnvFilesFinder(searchPaths, exclude)
	found, err := finder.FindEnvFiles()
	if err != nil {
		t.Fatalf("Failed to find env files: %v", err)
	}

	// Should only find root .env, not the one in node_modules
	if len(found) != 1 {
		t.Errorf("Expected 1 env file (excluding node_modules), found %d", len(found))
	}

	if len(found) > 0 && found[0] != rootEnv {
		t.Error("Should find root .env file")
	}
}

func TestFindPemFiles(t *testing.T) {
	tempDir := t.TempDir()

	// Create test .pem files
	pemFiles := []string{
		filepath.Join(tempDir, "cert.pem"),
		filepath.Join(tempDir, "keys", "private.pem"),
		filepath.Join(tempDir, "ssl", "server.pem"),
	}

	for _, pemFile := range pemFiles {
		if err := os.MkdirAll(filepath.Dir(pemFile), 0755); err != nil {
			t.Fatalf("Failed to create dir: %v", err)
		}
		if err := os.WriteFile(pemFile, []byte("-----BEGIN CERTIFICATE-----"), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", pemFile, err)
		}
	}

	searchPaths := []string{tempDir}
	exclude := []string{}

	finder := NewEnvFilesFinder(searchPaths, exclude)
	found, err := finder.FindPemFiles()
	if err != nil {
		t.Fatalf("Failed to find pem files: %v", err)
	}

	if len(found) != len(pemFiles) {
		t.Errorf("Expected %d pem files, found %d", len(pemFiles), len(found))
	}
}

func TestFindPemFilesWithExclusions(t *testing.T) {
	tempDir := t.TempDir()

	// Create .pem in root
	rootPem := filepath.Join(tempDir, "root.pem")
	if err := os.WriteFile(rootPem, []byte("CERT"), 0644); err != nil {
		t.Fatalf("Failed to create root pem: %v", err)
	}

	// Create .pem in excluded directory
	vendorDir := filepath.Join(tempDir, "vendor", "lib")
	if err := os.MkdirAll(vendorDir, 0755); err != nil {
		t.Fatalf("Failed to create vendor dir: %v", err)
	}

	vendorPem := filepath.Join(vendorDir, "vendor.pem")
	if err := os.WriteFile(vendorPem, []byte("CERT"), 0644); err != nil {
		t.Fatalf("Failed to create vendor pem: %v", err)
	}

	searchPaths := []string{tempDir}
	exclude := []string{"*/vendor/*"}

	finder := NewEnvFilesFinder(searchPaths, exclude)
	found, err := finder.FindPemFiles()
	if err != nil {
		t.Fatalf("Failed to find pem files: %v", err)
	}

	if len(found) != 1 {
		t.Errorf("Expected 1 pem file (excluding vendor), found %d", len(found))
	}

	if len(found) > 0 && found[0] != rootPem {
		t.Error("Should find root pem file only")
	}
}

func TestEmptySearchPaths(t *testing.T) {
	searchPaths := []string{}
	exclude := []string{}

	finder := NewEnvFilesFinder(searchPaths, exclude)
	found, err := finder.FindEnvFiles()
	if err != nil {
		t.Fatalf("Should not error with empty search paths: %v", err)
	}

	if len(found) != 0 {
		t.Error("Should find no files with empty search paths")
	}
}

func TestNonexistentSearchPath(t *testing.T) {
	tempDir := t.TempDir()
	nonexistent := filepath.Join(tempDir, "nonexistent")

	searchPaths := []string{nonexistent}
	exclude := []string{}

	finder := NewEnvFilesFinder(searchPaths, exclude)
	found, err := finder.FindEnvFiles()

	// Should handle gracefully (either error or empty result)
	if err == nil && len(found) != 0 {
		t.Error("Should return empty results for nonexistent path")
	}
}
