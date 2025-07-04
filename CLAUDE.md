# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

RSS Comb is a Go server application that acts as a proxy between existing RSS/Atom feeds and RSS reader applications. It provides feed normalization, deduplication, and content filtering capabilities through YAML-based configuration files.

## Development Environment

### Prerequisites
- Go 1.24+
- PostgreSQL 17+
- Redis 7+
- Docker & Docker Compose

### Common Commands

#### Development Setup
```bash
# Start development dependencies
make dev-up

# Stop development dependencies  
make dev-down

# Build the application
make build

# Run with development dependencies
make run

# Run database migrations
make migrate

# Run tests
make test
```

#### Docker Commands
```bash
# Start full stack in development
docker-compose up -d

# Production deployment
docker-compose -f docker-compose.prod.yml up -d

# Build production image
docker build -f docker/Dockerfile -t rss-comb:latest .
```

#### Database Operations
```bash
# Run migrations
migrate -path migrations -database "postgres://rss_user:rss_password@localhost:5432/rss_comb?sslmode=disable" up

# Create new migration
migrate create -ext sql -dir migrations -seq migration_name
```

## Architecture

### Core Components

1. **Configuration System** (`internal/config/`)
   - YAML-based feed configuration loading
   - Validation and default value handling
   - Hot-reload capability for configuration changes

2. **Feed Processing Engine** (`internal/feed/`)
   - HTTP feed fetching with timeout and retry logic
   - Content filtering based on configurable rules
   - Deduplication using content hashing

3. **Parser** (`internal/parser/`)
   - Universal RSS/Atom feed parsing using gofeed
   - Normalization of different feed formats
   - Content hash generation for deduplication

4. **Database Layer** (`internal/database/`)
   - PostgreSQL with UUID primary keys
   - Separate repositories for feeds and items
   - Optimized queries with proper indexing

5. **Caching Layer** (`internal/cache/`)
   - Redis-based caching for processed feeds
   - Configurable TTL per feed
   - Cache invalidation on feed updates

6. **Background Scheduler** (`internal/scheduler/`)
   - Worker pool for concurrent feed processing
   - Database-driven scheduling with next_fetch timestamps
   - Graceful shutdown handling

7. **HTTP API** (`internal/api/`)
   - RESTful endpoints for feed access
   - RSS 2.0 output generation
   - Cache headers and redirect handling

### Data Flow

1. Feed configurations loaded from `feeds/*.yaml`
2. Feeds registered in database with metadata
3. Background scheduler processes feeds based on refresh intervals
4. Items parsed, filtered, and deduplicated before storage
5. HTTP API serves processed feeds with caching

### Database Schema

**feeds table:**
- Stores feed metadata and processing status
- Tracks last_fetched, last_success, next_fetch timestamps
- Links to configuration files

**feed_items table:**
- Normalized item data with content hashing
- Filtering and deduplication flags
- JSONB storage for raw feed data
- Optimized indexes for common queries

## Configuration

### Feed Configuration Format (`feeds/*.yaml`)
```yaml
feed:
  url: "https://example.com/feed.xml"
  name: "Feed Name"

settings:
  enabled: true
  deduplication: true
  refresh_interval: 3600  # seconds
  max_items: 50
  timeout: 30            # seconds

filters:
  - field: "title"
    includes: ["keyword"]
    excludes: ["spam"]
```

### Environment Variables
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`
- `REDIS_ADDR`
- `FEEDS_DIR` (default: ./feeds)
- `PORT` (default: 8080)
- `CACHE_DURATION` (default: 300 seconds - global cache duration for all feeds)
- `USER_AGENT` (default: "RSS Comb/1.0" - global user agent for all feeds)

## Testing

### Unit Tests
```bash
# Run all tests
go test -v ./...

# Run tests with coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Integration Testing
- Test database migrations and schema
- Verify feed parsing for different formats
- Test filtering and deduplication logic
- API endpoint testing with real feeds

### Manual Testing
1. Create test feed configuration in `feeds/test.yaml`
2. Start services with `make run`
3. Monitor logs for feed processing
4. Test API endpoint: `curl "http://localhost:8080/feed?url=<feed-url>"`

## Deployment

### Production Deployment
1. Configure environment variables in `.env`
2. Run `./scripts/deploy.sh` for full deployment
3. Monitor logs: `docker-compose logs -f app`

### Health Monitoring
- Health check endpoint: `/health`
- Database connection monitoring
- Feed processing metrics in logs
- Redis cache hit/miss ratios

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
- Implement proper cache invalidation strategies

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

### Cache Issues
- Verify Redis connection and memory usage
- Check cache duration configuration
- Monitor cache hit/miss ratios
- Clear cache if needed: `redis-cli FLUSHDB`

## API Endpoints

### `GET /feed?url=<feed-url>`
- Returns processed RSS feed
- Supports caching with `X-Cache` header
- Redirects to original for unregistered feeds
- Returns empty feed template for not-yet-processed feeds

### `GET /health`
- Returns system health status
- Includes count of configured feeds
- Used for monitoring and load balancer health checks