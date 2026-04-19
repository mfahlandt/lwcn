package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/generative-ai-go/genai"
	"github.com/mfahlandt/lwcn/internal/models"
	"google.golang.org/api/option"
)

// DefaultAPITimeout is the default timeout for Gemini API calls
const DefaultAPITimeout = 5 * time.Minute

// neutralityPolicy is a shared, strict editorial policy injected into every
// Gemini prompt. It enforces a neutral, fact-driven, journalistic tone and
// explicitly forbids marketing language, vendor pitches and subjective opinions
// from blog posts or community members. This is intentionally prescriptive so
// the output stays consistent across newsletter and LinkedIn formats.
const neutralityPolicy = `
STRICT EDITORIAL / NEUTRALITY POLICY (MANDATORY, APPLIES TO EVERY SECTION):

You are acting as a NEUTRAL TECHNICAL JOURNALIST, not as a marketer, evangelist
or vendor. Your output MUST read like a factual changelog / tech news brief.

1. NO MARKETING SPEAK. Do NOT use (or translate equivalents of) hype words such as:
   "groundbreaking", "revolutionary", "innovative", "game-changing",
   "cutting-edge", "next-generation", "world-class", "best-in-class",
   "seamless", "effortless", "powerful", "amazing", "exciting",
   "unlock", "supercharge", "empower", "delight", "leverage",
   "mission-critical", "enterprise-grade", "industry-leading",
   "blazing-fast", "lightning-fast", "state-of-the-art", "paradigm shift".
   Also avoid vague superlatives ("the best", "the most advanced", "massive win").

2. NO VENDOR PITCHES. Do NOT copy promotional phrasing from release notes,
   company blogs, press releases or sponsored posts. Strip out any "why you
   should use X" framing. Do not advocate for products.

3. FACT FOCUS ONLY. For every release/news item, extract only:
   - WHAT CHANGED: features added, bugs fixed, deprecations, removals,
     breaking changes, CVEs/security fixes, performance numbers with units,
     API changes, default behavior changes.
   - WHAT IT MEANS TECHNICALLY: a short, concrete technical implication
     (e.g. "reduces control-plane memory on large clusters", "breaks clients
     using the v1beta1 API", "requires Go 1.22+"). Keep it verifiable.

4. NO OPINIONS. Ignore subjective takes, hot takes, predictions, sentiment
   or editorializing from blog posts, Hacker News threads, or community
   members. Do NOT write "I think", "we believe", "this is great",
   "this is disappointing", "the community loves", "users will enjoy".
   Report what was said or changed, not whether it is good or bad.

5. ATTRIBUTE CLAIMS. If a non-factual statement must be included (e.g. a
   roadmap intent), attribute it: "The maintainers state that ...",
   "According to the release notes, ...". Never present opinion as fact.

6. TONE: concise, neutral, precise, past tense for events, present tense for
   behavior. Prefer verbs like "adds", "removes", "deprecates", "fixes",
   "changes the default", "introduces", "requires".

7. If you are unsure whether something is a fact or a pitch, OMIT IT.
`

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

// SocialShorts bundles all three short-format posts produced in a single
// Gemini call. Consolidating into one request saves free-tier quota
// (1 call instead of 3) and keeps all shorts consistent with each other.
type SocialShorts struct {
	LinkedInShort string `json:"linkedin_short"`
	Tweet         string `json:"tweet"`
	Bluesky       string `json:"bluesky"`
}

// GenerateSocialShorts produces the LinkedIn teaser, the tweet, and the
// Bluesky post in ONE Gemini call, all derived from the long LinkedIn article.
// The model returns strict JSON which is parsed into SocialShorts.
func (c *GeminiClient) GenerateSocialShorts(ctx context.Context, longPost, newsletterURL string) (*SocialShorts, error) {
	apiCtx, cancel := context.WithTimeout(ctx, DefaultAPITimeout)
	defer cancel()

	prompt := buildCombinedShortsPrompt(longPost, newsletterURL)
	resp, err := c.model.GenerateContent(apiCtx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to generate social shorts: %w", err)
	}
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content generated")
	}
	raw := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])

	jsonStr := extractJSON(raw)
	var out SocialShorts
	if err := json.Unmarshal([]byte(jsonStr), &out); err != nil {
		return nil, fmt.Errorf("failed to parse social shorts JSON: %w\nraw output:\n%s", err, raw)
	}
	out.LinkedInShort = strings.TrimSpace(out.LinkedInShort)
	out.Tweet = strings.TrimSpace(out.Tweet)
	out.Bluesky = strings.TrimSpace(out.Bluesky)
	if out.LinkedInShort == "" || out.Tweet == "" || out.Bluesky == "" {
		return nil, fmt.Errorf("social shorts JSON missing fields\nraw output:\n%s", raw)
	}
	return &out, nil
}

