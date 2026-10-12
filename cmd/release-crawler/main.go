package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/mfahlandt/lwcn/internal/config"
	"github.com/mfahlandt/lwcn/internal/models"
	"github.com/mfahlandt/lwcn/internal/news"
)

func main() {
	configPath := flag.String("config", "config/news-sources.yaml", "Path to news sources config")
	outputDir := flag.String("output", "data", "Output directory for news")
	reposPath := flag.String("repos", "config/repositories.yaml", "Tracked projects; their names count as open-source signal")
	flag.Parse()

	cfg, err := config.LoadNewsSources(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx := context.Background()
	windowStart, windowEnd, err := models.CompletedWeek(time.Now())
	if err != nil {
		log.Fatalf("Failed to determine publication window: %v", err)
	}
	var allNews []models.NewsItem

	// RSS Feeds
	log.Printf("Fetching %d RSS feeds...", len(cfg.RSSFeeds))
	rssClient := news.NewRSSClient()
	rssNews, err := rssClient.FetchAllFeeds(ctx, cfg.RSSFeeds)
	if err == nil {
		allNews = append(allNews, rssNews...)
		log.Printf("Found %d RSS items", len(rssNews))
	}

	// Scraping
	log.Printf("Scraping %d sources...", len(cfg.ScrapeSources))
	scraper := news.NewScraper()
	for _, source := range cfg.ScrapeSources {
		items, err := scraper.ScrapeHeise(ctx, source)
		if err != nil {
			log.Printf("Scraping %s failed: %v", source.Name, err)
			continue
		}
		allNews = append(allNews, items...)
		log.Printf("Found %d items from %s", len(items), source.Name)
	}

	// Hacker News
	if cfg.HackerNews.Enabled {
		log.Println("Searching Hacker News...")
		hnClient := news.NewHackerNewsClient()
		hnNews, err := hnClient.Search(ctx, cfg.HackerNews.Keywords)
		if err == nil {
			allNews = append(allNews, hnNews...)
			log.Printf("Found %d Hacker News items", len(hnNews))
		}
	}

	allNews, outsideWindow := models.FilterNewsInRange(allNews, windowStart, windowEnd)
	log.Printf("Dropped %d news items outside publication window [%s, %s]", len(outsideWindow), windowStart.Format(time.RFC3339), windowEnd.Format(time.RFC3339))
	for _, item := range outsideWindow {
		log.Printf("  - [%s] %s (published: %s)", item.Source, item.Title, item.PublishedAt.Format(time.RFC3339))
	}

	var projectNames []string
	if repos, err := config.LoadRepositories(*reposPath); err != nil {
		log.Printf("Could not load tracked projects from %s (project names won't count as open-source signal): %v", *reposPath, err)
	} else {
		for _, r := range repos.Repositories {
			projectNames = append(projectNames, r.Name)
		}
	}

	filter := news.NewNewsFilter(news.FilterOptions{
		ExcludeKeywords:   cfg.ExcludeKeywords,
		OSSKeywords:       cfg.OSSKeywords,
		ProjectNames:      projectNames,
		ProjectNameIgnore: cfg.ProjectNameIgnore,
	})
	var dropped []news.DroppedItem
	allNews, dropped = filter.Apply(allNews)
	log.Printf("Dropped %d vendor / non-open-source news items", len(dropped))
	for _, d := range dropped {
		log.Printf("  - [%s] %s (%s)", d.Item.Source, d.Item.Title, d.Reason)
	}

	log.Printf("Total: %d news items", len(allNews))

	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	filename := fmt.Sprintf("news-%s.json", time.Now().Format("2006-01-02"))
	outputPath := filepath.Join(*outputDir, filename)

	data, err := json.MarshalIndent(allNews, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal news: %v", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		log.Fatalf("Failed to write output: %v", err)
	}

	log.Printf("News saved to %s", outputPath)
}
