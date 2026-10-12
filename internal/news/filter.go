package news

import (
	"regexp"
	"strings"

	"github.com/mfahlandt/lwcn/internal/models"
)

// FilterOptions configures the open-source / vendor news filter.
type FilterOptions struct {
	// ExcludeKeywords: title terms marking vendor news (dropped unless the
	// item carries an open-source signal).
	ExcludeKeywords []string
	// OSSKeywords: terms in title or description that mark open-source news.
	OSSKeywords []string
	// ProjectNames: tracked open-source project names; a match in the TITLE
	// counts as open-source signal.
	ProjectNames []string
	// ProjectNameIgnore: project names too generic to count as a signal.
	ProjectNameIgnore []string
}

// NewsFilter keeps LWCN focused on open source:
//   - items from vendor feeds are kept only with an open-source signal
//   - items from feeds requiring OSS are also kept only with an open-source signal
//   - other items whose title hits an exclude keyword are dropped unless they
//     carry an open-source signal
type NewsFilter struct {
	exclude  *regexp.Regexp
	ossText  *regexp.Regexp
	ossTitle *regexp.Regexp
}

// DroppedItem is a filtered-out news item plus the reason.
type DroppedItem struct {
	Item   models.NewsItem
	Reason string
}

func NewNewsFilter(opts FilterOptions) *NewsFilter {
	ignore := map[string]bool{}
	for _, n := range opts.ProjectNameIgnore {
		ignore[strings.ToLower(strings.TrimSpace(n))] = true
	}
	var names []string
	for _, n := range opts.ProjectNames {
		for _, alias := range projectAliases(n) {
			if len(alias) >= 3 && !ignore[strings.ToLower(alias)] {
				names = append(names, alias)
			}
		}
	}
	return &NewsFilter{
		exclude:  compileTerms(opts.ExcludeKeywords),
		ossText:  compileTerms(opts.OSSKeywords),
		ossTitle: compileTerms(names),
	}
}

// projectAliases splits "Open Policy Agent (OPA)" into both names.
func projectAliases(name string) []string {
	name = strings.TrimSpace(name)
	if i := strings.Index(name, "("); i > 0 && strings.HasSuffix(name, ")") {
		return []string{strings.TrimSpace(name[:i]), strings.TrimSpace(name[i+1 : len(name)-1])}
	}
	return []string{name}
}

func compileTerms(terms []string) *regexp.Regexp {
	var parts []string
	for _, t := range terms {
		if t = strings.TrimSpace(t); t != "" {
			parts = append(parts, regexp.QuoteMeta(t))
		}
	}
	if len(parts) == 0 {
		return nil
	}
	return regexp.MustCompile(`(?i)\b(?:` + strings.Join(parts, "|") + `)\b`)
}

func matches(re *regexp.Regexp, s string) bool {
	return re != nil && re.MatchString(s)
}

// builtinOSSRe matches all spellings of "open source" (open-source,
// open sourced, opensourcing, ...), so the config cannot miss a variant.
var builtinOSSRe = regexp.MustCompile(`(?i)\bopen[- ]?sourc(?:e|ed|es|ing)\b`)

// HasOSSSignal reports whether the item is recognizably about open source.
func (f *NewsFilter) HasOSSSignal(item models.NewsItem) bool {
	text := item.Title + "\n" + item.Description
	return builtinOSSRe.MatchString(text) || matches(f.ossText, text) || matches(f.ossTitle, item.Title)
}

// Apply splits items into kept and dropped.
func (f *NewsFilter) Apply(items []models.NewsItem) (kept []models.NewsItem, dropped []DroppedItem) {
	for _, item := range items {
		oss := f.HasOSSSignal(item)
		switch {
		case item.Vendor && !oss:
			dropped = append(dropped, DroppedItem{item, "vendor feed without open-source signal"})
		case item.RequireOSS && !oss:
			dropped = append(dropped, DroppedItem{item, "feed requires open-source signal"})
		case !item.Vendor && !oss && matches(f.exclude, item.Title):
			dropped = append(dropped, DroppedItem{item, "vendor keyword in title"})
		default:
			kept = append(kept, item)
		}
	}
	return kept, dropped
}
