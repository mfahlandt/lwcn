package models

import "time"

const WeekDuration = 7 * 24 * time.Hour

// CompletedWeek returns the previous calendar week, bounded by Monday 00:00 UTC.
func CompletedWeek(now time.Time) (start, end time.Time, err error) {
	local := now.UTC()
	daysSinceMonday := (int(local.Weekday()) + 6) % 7
	end = time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -daysSinceMonday)
	return end.AddDate(0, 0, -7), end, nil
}

// PublishedInRange requires an actual publication timestamp, not a crawl or update date.
func PublishedInRange(published, start, end time.Time) bool {
	return !published.IsZero() && !published.Before(start) && published.Before(end)
}

func FilterNewsInRange(items []NewsItem, start, end time.Time) (kept, dropped []NewsItem) {
	for _, item := range items {
		if PublishedInRange(item.PublishedAt, start, end) {
			kept = append(kept, item)
		} else {
			dropped = append(dropped, item)
		}
	}
	return kept, dropped
}

func FilterReleasesInRange(items []Release, start, end time.Time) (kept, dropped []Release) {
	for _, item := range items {
		if PublishedInRange(item.PublishedAt, start, end) {
			kept = append(kept, item)
		} else {
			dropped = append(dropped, item)
		}
	}
	return kept, dropped
}

func FilterStatsInRange(items []RepoStats, start, end time.Time) (kept, dropped []RepoStats) {
	for _, item := range items {
		if !item.WindowFrom.IsZero() && !item.WindowTo.IsZero() &&
			!item.WindowFrom.Before(start) && item.WindowFrom.Before(item.WindowTo) &&
			!item.WindowTo.After(end) {
			kept = append(kept, item)
		} else {
			dropped = append(dropped, item)
		}
	}
	return kept, dropped
}
