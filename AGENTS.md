# AGENTS.md - AI Assistant Guidelines

This file provides context and guidelines for AI assistants working with this codebase.

## Project Overview

**LWCN (Last Week in Cloud Native)** is an automated newsletter generation system that:
1. Crawls news from multiple sources
2. Fetches GitHub releases from 100+ CNCF ecosystem repositories
3. Processes content using Google Gemini AI to generate newsletter summaries and LinkedIn posts
4. Publishes content via a Hugo static website at [https://lwcn.dev](https://lwcn.dev)

## Tech Stack

- **Language**: Go 1.21+
- **Website**: Hugo with PaperMod theme
- **AI**: Google Gemini API
- **CI/CD**: GitHub Actions
- **Hosting**: GitHub Pages

## Project Structure

``` 
lwcn/
├── cmd/
│   ├── release-crawler/    # News crawler CLI (RSS, Heise, HackerNews)
│   │   └── main.go
│   ├── github-releases/    # GitHub releases crawler CLI
│   │   └── main.go
│   ├── ai-processor/       # AI newsletter generator CLI
│   │   └── main.go
│   ├── backfill-newsletter/ # Tool to generate historical newsletters
│   │   └── main.go
│   └── debug-heise/        # Debug tool for Heise scraping
│       └── main.go
├── internal/
│   ├── ai/                 # Gemini AI integration (draft.go, gemini.go)
│   ├── config/             # Configuration loader
│   ├── github/             # GitHub API client
│   ├── models/             # Data models (news, releases, newsletter)
│   └── news/               # News fetchers (rss.go, scraper.go, hackernews.go)
├── config/
│   ├── repositories.yaml   # GitHub repos to track (100+ CNCF projects)
│   └── news-sources.yaml   # RSS feeds, scrape sources, HackerNews keywords
├── data/                   # Generated JSON data (gitignored)
├── website/
│   ├── hugo.toml           # Hugo configuration
│   ├── content/
│   │   ├── newsletter/     # Generated newsletter posts
│   │   ├── impressum.md    # Legal imprint (German law)
│   │   └── datenschutz.md  # Privacy policy (GDPR)
│   ├── layouts/
│   │   ├── _default/       # Base templates (baseof.html, single.html)
│   │   ├── newsletter/     # Newsletter templates (list.html, single.html, rss.xml)
│   │   └── partials/
│   │       ├── header.html
│   │       ├── footer.html
│   │       ├── newsletter-card.html
│   │       ├── cookie-consent.html
│   │       └── cookie-consent-js.html
│   └── static/
│       ├── css/style.css
│       └── js/cookie-consent.js
├── .github/
│   └── workflows/
│       ├── deploy.yml      # Hugo site deployment
│       ├── newsletter.yml  # Weekly newsletter generation
│       └── test.yml        # Go tests and linting
├── Makefile
├── README.md
└── AGENTS.md               # This file
```

## Coding Conventions

### Go Code

- Follow standard Go conventions and `gofmt`
- Use meaningful package names
- Error handling: always handle errors, don't ignore them
- Use structured logging where appropriate
- Keep functions small and focused

```go
// Example error handling pattern
result, err := doSomething()
if err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}
```

### Hugo/HTML

- Use Hugo partials for reusable components
- Keep JavaScript in separate `.js` files (not inline in templates)
- CSS can be inline in partials for component-specific styles
- Use Hugo's built-in functions where possible

### Configuration

- Environment variables for secrets (GITHUB_TOKEN, GEMINI_API_KEY)
- YAML files for static configuration (repositories.yaml)
- TOML for Hugo configuration (hugo.toml)

## Important Files

| File | Purpose |
|------|---------|
| `cmd/release-crawler/main.go` | Entry point for news crawler (RSS, Heise, HackerNews) |
| `cmd/github-releases/main.go` | Entry point for GitHub releases crawler |
| `cmd/ai-processor/main.go` | Entry point for AI newsletter generator |
| `internal/ai/gemini.go` | Gemini API client |
| `internal/ai/draft.go` | Newsletter draft generation logic |
| `internal/news/rss.go` | RSS feed parser |
| `internal/news/scraper.go` | Heise web scraper |
| `internal/news/hackernews.go` | Hacker News API client |
| `config/repositories.yaml` | List of GitHub repos to monitor |
| `config/news-sources.yaml` | RSS feeds, scrape sources, HackerNews config |
| `website/hugo.toml` | Hugo site configuration |
| `website/layouts/_default/baseof.html` | Base HTML template |
| `website/layouts/newsletter/single.html` | Newsletter page template |
| `.github/workflows/*.yml` | CI/CD pipelines |

## Common Tasks

### Adding a New Repository to Track

Edit `config/repositories.yaml`:
```yaml
repositories:
  - owner: kubernetes
    repo: kubernetes
  - owner: new-owner    # Add new entry
    repo: new-repo
```

### Adding a New RSS Feed

Edit `config/news-sources.yaml`:
```yaml
rss_feeds:
  - name: "CNCF Blog"
    url: "https://www.cncf.io/feed/"
  - name: "New Feed"     # Add new entry
    url: "https://example.com/feed/"
```

### Adding Hacker News Keywords

Edit `config/news-sources.yaml`:
```yaml
hackernews:
  enabled: true
  keywords:
    - "kubernetes"
    - "new-keyword"      # Add new keyword
```

### Creating a New Hugo Partial

1. Create file in `website/layouts/partials/`
2. Include in template with `{{ partial "filename.html" . }}`

### Adding New CLI Flags

Use Go's `flag` package:
```go
var configPath = flag.String("config", "config/repositories.yaml", "Path to config file")
flag.Parse()
```

## Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `GITHUB_TOKEN` | Yes | GitHub PAT for API access |
| `GEMINI_API_KEY` | Yes | Google AI Studio API key |

## Build & Run Commands

```bash
# Build all binaries
make build

# Crawling
make crawl-news          # Fetch news from RSS, Heise, HackerNews
make crawl-releases      # Fetch GitHub releases from CNCF repos
make crawl-all           # Fetch all sources

# Newsletter Generation
make generate-newsletter # Generate newsletter draft with AI
make generate-linkedin   # Generate only LinkedIn post
make newsletter          # Full workflow (crawl + generate)

# Backfill Historical Newsletters
make backfill            # Generate newsletters for past 3 weeks
make backfill WEEKS=5    # Generate newsletters for past 5 weeks

# Hugo
make hugo-serve          # Start Hugo dev server
make hugo-build          # Build Hugo site for production

# Code Quality
make fmt                 # Format Go code
make lint                # Run linter
make test                # Run tests
make deps                # Download dependencies
```

## Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...
```

## Legal Compliance (GDPR)

This project includes GDPR-compliant features:
- Cookie consent banner (must be accepted before Google Analytics loads)
- Privacy policy page (`/datenschutz/`)
- Imprint page (`/impressum/`)
- IP anonymization enabled for Google Analytics

## API Rate Limits

- **GitHub API**: 5000 requests/hour with token, 60/hour without
- **Gemini API**: Check current limits at Google AI Studio

## Notes for AI Assistants

1. **Hugo Templates**: Avoid complex Go template logic inside `<script>` tags - it causes parsing errors. Use separate `.js` files instead.

2. **German Legal Requirements**: This project targets German users, so Impressum and Datenschutz pages are legally required.

3. **Privacy First**: Always load analytics only after user consent. The cookie consent system uses both localStorage and cookies.

4. **File Paths**: Use forward slashes in code, even on Windows.

5. **Generated Content**: Files in `data/` and `website/content/newsletter/` are auto-generated - don't manually edit them.

6. **Three CLIs**: The project has three separate CLI tools:
   - `release-crawler`: Fetches news (RSS, Heise, HackerNews) → outputs to `data/news-YYYY-MM-DD.json`
   - `github-releases`: Fetches GitHub releases → outputs to `data/releases-YYYY-MM-DD.json`
   - `ai-processor`: Generates newsletter from data files → outputs to `website/content/newsletter/`

7. **LinkedIn Posts**: The AI processor can generate LinkedIn posts separately with the `-linkedin` flag.

8. **Weekly Workflow**: The newsletter workflow runs every Monday at 6:00 UTC via GitHub Actions and creates a PR for review.

