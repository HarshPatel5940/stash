package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type IncrementalConfig struct {
	Enabled            bool   `yaml:"enabled" mapstructure:"enabled"`
	FullBackupInterval string `yaml:"full_backup_interval" mapstructure:"full_backup_interval"` // e.g., "7d"
	AutoMergeThreshold int    `yaml:"auto_merge_threshold" mapstructure:"auto_merge_threshold"`
}

type CloudConfig struct {
	Enabled  bool   `yaml:"enabled" mapstructure:"enabled"`
	Provider string `yaml:"provider" mapstructure:"provider"` // "s3" for S3-compatible storage
	Bucket   string `yaml:"bucket" mapstructure:"bucket"`
	Region   string `yaml:"region" mapstructure:"region"`
	Endpoint string `yaml:"endpoint,omitempty" mapstructure:"endpoint"` // Custom endpoint for B2, R2, MinIO, etc.
	Prefix   string `yaml:"prefix,omitempty" mapstructure:"prefix"`     // Path prefix for backups
}

type Config struct {
	SearchPaths        []string           `yaml:"search_paths" mapstructure:"search_paths"`
	Exclude            []string           `yaml:"exclude" mapstructure:"exclude"`
	AdditionalDotfiles []string           `yaml:"additional_dotfiles" mapstructure:"additional_dotfiles"`
	BackupDir          string             `yaml:"backup_dir" mapstructure:"backup_dir"`
	EncryptionKey      string             `yaml:"encryption_key" mapstructure:"encryption_key"`
	Incremental        *IncrementalConfig `yaml:"incremental,omitempty" mapstructure:"incremental"`
	Cloud              *CloudConfig       `yaml:"cloud,omitempty" mapstructure:"cloud"`
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
			Enabled:            false, // Opt-in feature
			FullBackupInterval: "7d",  // Force full backup every 7 days
			AutoMergeThreshold: 5,     // Merge after 5 incrementals
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
}

func expandPath(path, homeDir string) string {
	if len(path) > 0 && path[0] == '~' {
		return filepath.Join(homeDir, path[1:])
	}
	return path
}
