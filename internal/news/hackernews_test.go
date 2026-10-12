package news

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mfahlandt/lwcn/internal/models"
)

type hnTestTransport struct {
	target string
}

func (t hnTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	redirected := req.Clone(req.Context())
	target, err := http.NewRequestWithContext(req.Context(), req.Method, t.target, nil)
	if err != nil {
		return nil, err
	}
	redirected.URL = target.URL
	return http.DefaultTransport.RoundTrip(redirected)
}

func TestHackerNewsPublicationWindow(t *testing.T) {
	start, now, err := models.CompletedWeek(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits := []hnHit{
			{Title: "kubernetes recent", URL: "https://example.com/recent", CreatedAt: now.Add(-time.Hour).Format(time.RFC3339), Points: 10},
			{Title: "kubernetes old", CreatedAt: start.Add(-time.Hour).Format(time.RFC3339), Points: 10},
			{Title: "kubernetes future", CreatedAt: now.Add(time.Hour).Format(time.RFC3339), Points: 10},
			{Title: "kubernetes invalid", CreatedAt: "invalid", Points: 10},
			{Title: "kubernetes missing", Points: 10},
		}
		if err := json.NewEncoder(w).Encode(hnSearchResponse{Hits: hits}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
	defer server.Close()
	client := &HackerNewsClient{client: &http.Client{Transport: hnTestTransport{target: server.URL}}}
	items, err := client.Search(context.Background(), []string{"kubernetes"})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].URL != "https://example.com/recent" {
		t.Fatalf("expected only recent story, got %+v", items)
	}
}
