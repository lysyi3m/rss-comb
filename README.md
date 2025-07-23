# RSS Comb

[![CI/CD](https://github.com/lysyi3m/rss-comb/actions/workflows/ci.yml/badge.svg)](https://github.com/lysyi3m/rss-comb/actions/workflows/ci.yml)
[![Docker](https://img.shields.io/badge/docker-ghcr.io%2Flysyi3m%2Frss--comb-blue)](https://github.com/lysyi3m/rss-comb/pkgs/container/rss-comb)
[![Go Version](https://img.shields.io/badge/go-1.24+-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

RSS Comb is a high-performance Go server application that acts as a proxy between existing RSS/Atom feeds and RSS reader applications. It provides feed normalization, automatic deduplication, flexible filtering and content extraction capabilities through YAML-based configuration files.

## Features

- **Feed Normalization**: Converts RSS 1.0, RSS 2.0, and Atom feeds to standardized RSS 2.0 format
- **Automatic Deduplication**: Automatically eliminates duplicate items based on content hashing
- **Content Extraction**: Intelligent full-text content extraction using [go-shiori/go-readability](https://github.com/go-shiori/go-readability)
- **Flexible Filtering**: Configurable content filtering using include/exclude rules
- **Background Processing**: Automated feed updates with configurable refresh intervals
- **Task Scheduling**: FIFO task queue with retry logic and background workers
- **Statistics & Monitoring**: Built-in stats endpoint and comprehensive logging
- **API Authentication**: Secure API endpoints with configurable access keys
- **Docker Ready**: Fully containerized with optimized multi-stage builds

## Quick Start

### Using Docker (Recommended)

1. **Pull the image**:
   ```bash
   # Latest version
   docker pull ghcr.io/lysyi3m/rss-comb:latest

   # Specific version (recommended for production)
   docker pull ghcr.io/lysyi3m/rss-comb:1.0.0
   ```

2. **Run with Docker Compose**:
   ```bash
   # Create docker-compose.yml
   cat > docker-compose.yml << EOF
   version: '3.8'
   services:
     rss-comb:
       image: ghcr.io/lysyi3m/rss-comb:latest
       ports:
         - "8080:8080"
       environment:
         - DB_HOST=db
         - DB_USER=rss_user
         - DB_PASSWORD=rss_password
         - DB_NAME=rss_comb
       volumes:
         - ./feeds:/app/feeds:ro
       depends_on:
         - db

     db:
       image: postgres:15-alpine
       environment:
         - POSTGRES_DB=rss_comb
         - POSTGRES_USER=rss_user
         - POSTGRES_PASSWORD=rss_password
       volumes:
         - postgres_data:/var/lib/postgresql/data

   volumes:
     postgres_data:
   EOF

   # Start services
   docker-compose up -d
   ```

3. **Access your feeds**:
   - Health check: `http://localhost:8080/health`
   - Feed example: `http://localhost:8080/feeds/your-feed-name` (based on YAML filename)

### Development Setup

1. **Prerequisites**:
   - Go 1.24+
   - PostgreSQL 15+
   - Docker & Docker Compose

2. **Clone and setup**:
   ```bash
   git clone https://github.com/lysyi3m/rss-comb.git
   cd rss-comb

   # Start development database
   make dev-db-up

   # Run application with development database (auto-starts DB with correct credentials)
   make dev-run
   ```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | localhost | Database host |
| `DB_PORT` | 5432 | Database port |
| `DB_USER` | rss_user | Database username |
| `DB_PASSWORD` | *required* | Database password |
| `DB_NAME` | rss_comb | Database name |
| `FEEDS_DIR` | ./feeds | Directory containing feed configuration files |
| `PORT` | 8080 | HTTP server port |
| `BASE_URL` | *empty* | Base URL for RSS self-referencing links |
| `WORKER_COUNT` | 5 | Number of background workers |
| `SCHEDULER_INTERVAL` | 30 | Scheduler interval in seconds |
| `API_ACCESS_KEY` | *optional* | API access key for authentication |
| `USER_AGENT` | "RSS Comb/1.0" | User agent for HTTP requests |
| `TZ` | UTC | Timezone for timestamps |
| `DEBUG` | false | Enable debug logging |

### Feed Configuration

Create YAML configuration files in the `feeds/` directory. Feed names are derived from filenames (e.g., `tech-news.yml` creates feed name `tech-news`):

```yaml
url: "https://example.com/feed.xml"

settings:
  enabled: true
  refresh_interval: 1800       # 30 minutes
  max_items: 50                # Limits RSS output items (all items stored in database)
  timeout: 30                  # seconds
  extract_content: false       # Enable automatic content extraction

filters:
  - field: "title"
    includes:
      - "technology"
      - "programming"
    excludes:
      - "advertisement"
      - "sponsored"
  - field: "description"
    excludes:
      - "clickbait"
  - field: "authors"
    includes:
      - "john doe"
    excludes:
      - "spammer"
```

**Key Configuration Notes:**
- Feed names are derived from filenames (remove `.yml` extension)
- Feed names are used in URLs: `/feeds/<name>`
- Feed names must be unique and URL-safe
- Feed titles are automatically extracted from the RSS/Atom source (no manual configuration needed)
- `max_items` limits RSS output only - all feed items are stored in database
- `extract_content: true` enables automatic full-text content extraction from article URLs
- Deduplication is automatic and always enabled
- Filters support `title`, `description`, `content`, `authors`, `link`, and `categories` fields

## API Endpoints

### Public Endpoints

- **`GET /feeds/<name>`** - Get processed RSS feed by feed name (derived from filename)
- **`GET /health`** - Application health check and statistics

### Authenticated Endpoints

Require `X-API-Key` header or `Authorization: Bearer <token>`:

- **`GET /api/feeds`** - List all configured feeds with status
- **`GET /api/feeds/<name>/details`** - Detailed feed information and statistics
- **`POST /api/feeds/<name>/reload`** - Reload configuration and re-apply filters to all feed items

### Example API Usage

```bash
# Get health check and statistics
curl http://localhost:8080/health

# Access processed feed (name derived from filename tech-news.yml)
curl http://localhost:8080/feeds/tech-news

# List all feeds (with API key)
curl -H "X-API-Key: your-api-key" http://localhost:8080/api/feeds

# Reload configuration and re-apply filters
curl -X POST -H "X-API-Key: your-api-key" http://localhost:8080/api/feeds/tech-news/reload
```

## Development

### Available Commands

```bash
# Development database management
make dev-db-up      # Start PostgreSQL development database
make dev-db-down    # Stop development database
make dev-db-logs    # View development database logs

# Development
make dev-build      # Build the application
make dev-run        # Run with development database (auto-starts DB with correct credentials)
make dev-test       # Run all tests

# Cleanup (important to prevent conflicts)
make dev-stop       # Stop development processes
make dev-clean      # Complete development cleanup: processes + containers + caches
```

### Testing

```bash
# Run all tests
make dev-test

# Run tests with coverage
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test -v ./app/cfg
```

### Database Migrations

Database migrations are embedded in the application binary and run automatically on startup.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request. For major changes, please open an issue first to discuss what you would like to change.

## Links

- [GitHub Repository](https://github.com/lysyi3m/rss-comb)
- [Docker Images](https://github.com/lysyi3m/rss-comb/pkgs/container/rss-comb)
- [Issues & Bug Reports](https://github.com/lysyi3m/rss-comb/issues)
