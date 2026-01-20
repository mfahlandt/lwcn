# LWCN - Last Week in Cloud Native

A tool to automatically aggregate and summarize cloud-native project releases and news into a weekly newsletter, powered by AI.

## Features

- 🔍 **Release Crawler**: Automatically fetches releases from configured GitHub repositories
- 🤖 **AI Processor**: Uses Google Gemini AI to generate newsletter summaries
- 📰 **Hugo Website**: Generates a static website for the newsletter
- ⚙️ **GitHub Actions**: Fully automated pipeline for weekly newsletter generation

## Prerequisites

- Go 1.21+
- Hugo (for website generation)
- GitHub Token (for API access)
- Google Gemini API Key (for AI processing)

## Installation

```bash
# Clone the repository
git clone https://github.com/mfahlandt/lwcn.git
cd lwcn

# Build the release crawler
go build -o bin/release-crawler ./cmd/release-crawler

# Build the AI processor
go build -o bin/ai-processor ./cmd/ai-processor
```

## Configuration

### Repository Configuration

Configure the repositories to track in `config/repositories.yaml`.

### Google Analytics

1. Get your GA4 Measurement ID from [Google Analytics](https://analytics.google.com/)
2. Update `website/hugo.toml` with your Measurement ID:
   ```toml
   [services.googleAnalytics]
     ID = "G-XXXXXXXXXX"
   ```
3. The cookie consent banner will handle GDPR compliance automatically

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

### Release Crawler

Fetches releases from configured GitHub repositories:

```bash
# Build
go build -o bin/release-crawler ./cmd/release-crawler

# Run
GITHUB_TOKEN=ghp_xxx ./bin/release-crawler -config config/repositories.yaml -output data
```

### AI Processor

Processes releases and news to generate newsletter content:

```bash
# Build
go build -o bin/ai-processor ./cmd/ai-processor

# Run
GEMINI_API_KEY=xxx ./bin/ai-processor \
  -releases data/releases-2024-01-15.json \
  -news data/news-2024-01-15.json \
  -output website/content/newsletter
```

## GitHub Actions Setup

The project includes two automated workflows:

### Workflows

| Workflow | File | Trigger | Description |
|----------|------|---------|-------------|
| **Deploy** | `.github/workflows/deploy.yml` | Push to main | Builds and deploys Hugo site |
| **Newsletter** | `.github/workflows/newsletter.yml` | Weekly (Monday 6:00 UTC) or Manual | Crawls news, generates newsletter, creates PR |

### 1. Repository Secrets

Go to **Settings → Secrets and variables → Actions → Secrets** and add:

| Secret | Description | How to get |
|--------|-------------|------------|
| `GH_PAT` | GitHub Personal Access Token | [Create PAT](https://github.com/settings/tokens) with `repo` scope |
| `GEMINI_API_KEY` | Google AI Studio API Key | [Get API Key](https://aistudio.google.com/app/apikey) |

> **Note:** `GITHUB_TOKEN` is automatically provided but has limited permissions. `GH_PAT` is needed to create PRs.

### 2. Enable GitHub Pages

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
Crawl RSS feeds, Heise, Hacker News
    ↓
Crawl GitHub Releases (107 CNCF repos)
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
# Run the complete pipeline
make run-all

# Only crawl releases
make crawl-releases

# Start Hugo development server
make hugo-serve

# Show help
make help
```

## Project Structure

```
lwcn/
├── cmd/
│   ├── release-crawler/    # Release crawler CLI
│   └── ai-processor/       # AI processor CLI
├── config/
│   └── repositories.yaml   # Repository configuration
├── data/                   # Output data (releases, news)
├── website/
│   └── content/
│       └── newsletter/     # Generated newsletter content
├── Makefile
└── README.md
```

## License

MIT License
