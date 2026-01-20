package news

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/mfahlandt/lwcn/internal/models"
)

type Scraper struct {
	client *http.Client
}

func NewScraper() *Scraper {
	return &Scraper{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *Scraper) ScrapeHeise(ctx context.Context, source models.ScrapeSource) ([]models.NewsItem, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", source.URL, nil)
	if err != nil {
		return nil, err
	}

	// Use a proper browser User-Agent - Heise blocks simple bots
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "de-DE,de;q=0.9,en;q=0.8")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Heise returned status %d", resp.StatusCode)
		return nil, nil
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var items []models.NewsItem
	oneWeekAgo := time.Now().AddDate(0, 0, -7)

	// Heise uses data-component="TeaserContainer" for article teasers
	// Try multiple selectors for different page layouts
	selectors := []string{
		"article[data-component='TeaserContainer']",
		"[data-component='TeaserContainer']",
		"article[data-teaser-name]",
		"article",
	}

	var foundSelector string
	for _, selector := range selectors {
		count := doc.Find(selector).Length()
		if count > 0 {
			foundSelector = selector
			log.Printf("Heise: Using selector '%s' - found %d elements", selector, count)
			break
		}
	}

	if foundSelector == "" {
		log.Printf("Heise: No matching selectors found")
		return items, nil
	}

	doc.Find(foundSelector).Each(func(i int, sel *goquery.Selection) {
		// Try different ways to get the link and title
		var link, title, desc string
		var pubDate time.Time

		// Method 1: Look for TeaserLinkContainer
		linkSel := sel.Find("a[data-component='TeaserLinkContainer']").First()
		if linkSel.Length() == 0 {
			// Method 2: First link in the article
			linkSel = sel.Find("a").First()
		}

		if linkSel.Length() > 0 {
			link, _ = linkSel.Attr("href")
			// Title might be in the link or in a heading inside
			title = strings.TrimSpace(linkSel.Find("h2, h3, span").First().Text())
			if title == "" {
				title = strings.TrimSpace(linkSel.Text())
			}
		}

		// Look for description/synopsis
		descSel := sel.Find("p, [class*='synopsis'], [class*='description']").First()
		if descSel.Length() > 0 {
			desc = strings.TrimSpace(descSel.Text())
		}

		// Extract publication date from time element with datetime attribute
		timeSel := sel.Find("time[datetime], [datetime]").First()
		if timeSel.Length() > 0 {
			if datetime, exists := timeSel.Attr("datetime"); exists {
				// Parse ISO 8601 datetime with milliseconds (e.g., "2026-01-19T18:02:06.276Z")
				parsedTime, err := time.Parse("2006-01-02T15:04:05.000Z", datetime)
				if err != nil {
					// Try RFC3339 format
					parsedTime, err = time.Parse(time.RFC3339, datetime)
				}
				if err != nil {
					// Try alternative format without milliseconds
					parsedTime, err = time.Parse("2006-01-02T15:04:05Z", datetime)
				}
				if err == nil {
					pubDate = parsedTime
				}
			}
		}

		// Skip if no date found or older than 7 days
		if pubDate.IsZero() || pubDate.Before(oneWeekAgo) {
			return
		}

		// Skip if no title or link
		if title == "" || link == "" {
			return
		}

		// Clean up title (remove excessive whitespace)
		title = strings.Join(strings.Fields(title), " ")

		// Make link absolute
		if !strings.HasPrefix(link, "http") {
			link = "https://www.heise.de" + link
		}

		// Skip very short titles (likely navigation elements)
		if len(title) < 10 {
			return
		}

		items = append(items, models.NewsItem{
			Title:       title,
			URL:         link,
			Source:      source.Name,
			Description: desc,
			PublishedAt: pubDate,
			Category:    "news",
		})
	})

	log.Printf("Heise: Scraped %d items from %s", len(items), source.URL)
	return items, nil
}
