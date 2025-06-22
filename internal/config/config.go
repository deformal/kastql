package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/deformal/kastql/internal/storage"
)

// Config holds application configuration
type Config struct {
	RegistryFile string `json:"registry_file"`
	LogLevel     string `json:"log_level"`
	Port         int    `json:"port"`
	UIPort       int    `json:"ui_port"`
}

// DefaultConfig returns default configuration
func DefaultConfig() *Config {
	return &Config{
		RegistryFile: "kastql-registry.json",
		LogLevel:     "info",
		Port:         8080,
		UIPort:       3000,
	}
}

// LoadConfig loads configuration from file
func LoadConfig(configPath string) (*Config, error) {
	if configPath == "" {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// SaveConfig saves configuration to file
func SaveConfig(config *Config, configPath string) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// RegistryManager handles persistent registry storage
type RegistryManager struct {
	config       *Config
	registryPath string
}

// NewRegistryManager creates a new registry manager
func NewRegistryManager(config *Config) *RegistryManager {
	registryPath := config.RegistryFile
	if !filepath.IsAbs(registryPath) {
		// Make it relative to current directory
		registryPath = filepath.Join(".", registryPath)
	}

	return &RegistryManager{
		config:       config,
		registryPath: registryPath,
	}
}

// LoadRegistry loads the registry from persistent storage
func (rm *RegistryManager) LoadRegistry() (*storage.Registry, error) {
	registry := storage.NewRegistry()

	// Check if registry file exists
	if _, err := os.Stat(rm.registryPath); os.IsNotExist(err) {
		// File doesn't exist, return empty registry
		return registry, nil
	}

	// Read registry file
	data, err := os.ReadFile(rm.registryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry file: %w", err)
	}

	// Import registry data
	if err := registry.ImportRegistry(data); err != nil {
		return nil, fmt.Errorf("failed to import registry: %w", err)
	}

	return registry, nil
}

// SaveRegistry saves the registry to persistent storage
func (rm *RegistryManager) SaveRegistry(registry *storage.Registry) error {
	data, err := registry.ExportRegistry()
	if err != nil {
		return fmt.Errorf("failed to export registry: %w", err)
	}

	if err := os.WriteFile(rm.registryPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write registry file: %w", err)
	}

	return nil
}

// GetRegistryPath returns the registry file path
func (rm *RegistryManager) GetRegistryPath() string {
	return rm.registryPath
}
