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
	var releases []models.Release

	opts := &github.ListOptions{PerPage: 30}
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
		// Skip drafts and prereleases
		if r.GetDraft() {
			continue
		}

		if r.PublishedAt == nil || r.PublishedAt.Time.Before(oneWeekAgo) {
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
	var allReleases []models.Release

	for i, repo := range repos {
		log.Printf("[%d/%d] Fetching %s/%s...", i+1, len(repos), repo.Owner, repo.Repo)

		releases, err := c.GetReleasesLastWeek(ctx, repo.Owner, repo.Repo, repo.Category)
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
