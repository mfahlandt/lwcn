package models

import "time"

type NewsItem struct {
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Source      string    `json:"source"`
	Description string    `json:"description"`
	PublishedAt time.Time `json:"published_at"`
	Category    string    `json:"category"`
}

type RSSSource struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

type ScrapeSource struct {
	Name     string `yaml:"name"`
	URL      string `yaml:"url"`
	Selector string `yaml:"selector"`
}

type HackerNewsConfig struct {
	Enabled  bool     `yaml:"enabled"`
	Keywords []string `yaml:"keywords"`
}

type NewsSourceConfig struct {
	RSSFeeds      []RSSSource      `yaml:"rss_feeds"`
	ScrapeSources []ScrapeSource   `yaml:"scrape_sources"`
	HackerNews    HackerNewsConfig `yaml:"hackernews"`
}
