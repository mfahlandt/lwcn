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

func TestScrapeHeisePublicationWindow(t *testing.T) {
	start, now, err := models.CompletedWeek(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, date := range []string{
			now.Add(-time.Hour).Format(time.RFC3339),
			start.Add(-time.Hour).Format(time.RFC3339),
			now.Add(time.Hour).Format(time.RFC3339),
			"", "invalid",
		} {
			fmt.Fprintf(w, `<article><a href="/news/article"><h2>Cloud native news article</h2></a><time datetime="%s"></time></article>`, date)
		}
	}))
	defer server.Close()
	items, err := NewScraper().ScrapeHeise(context.Background(), models.ScrapeSource{Name: "Heise", URL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d items, want one recent article", len(items))
	}
}
