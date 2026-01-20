package main

// Backfill tool for generating newsletters for past weeks.
// Use this to regenerate historical newsletters.
// Run: go run ./cmd/backfill-newsletter -weeks 3

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
	"unicode/utf8"

	"github.com/joho/godotenv"
	"github.com/mfahlandt/lwcn/internal/ai"
	"github.com/mfahlandt/lwcn/internal/config"
	"github.com/mfahlandt/lwcn/internal/github"
	"github.com/mfahlandt/lwcn/internal/models"
	"gopkg.in/yaml.v3"
)

func main() {
	godotenv.Load()

	weeks := flag.Int("weeks", 3, "Number of past weeks to generate (1-10)")
	specificWeek := flag.Int("week", 0, "Generate a specific week number (overrides -weeks)")
	configPath := flag.String("config", "config/repositories.yaml", "Path to repositories config")
	outputDir := flag.String("output", "website/content/newsletter", "Output directory for newsletters")
	dataDir := flag.String("data", "data", "Output directory for data files")
	skipCrawl := flag.Bool("skip-crawl", false, "Skip crawling, use existing data files")
	flag.Parse()

	if *weeks < 1 || *weeks > 10 {
		log.Fatal("Weeks must be between 1 and 10")
	}

	githubToken := os.Getenv("GITHUB_TOKEN")
	geminiKey := os.Getenv("GEMINI_API_KEY")

	if githubToken == "" && !*skipCrawl {
		log.Fatal("GITHUB_TOKEN environment variable required for crawling")
	}
	if geminiKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable required")
	}

	ctx := context.Background()

	// Load repository config
	cfg, err := config.LoadRepositories(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Create Gemini client
	gemini, err := ai.NewGeminiClient(ctx, geminiKey)
	if err != nil {
		log.Fatalf("Failed to create Gemini client: %v", err)
	}
	defer gemini.Close()

	// Create GitHub client
	var ghClient *github.Client
	if !*skipCrawl {
		ghClient = github.NewClient(githubToken)
	}

	// Calculate week ranges
	// We generate for past completed weeks
	now := time.Now()
	currentYear, currentWeek := now.ISOWeek()

	// Determine which weeks to generate
	var weeksToGenerate []int
	if *specificWeek > 0 {
		// Generate a specific week
		weeksToGenerate = append(weeksToGenerate, *specificWeek)
	} else {
		// Generate past N weeks
		for i := *weeks; i >= 1; i-- {
			weekNum := currentWeek - i
			if weekNum > 0 {
				weeksToGenerate = append(weeksToGenerate, weekNum)
			}
		}
	}

	for _, targetWeek := range weeksToGenerate {
		// Calculate the Monday of the target week
		weekStart := getWeekStartForWeek(currentYear, targetWeek)
		weekEnd := weekStart.AddDate(0, 0, 7).Add(-time.Second) // Sunday 23:59:59

		year, week := weekStart.ISOWeek()

		log.Printf("\n" + strings.Repeat("=", 60))
		log.Printf("Generating Week %d (%s to %s)", week,
			weekStart.Format("2006-01-02"),
			weekEnd.Format("2006-01-02"))
		log.Printf(strings.Repeat("=", 60))

		var releases []models.Release

		if *skipCrawl {
			// Try to load existing data
			releasesFile := filepath.Join(*dataDir, fmt.Sprintf("releases-%d-week-%02d.json", year, week))
			if data, err := os.ReadFile(releasesFile); err == nil {
				json.Unmarshal(data, &releases)
				// Sanitize UTF-8 in release bodies
				for i := range releases {
					releases[i].Body = sanitizeUTF8(releases[i].Body)
					releases[i].Name = sanitizeUTF8(releases[i].Name)
				}
				log.Printf("Loaded %d releases from %s", len(releases), releasesFile)
			} else {
				log.Printf("No existing data for week %d, skipping", week)
				continue
			}
		} else {
			// Crawl releases for this week
			log.Printf("Crawling releases for week %d...", week)
			releases, err = ghClient.FetchReleasesInRange(ctx, cfg.Repositories, weekStart, weekEnd)
			if err != nil {
				log.Printf("Error crawling releases: %v", err)
				continue
			}
			log.Printf("Found %d releases", len(releases))

			// Save releases data
			os.MkdirAll(*dataDir, 0755)
			releasesFile := filepath.Join(*dataDir, fmt.Sprintf("releases-%d-week-%02d.json", year, week))
			data, _ := json.MarshalIndent(releases, "", "  ")
			os.WriteFile(releasesFile, data, 0644)
			log.Printf("Saved releases to %s", releasesFile)
		}

		if len(releases) == 0 {
			log.Printf("No releases for week %d, skipping newsletter generation", week)
			continue
		}

		// Generate newsletter
		log.Printf("Generating newsletter with Gemini AI...")
		newsletter, err := gemini.GenerateNewsletter(ctx, releases, nil)
		if err != nil {
			log.Printf("Error generating newsletter: %v", err)
			continue
		}

		// Save newsletter with correct week
		generator := NewBackfillDraftGenerator(*outputDir, year, week, weekStart)
		draftPath, err := generator.GenerateDraft(newsletter)
		if err != nil {
			log.Printf("Error saving newsletter: %v", err)
			continue
		}
		log.Printf("Newsletter saved to %s", draftPath)

		// Small delay between weeks to avoid rate limits
		time.Sleep(2 * time.Second)
	}

	log.Printf("\n" + strings.Repeat("=", 60))
	log.Printf("Backfill complete!")
	log.Printf(strings.Repeat("=", 60))
}

