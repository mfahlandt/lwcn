# LWCN - Last Week in Cloud Native

A fully automated newsletter generation system that aggregates Cloud Native releases and news, powered by AI.

**Live Site:** [https://lwcn.dev](https://lwcn.dev)

## Features

- 🔄 **CNCF Project Sync**: Automatically fetches the latest CNCF project list from the [Landscape API](https://landscape.cncf.io)
- 🔍 **News Crawler**: Aggregates news from multiple sources
- 📦 **GitHub Releases**: Tracks releases from CNCF ecosystem repositories + configurable additional projects
- 🤖 **AI Processor**: Uses Google Gemini AI to generate:
  - Weekly newsletter summaries with categorized content
  - Separate articles page for curated news
- 📰 **Hugo Website**: SEO-optimized static site with PaperMod theme
- 🍪 **GDPR Compliance**: Cookie consent banner, privacy policy, and IP anonymization
- ⚙️ **GitHub Actions**: Fully automated weekly pipeline with PR-based review workflow

## Prerequisites

- Go 1.21+
- Hugo Extended (for website generation)
- GitHub Personal Access Token (for API access)
- Google Gemini API Key (for AI processing)

## Installation

```bash
# Clone the repository
git clone https://github.com/mfahlandt/lwcn.git
cd lwcn

# Build all binaries
make build

# Or build individually:
go build -o bin/release-crawler ./cmd/release-crawler         # News crawler (RSS, Heise, HackerNews)
go build -o bin/github-releases ./cmd/github-releases         # GitHub releases crawler
go build -o bin/ai-processor ./cmd/ai-processor               # AI newsletter generator
go build -o bin/sync-cncf-projects ./cmd/sync-cncf-projects   # CNCF project sync
```

## Configuration

### Repository Configuration

The list of GitHub repositories to track is **auto-generated** from two sources:

1. **CNCF Landscape API** — all graduated, incubating, and sandbox projects are fetched automatically
2. **`config/additional-repos.yaml`** — manually curated non-CNCF repos (e.g., Podman, k9s, Trivy)

Run the sync to (re)generate `config/repositories.yaml`:

```bash
make sync-repos          # Fetch + merge → writes config/repositories.yaml
make sync-repos-dry      # Preview without writing
```

To add a non-CNCF project, edit `config/additional-repos.yaml`:

```yaml
additional_repositories:
  - owner: my-org
    repo: my-project
    name: My Project
    category: my-category
```

> **Note:** Do NOT edit `config/repositories.yaml` manually — it is auto-generated.

### News Sources Configuration

Configure news sources in `config/news-sources.yaml`:
- **RSS Feeds**: CNCF, Kubernetes, The New Stack, InfoQ, AWS, Azure, Google Cloud blogs
- **Scrape Sources**: Heise Cloud articles
- **Hacker News**: Keyword-filtered stories (kubernetes, cloud native, cncf, docker, etc.)

### Google Analytics & Cookie Consent

1. Get your GA4 Measurement ID from [Google Analytics](https://analytics.google.com/)
2. Update `website/hugo.toml`:
   ```toml
   [services.googleAnalytics]
     ID = "G-XXXXXXXXXX"
   ```
3. The cookie consent system:
   - Blocks all analytics until user accepts
   - Stores consent in localStorage + cookie for 1 year
   - Respects Do Not Track browser setting
   - Anonymizes IP addresses automatically

### Legal Pages

Update the following pages with your information:
- `website/content/impressum.md` - Imprint/Legal notice
- `website/content/datenschutz.md` - Privacy Policy

Replace all placeholder text (marked with `[brackets]`) with your actual information.

### Environment Variables

| Variable | Description |
|----------|-------------|
| `GITHUB_TOKEN` | GitHub Personal Access Token for API access |
| `GEMINI_API_KEY` | Google AI Studio API Key for AI processing |

## Usage

### News Crawler

Fetches news from RSS feeds, Heise, and Hacker News:

```bash
# Build
go build -o bin/release-crawler ./cmd/release-crawler

# Run
./bin/release-crawler -config config/news-sources.yaml -output data
```

### GitHub Releases Crawler

Fetches releases from configured GitHub repositories:

```bash
# Build
go build -o bin/github-releases ./cmd/github-releases

# Run
GITHUB_TOKEN=ghp_xxx ./bin/github-releases -config config/repositories.yaml -output data
```

### AI Processor

Processes releases and news to generate newsletter content:

```bash
# Build
go build -o bin/ai-processor ./cmd/ai-processor

# Run - Generate full newsletter
GEMINI_API_KEY=xxx ./bin/ai-processor -output website/content/newsletter

# Run - Generate only LinkedIn post
GEMINI_API_KEY=xxx ./bin/ai-processor -output website/content/newsletter -linkedin
```

## GitHub Actions Setup

The project includes four automated workflows:

### Workflows

| Workflow | File | Trigger | Description |
|----------|------|---------|-------------|
| **Deploy** | `.github/workflows/deploy.yml` | Push to main | Builds and deploys Hugo site to GitHub Pages |
| **Newsletter** | `.github/workflows/newsletter.yml` | Weekly (Monday 6:00 UTC) or Manual | Crawls news, generates newsletter, creates PR |
| **Send Email** | `.github/workflows/send-newsletter.yml` | Newsletter merged to main | Creates draft email in Buttondown |
| **Test** | `.github/workflows/test.yml` | On PR | Runs Go tests and linting |

### 1. Repository Secrets

Go to **Settings → Secrets and variables → Actions → Secrets** and add:

| Secret | Description | How to get |
|--------|-------------|------------|
| `GH_PAT` | GitHub Personal Access Token | [Create PAT](https://github.com/settings/tokens) with `repo` scope |
| `GEMINI_API_KEY` | Google AI Studio API Key | [Get API Key](https://aistudio.google.com/app/apikey) |
| `BUTTONDOWN_API_KEY` | Buttondown API Key (optional) | [Get API Key](https://buttondown.com/settings/api) |

> **Note:** `GITHUB_TOKEN` is automatically provided but has limited permissions. `GH_PAT` is needed to create PRs.

### 2. Email Newsletter Setup (Optional)

To enable email subscriptions via [Buttondown](https://buttondown.com) (free up to 100 subscribers):

1. Create a Buttondown account at [buttondown.com](https://buttondown.com)
2. Set your newsletter username to `lwcn` (or update `website/layouts/index.html`)
3. Get your API key from Settings → API
4. Add `BUTTONDOWN_API_KEY` to your repository secrets
5. When a newsletter is merged, a draft email is automatically created in Buttondown
6. Review and send from the Buttondown dashboard

### 3. Enable GitHub Pages

1. Go to **Settings → Pages**
2. Under **Source**, select **"GitHub Actions"**
3. Your site will be available at `https://yourusername.github.io/lwcn/`

### 3. Custom Domain (Optional)

1. Add your domain in **Settings → Pages → Custom domain**
2. Create a CNAME record pointing to `yourusername.github.io`
3. The workflow automatically adds the CNAME file

### 4. Manual Trigger

You can manually trigger the newsletter generation:

1. Go to **Actions → Generate Weekly Newsletter**
2. Click **Run workflow**
3. Optionally check "Skip AI generation" to only crawl data

### Workflow Details

**Weekly Newsletter Flow:**
```
Monday 6:00 UTC
    ↓
Sync CNCF projects from Landscape API
    ↓
Crawl RSS feeds, Heise, Hacker News
    ↓
Crawl GitHub Releases (CNCF + additional repos)
    ↓
Generate Newsletter with Gemini AI
    ↓
Create Pull Request for review
    ↓
Review & Merge PR
    ↓
Auto-deploy to GitHub Pages
```

## Local Development

```bash
# Build all binaries
make build

# Run tests
make test

# Clean build artifacts
make clean

# CNCF Project Sync
make sync-repos          # Sync CNCF projects from Landscape API + additional repos
make sync-repos-dry      # Dry-run sync (preview without writing)

# Crawling
make crawl-news          # Fetch news from RSS, Heise, HackerNews
make crawl-releases      # Fetch GitHub releases from CNCF repos
make crawl-all           # Fetch all sources

# Newsletter Generation
make generate-newsletter # Generate newsletter draft with AI
make generate-linkedin   # Generate only LinkedIn post
make newsletter          # Full workflow (sync + crawl + generate)

# Backfill Historical Newsletters
make backfill            # Generate newsletters for past 3 weeks
make backfill WEEKS=5    # Generate newsletters for past 5 weeks

# Hugo
make hugo-serve          # Start Hugo development server
make hugo-build          # Build Hugo site for production

# Code Quality
make fmt                 # Format Go code
make lint                # Run linter
make deps                # Download dependencies

# Show all available commands
make help
```

## Sponsored Content

LWCN supports clearly labeled sponsored / partner placements via the
`{{< sponsored >}}` Hugo shortcode. The full workflow (legal background,
shortcode parameters, disclosure checklist) is documented in
[`docs/SPONSORED_CONTENT.md`](docs/SPONSORED_CONTENT.md).

Key rules:

- **Never** insert sponsor copy into the AI-generated editorial sections.
- Use the shortcode at the end of the weekly Markdown file, under a
  `## 💼 Sponsored` heading.
- Outbound sponsor links automatically carry `rel="sponsored nofollow noopener"`.
- Public policy: <https://lwcn.dev/about/#independence--sponsorship>
- Legal disclosure: <https://lwcn.dev/impressum/#advertising--sponsored-content>

## Project Structure

```
lwcn/
├── cmd/
│   ├── release-crawler/    # News crawler CLI (RSS, Heise, HackerNews)
│   ├── github-releases/    # GitHub releases crawler CLI
│   ├── ai-processor/       # AI newsletter generator CLI
│   ├── backfill-newsletter/ # Tool to generate historical newsletters
│   ├── sync-cncf-projects/ # Syncs CNCF projects from Landscape API
│   └── debug-heise/        # Debug tool for Heise scraping
├── internal/
│   ├── ai/                 # Gemini AI integration
│   ├── cncf/               # CNCF Landscape API client
│   ├── config/             # Configuration loader
│   ├── github/             # GitHub API client
│   ├── models/             # Data models (news, releases, newsletter)
│   └── news/               # News fetchers (RSS, scraper, HackerNews)
├── config/
│   ├── repositories.yaml   # Auto-generated from CNCF Landscape + additional repos
│   ├── additional-repos.yaml # Manually curated non-CNCF repos to track
│   └── news-sources.yaml   # RSS feeds, scrape sources, HackerNews keywords
├── data/                   # Output data (releases, news JSON files)
├── website/
│   ├── hugo.toml           # Hugo configuration
│   ├── content/
│   │   ├── newsletter/     # Generated newsletter content
│   │   ├── impressum.md    # Legal imprint (German law)
│   │   └── datenschutz.md  # Privacy policy (GDPR)
│   ├── layouts/
│   │   ├── _default/       # Base templates
│   │   ├── newsletter/     # Newsletter templates (list, single, RSS)
│   │   └── partials/       # Reusable components
│   │       ├── cookie-consent.html
│   │       └── cookie-consent-js.html
│   └── static/
│       ├── css/style.css
│       └── js/cookie-consent.js
├── .github/
│   └── workflows/          # CI/CD pipelines
├── Makefile
├── AGENTS.md               # AI assistant guidelines
└── README.md
```

## GDPR Compliance

This project includes full GDPR compliance for German/EU users:

- **Cookie Consent Banner**: Blocks analytics until user accepts, with accept/decline options
- **Privacy Policy** (`/datenschutz/`): Explains data collection and user rights
- **Imprint** (`/impressum/`): Legal contact information (required by German law)
- **IP Anonymization**: Google Analytics configured with `anonymize_ip: true`
- **Do Not Track**: Respects browser DNT setting

## License

MIT License
