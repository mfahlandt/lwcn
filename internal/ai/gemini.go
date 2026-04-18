package ai

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/generative-ai-go/genai"
	"github.com/mfahlandt/lwcn/internal/models"
	"google.golang.org/api/option"
)

// DefaultAPITimeout is the default timeout for Gemini API calls
const DefaultAPITimeout = 5 * time.Minute

type GeminiClient struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

func NewGeminiClient(ctx context.Context, apiKey string) (*GeminiClient, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	// Use gemini-2.0-flash - the latest fast and cost-effective model
	model := client.GenerativeModel("gemini-2.5-flash")
	model.SetTemperature(0.7)
	model.SetTopP(0.9)

	return &GeminiClient{
		client: client,
		model:  model,
	}, nil
}

func (c *GeminiClient) Close() error {
	return c.client.Close()
}

func (c *GeminiClient) GenerateLinkedInPost(ctx context.Context, releases []models.Release, news []models.NewsItem) (string, error) {
	// Create a context with timeout for the API call
	apiCtx, cancel := context.WithTimeout(ctx, DefaultAPITimeout)
	defer cancel()

	prompt := buildLinkedInPrompt(releases, news)

	resp, err := c.model.GenerateContent(apiCtx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate LinkedIn post: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content generated")
	}

	content := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	return content, nil
}

// GenerateLinkedInShortPost creates a short teaser post. It is derived from the
// already-generated long LinkedIn newsletter post so that the highlights in the
// teaser match the content of the long article. Releases/news are passed only
// as fallback context.
func (c *GeminiClient) GenerateLinkedInShortPost(ctx context.Context, longPost string, releases []models.Release, news []models.NewsItem) (string, error) {
	apiCtx, cancel := context.WithTimeout(ctx, DefaultAPITimeout)
	defer cancel()

	prompt := buildLinkedInShortPrompt(longPost, releases, news)

	resp, err := c.model.GenerateContent(apiCtx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate LinkedIn short post: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content generated")
	}

	content := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	return content, nil
}

func (c *GeminiClient) GenerateNewsletter(ctx context.Context, releases []models.Release, news []models.NewsItem) (*models.Newsletter, error) {
	// Create a context with timeout for the API call
	apiCtx, cancel := context.WithTimeout(ctx, DefaultAPITimeout)
	defer cancel()

	prompt := buildPrompt(releases, news)

	resp, err := c.model.GenerateContent(apiCtx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content generated")
	}

	content := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])

	newsletter := &models.Newsletter{
		Content:   content,
		Releases:  releases,
		NewsItems: news,
	}

	return newsletter, nil
}

func buildPrompt(releases []models.Release, news []models.NewsItem) string {
	// Filter out pre-releases (RC, alpha, beta, test)
	stableReleases := filterStableReleases(releases)

	prompt := `You are a technical writer creating a weekly Cloud Native newsletter called "Last Week in Cloud Native" (LWCN).

Write a newsletter in Markdown format based on the following releases and news items.

IMPORTANT GUIDELINES:
1. Write in ENGLISH
2. DO NOT list individual news articles - instead, write a SUMMARY of what happened this week
3. Group news into themes/topics (e.g., "AI & Cloud Native", "Security Updates", "Kubernetes Ecosystem")
4. For releases: Only include the releases provided (already filtered to stable releases only)
5. Keep the tone professional but engaging
6. Use emojis sparingly for section headers
7. DO NOT wrap the output in markdown code blocks - output raw markdown directly
8. IMPORTANT: For each release, include a LINK to the release using the provided URL

STRUCTURE:

## 👋 Welcome
A brief 2-3 sentence intro summarizing the week's highlights.

## 🚀 Notable Releases
Group by category. For each release, use this format WITH A LINK:
- **[Project Name vX.Y.Z](RELEASE_URL)** - What's new in 1-2 sentences

Example:
- **[Cilium v1.18.6](https://github.com/cilium/cilium/releases/tag/v1.18.6)** - Publishes Helm charts to OCI registries.

## 📰 This Week in Cloud Native
Write 3-5 paragraphs summarizing the major themes and news from this week. DO NOT list individual articles.
Group related news into coherent narratives about:
- Major announcements and product launches
- Industry trends and developments  
- Security news and vulnerabilities
- Community and ecosystem updates

## 💬 Community Buzz
Summarize interesting discussions from Hacker News related to cloud native topics.
Write 2-3 sentences about what the community is talking about.

## 📊 Week in Numbers
- X stable releases across Y projects
- Key stats from the news

DO NOT add any "View all articles" link - this will be added automatically.

---

STABLE RELEASES (pre-filtered, no RC/alpha/beta):
`

	for _, r := range stableReleases {
		// Sanitize all fields to remove invalid UTF-8 characters
		body := sanitizeUTF8(r.Body)
		name := sanitizeUTF8(r.Name)
		prompt += fmt.Sprintf("\n- %s/%s %s (%s)\n  URL: %s\n  Name: %s\n  Notes: %s\n",
			r.RepoOwner, r.RepoName, r.TagName, r.Category, r.URL, name, truncateText(body, 500))
	}

	prompt += fmt.Sprintf("\nTotal: %d stable releases\n", len(stableReleases))

	prompt += "\n\nNEWS ITEMS (use these to write summaries, DO NOT list them individually):\n"

	for _, n := range news {
		title := sanitizeUTF8(n.Title)
		desc := sanitizeUTF8(n.Description)
		prompt += fmt.Sprintf("\n- [%s] %s\n  URL: %s\n  Description: %s\n",
			n.Source, title, n.URL, truncateText(desc, 200))
	}

	prompt += "\n\nGenerate the newsletter content in Markdown (no code blocks, raw markdown only):"

	// Final sanitization of entire prompt
	return sanitizeUTF8(prompt)
}

