package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type AppConfig struct {
	DownloadsDirectory string `yaml:"downloads_directory"`
	Proxy              string `yaml:"proxy"`
}

func ParseConfig(filePath string) (*AppConfig, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	var config AppConfig
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	return &config, nil
}
