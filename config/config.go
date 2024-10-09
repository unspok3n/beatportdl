package config

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
)

type AppConfig struct {
	DownloadsDirectory       string `yaml:"downloads_directory"`
	CreateReleaseDirectory   bool   `yaml:"create_release_directory"`
	TrackFileTemplate        string `yaml:"track_file_template"`
	ReleaseDirectoryTemplate string `yaml:"release_directory_template"`
	WhitespaceCharacter      string `yaml:"whitespace_character"`
	Proxy                    string `yaml:"proxy"`
}

func ParseConfig(filePath string) (*AppConfig, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	config := AppConfig{
		TrackFileTemplate:        "{number}. {artists} - {name} ({mix_name})",
		ReleaseDirectoryTemplate: "[{catalog_number}] {artists} - {name}",
		WhitespaceCharacter:      " ",
	}
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	return &config, nil
}