// filterStableReleases removes release candidates, alpha, beta, and test releases
func filterStableReleases(releases []models.Release) []models.Release {
	var stable []models.Release
	var filtered []string
	for _, r := range releases {
		// Only filter based on tag name, not GitHub's IsPrerelease flag
		// Some projects mark patch releases as prerelease incorrectly
		if !isPreRelease(r.TagName) {
			stable = append(stable, r)
		} else {
			filtered = append(filtered, fmt.Sprintf("%s/%s %s", r.RepoOwner, r.RepoName, r.TagName))
		}
	}
	log.Printf("Filtered %d pre-releases, keeping %d stable releases", len(filtered), len(stable))
	if len(filtered) > 0 {
		log.Printf("Filtered releases: %v", filtered)
	}
	return stable
}

// isPreRelease checks if a version tag indicates a pre-release
func isPreRelease(tag string) bool {
	tagLower := strings.ToLower(tag)
	preReleaseIndicators := []string{
		"-rc", "-alpha", "-beta", "-test", "-dev", "-preview", "-pre",
		"-next", "-canary", "-nightly", "-snapshot",
		"alpha.", "beta.", "test.", "dev.", "preview.",
		"edge-", // Linkerd edge releases
	}
	for _, indicator := range preReleaseIndicators {
		if strings.Contains(tagLower, indicator) {
			return true
		}
	}

	// Check for patterns like "rc1", "rc.1", "alpha1", etc. at the end
	suffixPatterns := []string{
		"rc", "alpha", "beta", "test", "dev", "preview", "pre", "next",
	}
	for _, pattern := range suffixPatterns {
		// Match patterns like -rc1, -rc.1, .rc1, .rc.1
		if strings.Contains(tagLower, "."+pattern) || strings.Contains(tagLower, "-"+pattern) {
			return true
		}
	}

	return false
}