// getWeekStart returns the Monday of the week that is 'weeksAgo' weeks before now
func getWeekStart(now time.Time, weeksAgo int) time.Time {
	// Get the Monday of the current week
	weekday := int(now.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday = 7
	}
	monday := now.AddDate(0, 0, -(weekday - 1))
	monday = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location())

	// Go back 'weeksAgo' weeks
	return monday.AddDate(0, 0, -7*weeksAgo)
}

// getWeekStartForWeek returns the Monday of a specific ISO week number
func getWeekStartForWeek(year, week int) time.Time {
	// Find January 4th of the year (always in week 1 per ISO 8601)
	jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, time.Local)

	// Find the Monday of week 1
	weekday := int(jan4.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	week1Monday := jan4.AddDate(0, 0, -(weekday - 1))

	// Add (week - 1) weeks to get to the target week
	return week1Monday.AddDate(0, 0, (week-1)*7)
}

// BackfillDraftGenerator generates drafts for specific weeks
type BackfillDraftGenerator struct {
	outputDir string
	year      int
	week      int
	weekStart time.Time
}

func NewBackfillDraftGenerator(outputDir string, year, week int, weekStart time.Time) *BackfillDraftGenerator {
	return &BackfillDraftGenerator{
		outputDir: outputDir,
		year:      year,
		week:      week,
		weekStart: weekStart,
	}
}

func (g *BackfillDraftGenerator) GenerateDraft(newsletter *models.Newsletter) (string, error) {
	title := fmt.Sprintf("Week %d - %s", g.week, g.weekStart.Format("January 2006"))
	filename := fmt.Sprintf("%d-week-%02d.md", g.year, g.week)

	metadata := models.DraftMetadata{
		Title:       title,
		Date:        g.weekStart.Format("2006-01-02"),
		Draft:       false,
		Summary:     g.generateSummary(newsletter),
		Description: fmt.Sprintf("Cloud Native Newsletter Week %d %d: Kubernetes ecosystem releases and news.", g.week, g.year),
		Keywords:    []string{"Cloud Native Newsletter", "Kubernetes Releases", "CNCF Projects"},
		Highlights:  g.extractHighlights(newsletter),
	}

	frontmatter, err := yaml.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("failed to marshal frontmatter: %w", err)
	}

	// Add article link
	contentWithLink := newsletter.Content
	contentWithLink = strings.TrimRight(contentWithLink, "\n\r\t ")
	articlesURL := fmt.Sprintf("/newsletter/%d-week-%02d/articles/", g.year, g.week)
	contentWithLink += fmt.Sprintf("\n\n📚 **[View all articles from this week →](%s)**\n", articlesURL)

	content := fmt.Sprintf("---\n%s---\n\n%s", string(frontmatter), contentWithLink)

	outputPath := filepath.Join(g.outputDir, filename)
	if err := os.MkdirAll(g.outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write draft: %w", err)
	}

	return outputPath, nil
}

func (g *BackfillDraftGenerator) generateSummary(newsletter *models.Newsletter) string {
	releaseCount := len(newsletter.Releases)
	return fmt.Sprintf("This week: %d releases from the Cloud Native ecosystem.", releaseCount)
}

func (g *BackfillDraftGenerator) extractHighlights(newsletter *models.Newsletter) []string {
	highlights := make(map[string]bool)
	for _, r := range newsletter.Releases {
		highlights[r.RepoName] = true
	}

	result := []string{}
	for name := range highlights {
		result = append(result, name)
		if len(result) >= 5 {
			break
		}
	}
	return result
}

// sanitizeUTF8 removes invalid UTF-8 characters from a string
func sanitizeUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	// Replace invalid UTF-8 sequences with empty string
	v := make([]rune, 0, len(s))
	for i, r := range s {
		if r == utf8.RuneError {
			_, size := utf8.DecodeRuneInString(s[i:])
			if size == 1 {
				continue // skip invalid byte
			}
		}
		v = append(v, r)
	}
	return string(v)
}
