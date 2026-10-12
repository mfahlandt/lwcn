package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	gh "github.com/google/go-github/v60/github"
	"github.com/mfahlandt/lwcn/internal/models"
)

func TestGetReleasesPublicationWindow(t *testing.T) {
	end := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	start := end.Add(-models.WeekDuration)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var releases []gh.RepositoryRelease
		for _, date := range []time.Time{start, end, start.Add(-time.Second), end.Add(time.Second), {}} {
			releases = append(releases, gh.RepositoryRelease{
				TagName: gh.String(date.String()), PublishedAt: &gh.Timestamp{Time: date},
			})
		}
		releases = append(releases,
			gh.RepositoryRelease{TagName: gh.String("missing date")},
			gh.RepositoryRelease{TagName: gh.String("draft"), Draft: gh.Bool(true), PublishedAt: &gh.Timestamp{Time: end}},
		)
		if err := json.NewEncoder(w).Encode(releases); err != nil {
			t.Errorf("encode releases: %v", err)
		}
	}))
	defer server.Close()
	api := gh.NewClient(server.Client())
	base, err := url.Parse(server.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	api.BaseURL = base
	client := &Client{gh: api}
	releases, err := client.GetReleasesInRange(context.Background(), "owner", "repo", "security", start, end)
	if err != nil {
		t.Fatal(err)
	}
	if len(releases) != 1 || !releases[0].PublishedAt.Equal(start) {
		t.Fatalf("expected only inclusive start boundary, got %+v", releases)
	}
}
