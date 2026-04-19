package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/mfahlandt/lwcn/internal/ai"
	"github.com/mfahlandt/lwcn/internal/models"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	releasesFile := flag.String("releases", "", "Path to releases JSON file")
	newsFile := flag.String("news", "", "Path to news JSON file")
	outputDir := flag.String("output", "website/content/newsletter", "Output directory for drafts")
	linkedinOnly := flag.Bool("linkedin", false, "Generate only LinkedIn post")
	flag.Parse()

	log.Println("Starting AI Newsletter Generator...")

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable required. Set it in .env file or environment.")
	}
	log.Println("GEMINI_API_KEY found")

	releases, err := loadReleases(*releasesFile)
	if err != nil {
		log.Fatalf("Failed to load releases: %v", err)
	}
	log.Printf("Loaded %d releases", len(releases))

	news, err := loadNews(*newsFile)
	if err != nil {
		log.Fatalf("Failed to load news: %v", err)
	}
	log.Printf("Loaded %d news items", len(news))

	// Load neutral activity stats (optional — missing file is OK)
	stats, err := loadStats("")
	if err != nil {
		log.Printf("No stats file loaded: %v", err)
	} else {
		log.Printf("Loaded %d repo stats entries", len(stats))
	}

	ctx := context.Background()

	gemini, err := ai.NewGeminiClient(ctx, apiKey)
	if err != nil {
		log.Fatalf("Failed to create Gemini client: %v", err)
	}
	defer gemini.Close()

	// Generate LinkedIn post
	if *linkedinOnly {
		generateLinkedInPosts(ctx, gemini, releases, news, *outputDir)
		return
	}

	// Generate full newsletter
	log.Println("Generating newsletter with Gemini...")
	newsletter, err := gemini.GenerateNewsletter(ctx, releases, news, stats)
	if err != nil {
		log.Fatalf("Failed to generate newsletter: %v", err)
	}

	generator := ai.NewDraftGenerator(*outputDir)
	draftPath, err := generator.GenerateDraft(newsletter)
	if err != nil {
		log.Fatalf("Failed to generate draft: %v", err)
	}

	log.Printf("Newsletter draft created: %s", draftPath)

	// Also generate LinkedIn posts (newsletter + short teaser)
	generateLinkedInPosts(ctx, gemini, releases, news, *outputDir)
}

func generateLinkedInPosts(ctx context.Context, gemini *ai.GeminiClient, releases []models.Release, news []models.NewsItem, outputDir string) {
	// Generate LinkedIn Newsletter post (longer version with website link)
	log.Println("Generating LinkedIn newsletter post with Gemini...")
	linkedinPost, err := gemini.GenerateLinkedInPost(ctx, releases, news)
	if err != nil {
		log.Printf("Warning: Failed to generate LinkedIn newsletter post: %v", err)
		return
	}

	linkedinPath := saveLinkedInFile(outputDir, linkedinPost, "linkedin")
	log.Printf("LinkedIn newsletter post created: %s", linkedinPath)
	fmt.Println("\n--- LinkedIn Newsletter Post ---")
	fmt.Println(linkedinPost)
	fmt.Println("--- End ---")

	// Generate all three short-format posts (LinkedIn teaser, tweet, bluesky)
	// in a SINGLE combined Gemini call to save free-tier quota.
	now := time.Now()
	year, week := now.ISOWeek()
	newsletterURL := fmt.Sprintf("https://lwcn.dev/newsletter/%d-week-%02d/", year, week)

	log.Println("Generating short-format posts (LinkedIn teaser + tweet + bluesky) in ONE Gemini call...")
	shorts, err := gemini.GenerateSocialShorts(ctx, linkedinPost, newsletterURL)
	if err != nil {
		log.Printf("Warning: Failed to generate social shorts: %v", err)
		return
	}

	shortPath := saveLinkedInFile(outputDir, shorts.LinkedInShort, "linkedin-short")
	log.Printf("LinkedIn short post created (%d chars): %s", len([]rune(shorts.LinkedInShort)), shortPath)

	tweetPath := saveLinkedInFile(outputDir, strings.TrimSpace(shorts.Tweet), "tweet")
	log.Printf("Tweet created (%d chars body): %s", len([]rune(shorts.Tweet)), tweetPath)

	bskyPath := saveLinkedInFile(outputDir, strings.TrimSpace(shorts.Bluesky), "bluesky")
	log.Printf("Bluesky post created (%d chars body): %s", len([]rune(shorts.Bluesky)), bskyPath)

	fmt.Println("\n--- LinkedIn Short ---")
	fmt.Println(shorts.LinkedInShort)
	fmt.Println("\n--- Tweet ---")
	fmt.Println(shorts.Tweet)
	fmt.Println("\n--- Bluesky ---")
	fmt.Println(shorts.Bluesky)
	fmt.Println("--- End ---")
}

func saveLinkedInFile(outputDir, content, suffix string) string {
	now := time.Now()
	year, week := now.ISOWeek()
	filename := fmt.Sprintf("%d-week-%02d-%s.txt", year, week, suffix)
	outputPath := filepath.Join(outputDir, filename)

	os.MkdirAll(outputDir, 0755)
	os.WriteFile(outputPath, []byte(content), 0644)

	return outputPath
}

func loadReleases(path string) ([]models.Release, error) {
	if path == "" {
		path = findLatestFile("data", "releases-")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var releases []models.Release
	if err := json.Unmarshal(data, &releases); err != nil {
		return nil, err
	}

	return releases, nil
}

func loadNews(path string) ([]models.NewsItem, error) {
	if path == "" {
		path = findLatestFile("data", "news-")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var news []models.NewsItem
	if err := json.Unmarshal(data, &news); err != nil {
		return nil, err
	}

	return news, nil
}

// loadStats loads the most recent repo activity stats file (stats-*.json).
// Missing file is not a fatal error — stats are optional context for the
// "Numbers of the Week" section.
func loadStats(path string) ([]models.RepoStats, error) {
	if path == "" {
		path = findLatestFile("data", "stats-")
	}
	if path == "" {
		return nil, fmt.Errorf("no stats file found")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var stats []models.RepoStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil, err
	}
	return stats, nil
}

func findLatestFile(dir, prefix string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	var latest string
	for _, entry := range entries {
		if !entry.IsDir() && len(entry.Name()) > len(prefix) && entry.Name()[:len(prefix)] == prefix {
			latest = filepath.Join(dir, entry.Name())
		}
	}

	return latest
}
