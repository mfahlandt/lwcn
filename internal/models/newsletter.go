package models

import "time"

type Newsletter struct {
	Title      string     `json:"title"`
	WeekStart  time.Time  `json:"week_start"`
	WeekEnd    time.Time  `json:"week_end"`
	Content    string     `json:"content"`
	Summary    string     `json:"summary"`
	Highlights []string   `json:"highlights"`
	Releases   []Release  `json:"releases"`
	NewsItems  []NewsItem `json:"news_items"`
}

type DraftMetadata struct {
	Title      string   `yaml:"title"`
	Date       string   `yaml:"date"`
	Draft      bool     `yaml:"draft"`
	Summary    string   `yaml:"summary"`
	Highlights []string `yaml:"highlights"`
}
