package models

import "time"

// RepoStats captures factual activity metrics for a single repository over a
// defined time window. All fields are raw counts — no ranking, no opinion —
// so they can be rendered neutrally in the newsletter ("Numbers of the Week").
type RepoStats struct {
	RepoOwner  string    `json:"repo_owner"`
	RepoName   string    `json:"repo_name"`
	Category   string    `json:"category,omitempty"`
	Commits    int       `json:"commits"`
	MergedPRs  int       `json:"merged_prs"`
	OpenedPRs  int       `json:"opened_prs,omitempty"`
	WindowFrom time.Time `json:"window_from"`
	WindowTo   time.Time `json:"window_to"`
}
