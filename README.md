# RSS Comb

RSS Comb is a server application that acts as a proxy between existing RSS/Atom feeds and RSS reader applications. It provides feed normalization, deduplication, and content filtering capabilities through YAML-based configuration files.

## Features

- **Feed Normalization**: Converts RSS 1.0, RSS 2.0, and Atom feeds to a standardized format
- **Deduplication**: Eliminates duplicate items based on content hashing
- **Content Filtering**: Flexible filtering system using configurable rules
- **Background Processing**: Automated feed updates with configurable intervals

## Quick Start

1. **Start development dependencies**:
   ```bash
   make dev-up
   ```

2. **Build the application**:
   ```bash
   make build
   ```

3. **Run the application**:
   ```bash
   make run
   ```

## Configuration

### Environment Variables

Configure the application using environment variables or command-line flags:

```bash
# Database configuration
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=rss_user
export DB_PASSWORD=your_password
export DB_NAME=rss_comb

# Application configuration
export FEEDS_DIR=./feeds
export PORT=8080
export WORKER_COUNT=5
export SCHEDULER_INTERVAL=30
export API_ACCESS_KEY=your_api_key
export USER_AGENT="RSS Comb/1.0"
```

See all available options with: `go run app/main.go --help`

### Feed Configuration

Create YML configuration files in the `feeds/` directory:

```yaml
feed:
  id: "example"           # Unique identifier for URL routing
  url: "https://example.com/feed.xml"
  name: "Example Feed"

settings:
  enabled: true
  deduplication: true
  refresh_interval: 1800  # 30 minutes (recommended)
  max_items: 50
  timeout: 30            # seconds

filters:
  - field: "title"
    includes:
      - "technology"
    excludes:
      - "advertisement"
```

## API Endpoints

### Public Endpoints
- `GET /feeds/<id>` - Get processed RSS feed by feed ID
- `GET /health` - Health check endpoint
- `GET /stats` - Application statistics

### API Endpoints (require API key)
- `GET /api/feeds` - List all configured feeds
- `GET /api/feeds/<id>/details` - Get detailed information about a specific feed
- `POST /api/feeds/<id>/refilter` - Re-apply filters to a specific feed

## Development

### Prerequisites

- Go 1.24+
- PostgreSQL 17+
- Docker & Docker Compose

### Commands

```bash
# Start development dependencies
make dev-up

# Stop development dependencies
make dev-down

# Run tests
make test

# Build application
make build

# Run application
make run

# Run database migrations
make migrate
```

## License

MIT License