package config

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
	"os/exec"
)

type AppConfig struct {
	Username      string `yaml:"username,omitempty"`
	Password      string `yaml:"password,omitempty"`
	Quality       string `yaml:"quality,omitempty"`
	WriteErrorLog bool   `yaml:"write_error_log,omitempty"`
	ShowProgress  bool   `yaml:"show_progress,omitempty"`

	MaxGlobalWorkers   int `yaml:"max_global_workers,omitempty"`
	MaxDownloadWorkers int `yaml:"max_download_workers,omitempty"`

	DownloadsDirectory string `yaml:"downloads_directory,omitempty"`
	SortByContext      bool   `yaml:"sort_by_context,omitempty"`
	SortByLabel        bool   `yaml:"sort_by_label,omitempty"`

	ReleaseDirectoryTemplate string `yaml:"release_directory_template,omitempty"`
	TrackFileTemplate        string `yaml:"track_file_template,omitempty"`
	WhitespaceCharacter      string `yaml:"whitespace_character,omitempty"`
	ArtistsLimit             int    `yaml:"artists_limit,omitempty"`
	ArtistsShortForm         string `yaml:"artists_short_form,omitempty"`
	KeySystem                string `yaml:"key_system,omitempty"`

	CoverSize string `yaml:"cover_size,omitempty"`
	KeepCover bool   `yaml:"keep_cover,omitempty"`
	FixTags   bool   `yaml:"fix_tags,omitempty"`

	Proxy string `yaml:"proxy,omitempty"`
}

const (
	DefaultCoverSize = "1400x1400"
)

var (
	SupportedKeySystems = []string{
		"openkey",
		"openkey-short",
		"camelot",
	}
)

func PermittedValue[T comparable](value T, permittedValues ...T) bool {
	for i := range permittedValues {
		if value == permittedValues[i] {
			return true
		}
	}
	return false
}

func FFMPEGInstalled() bool {
	_, err := exec.LookPath("ffmpeg")
	return err == nil
}

func Parse(filePath string) (*AppConfig, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	config := AppConfig{
		Quality:                  "lossless",
		CoverSize:                DefaultCoverSize,
		TrackFileTemplate:        "{number}. {artists} - {name} ({mix_name})",
		ReleaseDirectoryTemplate: "[{catalog_number}] {artists} - {name}",
		ArtistsLimit:             3,
		ArtistsShortForm:         "VA",
		KeySystem:                "openkey-short",
		FixTags:                  true,
		ShowProgress:             true,
		MaxGlobalWorkers:         15,
		MaxDownloadWorkers:       15,
	}
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	if config.Username == "" || config.Password == "" {
		return nil, fmt.Errorf("username or password is not provided")
	}

	if config.Quality == "medium-hls" && !FFMPEGInstalled() {
		return nil, errors.New("ffmpeg not found")
	}

	if !PermittedValue(config.KeySystem, SupportedKeySystems...) {
		return nil, fmt.Errorf("invalid key system")
	}

	if config.DownloadsDirectory == "" {
		return nil, fmt.Errorf("no downloads directory provided")
	}

	return &config, nil
}

func (c *AppConfig) Save(filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()
	encoder := yaml.NewEncoder(file)
	if err := encoder.Encode(&c); err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	return nil
}
