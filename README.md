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

Create YAML configuration files in the `feeds/` directory:

```yaml
feed:
  url: "https://example.com/feed.xml"
  name: "Example Feed"

settings:
  enabled: true
  deduplication: true
  refresh_interval: 3600  # seconds
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

- `GET /feed?url=<feed-url>` - Get processed RSS feed
- `GET /health` - Health check endpoint

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