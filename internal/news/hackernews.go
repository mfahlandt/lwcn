package news

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mfahlandt/lwcn/internal/models"
)

type HackerNewsClient struct {
	client *http.Client
}

type hnSearchResponse struct {
	Hits             []hnHit `json:"hits"`
	NbHits           int     `json:"nbHits"`
	ProcessingTimeMS int     `json:"processingTimeMS"`
}

type hnHit struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	StoryText   string `json:"story_text"`
	ObjectID    string `json:"objectID"`
	CreatedAt   string `json:"created_at"`
	Points      int    `json:"points"`
	NumComments int    `json:"num_comments"`
}

func NewHackerNewsClient() *HackerNewsClient {
	return &HackerNewsClient{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *HackerNewsClient) Search(ctx context.Context, keywords []string) ([]models.NewsItem, error) {
	var allItems []models.NewsItem
	oneWeekAgo := time.Now().AddDate(0, 0, -7)
	timestamp := oneWeekAgo.Unix()

	// Strategy 1: Search by individual keywords
	for _, keyword := range keywords {
		items, err := c.searchKeyword(ctx, keyword, timestamp)
		if err != nil {
			log.Printf("HN search for '%s' failed: %v", keyword, err)
			continue
		}
		log.Printf("HN: Found %d items for keyword '%s'", len(items), keyword)
		allItems = append(allItems, items...)
	}

	// Strategy 2: Search for recent popular stories with combined query
	combinedQueries := []string{
		"kubernetes OR k8s",
		"docker container",
		"cloud infrastructure",
		"devops platform",
	}
	for _, query := range combinedQueries {
		items, err := c.searchKeyword(ctx, query, timestamp)
		if err != nil {
			continue
		}
		log.Printf("HN: Found %d items for combined query '%s'", len(items), query)
		allItems = append(allItems, items...)
	}

	// Strategy 3: Search front page stories (most popular)
	frontPageItems, err := c.searchFrontPage(ctx, timestamp)
	if err == nil {
		// Filter front page for cloud-native relevance
		relevantItems := c.filterRelevant(frontPageItems, keywords)
		log.Printf("HN: Found %d relevant front page items", len(relevantItems))
		allItems = append(allItems, relevantItems...)
	}

	deduplicated := deduplicate(allItems)
	log.Printf("HN: Total unique items after deduplication: %d", len(deduplicated))
	return deduplicated, nil
}

func (c *HackerNewsClient) searchKeyword(ctx context.Context, keyword string, timestamp int64) ([]models.NewsItem, error) {
	query := url.QueryEscape(keyword)
	// Use search for recent items - numericFilters needs proper URL encoding
	numericFilter := url.QueryEscape(fmt.Sprintf("created_at_i>%d", timestamp))
	apiURL := fmt.Sprintf(
		"https://hn.algolia.com/api/v1/search?query=%s&tags=story&numericFilters=%s&hitsPerPage=50",
		query, numericFilter,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "LWCN-Bot/1.0 (Last Week in Cloud Native)")
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HN API returned status %d for URL: %s", resp.StatusCode, apiURL)
	}

	var result hnSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var items []models.NewsItem
	for _, hit := range result.Hits {
		// Lower threshold to 3 points OR at least 2 comments
		if hit.Points < 3 && hit.NumComments < 2 {
			continue
		}

		itemURL := hit.URL
		if itemURL == "" {
			itemURL = fmt.Sprintf("https://news.ycombinator.com/item?id=%s", hit.ObjectID)
		}

		pubTime, _ := time.Parse(time.RFC3339, hit.CreatedAt)

		items = append(items, models.NewsItem{
			Title:       hit.Title,
			URL:         itemURL,
			Source:      "Hacker News",
			Description: truncate(hit.StoryText, 200),
			PublishedAt: pubTime,
			Category:    "community",
		})
	}

	return items, nil
}

func (c *HackerNewsClient) searchFrontPage(ctx context.Context, timestamp int64) ([]models.NewsItem, error) {
	// Get recent popular stories - numericFilters needs URL encoding
	numericFilter := url.QueryEscape(fmt.Sprintf("created_at_i>%d", timestamp))
	apiURL := fmt.Sprintf(
		"https://hn.algolia.com/api/v1/search?tags=front_page&numericFilters=%s&hitsPerPage=100",
		numericFilter,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "LWCN-Bot/1.0 (Last Week in Cloud Native)")
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HN API returned status %d", resp.StatusCode)
	}

	var result hnSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	var items []models.NewsItem
	for _, hit := range result.Hits {
		itemURL := hit.URL
		if itemURL == "" {
			itemURL = fmt.Sprintf("https://news.ycombinator.com/item?id=%s", hit.ObjectID)
		}

		pubTime, _ := time.Parse(time.RFC3339, hit.CreatedAt)

		items = append(items, models.NewsItem{
			Title:       hit.Title,
			URL:         itemURL,
			Source:      "Hacker News",
			Description: truncate(hit.StoryText, 200),
			PublishedAt: pubTime,
			Category:    "community",
		})
	}

	return items, nil
}

func (c *HackerNewsClient) filterRelevant(items []models.NewsItem, keywords []string) []models.NewsItem {
	var relevant []models.NewsItem
	for _, item := range items {
		titleLower := strings.ToLower(item.Title)
		descLower := strings.ToLower(item.Description)

		for _, kw := range keywords {
			kwLower := strings.ToLower(kw)
			if strings.Contains(titleLower, kwLower) || strings.Contains(descLower, kwLower) {
				relevant = append(relevant, item)
				break
			}
		}
	}
	return relevant
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func deduplicate(items []models.NewsItem) []models.NewsItem {
	seen := make(map[string]bool)
	var result []models.NewsItem

	for _, item := range items {
		key := strings.ToLower(item.Title)
		if !seen[key] {
			seen[key] = true
			result = append(result, item)
		}
	}
	return result
}
