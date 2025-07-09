# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

RSS Comb is a Go server application that acts as a proxy between existing RSS/Atom feeds and RSS reader applications. It provides feed normalization, deduplication, and content filtering capabilities through YAML-based configuration files.

## Development Environment

### Prerequisites
- Go 1.24+
- PostgreSQL 17+
- Docker & Docker Compose

### Common Commands

#### Development Setup
```bash
# Start database
make db-up

# Stop database
make db-down

# View database logs
make db-logs

# Build the application
make build

# Run with development dependencies
make run

# Database migrations run automatically on startup
# To disable: DISABLE_MIGRATE=true make run

# Run tests
make test
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
# Database migrations are now handled automatically by the application on startup
# To disable auto-migration, use --disable-migrate flag or set DISABLE_MIGRATE=true

# Create new migration files in app/database/migrations/
# Follow the naming convention: NNN_description.up.sql and NNN_description.down.sql
```

## Project Structure

```
rss-comb/
├── Dockerfile                 # Production container build
├── Makefile                  # Development commands
├── app/                      # Main application code
│   ├── main.go              # Application entry point
│   ├── api/                 # HTTP handlers and server
│   ├── config/              # Configuration loading
│   ├── database/            # Database connections, repositories, and embedded migrations
│   ├── feed/                # Feed processing logic
│   ├── parser/              # RSS/Atom parsing
│   └── scheduler/           # Background job scheduling
├── feeds/                    # Feed configuration files (*.yml)
├── docker-compose.yml       # Development database service
├── .github/workflows/       # CI/CD automation
└── go.mod                   # Go module definition
```

## Architecture

### Core Components

1. **Main Application** (`app/main.go`)
   - Application entry point and configuration loading
   - go-flags based environment variable and command-line flag parsing
   - Server initialization and graceful shutdown handling

2. **Configuration System** (`app/config/`)
   - YAML-based feed configuration loading
   - Validation and default value handling
   - Hot-reload capability for configuration changes

3. **Feed Processing Engine** (`app/feed/`)
   - HTTP feed fetching with timeout and retry logic
   - Content filtering based on configurable rules
   - Deduplication using content hashing

4. **Parser** (`app/parser/`)
   - Universal RSS/Atom feed parsing using gofeed
   - Normalization of different feed formats
   - Content hash generation for deduplication

5. **Database Layer** (`app/database/`)
   - PostgreSQL with UUID primary keys
   - Separate repositories for feeds and items
   - Optimized queries with proper indexing
   - Embedded migrations with automatic execution on startup

6. **Background Scheduler** (`app/scheduler/`)
   - Worker pool for concurrent feed processing
   - Database-driven scheduling with next_fetch timestamps
   - Graceful shutdown handling

7. **HTTP API** (`app/api/`)
   - RESTful endpoints for feed access
   - RSS 2.0 output generation
   - Direct database queries for real-time data

### Data Flow

1. Feed configurations loaded from `feeds/*.yml`
2. Feeds registered in database with metadata
3. Background scheduler processes feeds based on refresh intervals
4. Items parsed, filtered, and deduplicated before storage
5. HTTP API serves processed feeds directly from database

### Database Schema

**feeds table:**
- Stores feed metadata and processing status
- Tracks last_fetched, last_success, next_fetch timestamps
- Links to configuration files

**feed_items table:**
- Normalized item data with content hashing
- Filtering and deduplication flags
- Optimized indexes for common queries

## Detailed Architecture

### Database Schema Details
- **feeds table**: metadata, processing status, timestamps
- **feed_items table**: normalized items with content hashing
- Key relationships: feeds.id → feed_items.feed_id
- Indexes: feed_id, published_at, content_hash

### Repository Layer (`app/database/`)
- `connection.go`: PostgreSQL connection management
- `feed_repository.go`: CRUD operations for feeds
- `item_repository.go`: CRUD operations for feed items
- `interfaces.go`: Database interface definitions
- `models.go`: Database model structs
- `migrations.go`: Embedded migration management
- `migrations/`: SQL migration files

### Configuration Loading (`app/config/`)
- `loader.go`: Configuration loading logic
- `loader_test.go`: Tests for configuration loading
- `types.go`: Configuration structs and types
- Watches `feeds/*.yml` files for changes

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
- **Feed configs**: Mounted `feeds/*.yml` files
- **Deployment**: Automated via GitHub Actions on release tags

### How to verify "app is running"
- **Dev**: `curl localhost:8080/stats` returns 200
- **Prod**: `curl localhost:8080/stats` returns 200
- **Database**: `make test` passes, or manual query works

## Work Verification Process

### After making changes:
1. **Build & Test**: `make build && make test`
2. **Local verification**: `make run` then test endpoints
3. **Docker verification**: `docker build -f Dockerfile -t rss-comb:latest .` then test locally
4. **Commit changes**: Only when explicitly requested
5. **Cleanup**: `make clean`

