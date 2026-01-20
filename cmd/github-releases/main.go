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

	"github.com/joho/godotenv"
	"github.com/mfahlandt/lwcn/internal/config"
	"github.com/mfahlandt/lwcn/internal/github"
)

func main() {
	// Load .env file
	godotenv.Load()

	configPath := flag.String("config", "config/repositories.yaml", "Path to repositories config")
	outputDir := flag.String("output", "data", "Output directory for releases")
	flag.Parse()

	// Get GitHub token from environment
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("GITHUB_TOKEN environment variable is required")
	}

	cfg, err := config.LoadRepositories(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx := context.Background()
	client := github.NewClient(token)

	log.Printf("Fetching releases from %d repositories...", len(cfg.Repositories))

	releases, err := client.FetchAllReleases(ctx, cfg.Repositories)
	if err != nil {
		log.Fatalf("Failed to fetch releases: %v", err)
	}

	log.Printf("Found %d releases from last 7 days", len(releases))

	// Group releases by category for logging
	categoryCount := make(map[string]int)
	for _, r := range releases {
		categoryCount[r.Category]++
	}
	for cat, count := range categoryCount {
		log.Printf("  - %s: %d releases", cat, count)
	}

	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	filename := fmt.Sprintf("releases-%s.json", time.Now().Format("2006-01-02"))
	outputPath := filepath.Join(*outputDir, filename)

	data, err := json.MarshalIndent(releases, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal releases: %v", err)
	}

	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		log.Fatalf("Failed to write output: %v", err)
	}

	log.Printf("Releases saved to %s", outputPath)
}
