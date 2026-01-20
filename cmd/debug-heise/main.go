package main

// Debug tool for analyzing Heise website HTML structure.
// Use this when scraping stops working due to HTML changes.
// Run: go run ./cmd/debug-heise

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func main() {
	client := &http.Client{Timeout: 30 * time.Second}

	// Test different Heise URLs
	urls := []string{
		"https://www.heise.de/thema/Cloud",
		"https://www.heise.de/news/",
	}

	for _, url := range urls {
		fmt.Printf("\n=== Testing: %s ===\n", url)

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Printf("Request error: %v\n", err)
			continue
		}

		// Use a proper browser User-Agent
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		req.Header.Set("Accept-Language", "de-DE,de;q=0.9,en;q=0.8")

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("HTTP error: %v\n", err)
			continue
		}

		fmt.Printf("Status: %d\n", resp.StatusCode)

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		fmt.Printf("Body length: %d bytes\n", len(body))

		// Parse with goquery
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(body)))
		if err != nil {
			fmt.Printf("Parse error: %v\n", err)
			continue
		}

		// Try different selectors
		selectors := []string{
			"article",
			"article.a-article-teaser",
			".a-article-teaser",
			"[class*='teaser']",
			"[class*='article']",
			".teaser",
			"a-teaser",
			"[data-component='Teaser']",
			"[data-component='TeaserContainer']",
		}

		for _, sel := range selectors {
			count := doc.Find(sel).Length()
			if count > 0 {
				fmt.Printf("Selector '%s': %d elements found\n", sel, count)

				// Show first element details
				doc.Find(sel).First().Each(func(i int, s *goquery.Selection) {
					html, _ := s.Html()
					if len(html) > 500 {
						html = html[:500] + "..."
					}
					fmt.Printf("First element HTML:\n%s\n", html)

					// Look for date/time elements
					s.Find("time, [datetime], [class*='date'], [class*='time']").Each(func(j int, timeSel *goquery.Selection) {
						datetime, _ := timeSel.Attr("datetime")
						text := timeSel.Text()
						fmt.Printf("  Date element found: datetime='%s', text='%s'\n", datetime, text)
					})
				})
			}
		}

		// Also print a snippet of the raw HTML to understand structure
		if len(body) > 2000 {
			// Find article section
			htmlStr := string(body)
			if idx := strings.Index(htmlStr, "article"); idx > 0 {
				start := idx - 100
				if start < 0 {
					start = 0
				}
				end := idx + 500
				if end > len(htmlStr) {
					end = len(htmlStr)
				}
				fmt.Printf("\n--- HTML around 'article': ---\n%s\n", htmlStr[start:end])
			}
		}
	}
}
