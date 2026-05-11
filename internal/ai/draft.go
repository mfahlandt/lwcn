package ai

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/mfahlandt/lwcn/internal/models"
	"gopkg.in/yaml.v3"
)

type DraftGenerator struct {
	outputDir string
}

func NewDraftGenerator(outputDir string) *DraftGenerator {
	return &DraftGenerator{outputDir: outputDir}
}

func (g *DraftGenerator) GenerateDraft(newsletter *models.Newsletter) (string, error) {
	now := time.Now()
	year, week := now.ISOWeek()

	// Compute the Monday–Sunday date range for this ISO week.
	// ISOWeek: Monday = 1 … Sunday = 7.
	weekday := now.Weekday() // Sunday=0, Monday=1 … Saturday=6
	if weekday == 0 {
		weekday = 7
	}
	monday := now.AddDate(0, 0, -int(weekday-1))
	sunday := monday.AddDate(0, 0, 6)

	// Build a unique title with date range:
	//   "Week 18, May 5-11, 2026"  (same month)
	//   "Week 5, Jan 27 - Feb 2, 2026"  (cross-month)
	var title string
	if monday.Month() == sunday.Month() {
		title = fmt.Sprintf("Week %d, %s %d-%d, %d",
			week, monday.Format("Jan"), monday.Day(), sunday.Day(), year)
	} else {
		title = fmt.Sprintf("Week %d, %s %d - %s %d, %d",
			week, monday.Format("Jan"), monday.Day(),
			sunday.Format("Jan"), sunday.Day(), year)
	}

	filename := fmt.Sprintf("%d-week-%02d.md", year, week)

	metadata := models.DraftMetadata{
		Title:       title,
		Date:        now.Format("2006-01-02"),
		Lastmod:     now.Format("2006-01-02"),
		Draft:       false,
		Summary:     generateSummary(newsletter),
		Description: generateSEODescription(newsletter, week, year),
		Keywords:    generateKeywords(newsletter),
		Highlights:  extractHighlights(newsletter),
		Sitemap: &models.SitemapMetadata{
			Priority:   0.9,
			ChangeFreq: "weekly",
		},
	}

	frontmatter, err := yaml.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("failed to marshal frontmatter: %w", err)
	}

	// Remove any AI-generated article link (various patterns) and add the correct one
	contentWithLink := newsletter.Content
	// Use regex to remove any "View all articles" link regardless of URL format
	articlesLinkRe := regexp.MustCompile(`(?m)\n*📚\s*\*{0,2}\[View all articles[^\]]*\]\([^)]*\)\*{0,2}\s*\n*`)
	contentWithLink = articlesLinkRe.ReplaceAllString(contentWithLink, "")
	contentWithLink = strings.TrimRight(contentWithLink, "\n\r\t ")
	// Always use the full absolute URL with domain
	articlesURL := fmt.Sprintf("https://lwcn.dev/newsletter/%d-week-%02d/articles/", year, week)
	contentWithLink += fmt.Sprintf("\n\n📚 **[View all articles from this week →](%s)**\n", articlesURL)

	content := fmt.Sprintf("---\n%s---\n\n%s", string(frontmatter), contentWithLink)

	outputPath := filepath.Join(g.outputDir, filename)
	if err := os.MkdirAll(g.outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write draft: %w", err)
	}

	// Also generate the articles page
	articlesPath, err := g.generateArticlesPage(newsletter, year, week)
	if err != nil {
		return outputPath, fmt.Errorf("newsletter created but articles page failed: %w", err)
	}
	_ = articlesPath // logged elsewhere

	return outputPath, nil
}

func (g *DraftGenerator) generateArticlesPage(newsletter *models.Newsletter, year, week int) (string, error) {
	now := time.Now()

	// Create articles as a separate file (not a folder)
	filename := fmt.Sprintf("%d-week-%02d-articles.md", year, week)

	// Group news by source
	newsBySource := make(map[string][]models.NewsItem)
	for _, item := range newsletter.NewsItems {
		newsBySource[item.Source] = append(newsBySource[item.Source], item)
	}

	// Build the articles content
	var content strings.Builder
	content.WriteString(fmt.Sprintf(`---
title: "Week %d Articles - %s"
date: "%s"
draft: false
noindex: true
url: "/newsletter/%d-week-%02d/articles/"
sitemap:
  priority: 0.3
  changefreq: never
build:
  list: never
  publishResources: true
  render: always
outputs:
  - html
---

# 📚 All Articles from Week %d

A complete list of all cloud native articles and news from this week.

`, week, now.Format("January 2006"), now.Format("2006-01-02"), year, week, week))

	// Get sorted source names
	sources := make([]string, 0, len(newsBySource))
	for source := range newsBySource {
		sources = append(sources, source)
	}
	sort.Strings(sources)

	// Write articles grouped by source
	for _, source := range sources {
		items := newsBySource[source]
		content.WriteString(fmt.Sprintf("## %s (%d)\n\n", source, len(items)))

		for _, item := range items {
			title := item.Title
			if title == "" {
				title = "Untitled"
			}
			content.WriteString(fmt.Sprintf("- [%s](%s)\n", title, item.URL))
		}
		content.WriteString("\n")
	}

	content.WriteString(fmt.Sprintf("\n---\n\n[← Back to Newsletter](https://lwcn.dev/newsletter/%d-week-%02d/)\n", year, week))

	// Write the file
	outputPath := filepath.Join(g.outputDir, filename)
	if err := os.WriteFile(outputPath, []byte(content.String()), 0644); err != nil {
		return "", fmt.Errorf("failed to write articles page: %w", err)
	}

	return outputPath, nil
}

func generateSummary(newsletter *models.Newsletter) string {
	releaseCount := len(newsletter.Releases)
	newsCount := len(newsletter.NewsItems)

	majorReleases := []string{}
	for _, r := range newsletter.Releases {
		if isMajorRelease(r.TagName) {
			majorReleases = append(majorReleases, fmt.Sprintf("%s %s", r.RepoName, r.TagName))
		}
	}

	summary := fmt.Sprintf("This week: %d releases, %d news items.", releaseCount, newsCount)
	if len(majorReleases) > 0 {
		summary += fmt.Sprintf(" Notable: %s.", strings.Join(majorReleases[:min(3, len(majorReleases))], ", "))
	}

	return summary
}

func extractHighlights(newsletter *models.Newsletter) []string {
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

func isMajorRelease(tag string) bool {
	tag = strings.TrimPrefix(tag, "v")
	parts := strings.Split(tag, ".")
	if len(parts) >= 2 {
		return parts[1] == "0" && (len(parts) < 3 || parts[2] == "0")
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// generateSEODescription creates a SEO-optimized description for the newsletter
func generateSEODescription(newsletter *models.Newsletter, week, year int) string {
	highlights := extractHighlights(newsletter)
	highlightStr := ""
	if len(highlights) > 0 {
		maxHighlights := min(4, len(highlights))
		highlightStr = strings.Join(highlights[:maxHighlights], ", ")
	}

	return fmt.Sprintf("Cloud Native Newsletter Week %d %d: %s and more Kubernetes ecosystem news.", week, year, highlightStr)
}

// generateKeywords extracts relevant keywords from the newsletter for SEO
func generateKeywords(newsletter *models.Newsletter) []string {
	keywords := []string{
		"Cloud Native Newsletter",
		"Kubernetes Releases",
		"CNCF Projects",
	}

	// Add highlights as keywords
	highlights := extractHighlights(newsletter)
	for _, h := range highlights {
		keywords = append(keywords, h)
	}

	return keywords
}
