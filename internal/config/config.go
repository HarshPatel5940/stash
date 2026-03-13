package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// IncrementalConfig controls incremental backup behavior
type IncrementalConfig struct {
	Enabled            bool   `yaml:"enabled" mapstructure:"enabled"`
	FullBackupInterval string `yaml:"full_backup_interval" mapstructure:"full_backup_interval"`
	AutoMergeThreshold int    `yaml:"auto_merge_threshold" mapstructure:"auto_merge_threshold"`
}

// CloudConfig controls cloud sync settings
type CloudConfig struct {
	Enabled  bool   `yaml:"enabled" mapstructure:"enabled"`
	Provider string `yaml:"provider" mapstructure:"provider"`
	Bucket   string `yaml:"bucket" mapstructure:"bucket"`
	Region   string `yaml:"region" mapstructure:"region"`
	Endpoint string `yaml:"endpoint,omitempty" mapstructure:"endpoint"`
	Prefix   string `yaml:"prefix,omitempty" mapstructure:"prefix"`
}

// BackupConfig controls backup retention and behavior
type BackupConfig struct {
	KeepCount   int  `yaml:"keep_count" mapstructure:"keep_count"`
	AutoCleanup bool `yaml:"auto_cleanup" mapstructure:"auto_cleanup"`
}

// DotfilesConfig controls which dotfiles are backed up
type DotfilesConfig struct {
	Additional  []string `yaml:"additional" mapstructure:"additional"`
	IgnoredDirs []string `yaml:"ignored_dirs" mapstructure:"ignored_dirs"`
}

// SecretsConfig controls which secret directories are backed up
type SecretsConfig struct {
	Directories []string `yaml:"directories" mapstructure:"directories"`
}

// ShellHistoryConfig controls shell history backup
type ShellHistoryConfig struct {
	Files []string `yaml:"files" mapstructure:"files"`
}

// GitConfig controls git repository scanning
type GitConfig struct {
	SearchDirs []string `yaml:"search_dirs" mapstructure:"search_dirs"`
	MaxDepth   int      `yaml:"max_depth" mapstructure:"max_depth"`
	SkipDirs   []string `yaml:"skip_dirs" mapstructure:"skip_dirs"`
}

// MacOSDefaultsConfig controls which macOS preferences to backup
type MacOSDefaultsConfig struct {
	Enabled bool     `yaml:"enabled" mapstructure:"enabled"`
	Domains []string `yaml:"domains" mapstructure:"domains"`
}

// BrowsersConfig controls browser data backup
type BrowsersConfig struct {
	Enabled bool     `yaml:"enabled" mapstructure:"enabled"`
	Include []string `yaml:"include,omitempty" mapstructure:"include"`
}

// RestoreConfig controls restore behavior
type RestoreConfig struct {
	UseTUI              bool `yaml:"use_tui" mapstructure:"use_tui"`
	FilePickerThreshold int  `yaml:"file_picker_threshold" mapstructure:"file_picker_threshold"`
}

// DiffConfig controls diff display
type DiffConfig struct {
	DisplayLimit int `yaml:"display_limit" mapstructure:"display_limit"`
}

// Config is the main configuration structure
type Config struct {
	// Existing fields
	SearchPaths        []string           `yaml:"search_paths" mapstructure:"search_paths"`
	Exclude            []string           `yaml:"exclude" mapstructure:"exclude"`
	AdditionalDotfiles []string           `yaml:"additional_dotfiles" mapstructure:"additional_dotfiles"`
	BackupDir          string             `yaml:"backup_dir" mapstructure:"backup_dir"`
	EncryptionKey      string             `yaml:"encryption_key" mapstructure:"encryption_key"`
	Incremental        *IncrementalConfig `yaml:"incremental,omitempty" mapstructure:"incremental"`
	Cloud              *CloudConfig       `yaml:"cloud,omitempty" mapstructure:"cloud"`

	// New configurable sections
	Backup        *BackupConfig        `yaml:"backup,omitempty" mapstructure:"backup"`
	Dotfiles      *DotfilesConfig      `yaml:"dotfiles,omitempty" mapstructure:"dotfiles"`
	Secrets       *SecretsConfig       `yaml:"secrets,omitempty" mapstructure:"secrets"`
	ShellHistory  *ShellHistoryConfig  `yaml:"shell_history,omitempty" mapstructure:"shell_history"`
	Git           *GitConfig           `yaml:"git,omitempty" mapstructure:"git"`
	MacOSDefaults *MacOSDefaultsConfig `yaml:"macos_defaults,omitempty" mapstructure:"macos_defaults"`
	Browsers      *BrowsersConfig      `yaml:"browsers,omitempty" mapstructure:"browsers"`
	Restore       *RestoreConfig       `yaml:"restore,omitempty" mapstructure:"restore"`
	Diff          *DiffConfig          `yaml:"diff,omitempty" mapstructure:"diff"`
}

