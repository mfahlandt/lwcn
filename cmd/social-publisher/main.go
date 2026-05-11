// Command social-publisher posts a short teaser about the latest LWCN edition
// to Bluesky and X (Twitter) when a new newsletter goes live.
//
// Source text: the already-generated LinkedIn SHORT post
// (<content-dir>/YYYY-week-XX-linkedin-short.txt). That file is produced by
// ai-processor under the strict neutrality policy, so it already contains a
// concise, factual, marketing-free teaser.
//
// Credentials (all optional — a platform is skipped if its vars are unset):
//
//	BLUESKY_HANDLE         e.g. "lwcn.bsky.social"
//	BLUESKY_APP_PASSWORD
//	X_API_KEY
//	X_API_SECRET
//	X_ACCESS_TOKEN
//	X_ACCESS_SECRET
//
// Usage:
//
//	social-publisher [-content-dir website/content/newsletter] [-week N] [-year N] [-dry-run]
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/mfahlandt/lwcn/internal/social"
)

func main() {
	_ = godotenv.Load()

	contentDir := flag.String("content-dir", "website/content/newsletter", "Directory containing the linkedin-short.txt file")
	weekFlag := flag.Int("week", 0, "ISO week number (default: current ISO week)")
	yearFlag := flag.Int("year", 0, "ISO year (default: current ISO year)")
	dryRun := flag.Bool("dry-run", false, "Print what would be posted without actually posting")
	flag.Parse()

	now := time.Now()
	year, week := now.ISOWeek()
	if *yearFlag > 0 {
		year = *yearFlag
	}
	if *weekFlag > 0 {
		week = *weekFlag
	}

	shortPath := filepath.Join(*contentDir, fmt.Sprintf("%d-week-%02d-linkedin-short.txt", year, week))
	tweetPath := filepath.Join(*contentDir, fmt.Sprintf("%d-week-%02d-tweet.txt", year, week))
	blueskyPath := filepath.Join(*contentDir, fmt.Sprintf("%d-week-%02d-bluesky.txt", year, week))

	// Prefer platform-native files (already within char limits); fall back to
	// the LinkedIn short post (which the social package will truncate).
	tweetText := readFirst(tweetPath, shortPath)
	blueskyText := readFirst(blueskyPath, shortPath)
	if tweetText == "" && blueskyText == "" {
		log.Fatalf("No teaser text found. Expected one of:\n  %s\n  %s\n  %s", tweetPath, blueskyPath, shortPath)
	}

	url := fmt.Sprintf("https://lwcn.dev/newsletter/%d-week-%02d/", year, week)
	tweetPost := social.Post{Text: tweetText, URL: url}
	blueskyPost := social.Post{Text: blueskyText, URL: url}

	log.Printf("Publishing social teaser for Year %d Week %02d → %s", year, week, url)

	if *dryRun {
		fmt.Println("--- DRY RUN: X / Twitter ---")
		fmt.Println(tweetPost.Text)
		fmt.Println(">>>", tweetPost.URL)
		fmt.Println("--- DRY RUN: Bluesky ---")
		fmt.Println(blueskyPost.Text)
		fmt.Println(">>>", blueskyPost.URL)
		fmt.Println("--- end ---")
		return
	}

	publishBluesky(blueskyPost)
	publishX(tweetPost)
}

// readFirst returns the content of the first existing, non-empty file.
func readFirst(paths ...string) string {
	for _, p := range paths {
		if p == "" {
			continue
		}
		if b, err := os.ReadFile(p); err == nil && len(b) > 0 {
			return string(b)
		}
	}
	return ""
}

func publishBluesky(p social.Post) {
	handle := os.Getenv("BLUESKY_HANDLE")
	pw := os.Getenv("BLUESKY_APP_PASSWORD")
	if handle == "" || pw == "" {
		log.Printf("Bluesky: credentials not set — skipping")
		return
	}
	client := &social.BlueskyClient{Handle: handle, AppPassword: pw}
	if err := client.Post(p); err != nil {
		log.Printf("Bluesky: post failed: %v", err)
		return
	}
	log.Printf("Bluesky: post published ✅")
}

func publishX(p social.Post) {
	// Explicit opt-out without having to delete the X_* secrets. Useful while
	// the X Free tier has no posting credits (HTTP 402 CreditsDepleted) and
	// you only want to post to Bluesky.
	if strings.EqualFold(os.Getenv("X_ENABLED"), "false") {
		log.Printf("X: disabled via X_ENABLED=false — skipping")
		return
	}
	key := os.Getenv("X_API_KEY")
	secret := os.Getenv("X_API_SECRET")
	token := os.Getenv("X_ACCESS_TOKEN")
	tokenSecret := os.Getenv("X_ACCESS_SECRET")
	if key == "" || secret == "" || token == "" || tokenSecret == "" {
		log.Printf("X: credentials not set — skipping")
		return
	}
	client := &social.XClient{APIKey: key, APISecret: secret, AccessToken: token, AccessSecret: tokenSecret}
	if err := client.Post(p); err != nil {
		log.Printf("X: post failed: %v", err)
		return
	}
	log.Printf("X: tweet published ✅")
}
