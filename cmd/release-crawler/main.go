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
	flag.Parse()

	cfg, err := config.LoadNewsSources(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx := context.Background()
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
