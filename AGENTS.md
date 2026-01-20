# AGENT.md - AI Assistant Guidelines

This file provides context and guidelines for AI assistants working with this codebase.

## Project Overview

**LWCN (Last Week in Cloud Native)** is an automated newsletter generation system that:
1. Crawls GitHub releases from cloud-native projects
2. Processes releases using Google Gemini AI to generate summaries
3. Publishes content via a Hugo static website

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
│   ├── release-crawler/    # CLI to fetch GitHub releases
│   │   └── main.go
│   └── ai-processor/       # CLI to process releases with AI
│       └── main.go
├── internal/               # Shared internal packages
├── config/
│   └── repositories.yaml   # List of repos to track
├── data/                   # Generated JSON data (gitignored)
├── website/
│   ├── hugo.toml           # Hugo configuration
│   ├── content/
│   │   ├── newsletter/     # Generated newsletter posts
│   │   ├── impressum.md    # Legal imprint (German law)
│   │   └── datenschutz.md  # Privacy policy (GDPR)
│   ├── layouts/
│   │   ├── _default/
│   │   └── partials/
│   │       ├── header.html
│   │       ├── footer.html
│   │       ├── cookie-consent.html
│   │       └── cookie-consent-js.html
│   └── static/
│       └── js/
│           └── cookie-consent.js
├── .github/
│   └── workflows/          # GitHub Actions workflows
├── Makefile
├── README.md
└── AGENT.md               # This file
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
| `cmd/release-crawler/main.go` | Entry point for release crawler |
| `cmd/ai-processor/main.go` | Entry point for AI processor |
| `config/repositories.yaml` | List of GitHub repos to monitor |
| `website/hugo.toml` | Hugo site configuration |
| `website/layouts/_default/baseof.html` | Base HTML template |
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

# Run release crawler
make crawl-releases

# Run AI processor
make process-releases

# Start Hugo dev server
make hugo-serve

# Full pipeline
make run-all
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

**Important**: The legal pages contain placeholder text that must be replaced with actual information before deployment.

## API Rate Limits

- **GitHub API**: 5000 requests/hour with token, 60/hour without
- **Gemini API**: Check current limits at Google AI Studio

## Notes for AI Assistants

1. **Hugo Templates**: Avoid complex Go template logic inside `<script>` tags - it causes parsing errors. Use separate `.js` files instead.

2. **German Legal Requirements**: This project targets German users, so Impressum and Datenschutz pages are legally required.

3. **Privacy First**: Always load analytics only after user consent.

4. **File Paths**: Use forward slashes in code, even on Windows.

5. **Generated Content**: Files in `data/` and `website/content/newsletter/` are auto-generated - don't manually edit them.
