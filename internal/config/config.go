package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Auth     AuthConfig     `yaml:"auth"`
	Services []Service      `yaml:"services"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type DatabaseConfig struct {
	MetadataPath string `yaml:"metadata_path"`
	MetricsPath  string `yaml:"metrics_path"`
}

type AuthConfig struct {
	JWTSecret   string `yaml:"jwt_secret"`
	JWTHeader   string `yaml:"jwt_header"`
	RoleClaim   string `yaml:"role_claim"`
	DefaultRole string `yaml:"default_role"`
}

type Service struct {
	Name    string            `yaml:"name"`
	URL     string            `yaml:"url"`
	Type    string            `yaml:"type"` // "federation" or "stitching"
	Headers map[string]string `yaml:"headers"`
	Enabled bool              `yaml:"enabled"`
}

func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}

	// Expand ${ENV_VAR} references so secrets can come from the environment.
	expanded := os.ExpandEnv(string(raw))

	cfg := defaults()
	if err := yaml.Unmarshal([]byte(expanded), cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	return cfg, nil
}

func defaults() *Config {
	return &Config{
		Server: ServerConfig{
			Port: 8080,
		},
		Database: DatabaseConfig{
			MetadataPath: "metadata.db",
			MetricsPath:  "metrics.db",
		},
		Auth: AuthConfig{
			JWTHeader:   "Authorization",
			RoleClaim:   "x-kastql-role",
			DefaultRole: "public",
		},
	}
}
