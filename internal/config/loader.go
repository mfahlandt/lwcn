package config

import (
	"os"

	"github.com/mfahlandt/lwcn/internal/models"
	"gopkg.in/yaml.v3"
)

func LoadRepositories(path string) (*models.RepositoryConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config models.RepositoryConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func LoadNewsSources(path string) (*models.NewsSourceConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config models.NewsSourceConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
