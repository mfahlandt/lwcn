package news

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mfahlandt/lwcn/internal/models"
)

func TestFetchFeedRequireOSS(t *testing.T) {
	_, end, err := models.CompletedWeek(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		fmt.Fprintf(w, `<?xml version="1.0"?>
<rss version="2.0"><channel><title>TechTarget</title>
<link>https://example.com</link><description>News</description>
<item><title>Supply chain security</title><link>https://example.com/oss</link>
<description>Open source dependency controls</description><pubDate>%s</pubDate></item>
<item><title>Enterprise AI budgets</title><link>https://example.com/ai</link>
<description>Corporate spending</description><pubDate>%s</pubDate></item>
</channel></rss>`, end.Add(-time.Hour).Format(time.RFC1123Z), end.Add(-time.Hour).Format(time.RFC1123Z))
	}))
	defer server.Close()

	for _, requireOSS := range []bool{false, true} {
		t.Run(fmt.Sprintf("require_oss=%v", requireOSS), func(t *testing.T) {
			items, err := NewRSSClient().FetchFeed(context.Background(), models.RSSSource{
				Name: "TechTarget", URL: server.URL, RequireOSS: requireOSS,
			})
			if err != nil {
				t.Fatal(err)
			}
			if len(items) != 2 {
				t.Fatalf("got %d items, want 2", len(items))
			}
			for _, item := range items {
				if item.RequireOSS != requireOSS || item.Vendor || item.Source != "TechTarget" {
					t.Errorf("incorrect source metadata: %+v", item)
				}
			}
			kept, _ := NewNewsFilter(FilterOptions{}).Apply(items)
			want := 2
			if requireOSS {
				want = 1
			}
			if len(kept) != want {
				t.Fatalf("kept %d items, want %d", len(kept), want)
			}
			if kept[0].URL != "https://example.com/oss" {
				t.Errorf("unexpected article: %+v", kept[0])
			}
		})
	}
}

func TestFetchFeedPublicationDates(t *testing.T) {
	start, now, err := models.CompletedWeek(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		date string
		keep bool
	}{
		{"recent", now.Add(-time.Hour).Format(time.RFC1123Z), true},
		{"before the week", start.Add(-time.Hour).Format(time.RFC1123Z), false},
		{"start boundary", start.Format(time.RFC1123Z), true},
		{"end boundary", now.Format(time.RFC1123Z), false},
		{"future", now.Add(time.Hour).Format(time.RFC1123Z), false},
		{"missing", "", false},
		{"invalid", "not a date", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/rss+xml")
				fmt.Fprintf(w, `<rss version="2.0"><channel><title>News</title>
<link>https://example.com</link><description>News</description>
<item><title>Article</title><link>https://example.com/article</link><pubDate>%s</pubDate></item>
</channel></rss>`, tc.date)
			}))
			defer server.Close()
			items, err := NewRSSClient().FetchFeed(context.Background(), models.RSSSource{Name: "Test", URL: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			if (len(items) == 1) != tc.keep {
				t.Fatalf("got %d items, want keep=%v", len(items), tc.keep)
			}
		})
	}
}

func TestFetchFeedDoesNotUseUpdateDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		fmt.Fprintf(w, `<feed xmlns="http://www.w3.org/2005/Atom">
<title>News</title><id>https://example.com</id><updated>%s</updated>
<entry><title>Undated article</title><id>https://example.com/article</id>
<link href="https://example.com/article"/><updated>%s</updated></entry></feed>`,
			time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))
	}))
	defer server.Close()
	items, err := NewRSSClient().FetchFeed(context.Background(), models.RSSSource{Name: "Test", URL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("update-only article accepted: %+v", items)
	}
}
