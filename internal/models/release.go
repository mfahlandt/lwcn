package models

import "time"

type Release struct {
	RepoOwner    string    `json:"repo_owner"`
	RepoName     string    `json:"repo_name"`
	TagName      string    `json:"tag_name"`
	Name         string    `json:"name"`
	Body         string    `json:"body"`
	URL          string    `json:"url"`
	PublishedAt  time.Time `json:"published_at"`
	Category     string    `json:"category"`
	IsPrerelease bool      `json:"is_prerelease,omitempty"`
}

type Repository struct {
	Owner    string `yaml:"owner"`
	Repo     string `yaml:"repo"`
	Name     string `yaml:"name"`
	Category string `yaml:"category"`
}

type RepositoryConfig struct {
	Repositories []Repository `yaml:"repositories"`
}
