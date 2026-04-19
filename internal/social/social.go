// Package social publishes short posts to Bluesky and X (Twitter) when a new
// LWCN newsletter edition goes live.
//
// Design goals:
//   - stdlib only (no new dependencies)
//   - platform-specific length limits handled once, centrally
//   - graceful degradation: if a platform's credentials are missing, skip that
//     platform but continue with the others
//   - neutral content: reuses the already-generated, editorial-policy-compliant
//     LinkedIn short post as the source text
package social

import (
	"strings"
	"unicode/utf8"
)

// Post is the normalized input passed to every platform publisher.
type Post struct {
	// Text is the short post body (without the trailing URL).
	Text string
	// URL is the link to the new newsletter edition.
	URL string
}

// truncateForPlatform trims text to fit maxChars total, including a trailing
// space + URL. Uses rune-aware truncation so emoji/UTF-8 are not cut in half.
// Appends an ellipsis if the text had to be shortened.
func truncateForPlatform(text, url string, maxChars int) string {
	text = strings.TrimSpace(text)
	// Reserve space for " \nURL"
	reserve := utf8.RuneCountInString(url) + 2 // newline + space before URL
	budget := maxChars - reserve
	if budget < 20 {
		// URL alone is longer than budget — just post the URL.
		return url
	}
	if utf8.RuneCountInString(text) <= budget {
		return text + "\n\n" + url
	}
	// Rune-aware truncate
	runes := []rune(text)
	// -1 for ellipsis
	runes = runes[:budget-1]
	return strings.TrimSpace(string(runes)) + "…\n\n" + url
}
