package ai

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mfahlandt/lwcn/internal/models"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadPreviousCoverage(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "2026-week-36.md", "---\ntitle: x\n---\n\n## 👋 Welcome\nK8s 1.37 released.\n\n## 🚀 Notable Releases\n- **[Kubernetes v1.37.0](https://github.com/kubernetes/kubernetes/releases/tag/v1.37.0)** - New.\n\n## 📰 This Week in Cloud Native\nRelease week.\n\n## 💬 Community Buzz\nHN stuff.\n")
	writeFile(t, dir, "2026-week-37.md", "## 👋 Welcome\nWeek 37.\n\n## 📰 This Week in Cloud Native\nPatches.\n")
	writeFile(t, dir, "2026-week-38.md", "## 👋 Welcome\nWeek 38.\n")
	writeFile(t, dir, "2026-week-38-articles.md", "kubernetes/kubernetes/releases/tag/v1.99.0")
	// Current week must be ignored (re-runs overwrite it).
	writeFile(t, dir, "2026-week-39.md", "## 👋 Welcome\nCurrent.\n")

	now := time.Date(2026, 9, 23, 6, 0, 0, 0, time.UTC) // ISO week 39
	cov, err := LoadPreviousCoverage(dir, now, 2)
	if err != nil {
		t.Fatal(err)
	}

	if len(cov.Editions) != 2 || cov.Editions[0].Name != "2026-week-37.md" || cov.Editions[1].Name != "2026-week-38.md" {
		t.Fatalf("unexpected editions: %+v", cov.Editions)
	}
	if !strings.Contains(cov.Editions[0].Summary, "Week 37.") || !strings.Contains(cov.Editions[0].Summary, "Patches.") {
		t.Errorf("summary missing sections: %q", cov.Editions[0].Summary)
	}
	if !cov.CoveredK8sMinors[37] {
		t.Error("v1.37 should be covered (found in older edition outside the summary window)")
	}
	if cov.CoveredK8sMinors[99] {
		t.Error("articles pages must be ignored")
	}
}

func TestFilterRepeatedReleaseNews(t *testing.T) {
	cov := &PreviousCoverage{CoveredK8sMinors: map[int]bool{37: true}}
	news := []models.NewsItem{
		{Title: "Kubernetes v1.37: Tracking When a PersistentVolumeClaim Was Last Used (Beta)"},
		{Title: "Kubernetes 1.37 brings rootless kubelet"},
		{Title: "Kubernetes v1.37.0 deep dive"},
		{Title: "Kubernetes v1.37.1 fixes CVE-2026-1234"},
		{Title: "Kubernetes v1.38: Release schedule"},
		{Title: "Spotlight on SIG Apps"},
	}
	kept, dropped := FilterRepeatedReleaseNews(news, cov)
	if len(dropped) != 3 {
		t.Errorf("dropped %d, want 3: %+v", len(dropped), dropped)
	}
	if len(kept) != 3 {
		t.Errorf("kept %d, want 3: %+v", len(kept), kept)
	}

	if k, d := FilterRepeatedReleaseNews(news, nil); len(k) != len(news) || d != nil {
		t.Error("nil coverage must keep everything")
	}
}
