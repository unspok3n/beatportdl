package config

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
)

type AppConfig struct {
	Username                 string `yaml:"username"`
	Password                 string `yaml:"password"`
	DownloadsDirectory       string `yaml:"downloads_directory"`
	CreateReleaseDirectory   bool   `yaml:"create_release_directory"`
	CoverSize                string `yaml:"cover_size"`
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

	if config.Username == "" || config.Password == "" {
		return nil, fmt.Errorf("username or password is not provided")
	}

	return &config, nil
}
