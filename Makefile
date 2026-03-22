.PHONY: build test clean crawl-news crawl-releases crawl-all generate-newsletter hugo-serve hugo-build sync-repos

# Binaries (Windows uses .exe extension)
ifeq ($(OS),Windows_NT)
    NEWS_CRAWLER = bin/release-crawler.exe
    GITHUB_RELEASES = bin/github-releases.exe
    AI_PROCESSOR = bin/ai-processor.exe
    BACKFILL = bin/backfill-newsletter.exe
    SYNC_CNCF = bin/sync-cncf-projects.exe
    MKDIR = if not exist bin mkdir bin
    RM_BIN = if exist bin rmdir /s /q bin
    RM_PUBLIC = if exist website\public rmdir /s /q website\public
else
    NEWS_CRAWLER = bin/release-crawler
    GITHUB_RELEASES = bin/github-releases
    AI_PROCESSOR = bin/ai-processor
    BACKFILL = bin/backfill-newsletter
    SYNC_CNCF = bin/sync-cncf-projects
    MKDIR = mkdir -p bin
    RM_BIN = rm -rf bin/
    RM_PUBLIC = rm -rf website/public/
endif

# Directories
DATA_DIR = data
CONTENT_DIR = website/content/newsletter

# Build all binaries
build:
	@echo "Building binaries..."
	go mod tidy
	@$(MKDIR)
	go build -o $(NEWS_CRAWLER) ./cmd/release-crawler
	go build -o $(GITHUB_RELEASES) ./cmd/github-releases
	go build -o $(AI_PROCESSOR) ./cmd/ai-processor
	go build -o $(BACKFILL) ./cmd/backfill-newsletter
	go build -o $(SYNC_CNCF) ./cmd/sync-cncf-projects

# Run tests
test:
	go test -v ./...

# Clean build artifacts
clean:
	$(RM_BIN)
	$(RM_PUBLIC)

# Sync CNCF project list from Landscape API + additional repos
sync-repos: build
	@echo "Syncing CNCF project list from Landscape API..."
	$(SYNC_CNCF) -output config/repositories.yaml -additional config/additional-repos.yaml
	@echo "Repository list updated. Review config/repositories.yaml"

# Sync CNCF project list (dry-run, no file changes)
sync-repos-dry: build
	@echo "Dry-run: syncing CNCF project list..."
	$(SYNC_CNCF) -output config/repositories.yaml -additional config/additional-repos.yaml -dry-run

# Crawl news from RSS, Heise, HackerNews
crawl-news: build
	@echo "Crawling news..."
	$(NEWS_CRAWLER) -config config/news-sources.yaml -output $(DATA_DIR)

# Crawl GitHub releases from CNCF repositories
crawl-releases: build
	@echo "Crawling GitHub releases..."
	$(GITHUB_RELEASES) -config config/repositories.yaml -output $(DATA_DIR)

# Crawl all sources (news + releases)
crawl-all: crawl-news crawl-releases
	@echo "All sources crawled."

# Generate newsletter draft using AI
generate-newsletter: build
	@echo "Generating newsletter draft with Gemini AI..."
	$(AI_PROCESSOR) -output $(CONTENT_DIR)

# Generate LinkedIn posts (newsletter article + short teaser)
generate-linkedin: build
	@echo "Generating LinkedIn posts (newsletter + short teaser) with Gemini AI..."
	$(AI_PROCESSOR) -output $(CONTENT_DIR) -linkedin

# Full workflow: sync repos, crawl all sources, and generate newsletter
newsletter: sync-repos crawl-all generate-newsletter
	@echo "Newsletter draft generated!"
	@echo "Review the draft in $(CONTENT_DIR)"
	@echo "Run 'make hugo-serve' to preview"

# Backfill past weeks (default: 3 weeks)
# Usage: make backfill WEEKS=3
WEEKS ?= 3
backfill: build
	@echo "Generating newsletters for past $(WEEKS) weeks..."
	$(BACKFILL) -weeks $(WEEKS) -output $(CONTENT_DIR) -data $(DATA_DIR)

# Debug: Show what's being loaded
debug-config:
	@echo "=== Config file content ==="
ifeq ($(OS),Windows_NT)
	@type config\repositories.yaml
else
	@cat config/repositories.yaml
endif

# Hugo development server
hugo-serve:
	cd website && hugo server -D

# Hugo build
hugo-build:
	cd website && hugo --minify

# Download dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Show help
help:
	@echo "Available targets:"
	@echo "  build               - Build all binaries"
	@echo "  test                - Run tests"
	@echo "  clean               - Remove build artifacts"
	@echo ""
	@echo "CNCF Sync:"
	@echo "  sync-repos          - Sync CNCF projects from Landscape API + additional repos"
	@echo "  sync-repos-dry      - Dry-run sync (preview without writing)"
	@echo ""
	@echo "Crawling:"
	@echo "  crawl-news          - Fetch news from RSS, Heise, HackerNews"
	@echo "  crawl-releases      - Fetch GitHub releases from CNCF repos"
	@echo "  crawl-all           - Fetch all sources"
	@echo ""
	@echo "Newsletter Generation:"
	@echo "  generate-newsletter - Generate newsletter draft with AI"
	@echo "  newsletter          - Full workflow (sync + crawl + generate)"
	@echo ""
	@echo "Hugo:"
	@echo "  hugo-serve          - Start Hugo development server"
	@echo "  hugo-build          - Build Hugo site for production"
	@echo ""
	@echo "Environment variables:"
	@echo "  GITHUB_TOKEN        - Required for crawl-releases"
	@echo "  GEMINI_API_KEY      - Required for generate-newsletter"
	@echo ""
	@echo "Utilities:"
	@echo "  deps           - Download Go dependencies"
	@echo "  fmt            - Format Go code"
	@echo "  lint           - Run linter"
