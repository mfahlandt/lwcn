package models

import "time"

type NewsItem struct {
	Title       string    `json:"title"`
	URL         string    `json:"url"`
	Source      string    `json:"source"`
	Description string    `json:"description"`
	PublishedAt time.Time `json:"published_at"`
	Category    string    `json:"category"`
	// Vendor marks items from vendor / cloud-provider blogs.
	Vendor     bool `json:"vendor,omitempty"`
	RequireOSS bool `json:"require_oss,omitempty"`
}

type RSSSource struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
	// Vendor feeds only pass items that carry an open-source signal.
	Vendor bool `yaml:"vendor"`
	// RequireOSS restricts broad editorial feeds without labeling them as vendors.
	RequireOSS bool `yaml:"require_oss"`
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
	// ExcludeKeywords drops items whose title mentions one of these terms,
	// unless the item carries an open-source signal.
	ExcludeKeywords []string `yaml:"exclude_keywords"`
	// OSSKeywords are terms (title or description) that mark an item as
	// open-source news.
	OSSKeywords []string `yaml:"oss_keywords"`
	// ProjectNameIgnore lists tracked project names too generic to count as
	// an open-source signal.
	ProjectNameIgnore []string `yaml:"project_name_ignore"`
}
