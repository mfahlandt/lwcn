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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		fmt.Fprintf(w, `<?xml version="1.0"?>
<rss version="2.0"><channel><title>TechTarget</title>
<link>https://example.com</link><description>News</description>
<item><title>Supply chain security</title><link>https://example.com/oss</link>
<description>Open source dependency controls</description><pubDate>%s</pubDate></item>
<item><title>Enterprise AI budgets</title><link>https://example.com/ai</link>
<description>Corporate spending</description><pubDate>%s</pubDate></item>
</channel></rss>`, time.Now().Format(time.RFC1123Z), time.Now().Format(time.RFC1123Z))
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
