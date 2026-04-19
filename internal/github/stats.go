package github

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/go-github/v60/github"
	"github.com/mfahlandt/lwcn/internal/models"
)

// GetRepoStats fetches neutral activity metrics for a repo in [start, end]:
// number of commits on the default branch and number of merged pull requests.
// Counts only — no sentiment, no opinion.
func (c *Client) GetRepoStats(ctx context.Context, owner, repo, category string, start, end time.Time) (models.RepoStats, error) {
	stats := models.RepoStats{
		RepoOwner:  owner,
		RepoName:   repo,
		Category:   category,
		WindowFrom: start,
		WindowTo:   end,
	}

	// --- Commits on default branch ---
	commitOpts := &github.CommitsListOptions{
		Since:       start,
		Until:       end,
		ListOptions: github.ListOptions{PerPage: 100},
	}

	commits := 0
	for {
		page, resp, err := c.gh.Repositories.ListCommits(ctx, owner, repo, commitOpts)
		if err != nil {
			// 409 = empty repo, 404 = access issue — keep the stats row but with zeros
			if resp != nil && (resp.StatusCode == 409 || resp.StatusCode == 404) {
				break
			}
			return stats, fmt.Errorf("list commits %s/%s: %w", owner, repo, err)
		}
		commits += len(page)
		if resp == nil || resp.NextPage == 0 {
			break
		}
		commitOpts.Page = resp.NextPage
	}
	stats.Commits = commits

	// --- Merged PRs (via search, returns TotalCount in a single call) ---
	// GitHub search rate limit is 30 req/min authenticated; caller adds delay.
	dateRange := fmt.Sprintf("%s..%s", start.UTC().Format("2006-01-02T15:04:05Z"), end.UTC().Format("2006-01-02T15:04:05Z"))
	mergedQuery := fmt.Sprintf("repo:%s/%s is:pr is:merged merged:%s", owner, repo, dateRange)
	if res, _, err := c.gh.Search.Issues(ctx, mergedQuery, &github.SearchOptions{ListOptions: github.ListOptions{PerPage: 1}}); err == nil && res != nil {
		stats.MergedPRs = res.GetTotal()
	} else if err != nil {
		log.Printf("  merged-PR search failed for %s/%s: %v", owner, repo, err)
	}

	// --- Opened PRs (optional neutral metric) ---
	openedQuery := fmt.Sprintf("repo:%s/%s is:pr created:%s", owner, repo, dateRange)
	if res, _, err := c.gh.Search.Issues(ctx, openedQuery, &github.SearchOptions{ListOptions: github.ListOptions{PerPage: 1}}); err == nil && res != nil {
		stats.OpenedPRs = res.GetTotal()
	} else if err != nil {
		log.Printf("  opened-PR search failed for %s/%s: %v", owner, repo, err)
	}

	return stats, nil
}

// FetchAllStats iterates over repos and collects stats for the given window.
// Respects search-API rate limits with a conservative delay (~2.5s per repo
// ≈ 24 req/min, 2 search calls each -> stays under 30/min).
func (c *Client) FetchAllStats(ctx context.Context, repos []models.Repository, start, end time.Time) []models.RepoStats {
	log.Printf("Collecting repo stats from %s to %s for %d repos...",
		start.Format("2006-01-02"), end.Format("2006-01-02"), len(repos))

	out := make([]models.RepoStats, 0, len(repos))
	for i, repo := range repos {
		log.Printf("[%d/%d] stats: %s/%s", i+1, len(repos), repo.Owner, repo.Repo)
		s, err := c.GetRepoStats(ctx, repo.Owner, repo.Repo, repo.Category, start, end)
		if err != nil {
			log.Printf("  error: %v", err)
			continue
		}
		if s.Commits > 0 || s.MergedPRs > 0 || s.OpenedPRs > 0 {
			out = append(out, s)
		}
		// Throttle to stay under the 30 req/min search API limit.
		time.Sleep(2500 * time.Millisecond)
	}
	return out
}
