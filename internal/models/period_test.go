package models

import (
	"testing"
	"time"
)

func TestPublicationWindow(t *testing.T) {
	end := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	start := end.Add(-WeekDuration)
	for _, tc := range []struct {
		name string
		date time.Time
		want bool
	}{
		{"missing", time.Time{}, false},
		{"too old", start.Add(-time.Nanosecond), false},
		{"start boundary", start, true},
		{"recent", end.Add(-time.Hour), true},
		{"end boundary", end, false},
		{"future", end.Add(time.Nanosecond), false},
		{"different timezone", start.In(time.FixedZone("offset", 7200)), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := PublishedInRange(tc.date, start, end); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
			news, droppedNews := FilterNewsInRange([]NewsItem{{PublishedAt: tc.date}}, start, end)
			releases, droppedReleases := FilterReleasesInRange([]Release{{PublishedAt: tc.date}}, start, end)
			if (len(news) == 1) != tc.want || (len(releases) == 1) != tc.want ||
				len(news)+len(droppedNews) != 1 || len(releases)+len(droppedReleases) != 1 {
				t.Fatal("news/release filters disagree with publication window")
			}
		})
	}
}

func TestStatsWindow(t *testing.T) {
	end := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	start := end.Add(-WeekDuration)
	items := []RepoStats{
		{WindowFrom: start, WindowTo: end},
		{WindowFrom: start.Add(-time.Second), WindowTo: end},
		{WindowFrom: start, WindowTo: end.Add(time.Second)},
		{WindowFrom: end, WindowTo: start},
		{WindowTo: end},
		{WindowFrom: start},
	}
	kept, dropped := FilterStatsInRange(items, start, end)
	if len(kept) != 1 || len(dropped) != 5 || kept[0] != items[0] {
		t.Fatalf("kept=%v dropped=%v", kept, dropped)
	}
}

func TestCompletedWeek(t *testing.T) {
	for _, tc := range []struct {
		now, start, end string
	}{
		{"2026-10-12T06:00:00Z", "2026-10-05T00:00:00Z", "2026-10-12T00:00:00Z"},
		{"2026-10-15T14:00:00Z", "2026-10-05T00:00:00Z", "2026-10-12T00:00:00Z"},
		{"2026-10-12T00:00:00Z", "2026-10-05T00:00:00Z", "2026-10-12T00:00:00Z"},
		{"2026-10-11T23:59:59Z", "2026-09-28T00:00:00Z", "2026-10-05T00:00:00Z"},
		{"2026-10-12T00:00:00+02:00", "2026-09-28T00:00:00Z", "2026-10-05T00:00:00Z"},
		{"2026-10-12T02:00:00+02:00", "2026-10-05T00:00:00Z", "2026-10-12T00:00:00Z"},
		{"2026-10-26T12:00:00Z", "2026-10-19T00:00:00Z", "2026-10-26T00:00:00Z"},
		{"2026-03-30T12:00:00Z", "2026-03-23T00:00:00Z", "2026-03-30T00:00:00Z"},
		{"2026-01-01T12:00:00Z", "2025-12-22T00:00:00Z", "2025-12-29T00:00:00Z"},
	} {
		t.Run(tc.now, func(t *testing.T) {
			now, err := time.Parse(time.RFC3339, tc.now)
			if err != nil {
				t.Fatal(err)
			}
			start, end, err := CompletedWeek(now)
			if err != nil {
				t.Fatal(err)
			}
			if start.Format(time.RFC3339) != tc.start || end.Format(time.RFC3339) != tc.end || end.Sub(start) != WeekDuration {
				t.Fatalf("unexpected week: %s to %s (%v)", start, end, end.Sub(start))
			}
		})
	}
}