func truncateText(s string, max int) string {
	// First sanitize the text to remove invalid UTF-8
	s = sanitizeUTF8(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// sanitizeUTF8 removes invalid UTF-8 characters from a string
func sanitizeUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	// Replace invalid UTF-8 sequences with empty string
	v := make([]rune, 0, len(s))
	for i, r := range s {
		if r == utf8.RuneError {
			_, size := utf8.DecodeRuneInString(s[i:])
			if size == 1 {
				continue // skip invalid byte
			}
		}
		v = append(v, r)
	}
	return string(v)
}

func buildLinkedInPrompt(releases []models.Release, news []models.NewsItem) string {
	stableReleases := filterStableReleases(releases)

	now := time.Now()
	year, week := now.ISOWeek()
	newsletterURL := fmt.Sprintf("https://lwcn.dev/newsletter/%d-week-%02d/", year, week)

	prompt := `You are writing a LinkedIn newsletter article for a Cloud Native expert and Kubernetes enthusiast.
This will be published as a LinkedIn Newsletter article, so it can be slightly longer and more detailed than a regular post.

WRITING STYLE (match this tone and format):
- Start with an emoji and attention-grabbing headline about the week's biggest story
- Use storytelling and metaphors (like Norse mythology, nature themes, tech analogies)
- Be personal ("On a personal note...", "This is a massive win...") but technical
- Structure with emojis as section headers (🚀, 🧠, 🔄, 🛡️, ⚠️)
- Include technical depth with focus but keep it accessible 
- Add a call-to-action question at the end
- End with relevant hashtags: #Kubernetes #CloudNative #OpenSource #K8s #DevOps

EXAMPLE STYLE (from previous posts):
"🌳 Kubernetes 1.35 is released and is called "Timbernetes"!
When you look at the Logo, it shows the World Tree Yggdrasil. If you are not familiar with Norse mythology, Yggdrasil is the immense, sacred ash tree..."

"🚀 The biggest feature moving to Stable is without doubt In-Place Pod Resizing. You can now adjust CPU and memory resources for running Pods without restarting them. No more disruption for vertical scaling. This is a massive win for stateful workloads!"

STRUCTURE:
1. Hook with emoji + main headline (biggest news of the week)
2. Creative storytelling/context about why this matters
3. 🚀 Key releases with explanations of whats the newest and most relevant features
4. 📰 Notable news/trends summary (2-4 sentences)
5. 💬 Community highlights (if interesting)
6. Personal take or call-to-action question
7. A line saying: "📖 Read the full newsletter with all releases and articles: NEWSLETTER_URL"
8. Hashtags

IMPORTANT:
- This is for a LinkedIn NEWSLETTER so it can be up to 4000 characters
- Focus on 3-5 most impactful releases/news items
- Be enthusiastic but authentic and not over the top
- Do not use the tree metapher from the example create something unique that fits the news
- NO markdown formatting (no ** or ## ) - plain text with emojis only
- do not repeat the highlight topic in the notable news/trends block take a different topic there 
- the personal note should also not duplicate topics
- DO NOT Have duplications in the topics - no repeat in the blocks
- MUST include at the bottom before the hashtags: "📖 Read the full newsletter with all releases and articles: NEWSLETTER_URL"

Replace NEWSLETTER_URL with: ` + newsletterURL + `

---

THIS WEEK'S RELEASES:
`

	// Group releases by category and pick top ones
	releasesByCategory := make(map[string][]models.Release)
	for _, r := range stableReleases {
		releasesByCategory[r.Category] = append(releasesByCategory[r.Category], r)
	}

	for category, releases := range releasesByCategory {
		prompt += fmt.Sprintf("\n%s:\n", strings.ToUpper(category))
		for _, r := range releases {
			body := sanitizeUTF8(r.Body)
			prompt += fmt.Sprintf("- %s %s: %s\n", r.RepoName, r.TagName, truncateText(body, 300))
		}
	}

	prompt += fmt.Sprintf("\nTotal: %d stable releases this week\n", len(stableReleases))

	prompt += "\n\nKEY NEWS ITEMS:\n"

	// Only include non-HN news for LinkedIn (more professional sources)
	newsCount := 0
	for _, n := range news {
		if n.Source != "Hacker News" && newsCount < 15 {
			title := sanitizeUTF8(n.Title)
			prompt += fmt.Sprintf("- [%s] %s\n", n.Source, title)
			newsCount++
		}
	}

	prompt += "\n\nGenerate the LinkedIn newsletter article (plain text, no markdown, include the website link at the bottom):"

	// Final sanitization of entire prompt
	return sanitizeUTF8(prompt)
}

func buildLinkedInShortPrompt(longPost string, releases []models.Release, news []models.NewsItem) string {
	stableReleases := filterStableReleases(releases)

	now := time.Now()
	year, week := now.ISOWeek()

	prompt := fmt.Sprintf(`You are writing a SHORT LinkedIn teaser post to promote this week's edition of the "Last Week in Cloud Native" (LWCN) LinkedIn newsletter.

The teaser MUST be based on the long LinkedIn newsletter article provided below. Pick the SAME 2-3 top highlights that the long article focuses on (hook/main story + 2-3 key points). Do NOT introduce different topics.

RULES:
- Maximum 500 characters (very short!)
- Start with an emoji and a catchy one-liner matching the long article's hook
- Mention 2-3 key highlights in bullet points with emojis (use the SAME highlights as the long article)
- End with a call-to-action to read the full newsletter
- End with the text: "👉 Link in the comments!" or "👉 Check out the latest edition in my newsletter!"
- NO markdown formatting - plain text with emojis only
- NO hashtags in this short version (save space)
- Be enthusiastic but concise
- This is Week %d of %d

EXAMPLE:
"🚀 Kubernetes 1.35 just dropped!

Key highlights this week:
⚡ In-Place Pod Resizing goes stable
🛡️ Critical Envoy security patches
📦 Harbor 2.15 with smarter garbage collection

All the details in this week's Last Week in Cloud Native newsletter! 👉 Check it out!"

---

LONG LINKEDIN NEWSLETTER ARTICLE (summarize THIS into the teaser, keep the same focus topics):
"""
%s
"""

`, week, year, sanitizeUTF8(longPost))

	// Provide brief context as fallback/reference only
	prompt += fmt.Sprintf("\nREFERENCE CONTEXT (only use if the long article is empty): %d stable releases this week.\n", len(stableReleases))
	count := 0
	for _, r := range stableReleases {
		if count >= 5 {
			break
		}
		prompt += fmt.Sprintf("- %s %s (%s)\n", r.RepoName, r.TagName, r.Category)
		count++
	}
	newsCount := 0
	for _, n := range news {
		if n.Source != "Hacker News" && newsCount < 3 {
			prompt += fmt.Sprintf("- %s\n", sanitizeUTF8(n.Title))
			newsCount++
		}
	}

	prompt += "\nGenerate the short LinkedIn teaser post (plain text, under 500 characters, same highlights as the long article):"

	return sanitizeUTF8(prompt)
}
