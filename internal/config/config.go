package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/deformal/kastql/internal/storage"
)

type Config struct {
	RegistryFile string `json:"registry_file"`
	LogLevel     string `json:"log_level"`
	Port         int    `json:"port"`
	UIPort       int    `json:"ui_port"`
}

func DefaultConfig() *Config {
	return &Config{
		RegistryFile: "kastql-registry.json",
		LogLevel:     "info",
		Port:         8080,
		UIPort:       3000,
	}
}

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

type RegistryManager struct {
	config       *Config
	registryPath string
}

func NewRegistryManager(config *Config) *RegistryManager {
	registryPath := config.RegistryFile
	if !filepath.IsAbs(registryPath) {
		registryPath = filepath.Join(".", registryPath)
	}

	return &RegistryManager{
		config:       config,
		registryPath: registryPath,
	}
}

func (rm *RegistryManager) LoadRegistry() (*storage.Registry, error) {
	registry := storage.NewRegistry()

	if _, err := os.Stat(rm.registryPath); os.IsNotExist(err) {
		return registry, nil
	}

	data, err := os.ReadFile(rm.registryPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read registry file: %w", err)
	}

	if err := registry.ImportRegistry(data); err != nil {
		return nil, fmt.Errorf("failed to import registry: %w", err)
	}

	return registry, nil
}

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

func (rm *RegistryManager) GetRegistryPath() string {
	return rm.registryPath
}
