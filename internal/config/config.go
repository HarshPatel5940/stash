package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type Config struct {
	SearchPaths        []string `yaml:"search_paths" mapstructure:"search_paths"`
	Exclude            []string `yaml:"exclude" mapstructure:"exclude"`
	AdditionalDotfiles []string `yaml:"additional_dotfiles" mapstructure:"additional_dotfiles"`
	BackupDir          string   `yaml:"backup_dir" mapstructure:"backup_dir"`
	EncryptionKey      string   `yaml:"encryption_key" mapstructure:"encryption_key"`
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
	}
}

func Load() (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(homeDir, ".stash.yaml")

	// If config doesn't exist, return default
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
