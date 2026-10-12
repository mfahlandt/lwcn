package news

import (
	"testing"

	"github.com/mfahlandt/lwcn/internal/models"
)

func TestNewsFilter(t *testing.T) {
	f := NewNewsFilter(FilterOptions{
		ExcludeKeywords:   []string{"AWS", "Amazon", "EKS"},
		OSSKeywords:       []string{"CNCF"},
		ProjectNames:      []string{"Kubernetes", "Kyverno", "Open Policy Agent (OPA)", "kind"},
		ProjectNameIgnore: []string{"Kubernetes", "kind"},
	})

	cases := []struct {
		item models.NewsItem
		keep bool
	}{
		// Vendor feeds: need an open-source signal.
		{models.NewsItem{Title: "Implement per-pod image pull permissions on Amazon EKS", Vendor: true}, false},
		{models.NewsItem{Title: "Running Kubernetes at the edge with Amazon EKS", Description: "Kubernetes everywhere", Vendor: true}, false},
		{models.NewsItem{Title: "Automating proxy injection with Kyverno on EKS Fargate", Vendor: true}, true},
		{models.NewsItem{Title: "Introducing TOLAP", Description: "An open-source access control library", Vendor: true}, true},
		{models.NewsItem{Title: "Policy as code with OPA", Vendor: true}, true},
		{models.NewsItem{Title: "Test locally with kind", Vendor: true}, false},
		// General sources: vendor keyword drops unless open-source signal.
		{models.NewsItem{Title: "AWS CloudWatch: KI soll bei Incidents mit ermitteln"}, false},
		{models.NewsItem{Title: "AWS open-sources Strands harness"}, true},
		{models.NewsItem{Title: "Laws of distributed systems"}, true},
		{models.NewsItem{Title: "Kubernetes v1.37: Tracking PVC usage"}, true},
	}
	for _, c := range cases {
		kept, _ := f.Apply([]models.NewsItem{c.item})
		if got := len(kept) == 1; got != c.keep {
			t.Errorf("%q (vendor=%v): keep=%v, want %v", c.item.Title, c.item.Vendor, got, c.keep)
		}
	}
}

func TestNewsFilterEmpty(t *testing.T) {
	f := NewNewsFilter(FilterOptions{})
	kept, dropped := f.Apply([]models.NewsItem{{Title: "AWS news"}, {Title: "vendor", Vendor: true}})
	if len(kept) != 1 || len(dropped) != 1 {
		t.Errorf("kept=%v dropped=%v", kept, dropped)
	}
}
