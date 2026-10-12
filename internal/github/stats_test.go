package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	gh "github.com/google/go-github/v60/github"
	"github.com/mfahlandt/lwcn/internal/models"
)

func TestStatsExcludeNextMonday(t *testing.T) {
	start, end, err := models.CompletedWeek(time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	lastSecond := end.Add(-time.Second).UTC().Format(time.RFC3339)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/commits") {
			if r.URL.Query().Get("until") != end.Add(-time.Second).Format(time.RFC3339) {
				t.Errorf("commit end: %s", r.URL.RawQuery)
			}
			w.Write([]byte("[]"))
		} else {
			if !strings.Contains(r.URL.Query().Get("q"), ".."+lastSecond) {
				t.Errorf("search end: %s", r.URL.Query().Get("q"))
			}
			w.Write([]byte(`{"total_count":0,"items":[]}`))
		}
	}))
	defer server.Close()
	api := gh.NewClient(server.Client())
	api.BaseURL, err = url.Parse(server.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	stats, err := (&Client{gh: api}).GetRepoStats(context.Background(), "owner", "repo", "security", start, end)
	if err != nil {
		t.Fatal(err)
	}
	if !stats.WindowFrom.Equal(start) || !stats.WindowTo.Equal(end) {
		t.Fatal("stats metadata does not preserve the common calendar window")
	}
}
