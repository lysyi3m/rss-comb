# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

RSS Comb is a Go server application that acts as a proxy between existing RSS/Atom feeds and RSS reader applications. It provides feed normalization, automatic deduplication, content filtering, and full iTunes podcast support through YAML-based configuration files.

The application features a clean, modular architecture with clear separation of concerns, dependency injection, and comprehensive testing. Key architectural features include a FeedType interface with type-specific parsers and builders (basic, podcast, youtube), a SQLite-backed job queue for background processing, and intelligent newest-item duplicate detection to skip unchanged feeds.

## Project Philosophy & Scope

### What RSS Comb IS
- **Single-purpose tool** - Fetch, filter, deduplicate, and serve RSS feeds
- **Personal/small-scale** - Designed for 10-50 feeds, not thousands
- **Self-hosted** - Single instance deployment (Docker/Docker Compose)
- **Simple operations** - Manual intervention is acceptable and expected
- **Minimalist** - Favor simplicity and maintainability over enterprise patterns

### What RSS Comb is NOT
- ❌ Not a SaaS platform requiring 99.99% uptime
- ❌ Not a distributed system with microservices
- ❌ Not handling thousands of feeds or millions of requests
- ❌ Not requiring automated recovery from all failure scenarios
- ❌ Not a system where manual intervention is prohibitive

### Design Principles

**KISS (Keep It Simple, Stupid)**
- Prefer simple solutions over clever abstractions
- Logs are often sufficient instead of metrics dashboards
- Manual fixes are acceptable for rare failures
- Restart-on-failure is a valid strategy

