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

	ctx := context.Background()

	gemini, err := ai.NewGeminiClient(ctx, apiKey)
	if err != nil {
		log.Fatalf("Failed to create Gemini client: %v", err)
	}
	defer gemini.Close()

	// Generate LinkedIn post
	if *linkedinOnly {
		log.Println("Generating LinkedIn post with Gemini...")
		linkedinPost, err := gemini.GenerateLinkedInPost(ctx, releases, news)
		if err != nil {
			log.Fatalf("Failed to generate LinkedIn post: %v", err)
		}

		linkedinPath := saveLinkedInPost(*outputDir, linkedinPost)
		log.Printf("LinkedIn post created: %s", linkedinPath)
		fmt.Println("\n--- LinkedIn Post ---")
		fmt.Println(linkedinPost)
		fmt.Println("--- End ---")
		return
	}

	// Generate full newsletter
	log.Println("Generating newsletter with Gemini...")
	newsletter, err := gemini.GenerateNewsletter(ctx, releases, news)
	if err != nil {
		log.Fatalf("Failed to generate newsletter: %v", err)
	}

	generator := ai.NewDraftGenerator(*outputDir)
	draftPath, err := generator.GenerateDraft(newsletter)
	if err != nil {
		log.Fatalf("Failed to generate draft: %v", err)
	}

	log.Printf("Newsletter draft created: %s", draftPath)

	// Also generate LinkedIn post
	log.Println("Generating LinkedIn post with Gemini...")
	linkedinPost, err := gemini.GenerateLinkedInPost(ctx, releases, news)
	if err != nil {
		log.Printf("Warning: Failed to generate LinkedIn post: %v", err)
	} else {
		linkedinPath := saveLinkedInPost(*outputDir, linkedinPost)
		log.Printf("LinkedIn post created: %s", linkedinPath)
	}
}

func saveLinkedInPost(outputDir, content string) string {
	now := time.Now()
	year, week := now.ISOWeek()
	filename := fmt.Sprintf("%d-week-%02d-linkedin.txt", year, week)
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
