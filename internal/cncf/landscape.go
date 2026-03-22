package cncf

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/mfahlandt/lwcn/internal/models"
)

const (
	// LandscapeAPIURL is the CNCF landscape API endpoint that returns all projects
	LandscapeAPIURL = "https://landscape.cncf.io/api/items/export?project=hosted"
)

// LandscapeItem represents a single project from the CNCF Landscape API
type LandscapeItem struct {
	Name        string `json:"name"`
	RepoURL     string `json:"repo_url"`
	Project     string `json:"project"` // "graduated", "incubating", "sandbox"
	Category    string `json:"category"`
	Subcategory string `json:"subcategory"`
}

// FetchCNCFProjects fetches the list of all CNCF projects from the Landscape API
// and converts them into Repository models suitable for release tracking.
func FetchCNCFProjects() ([]models.Repository, error) {
	log.Println("Fetching CNCF projects from Landscape API...")

	resp, err := http.Get(LandscapeAPIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch CNCF landscape: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CNCF landscape API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var items []LandscapeItem
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, fmt.Errorf("failed to parse landscape data: %w", err)
	}

	log.Printf("Found %d items from CNCF Landscape", len(items))

	var repos []models.Repository
	seen := make(map[string]bool)

	for _, item := range items {
		owner, repo, ok := parseGitHubURL(item.RepoURL)
		if !ok {
			continue
		}

		key := owner + "/" + repo
		if seen[key] {
			continue
		}
		seen[key] = true

		category := normalizeCategory(item.Subcategory, item.Category)

		repos = append(repos, models.Repository{
			Owner:      owner,
			Repo:       repo,
			Name:       item.Name,
			Category:   category,
			CNCFStatus: item.Project,
		})
	}

	log.Printf("Extracted %d unique GitHub repositories from CNCF projects", len(repos))

	return repos, nil
}

// parseGitHubURL extracts owner and repo from a GitHub URL.
// Supports formats like:
//   - https://github.com/owner/repo
//   - https://github.com/owner/repo.git
func parseGitHubURL(rawURL string) (owner, repo string, ok bool) {
	if rawURL == "" {
		return "", "", false
	}

	// Only handle github.com URLs
	if !strings.Contains(rawURL, "github.com") {
		return "", "", false
	}

	// Remove trailing .git
	rawURL = strings.TrimSuffix(rawURL, ".git")
	// Remove trailing slash
	rawURL = strings.TrimSuffix(rawURL, "/")

	// Extract path after github.com
	parts := strings.Split(rawURL, "github.com/")
	if len(parts) != 2 {
		return "", "", false
	}

	segments := strings.Split(parts[1], "/")
	if len(segments) < 2 {
		return "", "", false
	}

	return segments[0], segments[1], true
}

// normalizeCategory converts CNCF landscape categories to simpler categories
// used for the newsletter.
func normalizeCategory(subcategory, category string) string {
	sub := strings.ToLower(subcategory)
	cat := strings.ToLower(category)

	// Map subcategories to our internal categories
	categoryMap := map[string]string{
		"container runtime":                    "container-runtime",
		"cloud native storage":                 "storage",
		"container registry":                   "registry",
		"service mesh":                         "service-mesh",
		"api gateway":                          "networking",
		"service proxy":                        "networking",
		"cloud native network":                 "networking",
		"coordination & service discovery":     "networking",
		"scheduling & orchestration":           "orchestration",
		"monitoring":                           "monitoring",
		"logging":                              "logging",
		"tracing":                              "tracing",
		"observability":                        "observability",
		"chaos engineering":                    "chaos-engineering",
		"continuous integration & delivery":    "ci-cd",
		"continuous optimization":              "ci-cd",
		"security & compliance":                "security",
		"key management":                       "security",
		"streaming & messaging":                "messaging",
		"database":                             "database",
		"application definition & image build": "build",
		"automation & configuration":           "configuration",
		"package manager":                      "package-management",
		"serverless":                           "serverless",
		"installable platform":                 "platform",
		"platform":                             "platform",
		"certified kubernetes - distribution":  "distribution",
	}

	// Try subcategory first
	if mapped, ok := categoryMap[sub]; ok {
		return mapped
	}

	// Try category
	if mapped, ok := categoryMap[cat]; ok {
		return mapped
	}

	// Fallback: sanitize subcategory or category
	if sub != "" {
		return strings.ReplaceAll(sub, " ", "-")
	}
	if cat != "" {
		return strings.ReplaceAll(cat, " ", "-")
	}

	return "other"
}