func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	return &Config{
		SearchPaths: []string{
			filepath.Join(homeDir, "projects"),
			filepath.Join(homeDir, "work"),
			filepath.Join(homeDir, "Documents"),
		},
		Exclude: []string{
			"*/node_modules/*",
			"*/vendor/*",
			"*/.git/*",
			"*/.next/*",
			"*/dist/*",
			"*/build/*",
		},
		AdditionalDotfiles: []string{},
		BackupDir:          filepath.Join(homeDir, "stash-backups"),
		EncryptionKey:      filepath.Join(homeDir, ".stash.key"),
		Incremental: &IncrementalConfig{
			Enabled:            false,
			FullBackupInterval: "7d",
			AutoMergeThreshold: 5,
		},
		Backup: &BackupConfig{
			KeepCount:   5,
			AutoCleanup: true,
		},
		Dotfiles: &DotfilesConfig{
			Additional: []string{},
			IgnoredDirs: []string{
				".cache", ".local", ".npm", ".node_modules",
				".vscode", ".Trash", ".DS_Store", ".docker",
				".gem", ".cargo", ".rustup", ".gradle",
				".m2", ".android",
			},
		},
		Secrets: &SecretsConfig{
			Directories: []string{".ssh", ".gnupg", ".aws"},
		},
		ShellHistory: &ShellHistoryConfig{
			Files: []string{".zsh_history", ".bash_history", ".zhistory"},
		},
		Git: &GitConfig{
			SearchDirs: []string{
				filepath.Join(homeDir, "Documents"),
				filepath.Join(homeDir, "Projects"),
				filepath.Join(homeDir, "Code"),
				filepath.Join(homeDir, "Dev"),
				filepath.Join(homeDir, "workspace"),
				filepath.Join(homeDir, "github"),
			},
			MaxDepth: 5,
			SkipDirs: []string{
				"node_modules", ".npm", ".cache", "vendor",
				"venv", ".venv", "dist", "build",
				"Library", "Applications",
			},
		},
		MacOSDefaults: &MacOSDefaultsConfig{
			Enabled: true,
			Domains: []string{
				"com.apple.dock",
				"com.apple.finder",
				"NSGlobalDomain",
				"com.apple.HIToolbox",
				"com.apple.AppleMultitouchTrackpad",
				"com.apple.screencapture",
				"com.apple.Safari",
				"com.apple.menuextra.clock",
				"com.apple.systemuiserver",
				"com.apple.spaces",
				"com.apple.TextEdit",
				"com.apple.Terminal",
				"com.apple.ActivityMonitor",
				"com.apple.TimeMachine",
			},
		},
		Browsers: &BrowsersConfig{
			Enabled: true,
			Include: []string{}, // Empty means all supported browsers
		},
		Restore: &RestoreConfig{
			UseTUI:              true,
			FilePickerThreshold: 100,
		},
		Diff: &DiffConfig{
			DisplayLimit: 10,
		},
	}
}

