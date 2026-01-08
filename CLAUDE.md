# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

RSS Comb is a Go server application that acts as a proxy between existing RSS/Atom feeds and RSS reader applications. It provides feed normalization, automatic deduplication, content filtering, and full iTunes podcast support through YAML-based configuration files.

The application features a clean, modular architecture with clear separation of concerns, dependency injection, and comprehensive testing. Recent architectural improvements have focused on eliminating code duplication, improving interface design, optimizing configuration management, implementing intelligent feed content hash optimization for significant performance improvements, and adding complete iTunes podcast RSS extension support.

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
- PostgreSQL 17+
- Docker & Docker Compose

### Common Commands

#### Development Setup
```bash
# Start development database
make dev-db-up

# Stop development database
make dev-db-down

# View development database logs
make dev-db-logs

# Build the application
make dev-build

# Run with development database (auto-starts DB with correct credentials)
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
│   ├── main.go              # Application entry point with ticker-based feed processing
│   ├── api/                 # HTTP handlers and server
│   ├── cfg/                 # Application configuration management
│   ├── database/            # Database connections, repositories, and embedded migrations
│   ├── feed/                # Feed processing logic and configuration management
│   └── services/            # Feed processing service functions
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
   - YAML-based feed configuration loading and validation (`config_cache.go`)
   - Feed names automatically derived from filenames (e.g., `habr.yml` → `habr`)
   - In-memory configuration caching with name-based indexing
   - No redundant storage of file paths - configuration files derived from names
   - Feed-specific settings and filter management

4. **Feed Processing System** (`app/feed/`)
   - **Parsing** (`parsing.go`): `feed.Parse()` - Universal RSS/Atom feed parsing using gofeed and normalization
   - **Generation** (`generator.go`): `feed.GenerateRSS()` - RSS 2.0 XML output generation for API responses
   - **Extraction** (`extraction.go`): `feed.Extract()` - Intelligent full-text content extraction using go-shiori/go-readability for feeds lacking <content:encoded>
   - **Filtering** (`filtering.go`): `feed.Filter()` - Configurable content filtering with include/exclude rules
   - **Performance Optimization**: Feed content hash comparison skips item processing when content unchanged
   - Simple stateless functions without struct wrappers

5. **Database Layer** (`app/database/`)
   - PostgreSQL with UUID primary keys
   - Separate repositories for feeds and items
   - Optimized queries with proper indexing
   - Embedded migrations with automatic execution on startup

6. **Feed Processing Services** (`app/services/`)
   - Simple service functions without task abstractions
   - `ProcessFeed`: Fetches, parses, filters, extracts content, and stores feed items
   - `RefilterFeed`: Re-applies filters to all items for a feed (used by reload endpoint)
   - Sequential processing via ticker in main.go
   - Database-driven scheduling with next_fetch timestamps

7. **HTTP API** (`app/api/`)
   - RESTful endpoints for feed access
   - RSS 2.0 output generation via feed system
   - Direct database queries for real-time data
   - Clean constructor pattern with consistent argument order

### Data Flow

1. **Application Initialization**: `cfg.Load()` loads application configuration, passed explicitly to all components
2. **Feed Configuration Loading**: YAML files loaded from `feeds/*.yml` and stored in database at startup
3. **Database Sync**: Configuration changes automatically registered in database with hash-based change detection via PostgreSQL UPSERT
4. **Feed Processing Loop**: Ticker (every 30 seconds) queries database for enabled feeds with `next_fetch` due
5. **Feed Processing**: `services.ProcessFeed()` fetches feed data, parses RSS/Atom, filters, and deduplicates items with content hash optimization
6. **Content Extraction**: During processing, content extractor fetches and extracts full article content when `extract_content: true`
7. **Storage**: Items stored with filter status and content hashes for deduplication
8. **RSS Feed Access**: `/feeds/:name` endpoint generates RSS 2.0 XML from database using `feed.GenerateRSS()` with visible items
9. **Configuration Reload**: `/api/feeds/:name/reload` API endpoint reloads YAML, updates database, and synchronously refilters existing items

### Database Schema

**feeds table:**
- Stores feed metadata and processing status
- Tracks last_fetched_at, next_fetch_at timestamps
- Stores content_hash for feed change detection optimization
- Stores configuration (settings JSONB, filters JSONB, is_enabled, config_hash)
- Uses `name` field to match with configuration files

**feed_items table:**
- Normalized item data with content hashing
- Filtering and deduplication flags
- RSS enclosure support (url, length, type)
- Optimized indexes for common queries

## Detailed Architecture

### Database Schema Details
- **feeds table**: id, name, feed_url, title, link, description, image_url, language, last_fetched_at, next_fetch_at, feed_published_at, feed_updated_at, content_hash, is_enabled, settings (JSONB), filters (JSONB), config_hash, itunes_author, itunes_image, itunes_explicit, itunes_owner_name, itunes_owner_email, created_at, updated_at
- **feed_items table**: id, feed_id, guid, link, title, description, content, published_at, updated_at, authors, categories, is_filtered, content_hash, enclosure_url, enclosure_length, enclosure_type, itunes_duration, itunes_episode, itunes_season, itunes_episode_type, itunes_image, created_at
- **Key relationships**: feeds.id → feed_items.feed_id (UUID primary keys)
- **Indexes**: feed_id, published_at, content_hash, is_enabled for optimized queries
- **Constraints**: Unique (feed_id, guid) for item deduplication within feeds
- **iTunes Podcast Support**: All iTunes fields are nullable and automatically extracted from podcast RSS feeds via gofeed library's built-in iTunes extension support

### Feed Processing Layer (`app/feed/`)
- `config_loader.go`: Pure functions for loading and validating YAML configuration files
- `parsing.go`: `feed.Parse()` - RSS/Atom parsing and content normalization using gofeed, extracts feed timestamps and iTunes podcast metadata
- `extraction.go`: `feed.Extract()` - Intelligent HTML content extraction using go-shiori/go-readability library
- `filtering.go`: `feed.Filter()` - Configurable content filtering with include/exclude rules
- `generator.go`: `feed.GenerateRSS()` - RSS 2.0 XML output generation with conditional iTunes podcast namespace (only when podcast data present) and tags for API responses
- `types.go`: Feed data structures and models, configuration types
- **Performance**: Intelligent content hash comparison skips processing when feed unchanged
- **Architecture**: Database is single source of truth at runtime, YAML files loaded only at startup/reload
- **Design**: Simple stateless functions instead of struct wrappers for better Go idioms
- **iTunes Support**: Automatic extraction and generation of iTunes podcast RSS extensions (author, image, explicit, owner, duration, episode, season, type); iTunes namespace added conditionally only when podcast data is present

### Repository Layer (`app/database/`)
- `connection.go`: PostgreSQL connection management with pooling
- `feed_repository.go`: Feed operations with PostgreSQL UPSERT for efficient configuration sync
- `item_repository.go`: Item operations implementing all repository interfaces
- `types.go`: Database model structs (Feed, FeedItem, Item)
- `interfaces.go`: Clean interface definitions with segregated responsibilities
- `migrations.go`: Embedded migration management with 7 migration files
- `migrations/`: SQL files (001-007) handling schema evolution, including iTunes podcast support (007)
- Interface segregation principle: separate interfaces for different responsibilities

### Feed Processing Services Layer (`app/services/`)
- `process_feed.go`: Feed processing service function
  - Fetches RSS/Atom feed data
  - Parses and normalizes feed content
  - Applies filters and deduplication
  - Extracts full content when configured
  - Stores items in database
- `refilter_feed.go`: Feed refiltering service function (used by reload endpoint)
  - Retrieves all items for a feed
  - Re-applies current filters
  - Updates filter status in database
- **Design**: Simple service functions, no task abstractions or queues
- **Processing**: Sequential execution via ticker in main.go
- **Failure Handling**: Errors logged, feed will be retried on next ticker interval based on next_fetch_at

### Application Configuration System (`app/cfg/`)
- `types.go`: Application configuration structs and interface definitions
- `loader.go`: Configuration loading with environment/command-line parsing
- `loader_test.go`: Comprehensive tests for configuration loading and interface compliance
- Configuration passed explicitly via dependency injection (no global state)
- Integrated version management and timezone configuration


## Environment Guide

### Development Environment
- **Application**: Running locally via `make run`
- **Database**: PostgreSQL in Docker container (localhost:5432)
- **Database URL**: `postgres://rss_comb_dev_user:rss_comb_dev_password@localhost:5432/rss_comb_dev?sslmode=disable`
- **Feed configs**: Local `feeds/*.yml` files
- **Logs**: Console output

### Production Environment
- **Application**: Running in Docker container from GitHub Container Registry
- **Database**: External PostgreSQL instance (configured via environment variables)
- **Database URL**: Configured via `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`
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
- **Development**: `make dev-db-up`, `make dev-run`, `make dev-db-down`
- **Testing**: `make dev-test`
- **Building**: `make dev-build`
- **Cleanup**: `make dev-clean`
- **Never use direct docker/go commands** - always use Makefile targets
- **Migrations**: Handled automatically by the application (no separate migrate commands needed)

### Cleanup Commands
```bash
# Stop development database
make dev-db-down

# Stop ALL RSS Comb processes (including stray development builds)
make dev-stop

# Complete development cleanup: processes + containers + build caches
make dev-clean
```

**IMPORTANT: Always clean up after development work to prevent side effects:**
- Stray development processes can interfere with production instances
- Multiple instances may cause port conflicts and database connection issues
- Development processes often use different credentials that can generate error logs
- Use `make dev-clean` before switching between development and production work

## Configuration

### Feed Configuration Format (`feeds/*.yml`)
```yaml
url: "https://example.com/feed.xml"
enabled: true

settings:
  refresh_interval: 1800  # 30 minutes (recommended)
  max_items: 50              # Limits RSS output items (all items stored in database)
  timeout: 30            # seconds
  extract_content: true     # Enable automatic content extraction

filters:
  - field: "title"
    includes: ["keyword"]
    excludes: ["spam"]
  - field: "authors"
    includes: ["john doe"]
    excludes: ["spammer"]
```

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
- `DB_HOST` (default: localhost) - Database host
- `DB_PORT` (default: 5432) - Database port
- `DB_USER` (default: rss_user) - Database user
- `DB_PASSWORD` (required) - Database password
- `DB_NAME` (default: rss_comb) - Database name

**Application Configuration:**
- `FEEDS_DIR` (default: ./feeds) - Directory containing feed configuration files
- `PORT` (default: 8080) - HTTP server port
- `BASE_URL` (optional) - Public base URL for the service (e.g., https://feeds.example.com). When set, RSS feeds use this URL for self-referencing links instead of localhost:port. Ideal for production deployments behind proxies.
- `SCHEDULER_INTERVAL` (default: 30) - Feed processing ticker interval in seconds
- `API_ACCESS_KEY` (optional) - API access key for authentication
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
   - Run tests with PostgreSQL
   - Build multi-architecture Docker images (linux/amd64, linux/arm64)
   - Push to GitHub Container Registry with multiple tags:
     - `ghcr.io/lysyi3m/rss-comb:1.0.0` (exact version)
     - `ghcr.io/lysyi3m/rss-comb:1.0` (major.minor)
     - `ghcr.io/lysyi3m/rss-comb:1` (major)
     - `ghcr.io/lysyi3m/rss-comb:latest` (always latest)
4. Pull the image using any of the generated tags
5. Configure environment variables for your deployment
6. Run the container with your PostgreSQL database and feed configurations

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
- Implement proper connection pooling
- Use prepared statements for repeated queries
- Monitor query performance with EXPLAIN

### Performance Considerations
- **Feed Content Hash Optimization**: Compares SHA-256 hash of raw feed content to skip item processing when unchanged (100x performance improvement for unchanged feeds)
- Works universally with all RSS/Atom feeds regardless of timestamp support
- Implement connection pooling for HTTP clients
- Use worker pools for concurrent processing
- Monitor memory usage in feed parsing
- Database queries optimized with proper indexing on content_hash and feed_id

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
- Generates RSS XML from database using feed.GenerateRSS()
- Includes feed metadata and visible (non-filtered) items
- Respects max_items setting from feed configuration
- Returns with headers: Content-Type, X-Feed-Items, X-Feed-Name, X-Last-Updated

#### `GET /health`
- Returns application health status and statistics
- Includes feed counts and processing metrics

### API Endpoints (require API key)

#### `POST /api/feeds/<name>/reload`
- Reloads the configuration file for the specified feed and re-applies filters to all items
- Processes synchronously and returns when complete (typically fast)
- Requires X-API-Key header or Authorization: Bearer token
