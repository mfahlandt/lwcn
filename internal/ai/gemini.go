package ai

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/mfahlandt/lwcn/internal/models"
	"google.golang.org/api/option"
)

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
	prompt := buildLinkedInPrompt(releases, news)

	resp, err := c.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", fmt.Errorf("failed to generate LinkedIn post: %w", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content generated")
	}

	content := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])
	return content, nil
}

func (c *GeminiClient) GenerateNewsletter(ctx context.Context, releases []models.Release, news []models.NewsItem) (*models.Newsletter, error) {
	prompt := buildPrompt(releases, news)

	resp, err := c.model.GenerateContent(ctx, genai.Text(prompt))
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
		prompt += fmt.Sprintf("\n- %s/%s %s (%s)\n  URL: %s\n  Notes: %s\n",
			r.RepoOwner, r.RepoName, r.TagName, r.Category, r.URL, truncateText(r.Body, 500))
	}

	prompt += fmt.Sprintf("\nTotal: %d stable releases\n", len(stableReleases))

	prompt += "\n\nNEWS ITEMS (use these to write summaries, DO NOT list them individually):\n"

	for _, n := range news {
		prompt += fmt.Sprintf("\n- [%s] %s\n  URL: %s\n  Description: %s\n",
			n.Source, n.Title, n.URL, truncateText(n.Description, 200))
	}

	prompt += "\n\nGenerate the newsletter content in Markdown (no code blocks, raw markdown only):"

	return prompt
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
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func buildLinkedInPrompt(releases []models.Release, news []models.NewsItem) string {
	stableReleases := filterStableReleases(releases)

	prompt := `You are writing a LinkedIn post for a Cloud Native expert and Kubernetes enthusiast. 

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
7. Hashtags

IMPORTANT:
- Keep it under 3000 characters (LinkedIn limit)
- Focus on 3-5 most impactful releases/news items
- Be enthusiastic but authentic and not over the top
- Do not use the tree metapher from the example create something unique that fits the news
- NO markdown formatting (no ** or ## ) - plain text with emojis only
- do not repeat the highlight topic in the notable news/trends block take a different topic there 
- the personal note should also not duplicate topics
- DO NOT Have duplications in the topics - no repeat in the blocks

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
			prompt += fmt.Sprintf("- %s %s: %s\n", r.RepoName, r.TagName, truncateText(r.Body, 300))
		}
	}

	prompt += fmt.Sprintf("\nTotal: %d stable releases this week\n", len(stableReleases))

	prompt += "\n\nKEY NEWS ITEMS:\n"

	// Only include non-HN news for LinkedIn (more professional sources)
	newsCount := 0
	for _, n := range news {
		if n.Source != "Hacker News" && newsCount < 15 {
			prompt += fmt.Sprintf("- [%s] %s\n", n.Source, n.Title)
			newsCount++
		}
	}

	prompt += "\n\nGenerate the LinkedIn post (plain text, no markdown, under 3000 characters):"

	return prompt
}