**YAGNI (You Ain't Gonna Need It)**
- Don't add enterprise patterns "just in case"
- No circuit breakers, retry logic, or distributed tracing unless actually needed
- Avoid over-engineering for hypothetical scale

**Pragmatic Trade-offs**
- Favor code clarity over performance micro-optimizations
- Simple error messages over structured error hierarchies
- Direct solutions over abstraction layers
- Fewer dependencies over framework convenience

### When to Add Complexity

Only add complexity when you have **actual evidence** of need:
- ✅ Remove duplication when it creates maintenance burden
- ✅ Add abstractions when you have 3+ identical implementations
- ✅ Optimize when you measure actual performance problems
- ❌ Don't add patterns because they're "best practices" in enterprise contexts

### Scale Assumptions

Current design assumes:
- ~10-50 feeds total
- Feeds refresh every 15-60 minutes
- Single server instance
- Manual monitoring via logs
- Downtime measured in minutes/hours is acceptable
- One maintainer/operator

If these assumptions change significantly, revisit architectural decisions.

## Development Environment

### Prerequisites
- Go 1.24+
- Docker & Docker Compose (for yt-dlp only)

### Common Commands

#### Development Setup
```bash
# SQLite database is created automatically on first run — no setup needed

# Build the application
make dev-build

# Run with development SQLite database (created automatically in ./data/)
make dev-run

# Database migrations run automatically on startup

# Run tests
make dev-test
```

#### Docker Commands
```bash
# Build production image manually (if needed)
docker build -f Dockerfile -t rss-comb:latest .

# Build with custom PORT
docker build -f Dockerfile -t rss-comb:latest . --build-arg PORT=9000

# Note: Production images are automatically built via GitHub Actions on git tags
```

#### Database Operations
```bash
# Database migrations are handled automatically by the application on startup

# Create new migration files in app/database/migrations/
# Follow the naming convention: NNN_description.up.sql and NNN_description.down.sql
```

## Project Structure

```
rss-comb/
├── Dockerfile                 # Production container build
├── Makefile                  # Development commands
├── app/                      # Main application code
│   ├── main.go              # Application entry point and initialization
│   ├── api/                 # HTTP handlers and server
│   ├── cfg/                 # Application configuration management
│   ├── database/            # Database connections, repositories, and embedded migrations
│   ├── feed/                # Feed types, parsing, building, filtering, config management
│   ├── jobs/                # Worker pool, scheduler, and job handlers
│   └── media/               # yt-dlp integration and media file management
├── feeds/                    # Feed configuration files (*.yml)
├── docker-compose.yml       # Development database service
├── .github/workflows/       # CI/CD automation
└── go.mod                   # Go module definition
```

## Architecture

### Core Components

1. **Main Application** (`app/main.go`)
   - Application entry point and simplified initialization
   - Server initialization and graceful shutdown handling

2. **Application Configuration System** (`app/cfg/`)
   - Centralized application configuration management
   - Configuration passed explicitly via dependency injection (no global state)
   - Environment variable and command-line flag parsing using go-flags
   - Version management and build-time version injection
   - Timezone configuration and application-wide settings

3. **Feed Configuration System** (`app/feed/`)
   - YAML-based feed configuration loading and validation (`config_loader.go`)
   - Configuration sync to database (`config_sync.go`)
   - Feed names automatically derived from filenames (e.g., `habr.yml` → `habr`)
   - Feed type system: basic (default), podcast, youtube

4. **Feed Type System** (`app/feed/`)
   - **Interface** (`feed_type.go`): `FeedType` interface with `Parse()` and `Build()` methods; `ForType()` factory resolves type string to implementation
   - **Basic** (`basic.go`): Standard RSS/Atom parsing and RSS 2.0 generation, no iTunes metadata
   - **Podcast** (`podcast.go`): RSS/Atom with iTunes metadata preservation and enclosure passthrough
   - **YouTube** (`youtube.go`): YouTube Atom feed parsing with `media:group` extraction, RSS 2.0 generation with downloaded audio enclosures
   - **Shared helpers** (`helpers.go`): Common parsing/building utilities used by all types
   - **Extraction** (`extraction.go`): `feed.Extract()` - Full-text content extraction using go-shiori/go-readability
   - **Filtering** (`filtering.go`): `feed.Filter()` - Content filtering with substring and regex pattern support
   - **Refiltering** (`refilter.go`): `feed.Refilter()` - Re-applies filters on config reload

5. **Database Layer** (`app/database/`)
   - SQLite with text UUID primary keys
   - Separate repositories for feeds and items
   - Optimized queries with proper indexing
   - Embedded migrations with automatic execution on startup

6. **Job Queue System** (`app/jobs/`, `app/database/job_repository.go`)
   - SQLite-backed job queue with serialized access via single connection
   - Worker pool with configurable concurrency via `WORKER_COUNT`
   - Scheduler creates `fetch_feed` jobs for due feeds on each tick
   - Job types: `fetch_feed` (feed processing), `extract_content` (article extraction), `download_media` (yt-dlp audio download)
   - Automatic retry with configurable max retries per job type
   - Stale job recovery for crashed workers

7. **Media System** (`app/media/`)
   - Audio extraction from YouTube videos via configurable yt-dlp command (`YT_DLP_CMD`)
   - GUID-based file naming (YouTube video ID extracted from `yt:video:` GUID)
   - Three-layer dedup: DB lookup → filesystem check → download
   - Global media cleanup removes orphaned files not referenced by any feed
   - Supports local yt-dlp binary or Docker-based execution

8. **HTTP API** (`app/api/`)
   - RESTful endpoints for feed access
   - RSS 2.0 output generation via feed system
   - Direct database queries for real-time data
   - Clean constructor pattern with consistent argument order

### Data Flow

1. **Application Initialization**: `cfg.Load()` loads application configuration, passed explicitly to all components
2. **Feed Configuration Loading**: YAML files loaded from `feeds/*.yml` and stored in database at startup
3. **Database Sync**: Configuration changes automatically registered in database with hash-based change detection via UPSERT
4. **Job Scheduling**: Scheduler (every 30 seconds) queries database for enabled feeds with `next_fetch` due, creates `fetch_feed` jobs
5. **Feed Processing**: Worker pool claims jobs; fetches feed data, parses via `feed.ForType(typ).Parse()`, filters, deduplicates items, creates `extract_content` or `download_media` jobs for new items
6. **Content Extraction**: `extract_content` jobs fetch article HTML and extract clean text (items hidden until ready)
7. **Media Downloading**: `download_media` jobs run yt-dlp to extract audio from YouTube videos (items hidden until ready; failed items stay hidden)
8. **Storage**: Items stored with filter status, content hashes, and processing status columns
9. **RSS Feed Access**: `/feeds/:name` endpoint generates RSS 2.0 XML from database using `feed.ForType(typ).Build()` with visible items; media items get `<enclosure>` URLs pointing to `/media/`
10. **Configuration Reload**: `/api/feeds/:name/reload` API endpoint reloads YAML via `feed.ConfigSync()`, updates database, and synchronously refilters via `feed.Refilter()`

### Database Schema

**feeds table:**
- Stores feed metadata and processing status
- Tracks last_fetched_at, next_fetch_at timestamps
- Stores feed_type for type-specific parsing and building
- Stores configuration (settings JSON TEXT, filters JSON TEXT, is_enabled, config_hash)
- Uses `name` field to match with configuration files

**feed_items table:**
- Normalized item data with content hashing
- Filtering and deduplication flags
- RSS enclosure support (url, length, type)
- Optimized indexes for common queries

## Detailed Architecture

### Database Schema Details
- **feeds table**: id, name, feed_url, title, source_title, link, description, image_url, language, last_fetched_at, next_fetch_at, feed_published_at, feed_updated_at, feed_type, is_enabled, settings (JSON TEXT), filters (JSON TEXT), config_hash, itunes_author, itunes_image, itunes_explicit, itunes_owner_name, itunes_owner_email, created_at, updated_at
- **feed_items table**: id, feed_id, guid, link, title, description, content, published_at, updated_at, authors, categories, is_filtered, content_hash, enclosure_url, enclosure_length, enclosure_type, itunes_duration, itunes_episode, itunes_season, itunes_episode_type, itunes_image, content_extraction_status, media_status, media_path, media_size, created_at
- **jobs table**: id, job_type, feed_id, item_id (nullable), status, retries, max_retries, error_message, created_at, updated_at
- **Key relationships**: feeds.id → feed_items.feed_id, feeds.id → jobs.feed_id, feed_items.id → jobs.item_id (TEXT primary keys)
- **Indexes**: feed_id, published_at, content_hash, is_enabled, jobs pending/dedup indexes, media_path for cross-feed dedup
- **Constraints**: Unique (feed_id, guid) for item deduplication within feeds
- **iTunes Podcast Support**: All iTunes fields are nullable and automatically extracted from podcast RSS feeds via gofeed library's built-in iTunes extension support

### Feed Processing Layer (`app/feed/`)
- `feed_type.go`: `FeedType` interface with `Parse()` and `Build()` methods; `ForType()` factory function
- `basic.go`: `basicType` — standard RSS/Atom parsing and RSS 2.0 building, no iTunes metadata
- `podcast.go`: `podcastType` — RSS/Atom with iTunes metadata preservation and enclosure passthrough
- `youtube.go`: `youtubeType` — YouTube Atom parsing with `media:group` extraction; RSS 2.0 building with downloaded audio enclosures
- `helpers.go`: Shared parsing/building utilities (URL normalization, content hashing, XML element writing, channel header, iTunes elements)
- `config_loader.go`: Pure functions for loading and validating YAML configuration files
- `config_sync.go`: `ConfigSync()` — syncs YAML config to database via `LoadConfig` + `UpsertFeedConfig`
- `refilter.go`: `Refilter()` — re-applies filters to all items for a feed (used by reload endpoint)
- `extraction.go`: `feed.Extract()` — HTML content extraction using go-shiori/go-readability library
- `filtering.go`: `feed.Filter()` and `feed.ClearRegexCache()` — content filtering with substring and regex patterns; compiled regex cached in sync.Map
- `types.go`: Feed data structures, configuration types, Metadata type alias
- **Performance**: Newest-item duplicate check skips processing when no new items; regex patterns compiled once and cached
- **Architecture**: Database is single source of truth at runtime, YAML files loaded only at startup/reload
- **Design**: `FeedType` interface with type-specific implementations; each type owns its Parse and Build logic
- **iTunes Support**: Podcast and YouTube types extract and generate iTunes RSS extensions; basic type ignores iTunes data entirely

### Repository Layer (`app/database/`)
- `connection.go`: SQLite connection management with WAL mode
- `feed_repository.go`: Feed operations with UPSERT for efficient configuration sync
- `item_repository.go`: Item operations (upsert, dedup check, visibility queries, status updates)
- `types.go`: Database model structs (Feed, Item) with `DisplayTitle()`, `GetSettings()`, `GetFilters()` methods; JSON array helpers (`JSONStringSlice`, `encodeStringSlice`) for SQLite TEXT columns
- `job_repository.go`: Job queue operations (create, claim, complete, fail, reset stale)
- `migrations.go`: Embedded migration management
- `migrations/`: Consolidated SQLite schema in `001_initial_schema.sql` (feeds, feed_items, jobs tables with indexes)

### Job Queue System (`app/jobs/`)
- `worker.go`: Worker pool with configurable concurrency, polls for pending jobs
- `scheduler.go`: Ticker-based scheduler that creates `fetch_feed` jobs for due feeds and resets stale jobs
- `handlers.go`: Job handler factories — `FetchFeedHandler`, `ExtractContentHandler`, `DownloadMediaHandler`
- `process.go`: Feed processing logic — fetch, parse via `FeedType`, deduplicate, filter, create downstream jobs
- `fetch.go`: HTTP fetch utility used by feed processing and content extraction
- **Job types**: `fetch_feed` (max_retries=0, scheduler retries), `extract_content` (max_retries=3), `download_media` (max_retries=3)
- **Concurrency**: Serialized via `MaxOpenConns(1)` — safe concurrent access without row locking
- **Cleanup**: Completed and exhausted jobs are deleted; failure state captured on items

### Media System (`app/media/`)
- `downloader.go`: `Validate()`, `Download()`, `MediaFileID()` — yt-dlp integration via configurable command string
- `cleanup.go`: `CleanupMedia()` — deletes orphaned media files not referenced by any feed
- `filecheck.go`: `FileExists()` — filesystem existence check for dedup fallback
- **Command splitting**: `YT_DLP_CMD` supports multi-word values (e.g., `docker compose run --rm yt-dlp`) via `strings.Fields`
- **File naming**: YouTube video ID extracted from GUID (`yt:video:ID`), fallback to SHA-256 hash of GUID

### Application Configuration System (`app/cfg/`)
- `types.go`: Application configuration struct with go-flags tags for env/CLI parsing
- `loader.go`: Configuration loading with environment/command-line parsing
- Configuration passed explicitly via dependency injection (no global state)
- Integrated version management and timezone configuration


## Environment Guide

### Development Environment
- **Application**: Running locally via `make run`
- **Database**: SQLite file at `./data/rss-comb-dev.db` (created automatically)
- **Feed configs**: Local `feeds/*.yml` files
- **Logs**: Console output

### Production Environment
- **Application**: Running in Docker container from GitHub Container Registry
- **Database**: SQLite file (configured via `DB_PATH`, default `/app/data/rss-comb.db`)
- **Public URL**: Configure `BASE_URL` for proper RSS self-referencing links (e.g., `https://feeds.yourdomain.com`)
- **Feed configs**: Mounted `feeds/*.yml` files
- **Deployment**: Automated via GitHub Actions on release tags

### How to verify "app is running"
- **Dev**: `curl localhost:8080/health` returns 200
- **Prod**: `curl localhost:8080/health` returns 200
- **Database**: `make dev-test` passes, or manual query works

## Work Verification Process

### After making changes:
1. **Build & Test**: `make dev-build && make dev-test`
2. **Local verification**: `make dev-run` then test endpoints
3. **Docker verification**: `docker build -f Dockerfile -t rss-comb:latest .` then test locally
4. **Commit changes**: Only when explicitly requested
5. **MANDATORY cleanup**: `make dev-clean` (prevents interference with production instances)

### All operations should use Makefile:
- **Development**: `make dev-run`
- **Testing**: `make dev-test`
- **Building**: `make dev-build`
- **Cleanup**: `make dev-clean`
- **Never use direct docker/go commands** - always use Makefile targets
- **Migrations**: Handled automatically by the application (no separate migrate commands needed)

### Cleanup Commands
```bash
# Stop ALL RSS Comb processes (including stray development builds)
make dev-stop

# Complete development cleanup: processes + dev database + build caches
make dev-clean
```

**IMPORTANT: Always clean up after development work to prevent side effects:**
- Stray development processes can interfere with production instances
- Multiple instances may cause port conflicts
- Use `make dev-clean` before switching between development and production work

## Configuration

### Feed Configuration Format (`feeds/*.yml`)
```yaml
url: "https://example.com/feed.xml"
enabled: true
title: "Custom Feed Title"  # Optional: overrides source feed title
type: youtube               # Optional: "" (basic, default), "podcast", or "youtube"

settings:
  refresh_interval: 1800  # 30 minutes (recommended)
  max_items: 50           # Limits RSS output items (all items stored in database)
  timeout: 30             # seconds
  extract_content: true   # Enable automatic content extraction (basic type only)

filters:
  - field: "title"
    includes: ["keyword"]
    excludes: ["spam"]
  - field: "authors"
    includes: ["john doe"]
    excludes: ["spammer"]
```

**Feed Types:**
- **basic** (default, no `type:` needed): Standard RSS/Atom normalization with filtering and deduplication. Supports `extract_content`.
- **podcast**: Preserves iTunes podcast metadata and enclosures from source feed.
- **youtube**: Parses YouTube Atom feeds, downloads audio via yt-dlp, generates podcast RSS with media enclosures.

**Filter Pattern Types:**
RSS Comb supports two pattern matching modes that can be used together:

1. **Substring matching** (default): Case-insensitive substring search with Unicode normalization
   ```yaml
   excludes: ["weekly digest"]  # Matches any title containing "weekly digest"
   ```

2. **Regular expressions**: Patterns wrapped in `/slashes/` are treated as regex
   ```yaml
   excludes: ["/weekly|digest/"]      # Matches "weekly" OR "digest"
   includes: ["/^tech(nology)?/"]     # Starts with "tech" or "technology"
   ```

**Regex Features:**
- Automatically case-insensitive (uses `(?i)` flag)
- Compiled once and cached for performance
- Cache cleared on config reload for fresh state
- Invalid regex falls back to literal substring matching with warning
- Full Go regex syntax support (RE2)

**Example: Simplifying Large Filter Lists**
```yaml
# Before: Multiple similar patterns
excludes:
  - "Mobile development weekly"
  - "Security news weekly"
  - "TOP-5 events of the week"

# After: Single regex pattern
excludes:
  - "/weekly|week/"
```

See `docs/REGEX_PATTERNS.md` for comprehensive examples and pattern reference.

**Important Notes:**
- The feed name is automatically derived from the filename (without `.yml` extension)
- Feed names must be unique
- Filenames should be URL-safe (alphanumeric, hyphens, underscores) since they become the feed name
- Feed URLs can be updated in configuration files at any time - the system automatically detects and applies URL changes
- Feeds with query parameters are fully supported since routing is based on feed names derived from filenames
- Authors field contains formatted strings like "email (name)" or "name" when email is not available

### Feed Configuration Architecture

RSS Comb uses a simplified, filesystem-driven approach to feed configuration that eliminates redundancy and improves maintainability:

**Key Principles:**
1. **Filename-Based Identity**: Feed names are derived from YAML filenames (e.g., `habr.yml` → `habr`)
2. **No Redundant Storage**: Configuration file paths are derived when needed (`feedsDir + name + ".yml"`)
3. **Single Source of Truth**: The filename uniquely identifies the feed across the entire system

**Benefits:**
- **Consistency**: Feed name is always derived from filename, preventing mismatches
- **Maintainability**: No need to maintain separate name and file path fields
- **Simplicity**: Configuration cache indexed by name for O(1) lookups
- **Flexibility**: Easy to rename feeds by simply renaming the file

**Migration from Legacy Systems:**
The system has evolved from using explicit `id` fields in YAML files to automatic name derivation. This eliminates the possibility of configuration inconsistencies between the filename and the internal identifier.

### Content Extraction Feature

RSS Comb includes automatic content extraction for feeds that don't provide full article content in their RSS feeds. This feature uses intelligent HTML parsing to extract clean, readable content from article web pages.

**Key Features:**
- **Inline Processing**: Runs during feed processing for new items
- **Intelligent Extraction**: Uses Mozilla's Readability algorithm (go-shiori/go-readability library)
- **Error Resilient**: Extraction failures don't affect item storage
- **Performance Optimized**: Only processes visible (non-filtered) items
- **Configurable**: Per-feed enable/disable with timeout controls

**Configuration Options:**
- `extract_content: true/false` - Enable/disable content extraction
- `max_items: 50` - Limits items stored per feed

**How It Works:**
1. Feed processing fetches and parses RSS/Atom feed
2. For each new, non-filtered item:
   - If `extract_content: true`, fetch article URL
   - Extract clean content using Readability algorithm
   - Store extracted content in item.Content field
3. Failed extractions are logged but don't block item storage
4. Original RSS content used as fallback if extraction fails

### Environment Variables
All configuration options support both environment variables and command-line flags:

**Database Configuration:**
- `DB_PATH` (default: ./data/rss-comb.db) - SQLite database file path

**Application Configuration:**
- `FEEDS_DIR` (default: ./feeds) - Directory containing feed configuration files
- `PORT` (default: 8080) - HTTP server port
- `BASE_URL` (optional) - Public base URL for the service (e.g., https://feeds.example.com). When set, RSS feeds use this URL for self-referencing links instead of localhost:port. Ideal for production deployments behind proxies.
- `SCHEDULER_INTERVAL` (default: 30) - Scheduler interval in seconds for creating feed processing jobs
- `WORKER_COUNT` (default: 5) - Number of concurrent workers for processing jobs (feed fetching, content extraction, media downloads)
- `API_ACCESS_KEY` (optional) - API access key for authentication
- `MEDIA_DIR` (default: ./media) - Directory for downloaded media files
- `YT_DLP_CMD` (default: "yt-dlp") - yt-dlp command; supports multi-word values for Docker (e.g., `docker compose run --rm yt-dlp`)
- `USER_AGENT` (default: "RSS Comb/1.0") - User agent string for HTTP requests
- `TZ` (default: "UTC") - Timezone for display timestamps in API responses and RSS feeds (e.g., UTC, America/New_York, Europe/London). Database operations always use UTC for consistency.

Use `./app/main.go --help` or `go run app/main.go --help` to see all available command-line flags.

## Testing

### Unit Tests
```bash
# Run all tests
go test -v ./app/...

# Run tests with coverage
go test -v -coverprofile=coverage.out ./app/...
go tool cover -html=coverage.out

# Run tests for specific package
go test -v ./app/api
go test -v ./app/cfg
go test -v ./app/database
```

### Integration Testing
- Test database migrations and schema
- Verify feed parsing for different formats
- Test filtering and deduplication logic
- API endpoint testing with real feeds

### Manual Testing
1. Enable the example feed in `feeds/example.yml` (set `enabled: true`)
2. Start services with `make run`
3. Monitor logs for feed processing
4. Test API endpoint: `curl -H "X-API-Key: your-key" "http://localhost:${PORT:-8080}/api/feeds/example"`
5. Reset example feed to disabled when done testing

## Deployment

### Production Deployment
1. Create a new git tag (e.g., `v1.0.0`)
2. Push the tag to GitHub: `git push origin v1.0.0`
3. GitHub Actions will automatically:
   - Run tests
   - Build multi-architecture Docker images (linux/amd64, linux/arm64)
   - Push to GitHub Container Registry with multiple tags:
     - `ghcr.io/lysyi3m/rss-comb:1.0.0` (exact version)
     - `ghcr.io/lysyi3m/rss-comb:1.0` (major.minor)
     - `ghcr.io/lysyi3m/rss-comb:1` (major)
     - `ghcr.io/lysyi3m/rss-comb:latest` (always latest)
4. Pull the image using any of the generated tags
5. Configure environment variables for your deployment
6. Run the container with your feed configurations (SQLite database is embedded)

### Monitoring
- Health endpoint: `/health`
- Database connection monitoring
- Feed processing metrics in logs
- Docker healthcheck can be disabled in docker-compose.yml with `healthcheck: { disable: true }`

## Development Guidelines

### Code Organization
- Use repository pattern for database operations
- Implement proper error handling with context
- Follow Go naming conventions and documentation standards
- Use interfaces for testability

### Code Comments Policy
- Write self-explanatory code that minimizes the need for comments
- Remove obvious comments that simply restate what the code does
- Use comments to explain "why" something is done, not "what" is done
- Focus comments on complex logic, non-obvious decisions, and important context
- Keep comments up-to-date with code changes - outdated comments are worse than no comments
- Avoid empty or useless comments that don't add value

### Database Guidelines
- Use transactions for multi-table operations
- SQLite uses single connection (`MaxOpenConns(1)`) — no connection pool needed
- WAL mode enabled for concurrent reads during writes
- Monitor query performance with `EXPLAIN QUERY PLAN`

### Performance Considerations
- **Newest-Item Duplicate Check**: After parsing, checks if the newest item already exists in the database; if so, skips all item processing (avoids false positives from metadata-only changes in feed XML)
- Connection pooling for HTTP clients
- Worker pool for concurrent job processing
- Database queries optimized with proper indexing on content_hash (item level) and feed_id

### Docker Optimization
- **Pinned Base Images**: Base images use specific versions for reproducible builds
- **Layer Caching**: Dockerfile structured to maximize cache hits (dependencies before source code)
- **Multi-stage Build**: Separate build and runtime environments for smaller final images
- **Package Management**: Alpine packages use latest available versions for compatibility
- **Build Context**: .dockerignore excludes unnecessary files for faster context transfer
- **Cache Strategy**: Default builds use cache; use `--no-cache` only when explicitly needed

## Common Issues

### Feed Not Updating
- Check feed configuration `enabled: true`
- Verify `next_fetch` timestamp in database
- Check application logs for processing errors
- Validate feed URL accessibility

### Missing Items
- Review filter configuration for over-filtering
- Deduplication is always enabled and automatic
- Examine `is_filtered` and `is_duplicate` flags in database
- Verify `max_items` setting (limits RSS output, not database storage)

### Feed URL Changes
- Feed URLs can be updated directly in configuration files (`.yml`)
- System automatically detects URL changes on startup and logs them
- Look for "Feed URL updated" messages in logs
- No manual database updates required - changes are applied automatically
- Supports feeds with query parameters since routing uses feed IDs

## API Endpoints

### Public Endpoints

#### `GET /feeds/<name>`
- Returns RSS 2.0 feed output for the specified feed
- Generates RSS XML from database using `feed.ForType(typ).Build()`
- Includes feed metadata and visible (non-filtered) items
- Respects max_items setting from feed configuration
- Returns with headers: Content-Type, X-Feed-Items, X-Feed-Name, X-Last-Updated

#### `GET /health`
- Returns application health status and statistics
- Includes feed counts and processing metrics

#### `GET /media/<filename>`
- Serves downloaded media files (MP3 audio from YouTube videos)
- Static file serving via gin with Content-Type, range requests, and caching headers
- Files named by YouTube video ID (e.g., `Wrgx6STAaWo.mp3`)

### API Endpoints (require API key)

#### `POST /api/feeds/<name>/reload`
- Reloads the configuration file for the specified feed and re-applies filters to all items
- Processes synchronously and returns when complete (typically fast)
- Requires X-API-Key header or Authorization: Bearer token