func Load() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(homeDir, ".stash.yaml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	cfg := DefaultConfig()
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func (c *Config) ExpandPaths() {
	homeDir, _ := os.UserHomeDir()

	for i, path := range c.SearchPaths {
		c.SearchPaths[i] = expandPath(path, homeDir)
	}

	c.BackupDir = expandPath(c.BackupDir, homeDir)
	c.EncryptionKey = expandPath(c.EncryptionKey, homeDir)

	// Expand git search dirs
	if c.Git != nil {
		for i, path := range c.Git.SearchDirs {
			c.Git.SearchDirs[i] = expandPath(path, homeDir)
		}
	}
}

func expandPath(path, homeDir string) string {
	if len(path) > 0 && path[0] == '~' {
		return filepath.Join(homeDir, path[1:])
	}
	return path
}

// Helper methods to safely access config values

// GetBackupKeepCount returns the number of backups to keep
func (c *Config) GetBackupKeepCount() int {
	if c.Backup != nil {
		return c.Backup.KeepCount
	}
	return 5
}

// GetGitMaxDepth returns the max depth for git scanning
func (c *Config) GetGitMaxDepth() int {
	if c.Git != nil {
		return c.Git.MaxDepth
	}
	return 5
}

// GetGitSearchDirs returns directories to search for git repos
func (c *Config) GetGitSearchDirs() []string {
	if c.Git != nil && len(c.Git.SearchDirs) > 0 {
		return c.Git.SearchDirs
	}
	homeDir, _ := os.UserHomeDir()
	return []string{
		filepath.Join(homeDir, "Documents"),
		filepath.Join(homeDir, "Projects"),
		filepath.Join(homeDir, "Code"),
	}
}

// GetGitSkipDirs returns directories to skip when scanning for git repos
func (c *Config) GetGitSkipDirs() []string {
	if c.Git != nil && len(c.Git.SkipDirs) > 0 {
		return c.Git.SkipDirs
	}
	return []string{"node_modules", ".npm", ".cache", "vendor", "venv", ".venv", "dist", "build", "Library", "Applications"}
}

// GetShellHistoryFiles returns shell history files to backup
func (c *Config) GetShellHistoryFiles() []string {
	if c.ShellHistory != nil && len(c.ShellHistory.Files) > 0 {
		return c.ShellHistory.Files
	}
	return []string{".zsh_history", ".bash_history", ".zhistory"}
}

// GetSecretDirs returns secret directories to backup
func (c *Config) GetSecretDirs() []string {
	if c.Secrets != nil && len(c.Secrets.Directories) > 0 {
		return c.Secrets.Directories
	}
	return []string{".ssh", ".gnupg", ".aws"}
}

// GetDotfilesIgnoredDirs returns directories to ignore when scanning dotfiles
func (c *Config) GetDotfilesIgnoredDirs() []string {
	if c.Dotfiles != nil && len(c.Dotfiles.IgnoredDirs) > 0 {
		return c.Dotfiles.IgnoredDirs
	}
	return []string{".cache", ".local", ".npm", ".vscode", ".Trash", ".DS_Store"}
}

// GetMacOSDefaultsDomains returns macOS preference domains to backup
func (c *Config) GetMacOSDefaultsDomains() []string {
	if c.MacOSDefaults != nil && len(c.MacOSDefaults.Domains) > 0 {
		return c.MacOSDefaults.Domains
	}
	return []string{"com.apple.dock", "com.apple.finder", "NSGlobalDomain"}
}

// IsMacOSDefaultsEnabled returns whether macOS defaults backup is enabled
func (c *Config) IsMacOSDefaultsEnabled() bool {
	if c.MacOSDefaults != nil {
		return c.MacOSDefaults.Enabled
	}
	return true
}

// IsBrowsersEnabled returns whether browser data backup is enabled
func (c *Config) IsBrowsersEnabled() bool {
	if c.Browsers != nil {
		return c.Browsers.Enabled
	}
	return true
}

// GetRestoreFilePickerThreshold returns the file count threshold for TUI picker
func (c *Config) GetRestoreFilePickerThreshold() int {
	if c.Restore != nil {
		return c.Restore.FilePickerThreshold
	}
	return 100
}

// IsRestoreTUIEnabled returns whether TUI is enabled for restore
func (c *Config) IsRestoreTUIEnabled() bool {
	if c.Restore != nil {
		return c.Restore.UseTUI
	}
	return true
}

// GetDiffDisplayLimit returns the display limit for diff output
func (c *Config) GetDiffDisplayLimit() int {
	if c.Diff != nil {
		return c.Diff.DisplayLimit
	}
	return 10
}
