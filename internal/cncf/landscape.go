package cncf

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mfahlandt/lwcn/internal/models"
	"gopkg.in/yaml.v3"
)

const LandscapeURL = "https://raw.githubusercontent.com/cncf/landscape/master/landscape.yml"

type landscapeItem struct {
	Name    string `yaml:"name"`
	Project string `yaml:"project"`
	RepoURL string `yaml:"repo_url"`
}

type landscapeSubcategory struct {
	Name  string          `yaml:"name"`
	Items []landscapeItem `yaml:"items"`
}

type landscapeCategory struct {
	Name          string                 `yaml:"name"`
	Subcategories []landscapeSubcategory `yaml:"subcategories"`
}

type landscape struct {
	Categories []landscapeCategory `yaml:"landscape"`
}

// These projects publish releases in a different repo than their landscape entry.
var releaseRepoOverrides = map[string]string{
	"OpenTelemetry":           "https://github.com/open-telemetry/opentelemetry-collector",
	"Confidential Containers": "https://github.com/confidential-containers/operator",
}

// FetchCNCFProjects reads active projects and their repos from the official landscape YAML.
func FetchCNCFProjects() ([]models.Repository, error) {
	return fetchCNCFProjects(&http.Client{Timeout: 30 * time.Second}, LandscapeURL)
}

func fetchCNCFProjects(client *http.Client, source string) ([]models.Repository, error) {
	log.Println("Fetching CNCF projects from landscape.yml...")
	resp, err := client.Get(source)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch CNCF landscape: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CNCF landscape returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read landscape response: %w", err)
	}
	return parseLandscape(body)
}

func parseLandscape(body []byte) ([]models.Repository, error) {
	var data landscape
	if err := yaml.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("failed to parse landscape YAML: %w", err)
	}

	var repos []models.Repository
	seen := make(map[string]bool)
	for _, category := range data.Categories {
		for _, subcategory := range category.Subcategories {
			for _, item := range subcategory.Items {
				switch item.Project {
				case "graduated", "incubating", "sandbox":
				default:
					continue
				}

				repoURL := item.RepoURL
				if override, ok := releaseRepoOverrides[item.Name]; ok {
					repoURL = override
				}
				owner, repo, err := parseGitHubRepoURL(repoURL)
				if err != nil {
					log.Printf("Skipping CNCF project %q: %v", item.Name, err)
					continue
				}
				if strings.TrimSpace(item.Name) == "" {
					return nil, fmt.Errorf("CNCF landscape has an unnamed project at %s", repoURL)
				}

				key := strings.ToLower(owner + "/" + repo)
				if seen[key] {
					continue
				}
				seen[key] = true
				repos = append(repos, models.Repository{
					Owner:      owner,
					Repo:       repo,
					Name:       item.Name,
					Category:   normalizeCategory(subcategory.Name, category.Name),
					CNCFStatus: item.Project,
				})
			}
		}
	}

	if len(repos) == 0 {
		return nil, fmt.Errorf("CNCF landscape contains no active projects with valid GitHub repositories")
	}
	log.Printf("Extracted %d GitHub repositories from CNCF projects", len(repos))
	return repos, nil
}

func parseGitHubRepoURL(raw string) (string, string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", fmt.Errorf("invalid repository URL %q: %w", raw, err)
	}
	if (u.Scheme != "https" && u.Scheme != "http") || !strings.EqualFold(u.Host, "github.com") || u.User != nil {
		return "", "", fmt.Errorf("not a GitHub repository URL: %q", raw)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("expected GitHub owner/repo URL: %q", raw)
	}
	repo := strings.TrimSuffix(parts[1], ".git")
	if repo == "" || parts[0] == "." || parts[0] == ".." || repo == "." || repo == ".." {
		return "", "", fmt.Errorf("invalid GitHub owner/repo URL: %q", raw)
	}
	return parts[0], repo, nil
}

// normalizeCategory converts CNCF landscape categories to simpler categories.
func normalizeCategory(subcategory, category string) string {
	sub := strings.ToLower(subcategory)

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
		"serverless":                           "serverless",
		"installable platform":                 "platform",
		"feature flagging":                     "observability",
		"ml serving":                           "ml-serving",
		"general orchestration":                "orchestration",
	}

	if mapped, ok := categoryMap[sub]; ok {
		return mapped
	}
	if sub != "" {
		return strings.ReplaceAll(sub, " ", "-")
	}
	if category != "" {
		return strings.ReplaceAll(strings.ToLower(category), " ", "-")
	}
	return "other"
}
