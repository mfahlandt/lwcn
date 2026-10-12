package ai

import (
	"strings"
	"testing"
	"time"

	"github.com/mfahlandt/lwcn/internal/models"
)

func TestPromptsIncludePublicationDates(t *testing.T) {
	releaseDate := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	newsDate := releaseDate.Add(time.Hour)
	releases := []models.Release{{RepoName: "project", TagName: "v1.0.0", PublishedAt: releaseDate}}
	news := []models.NewsItem{{Title: "Project announcement", Source: "Blog", PublishedAt: newsDate}}
	for name, prompt := range map[string]string{
		"newsletter": buildPrompt(releases, news, nil, nil),
		"linkedin":   buildLinkedInPrompt(releases, news),
	} {
		t.Run(name, func(t *testing.T) {
			for _, text := range []string{releaseDate.Format(time.RFC3339), newsDate.Format(time.RFC3339), "Publication dates do not make older events new"} {
				if !strings.Contains(prompt, text) {
					t.Errorf("prompt is missing %q", text)
				}
			}
		})
	}
}