### All operations should use Makefile:
- **Development**: `make db-up`, `make run`, `make db-down`
- **Testing**: `make test`
- **Building**: `make build`
- **Cleanup**: `make clean`
- **Never use direct docker/go commands** - always use Makefile targets
- **Migrations**: Handled automatically by the application (no separate migrate commands needed)

### Cleanup Commands
```bash
# Clean project-specific resources only (safe)
make clean

# Stop development database
make db-down
```

## Configuration

### Feed Configuration Format (`feeds/*.yml`)
```yaml
feed:
  id: "feed-identifier"     # Unique identifier for URL routing
  url: "https://example.com/feed.xml"
  title: "Feed Title"

settings:
  enabled: true
  deduplication: true
  refresh_interval: 1800  # 30 minutes (recommended)
  max_items: 50
  timeout: 30            # seconds

filters:
  - field: "title"
    includes: ["keyword"]
    excludes: ["spam"]
```

**Important Notes:**
- The `id` field is required and must be unique across all feed configurations
- Feed IDs are used in the URL schema: `/feeds/<id>`
- IDs should be URL-safe (alphanumeric, hyphens, underscores)
- Feed URLs can be updated in configuration files at any time - the system automatically detects and applies URL changes
- Feeds with query parameters are fully supported since routing is based on feed IDs, not URLs

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
- `WORKER_COUNT` (default: 5) - Number of background workers for feed processing
- `SCHEDULER_INTERVAL` (default: 30) - Scheduler interval in seconds
- `API_ACCESS_KEY` (optional) - API access key for authentication
- `USER_AGENT` (default: "RSS Comb/1.0") - User agent string for HTTP requests
- `TZ` (default: "UTC") - Timezone for display timestamps in API responses and RSS feeds (e.g., UTC, America/New_York, Europe/London). Database operations always use UTC for consistency.
- `DISABLE_MIGRATE` (default: false) - Disable automatic database migrations on startup

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
go test -v ./app/config
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
4. Test API endpoint: `curl "http://localhost:${PORT:-8080}/feeds/example"`
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
- Statistics endpoint: `/stats`
- Database connection monitoring
- Feed processing metrics in logs

## Development Guidelines

### Code Organization
- Use repository pattern for database operations
- Implement proper error handling with context
- Follow Go naming conventions and documentation standards
- Use interfaces for testability

### Database Guidelines
- Use transactions for multi-table operations
- Implement proper connection pooling
- Use prepared statements for repeated queries
- Monitor query performance with EXPLAIN

### Performance Considerations
- Implement connection pooling for HTTP clients
- Use worker pools for concurrent processing
- Monitor memory usage in feed parsing

### Docker Optimization
- **Pinned Dependencies**: All base images and packages use specific versions for reproducible builds
- **Layer Caching**: Dockerfile structured to maximize cache hits (dependencies before source code)
- **Multi-stage Build**: Separate build and runtime environments for smaller final images
- **Version Pinning**: All Alpine packages and Go tools use exact versions
- **Build Context**: .dockerignore excludes unnecessary files for faster context transfer
- **Cache Strategy**: Default builds use cache; use `--no-cache` only when explicitly needed

## Common Issues

### Feed Not Updating
- Check feed configuration `enabled: true`
- Verify `next_fetch` timestamp in database
- Check scheduler logs for processing errors
- Validate feed URL accessibility

### Missing Items
- Review filter configuration for over-filtering
- Check deduplication settings
- Examine `is_filtered` and `is_duplicate` flags in database
- Verify `max_items` setting

### Feed URL Changes
- Feed URLs can be updated directly in configuration files (`.yml`)
- System automatically detects URL changes on startup and logs them
- Look for "Feed URL updated" messages in logs
- No manual database updates required - changes are applied automatically
- Supports feeds with query parameters since routing uses feed IDs

## API Endpoints

### Public Endpoints

#### `GET /feeds/<id>`
- Returns processed RSS feed by feed ID
- Returns HTTP 404 for unknown feed IDs
- Returns empty feed template for not-yet-processed feeds

#### `GET /stats`
- Returns application statistics
- Includes feed counts and processing metrics

### API Endpoints (require API key)

#### `GET /api/feeds`
- Lists all configured feeds
- Returns feed configuration and status information
- Requires X-API-Key header or Authorization: Bearer token

#### `GET /api/feeds/<id>/details`
- Returns detailed information about a specific feed by ID
- Includes configuration, database status, and item statistics
- Requires X-API-Key header or Authorization: Bearer token

#### `POST /api/feeds/<id>/refilter`
- Re-applies filters to all items for a specific feed by ID
- Returns updated item counts and statistics
- Requires X-API-Key header or Authorization: Bearer token