// extractJSON tolerates responses wrapped in markdown code fences or with
// leading/trailing prose, and returns the first {...} block.
func extractJSON(s string) string {
	// Strip ```json ... ``` fences if present
	fence := regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)```")
	if m := fence.FindStringSubmatch(s); len(m) == 2 {
		s = m[1]
	}
	// Take first balanced {...} block
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end > start {
		return s[start : end+1]
	}
	return strings.TrimSpace(s)
}

// GenerateLinkedInPost generates the long-form LinkedIn newsletter article.
// This is the ONE "creative" call; the three short formats are produced from
// its output in a single combined call via GenerateSocialShorts.
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

// GenerateLinkedInShortPost is deprecated; use GenerateSocialShorts instead.
// Kept only so external callers (tests, older scripts) keep compiling.
func (c *GeminiClient) GenerateLinkedInShortPost(ctx context.Context, longPost string, _ []models.Release, _ []models.NewsItem) (string, error) {
	s, err := c.GenerateSocialShorts(ctx, longPost, "https://lwcn.dev/")
	if err != nil {
		return "", err
	}
	return s.LinkedInShort, nil
}

func (c *GeminiClient) GenerateNewsletter(ctx context.Context, releases []models.Release, news []models.NewsItem, stats []models.RepoStats) (*models.Newsletter, error) {
	// Create a context with timeout for the API call
	apiCtx, cancel := context.WithTimeout(ctx, DefaultAPITimeout)
	defer cancel()

	prompt := buildPrompt(releases, news, stats)

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

func buildPrompt(releases []models.Release, news []models.NewsItem, stats []models.RepoStats) string {
	// Filter out pre-releases (RC, alpha, beta, test)
	stableReleases := filterStableReleases(releases)

	prompt := `You are a technical writer creating a weekly Cloud Native newsletter called "Last Week in Cloud Native" (LWCN).

Write a newsletter in Markdown format based on the following releases and news items.
` + neutralityPolicy + `
IMPORTANT GUIDELINES:
1. Write in ENGLISH
2. DO NOT list individual news articles - instead, write a SUMMARY of what happened this week
3. Group news into themes/topics (e.g., "AI & Cloud Native", "Security Updates", "Kubernetes Ecosystem")
4. For releases: Only include the releases provided (already filtered to stable releases only)
5. Keep the tone professional, neutral and journalistic (see editorial policy above)
6. Use emojis sparingly for section headers
7. DO NOT wrap the output in markdown code blocks - output raw markdown directly
8. IMPORTANT: For each release, include a LINK to the release using the provided URL
9. For the news summary sections, report WHAT HAPPENED and technical implications only — no opinions, no hype, no vendor pitches
10. DO NOT insert sponsored, partner, promotional, advertising or "brought to you by" content of any kind. Do NOT add "[Sponsored]", "[Partner]", "Ad:", "Promoted:", "Sponsor:" tags, shortcodes ({{< sponsored ... >}}), or any block framed as paid placement. Sponsored/partner snippets are added post-generation by a human editor in a separate, clearly labeled block — NEVER by you.

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
Summarize which cloud native topics were discussed on Hacker News this week.
Report the TOPICS and FACTUAL SUBJECTS of the discussions only — do NOT repeat
opinions, hot takes, sentiment, praise or criticism from commenters. 2-3 sentences max.

## 📊 Numbers of the Week
Render the neutral, data-driven metrics provided in the "REPO ACTIVITY STATS" section below.
Use EXACTLY the numbers given — do NOT invent, estimate or round them. If a section's data is
empty, omit that sub-list. Format:

- Total stable releases: X across Y projects (computed from the releases list)
- Top 3 projects by commits this week:
  1. owner/repo — N commits
  2. owner/repo — N commits
  3. owner/repo — N commits
- Top 3 projects by merged pull requests this week:
  1. owner/repo — N merged PRs
  2. owner/repo — N merged PRs
  3. owner/repo — N merged PRs

No commentary, no ranking adjectives ("leading", "dominant", "busy") — just the numbers.

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

	// Inject neutral, pre-computed activity metrics for the "Numbers of the Week" section.
	prompt += "\n\nREPO ACTIVITY STATS (authoritative numbers — use EXACTLY these values, do not modify):\n"
	prompt += formatStatsForPrompt(stats)

	prompt += "\n\nGenerate the newsletter content in Markdown (no code blocks, raw markdown only):"

	// Final sanitization of entire prompt
	return sanitizeUTF8(prompt)
}

// formatStatsForPrompt renders the collected neutral metrics into a compact,
// deterministic block. It emits the top projects by commits and by merged PRs
// so the model can render them verbatim into the "Numbers of the Week" section.
func formatStatsForPrompt(stats []models.RepoStats) string {
	if len(stats) == 0 {
		return "(no stats collected — omit the top-3 sub-lists in the 'Numbers of the Week' section)\n"
	}

	byCommits := make([]models.RepoStats, len(stats))
	copy(byCommits, stats)
	sort.Slice(byCommits, func(i, j int) bool { return byCommits[i].Commits > byCommits[j].Commits })

	byMerged := make([]models.RepoStats, len(stats))
	copy(byMerged, stats)
	sort.Slice(byMerged, func(i, j int) bool { return byMerged[i].MergedPRs > byMerged[j].MergedPRs })

	var b strings.Builder
	b.WriteString("Top projects by commits (use these exact numbers):\n")
	for i, s := range byCommits {
		if i >= 3 || s.Commits == 0 {
			break
		}
		fmt.Fprintf(&b, "  %d. %s/%s — %d commits\n", i+1, s.RepoOwner, s.RepoName, s.Commits)
	}
	b.WriteString("Top projects by merged pull requests (use these exact numbers):\n")
	for i, s := range byMerged {
		if i >= 3 || s.MergedPRs == 0 {
			break
		}
		fmt.Fprintf(&b, "  %d. %s/%s — %d merged PRs\n", i+1, s.RepoOwner, s.RepoName, s.MergedPRs)
	}
	return b.String()
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

	prompt := `You are writing a LinkedIn newsletter article that summarizes this week in the Cloud Native / Kubernetes ecosystem.
This will be published as a LinkedIn Newsletter article, so it can be slightly longer and more detailed than a regular post.
` + neutralityPolicy + `
WRITING STYLE:
- Neutral, journalistic, fact-first. Treat this as a technical news digest, NOT a personal blog or marketing post.
- Start with an emoji and a FACTUAL headline stating the week's biggest concrete event
  (e.g. a named release, a CVE, a deprecation). No hype adjectives.
- Do NOT use storytelling framing, mythology, metaphors, analogies or personal anecdotes.
- Do NOT use phrases like "On a personal note", "massive win", "game changer", "exciting", "huge".
- Structure with emojis as section headers (🚀, 🧠, 🔄, 🛡️, ⚠️) — emojis are fine, hype is not.
- Include technical depth: version numbers, component names, behavior changes, CVE IDs where applicable.
- End with a neutral, open question that invites technical discussion (not a marketing CTA).
- End with relevant hashtags: #Kubernetes #CloudNative #OpenSource #K8s #DevOps

STRUCTURE:
1. Hook: emoji + factual one-line headline (the single most significant concrete event of the week)
2. Short neutral context (2-3 sentences) explaining WHAT changed and the technical implication — no metaphors, no hype
3. 🚀 Key releases: list the most relevant releases with the concrete new features, bug fixes, deprecations or breaking changes
4. 📰 Notable news/trends summary (2-4 sentences) — facts only, no opinions
5. 💬 Community highlights (only if there are factual topics worth noting; report TOPICS, not sentiment)
6. Closing neutral question for discussion
7. A line saying: "📖 Read the full newsletter with all releases and articles: NEWSLETTER_URL"
8. Hashtags

IMPORTANT:
- This is for a LinkedIn NEWSLETTER so it can be up to 4000 characters
- Focus on 3-5 most impactful releases/news items, selected by technical significance (breaking changes, security, GA milestones), NOT by marketing appeal
- NO markdown formatting (no ** or ## ) - plain text with emojis only
- Do NOT duplicate topics across sections
- Do NOT copy promotional phrasing from release notes or vendor blogs
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

// buildCombinedShortsPrompt asks Gemini to produce ALL THREE short-format
// posts (LinkedIn teaser, tweet, Bluesky skeet) in ONE call and return them
// as strict JSON. Saves ~3x the API quota on the free tier vs. separate calls.
//
// All three variants are derived from the same long LinkedIn article, so the
// hook/highlights stay consistent across platforms. Each variant is given its
// own character budget (text-only, URL is appended in post-processing).
func buildCombinedShortsPrompt(longPost, newsletterURL string) string {
	now := time.Now()
	year, week := now.ISOWeek()

	return sanitizeUTF8(fmt.Sprintf(`You are producing THREE short social posts that promote this week's
"Last Week in Cloud Native" (LWCN) newsletter edition (Year %d, Week %d).

All three posts MUST be derived from the LONG LinkedIn article below and use
the SAME hook / top 2-3 highlights (same project names, same version numbers,
same CVE IDs). Do NOT introduce new topics. Do NOT contradict each other.
%s
OUTPUT FORMAT — CRITICAL:
Respond with ONLY a single JSON object, no prose, no markdown code fences.
Exact schema:

{
  "linkedin_short": "<string>",
  "tweet": "<string>",
  "bluesky": "<string>"
}

CHARACTER BUDGETS (text only — the newsletter URL will be appended AFTER the text and is NOT in the budget):

- "linkedin_short":   up to 480 chars. 3-4 bullet-style highlight lines allowed.
                      End with: "👉 Check out the latest edition in my newsletter!"
                      No URL, no hashtags.

- "tweet":            up to 230 chars. Very terse. 1-3 ultra-short lines.
                      End with a short teaser like "Full breakdown:" (URL will be appended).
                      No hashtags, no @mentions.

- "bluesky":          up to 250 chars. Slightly more room than the tweet.
                      End with a short teaser like "Full breakdown:" (URL will be appended).
                      No hashtags, no @mentions.

COUNT CHARACTERS BEFORE FINALIZING each string. If a string is over its budget, shorten it.

CONTENT RULES (apply to all three):
- Neutral, factual tone. NO hype words ("revolutionary", "game-changing",
  "massive", "huge", "exciting", "amazing", "unlock", "supercharge",
  "seamless", "powerful", "blazing-fast", etc.).
- NO markdown (no **, ##, _, backticks).
- NO @mentions, NO hashtags.
- NO opinions, NO vendor pitches.
- Start each post with ONE emoji + a FACTUAL one-liner (name the project/version/event).
- Highlight bullets (if any) must state WHAT CHANGED (feature, fix, deprecation, CVE) — not why it is "great".
- Do NOT include the newsletter URL in any of the three strings — the URL (%s) will be appended by the publishing code.
- Do NOT wrap any string in quotes or code blocks beyond what JSON requires.

EXAMPLE output (structure only — write your own content):
{
  "linkedin_short": "🚀 Kubernetes 1.35 released.\n\nHighlights this week:\n⚡ In-Place Pod Resizing promoted to stable\n🛡️ Envoy security patches (CVE fixes)\n📦 Harbor 2.15 changes default garbage collection behavior\n\nFull breakdown in this week's Last Week in Cloud Native newsletter. 👉 Check out the latest edition in my newsletter!",
  "tweet": "🚀 Kubernetes 1.35 released.\n⚡ In-Place Pod Resizing → stable\n🛡️ Envoy CVE patches\nFull breakdown:",
  "bluesky": "🚀 Kubernetes 1.35 released.\n⚡ In-Place Pod Resizing promoted to stable\n🛡️ Envoy security patches (CVE fixes)\nFull breakdown:"
}

---

LONG LINKEDIN NEWSLETTER ARTICLE (source material — derive all three shorts from this):
"""
%s
"""

Return ONLY the JSON object described above.`,
		year, week,
		neutralityPolicy,
		newsletterURL,
		longPost,
	))
}
