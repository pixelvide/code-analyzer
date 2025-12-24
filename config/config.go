package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// AppConfig represents the application configuration
// AppConfig represents the application configuration
type AppConfig struct {
	Dir          string                    `yaml:"dir"`
	Output       string                    `yaml:"output"`
	GitLabReport string                    `yaml:"gitlab_report"`
	Analyzers    map[string]AnalyzerConfig `yaml:"analyzers"`
}

// AnalyzerConfig represents configuration for a specific analyzer
type AnalyzerConfig struct {
	Enabled  bool     `yaml:"enabled"`
	TopN     int      `yaml:"top"`
	Min      int      `yaml:"min"`
	MinRatio float64  `yaml:"min_ratio"`
	Sort     string   `yaml:"sort"`
	Exclude  []string `yaml:"exclude"`
}

// LoadConfig loads configuration from a YAML file
func LoadConfig(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := &AppConfig{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}

	return config, nil
}
