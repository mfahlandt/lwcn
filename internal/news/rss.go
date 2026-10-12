package news

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mfahlandt/lwcn/internal/models"
	"github.com/mmcdole/gofeed"
	"github.com/mmcdole/gofeed/atom"
)

type RSSClient struct {
	parser *gofeed.Parser
}

func NewRSSClient() *RSSClient {
	parser := gofeed.NewParser()
	parser.AtomTranslator = &publicationAtomTranslator{}
	return &RSSClient{
		parser: parser,
	}
}

type publicationAtomTranslator struct{}

func (t *publicationAtomTranslator) Translate(raw interface{}) (*gofeed.Feed, error) {
	source, ok := raw.(*atom.Feed)
	if !ok {
		return nil, fmt.Errorf("expected Atom feed, got %T", raw)
	}
	feed, err := (&gofeed.DefaultAtomTranslator{}).Translate(source)
	if err != nil {
		return nil, err
	}
	// gofeed otherwise substitutes updated for missing or invalid published dates.
	for i, entry := range source.Entries {
		feed.Items[i].Published = entry.Published
		feed.Items[i].PublishedParsed = entry.PublishedParsed
	}
	return feed, nil
}

func (c *RSSClient) FetchFeed(ctx context.Context, source models.RSSSource) ([]models.NewsItem, error) {
	feed, err := c.parser.ParseURLWithContext(source.URL, ctx)
	if err != nil {
		return nil, err
	}

	oneWeekAgo, now, err := models.CompletedWeek(time.Now())
	if err != nil {
		return nil, fmt.Errorf("determine publication window: %w", err)
	}
	var items []models.NewsItem

	for _, item := range feed.Items {
		if item.PublishedParsed == nil {
			log.Printf("RSS [%s]: skipping %q without a valid publication date", source.Name, item.Title)
			continue
		}
		pubDate := *item.PublishedParsed

		if !models.PublishedInRange(pubDate, oneWeekAgo, now) {
			continue
		}

		items = append(items, models.NewsItem{
			Title:       item.Title,
			URL:         item.Link,
			Source:      source.Name,
			Description: item.Description,
			PublishedAt: pubDate,
			Category:    "news",
			Vendor:      source.Vendor,
			RequireOSS:  source.RequireOSS,
		})
	}

	return items, nil
}

func (c *RSSClient) FetchAllFeeds(ctx context.Context, sources []models.RSSSource) ([]models.NewsItem, error) {
	var allItems []models.NewsItem

	for _, source := range sources {
		items, err := c.FetchFeed(ctx, source)
		if err != nil {
			log.Printf("RSS [%s]: fetch failed: %v", source.Name, err)
			continue
		}
		allItems = append(allItems, items...)
	}

	return allItems, nil
}
