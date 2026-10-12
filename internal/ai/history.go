package ai

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mfahlandt/lwcn/internal/models"
)

var (
	weeklyFileRe = regexp.MustCompile(`^(\d{4})-week-(\d{2})\.md$`)
	// Link to a Kubernetes minor release (vX.Y.0) inside a past newsletter.
	k8sMinorReleaseLinkRe = regexp.MustCompile(`kubernetes/kubernetes/releases/tag/v1\.(\d+)\.0\b`)
	// "Kubernetes v1.37: ...", "Kubernetes 1.37 brings ...", "Kubernetes v1.37.1 fixes ..."
	k8sVersionTitleRe = regexp.MustCompile(`(?i)\bkubernetes\s+v?1\.(\d+)(?:\.(\d+))?\b`)
)

// PreviousCoverage describes what earlier newsletter editions already reported.
type PreviousCoverage struct {
	// Editions holds the "Welcome" and "This Week in Cloud Native" sections
	// of recent editions, keyed by file name (oldest first).
	Editions []PreviousEdition
	// CoveredK8sMinors contains Kubernetes minor versions (e.g. 37 for v1.37)
	// whose .0 release already appeared in an earlier edition.
	CoveredK8sMinors map[int]bool
}

type PreviousEdition struct {
	Name    string
	Summary string
}

// LoadPreviousCoverage reads the last n weekly editions in dir that are older
// than the ISO week of now. All past editions are scanned for covered
// Kubernetes minor releases.
func LoadPreviousCoverage(dir string, now time.Time, n int) (*PreviousCoverage, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read newsletter dir: %w", err)
	}

	curYear, curWeek := now.ISOWeek()
	current := curYear*100 + curWeek

	type edition struct {
		key  int
		name string
	}
	var past []edition
	for _, e := range entries {
		m := weeklyFileRe.FindStringSubmatch(e.Name())
		if e.IsDir() || m == nil {
			continue
		}
		y, _ := strconv.Atoi(m[1])
		w, _ := strconv.Atoi(m[2])
		if key := y*100 + w; key < current {
			past = append(past, edition{key: key, name: e.Name()})
		}
	}
	sort.Slice(past, func(i, j int) bool { return past[i].key < past[j].key })

	cov := &PreviousCoverage{CoveredK8sMinors: map[int]bool{}}
	for i, ed := range past {
		data, err := os.ReadFile(filepath.Join(dir, ed.name))
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", ed.name, err)
		}
		content := string(data)
		for _, m := range k8sMinorReleaseLinkRe.FindAllStringSubmatch(content, -1) {
			if minor, err := strconv.Atoi(m[1]); err == nil {
				cov.CoveredK8sMinors[minor] = true
			}
		}
		if i >= len(past)-n {
			summary := strings.TrimSpace(extractSection(content, "Welcome") + "\n\n" +
				extractSection(content, "This Week in Cloud Native"))
			if summary != "" {
				cov.Editions = append(cov.Editions, PreviousEdition{Name: ed.name, Summary: summary})
			}
		}
	}
	return cov, nil
}

// extractSection returns the body of the first "## ..." heading containing
// title, up to the next "## " heading.
func extractSection(content, title string) string {
	lines := strings.Split(content, "\n")
	var b strings.Builder
	in := false
	for _, l := range lines {
		if strings.HasPrefix(l, "## ") {
			if in {
				break
			}
			in = strings.Contains(l, title)
			continue
		}
		if in {
			b.WriteString(l)
			b.WriteString("\n")
		}
	}
	return strings.TrimSpace(b.String())
}

// FilterRepeatedReleaseNews drops follow-up articles about a Kubernetes minor
// release that an earlier edition already covered (e.g. the weekly
// "Kubernetes v1.37: <feature>" deep-dive posts published for weeks after the
// release). Patch releases (v1.37.1+) are kept, as they are new events.
func FilterRepeatedReleaseNews(news []models.NewsItem, cov *PreviousCoverage) (kept, dropped []models.NewsItem) {
	if cov == nil || len(cov.CoveredK8sMinors) == 0 {
		return news, nil
	}
	for _, n := range news {
		if isRepeatedK8sReleaseTitle(n.Title, cov.CoveredK8sMinors) {
			dropped = append(dropped, n)
			continue
		}
		kept = append(kept, n)
	}
	return kept, dropped
}

func isRepeatedK8sReleaseTitle(title string, covered map[int]bool) bool {
	m := k8sVersionTitleRe.FindStringSubmatch(title)
	if m == nil {
		return false
	}
	if m[2] != "" && m[2] != "0" {
		return false
	}
	minor, err := strconv.Atoi(m[1])
	return err == nil && covered[minor]
}

// formatPreviousCoverage renders the previous editions for the prompt.
func formatPreviousCoverage(cov *PreviousCoverage) string {
	if cov == nil || len(cov.Editions) == 0 {
		return "(no previous editions available)\n"
	}
	var b strings.Builder
	for _, e := range cov.Editions {
		fmt.Fprintf(&b, "\n### %s\n%s\n", e.Name, truncateText(e.Summary, 4000))
	}
	return b.String()
}
