package ai

import (
	"fmt"
	"os"
	"path/filepath"
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

	title := fmt.Sprintf("Week %d - %s", week, now.Format("January 2006"))
	filename := fmt.Sprintf("%d-week-%02d.md", year, week)

	metadata := models.DraftMetadata{
		Title:       title,
		Date:        now.Format("2006-01-02"),
		Draft:       false,
		Summary:     generateSummary(newsletter),
		Description: generateSEODescription(newsletter, week, year),
		Keywords:    generateKeywords(newsletter),
		Highlights:  extractHighlights(newsletter),
	}

	frontmatter, err := yaml.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("failed to marshal frontmatter: %w", err)
	}

	// Replace any placeholder article link with the correct one
	contentWithLink := newsletter.Content
	// Remove any existing article link placeholder and add the correct one
	contentWithLink = strings.ReplaceAll(contentWithLink, "📚 **[View all articles from this week →](articles/)**", "")
	contentWithLink = strings.ReplaceAll(contentWithLink, "📚 **[View all articles from this week →](2026-week-XX/)**", "")
	// Trim trailing whitespace and add the correct link
	contentWithLink = strings.TrimRight(contentWithLink, "\n\r\t ")
	// Link to the articles page with the correct URL
	articlesURL := fmt.Sprintf("/newsletter/%d-week-%02d/articles/", year, week)
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
url: "/newsletter/%d-week-%02d/articles/"
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

	content.WriteString(fmt.Sprintf("\n---\n\n[← Back to Newsletter](/%d-week-%02d/)\n", year, week))

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
