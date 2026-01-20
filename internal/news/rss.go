package news

import (
	"context"
	"time"

	"github.com/mfahlandt/lwcn/internal/models"
	"github.com/mmcdole/gofeed"
)

type RSSClient struct {
	parser *gofeed.Parser
}

func NewRSSClient() *RSSClient {
	return &RSSClient{
		parser: gofeed.NewParser(),
	}
}

func (c *RSSClient) FetchFeed(ctx context.Context, source models.RSSSource) ([]models.NewsItem, error) {
	feed, err := c.parser.ParseURLWithContext(source.URL, ctx)
	if err != nil {
		return nil, err
	}

	oneWeekAgo := time.Now().AddDate(0, 0, -7)
	var items []models.NewsItem

	for _, item := range feed.Items {
		pubDate := time.Now()
		if item.PublishedParsed != nil {
			pubDate = *item.PublishedParsed
		}

		if pubDate.Before(oneWeekAgo) {
			continue
		}

		items = append(items, models.NewsItem{
			Title:       item.Title,
			URL:         item.Link,
			Source:      source.Name,
			Description: item.Description,
			PublishedAt: pubDate,
			Category:    "news",
		})
	}

	return items, nil
}

func (c *RSSClient) FetchAllFeeds(ctx context.Context, sources []models.RSSSource) ([]models.NewsItem, error) {
	var allItems []models.NewsItem

	for _, source := range sources {
		items, err := c.FetchFeed(ctx, source)
		if err != nil {
			continue
		}
		allItems = append(allItems, items...)
	}

	return allItems, nil
}
