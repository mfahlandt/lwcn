package github

import (
	"context"
	"log"
	"time"

	"github.com/google/go-github/v60/github"
	"github.com/mfahlandt/lwcn/internal/models"
	"golang.org/x/oauth2"
)

type Client struct {
	gh *github.Client
}

func NewClient(token string) *Client {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	return &Client{
		gh: github.NewClient(tc),
	}
}

func (c *Client) GetReleasesLastWeek(ctx context.Context, owner, repo, category string) ([]models.Release, error) {
	oneWeekAgo := time.Now().AddDate(0, 0, -7)
	return c.GetReleasesInRange(ctx, owner, repo, category, oneWeekAgo, time.Now())
}

// GetReleasesInRange fetches releases published within the given time range
func (c *Client) GetReleasesInRange(ctx context.Context, owner, repo, category string, start, end time.Time) ([]models.Release, error) {
	var releases []models.Release

	opts := &github.ListOptions{PerPage: 100}
	ghReleases, resp, err := c.gh.Repositories.ListReleases(ctx, owner, repo, opts)
	if err != nil {
		// Check for rate limit
		if resp != nil && resp.StatusCode == 403 {
			log.Printf("Rate limited for %s/%s, remaining: %d, reset: %v",
				owner, repo, resp.Rate.Remaining, resp.Rate.Reset.Time)
		}
		return nil, err
	}

	for _, r := range ghReleases {
		// Skip drafts
		if r.GetDraft() {
			continue
		}

		if r.PublishedAt == nil {
			continue
		}

		// Check if release is within the time range
		publishedAt := r.PublishedAt.Time
		if publishedAt.Before(start) || publishedAt.After(end) {
			continue
		}

		releaseName := r.GetName()
		if releaseName == "" {
			releaseName = r.GetTagName()
		}

		releases = append(releases, models.Release{
			RepoOwner:    owner,
			RepoName:     repo,
			TagName:      r.GetTagName(),
			Name:         releaseName,
			Body:         r.GetBody(),
			URL:          r.GetHTMLURL(),
			PublishedAt:  r.PublishedAt.Time,
			Category:     category,
			IsPrerelease: r.GetPrerelease(),
		})
	}

	return releases, nil
}

func (c *Client) FetchAllReleases(ctx context.Context, repos []models.Repository) ([]models.Release, error) {
	oneWeekAgo := time.Now().AddDate(0, 0, -7)
	return c.FetchReleasesInRange(ctx, repos, oneWeekAgo, time.Now())
}

// FetchReleasesInRange fetches all releases from all repos within the given time range
func (c *Client) FetchReleasesInRange(ctx context.Context, repos []models.Repository, start, end time.Time) ([]models.Release, error) {
	var allReleases []models.Release

	log.Printf("Fetching releases from %s to %s", start.Format("2006-01-02"), end.Format("2006-01-02"))

	for i, repo := range repos {
		log.Printf("[%d/%d] Fetching %s/%s...", i+1, len(repos), repo.Owner, repo.Repo)

		releases, err := c.GetReleasesInRange(ctx, repo.Owner, repo.Repo, repo.Category, start, end)
		if err != nil {
			log.Printf("  Error: %v", err)
			continue
		}

		if len(releases) > 0 {
			log.Printf("  Found %d releases", len(releases))
			for _, r := range releases {
				log.Printf("    - %s (%s)", r.TagName, r.PublishedAt.Format("2006-01-02"))
			}
		}

		allReleases = append(allReleases, releases...)

		// Small delay to avoid rate limiting
		time.Sleep(100 * time.Millisecond)
	}

	return allReleases, nil
}
